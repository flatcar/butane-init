package transpile

import (
	"fmt"
	"path"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/coreos/go-systemd/v22/unit"
	base "github.com/coreos/ignition/v2/butane/base/v0_5"
	"github.com/goccy/go-yaml/ast"
)

// cloud-init 26.2 fills omitted fields from these mount_default_fields.
var mountDefaults = []string{"", "", "auto", "defaults,nofail,x-systemd.after=cloud-init-network.service,_netdev", "0", "2"}

var (
	// cloud-init passes tags through to fstab unchanged. The character set
	// needs no escaping in udev symlinks, unit names, or unit files.
	mountTag = regexp.MustCompile(`^(LABEL|UUID)=([A-Za-z0-9._-]+)$`)
	// Partition nodes are also kept verbatim. cloud-init replaces a whole disk,
	// or a symlink to one, with its first partition if that exists at boot.
	mountPartition = regexp.MustCompile(`^/dev/(x?[shv]d[a-z]+[1-9][0-9]*|(nvme[0-9]+n[0-9]+|mmcblk[0-9]+)p[1-9][0-9]*)$`)
	mountPoint     = regexp.MustCompile(`^(/[A-Za-z0-9._-]+)+$`)
	mountOption    = regexp.MustCompile(`^[A-Za-z0-9_.:=/+@-]+$`)
	mountAfter     = regexp.MustCompile(`^x-systemd\.after=([A-Za-z0-9:_.@-]+\.(service|socket|target|mount|device|path|timer|swap))$`)
)

// Flatcar symlinks, read-only, virtual, and Flatcar-managed mount points.
var reservedMountPoints = []string{
	"/bin", "/boot", "/dev", "/etc", "/lib", "/lib64", "/media", "/oem", "/proc",
	"/run", "/sbin", "/sys", "/tmp", "/usr", "/var/lock", "/var/run",
}

// parseMounts converts the Cluster API mounts subset into systemd mount units,
// equivalent to the units systemd-fstab-generator creates from cloud-init's
// fstab lines. provisioned lists paths that cloud-init creates after mounting.
func parseMounts(document map[string]any, file *ast.File, provisioned []string) ([]base.Unit, ValidationErrors) {
	raw, exists := document["mounts"]
	if !exists {
		return nil, nil
	}
	items, ok := raw.([]any)
	if !ok {
		return nil, ValidationErrors{problem(file, "mounts", "must be a list")}
	}
	var units []base.Unit
	var problems ValidationErrors
	names := map[string]bool{}
	for i, item := range items {
		entryPath := fmt.Sprintf("mounts[%d]", i)
		fields, ok := item.([]any)
		if !ok || len(fields) < 2 || len(fields) > 6 {
			problems = append(problems, problem(file, entryPath, "must be a list of 2 to 6 fields: device, mount point, type, options, dump, and pass"))
			continue
		}
		values := append([]string(nil), mountDefaults...)
		valid := true
		for j, field := range fields {
			value, isString := field.(string)
			// Cluster API emits dump and pass unquoted, so YAML decodes them as integers.
			if number, isNumber := field.(uint64); isNumber && j >= 4 {
				value, isString = strconv.FormatUint(number, 10), true
			}
			if !isString || value == "" {
				problems = append(problems, problem(file, fmt.Sprintf("%s[%d]", entryPath, j), "must be a non-empty string"))
				valid = false
				continue
			}
			values[j] = value
		}
		if !valid {
			continue
		}
		fieldProblem := func(j int, message string) {
			problems = append(problems, problem(file, fmt.Sprintf("%s[%d]", entryPath, j), message))
		}

		device, where, format, options := values[0], values[1], values[2], values[3]
		what := device
		if tag := mountTag.FindStringSubmatch(device); tag != nil {
			what = "/dev/disk/by-" + strings.ToLower(tag[1]) + "/" + tag[2]
		} else if strings.HasPrefix(device, "LABEL=") || strings.HasPrefix(device, "UUID=") {
			fieldProblem(0, "label or UUID must contain only ASCII letters, digits, '.', '_', and '-'")
		} else if !mountPartition.MatchString(device) {
			fieldProblem(0, "must be LABEL=, UUID=, or a partition such as /dev/sdb1 or /dev/nvme1n1p1: cloud-init resolves aliases and short names at boot, and replaces a whole disk with its first partition when one exists")
		}

		name := unit.UnitNamePathEscape(where) + ".mount"
		if !mountPoint.MatchString(where) || path.Clean(where) != where {
			fieldProblem(1, "must be a clean absolute path below / of ASCII letters, digits, '.', '_', and '-'")
		} else if where == "/var" || slices.ContainsFunc(reservedMountPoints, func(reserved string) bool { return beneath(where, reserved) }) {
			fieldProblem(1, "unsupported Flatcar system, virtual, or symlinked mount point")
		} else if hidden := slices.IndexFunc(provisioned, func(other string) bool { return beneath(other, where) }); hidden != -1 {
			fieldProblem(1, fmt.Sprintf("would hide %q, which cloud-init creates after mounting", provisioned[hidden]))
		} else if len(name) > 255 {
			fieldProblem(1, "is too long for a systemd unit name")
		} else if names[name] {
			fieldProblem(1, "duplicates an earlier mount point")
		}
		names[name] = true

		switch format {
		case "auto", "ext4", "xfs", "btrfs", "vfat":
		case "swap":
			fieldProblem(2, "swap entries are unsupported")
		default:
			fieldProblem(2, "must be auto, ext4, xfs, btrfs, or vfat")
		}

		var after []string
		nofail, network := false, false
		for option := range strings.SplitSeq(options, ",") {
			dependency := mountAfter.FindStringSubmatch(option)
			switch {
			case !mountOption.MatchString(option):
				fieldProblem(3, "must be comma-separated mount options of ASCII letters, digits, and _.:=/+@-")
			case option == "nofail", option == "fail":
				// As in systemd, the last of nofail and fail wins.
				nofail = option == "nofail"
			case option == "_netdev":
				network = true
			case option == "noauto":
				fieldProblem(3, "noauto is unsupported")
			case dependency != nil:
				after = append(after, dependency[1])
			case strings.HasPrefix(option, "x-systemd."), strings.HasPrefix(option, "x-initrd."), strings.HasPrefix(option, "comment=systemd."):
				fieldProblem(3, fmt.Sprintf("systemd fstab option %q is unsupported", option))
			}
		}

		// dump(8) is not part of Flatcar, and systemd ignores the field.
		if values[4] != "0" {
			fieldProblem(4, "must be 0")
		}
		pass := values[5]
		if pass != "0" && pass != "1" && pass != "2" {
			fieldProblem(5, "must be 0, 1, or 2")
		}

		var contents strings.Builder
		// cloud-init mounts before runcmd starts; nofail mounts are not ordered
		// before local-fs.target or remote-fs.target, so order runcmd explicitly.
		contents.WriteString("[Unit]\nBefore=butane-init-runcmd.service\n")
		for _, dependency := range after {
			fmt.Fprintf(&contents, "After=%s\n", dependency)
		}
		if pass != "0" {
			fsck := "systemd-fsck@" + unit.UnitNamePathEscape(what) + ".service"
			fmt.Fprintf(&contents, "Requires=%s\nAfter=%s\n", fsck, fsck)
		}
		fmt.Fprintf(&contents, "\n[Mount]\nWhat=%s\nWhere=%s\n", what, where)
		if format != "auto" {
			fmt.Fprintf(&contents, "Type=%s\n", format)
		}
		// systemd reads nofail and _netdev from Options= itself.
		if options != "defaults" {
			fmt.Fprintf(&contents, "Options=%s\n", options)
		}
		install, target := "RequiredBy", "local-fs.target"
		if nofail {
			install = "WantedBy"
		}
		if network {
			target = "remote-fs.target"
		}
		fmt.Fprintf(&contents, "\n[Install]\n%s=%s\n", install, target)
		text := contents.String()
		enabled := true
		units = append(units, base.Unit{Name: name, Enabled: &enabled, Contents: &text})
	}
	if len(problems) > 0 {
		return nil, problems
	}
	return units, nil
}

// beneath reports whether target is root or lies beneath it.
func beneath(target, root string) bool {
	return target == root || strings.HasPrefix(target, root+"/")
}
