package transpile

import (
	"fmt"
	"path"
	"regexp"
	"strconv"
	"strings"
	"unicode"

	base "github.com/coreos/ignition/v2/butane/base/v0_5"
	"github.com/goccy/go-yaml/ast"
)

// cloud-init appends a partition number to the resolved device node. Only a
// top-level kernel name lets us derive that path without resolving symlinks.
var kernelDevice = regexp.MustCompile(`^/dev/[a-z][a-z0-9]*$`)

// maxLabelLength mirrors Ignition's per-format label limits, in bytes.
var maxLabelLength = map[string]int{"ext4": 16, "xfs": 12, "btrfs": 256, "vfat": 11, "swap": 15}

// parseFSSetup converts the Cluster API fs_setup subset into Butane filesystems.
func parseFSSetup(document map[string]any, file *ast.File) ([]base.Filesystem, ValidationErrors) {
	// Cluster API emits a bare fs_setup: when disk_setup has only partitions.
	raw := document["fs_setup"]
	if raw == nil {
		return nil, nil
	}
	items, ok := raw.([]any)
	if !ok {
		return nil, ValidationErrors{problem(file, "fs_setup", "must be a list")}
	}
	var filesystems []base.Filesystem
	var problems ValidationErrors
	devices := map[string]bool{}
	for i, item := range items {
		fieldPath := fmt.Sprintf("fs_setup[%d]", i)
		fields, ok := item.(map[string]any)
		if !ok {
			problems = append(problems, problem(file, fieldPath, "must be an object"))
			continue
		}
		for _, key := range sortedKeys(fields) {
			switch key {
			case "device", "filesystem", "label", "partition", "overwrite", "extra_opts":
			default:
				problems = append(problems, problem(file, fieldPath+"."+key, "unsupported field"))
			}
		}
		device := requiredString(fields, "device", fieldPath, file, &problems)
		if device != "" && (!strings.HasPrefix(device, "/dev/") || path.Clean(device) != device || strings.ContainsFunc(device, unicode.IsControl)) {
			problems = append(problems, problem(file, fieldPath+".device", "must be a clean absolute path beneath /dev; device aliases are unsupported"))
		}
		format := requiredString(fields, "filesystem", fieldPath, file, &problems)
		switch format {
		case "", "ext4", "xfs", "btrfs", "vfat", "swap":
		default:
			problems = append(problems, problem(file, fieldPath+".filesystem", "must be one of ext4, xfs, btrfs, vfat, or swap"))
		}
		// Cluster API emits a bare label: key when no label is configured.
		var label *string
		if raw := fields["label"]; raw != nil {
			value, ok := raw.(string)
			if !ok || value == "" {
				problems = append(problems, problem(file, fieldPath+".label", "must be a non-empty string or null"))
			} else {
				label = &value
				if limit := maxLabelLength[format]; limit > 0 && len(value) > limit {
					problems = append(problems, problem(file, fieldPath+".label", fmt.Sprintf("must be at most %d bytes for %s", limit, format)))
				}
			}
		}
		if raw, exists := fields["partition"]; !exists {
			problems = append(problems, problem(file, fieldPath+".partition", "is required: cloud-init's default \"any\" selects a device at boot time; use \"none\" or a partition number"))
		} else if number, ok := partitionNumber(raw); ok {
			if !kernelDevice.MatchString(device) {
				problems = append(problems, problem(file, fieldPath+".device", "must be a kernel device name such as /dev/sdb when partition is a number"))
			} else {
				// Like cloud-init, use the "p" separator after a trailing digit (nvme0n1p1).
				if unicode.IsDigit(rune(device[len(device)-1])) {
					device += "p"
				}
				device += strconv.Itoa(number)
			}
		} else if value, ok := raw.(string); !ok || !strings.EqualFold(value, "none") {
			problems = append(problems, problem(file, fieldPath+".partition", "must be \"none\" or a positive partition number; \"auto\" and \"any\" are unsupported"))
		}
		if device != "" {
			if devices[device] {
				problems = append(problems, problem(file, fieldPath+".device", fmt.Sprintf("duplicates filesystem device %q", device)))
			}
			devices[device] = true
		}
		overwrite := optionalBool(fields, "overwrite", fieldPath, file, &problems)
		var options []string
		if raw, exists := fields["extra_opts"]; exists {
			opts, ok := raw.([]any)
			if !ok {
				problems = append(problems, problem(file, fieldPath+".extra_opts", "must be a list"))
			}
			for j, opt := range opts {
				value, ok := opt.(string)
				if !ok || value == "" {
					problems = append(problems, problem(file, fmt.Sprintf("%s.extra_opts[%d]", fieldPath, j), "must be a non-empty string"))
					continue
				}
				options = append(options, value)
			}
		}
		// Without overwrite, Ignition reuses a matching filesystem, creates one
		// on a blank device, and fails instead of destroying any other filesystem.
		wipe := overwrite != nil && *overwrite
		filesystems = append(filesystems, base.Filesystem{
			Device: device, Format: &format, Label: label, Options: options, WipeFilesystem: &wipe,
		})
	}
	if len(problems) > 0 {
		return nil, problems
	}
	return filesystems, nil
}

// partitionNumber accepts the YAML integer or digit string Cluster API emits.
func partitionNumber(raw any) (int, bool) {
	var text string
	switch value := raw.(type) {
	case uint64:
		text = strconv.FormatUint(value, 10)
	case int64:
		text = strconv.FormatInt(value, 10)
	case string:
		text = value
	default:
		return 0, false
	}
	number, err := strconv.Atoi(text)
	if err != nil || number < 1 || strconv.Itoa(number) != text {
		return 0, false
	}
	return number, true
}
