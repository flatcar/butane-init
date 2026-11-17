package transpile

import (
	"path"
	"strings"
	"unicode"

	base "github.com/coreos/ignition/v2/butane/base/v0_5"
	"github.com/goccy/go-yaml/ast"
)

// parseDiskSetup converts the Cluster API disk_setup subset into Butane disks.
func parseDiskSetup(document map[string]any, file *ast.File) ([]base.Disk, ValidationErrors) {
	raw, exists := document["disk_setup"]
	// Cluster API emits a bare disk_setup: key when only fs_setup is configured.
	if !exists || raw == nil {
		return nil, nil
	}
	devices, ok := raw.(map[string]any)
	if !ok {
		return nil, ValidationErrors{problem(file, "disk_setup", "must be an object mapping devices to definitions")}
	}
	var disks []base.Disk
	var problems ValidationErrors
	for _, device := range sortedKeys(devices) {
		fieldPath := "disk_setup.'" + strings.ReplaceAll(device, "'", `\'`) + "'"
		if !strings.HasPrefix(device, "/dev/") || path.Clean(device) != device || strings.ContainsFunc(device, unicode.IsControl) {
			problems = append(problems, problem(file, fieldPath, "device must be a clean absolute path beneath /dev; device aliases are unsupported"))
		}
		fields, ok := devices[device].(map[string]any)
		if !ok {
			problems = append(problems, problem(file, fieldPath, "must be an object"))
			continue
		}
		for _, key := range sortedKeys(fields) {
			switch key {
			case "table_type", "layout", "overwrite":
			default:
				problems = append(problems, problem(file, fieldPath+"."+key, "unsupported field"))
			}
		}
		layout := optionalBool(fields, "layout", fieldPath, file, &problems)
		if _, exists := fields["layout"]; !exists {
			problems = append(problems, problem(file, fieldPath+".layout", "is required"))
		}
		// overwrite is validated but does not change the output; see README.md.
		optionalBool(fields, "overwrite", fieldPath, file, &problems)
		tableType := "mbr" // cloud-init's default
		if value := optionalString(fields, "table_type", fieldPath, file, &problems); value != nil {
			tableType = *value
			if tableType != "mbr" && tableType != "gpt" {
				problems = append(problems, problem(file, fieldPath+".table_type", "must be mbr or gpt"))
			}
		}
		// cloud-init leaves the device untouched when layout is false.
		if layout == nil || !*layout {
			continue
		}
		if tableType == "mbr" {
			problems = append(problems, problem(file, fieldPath+".table_type", "Ignition only creates GPT partition tables; set table_type: gpt (the default is mbr)"))
			continue
		}
		// Partition 1 without a start, size, or type keeps any existing
		// partition 1, like cloud-init keeps an existing layout, and otherwise
		// fills the largest free area. The table is never wiped because
		// cloud-init does not repartition a partitioned disk even with overwrite.
		wipe := false
		disks = append(disks, base.Disk{
			Device: device, WipeTable: &wipe, Partitions: []base.Partition{{Number: 1}},
		})
	}
	if len(problems) > 0 {
		return nil, problems
	}
	return disks, nil
}
