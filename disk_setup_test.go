package transpile_test

import (
	"errors"
	"testing"

	"github.com/flatcar/butane-init"
)

func TestTranspileDiskSetup(t *testing.T) {
	// Shaped like Cluster API's disk_setup template output.
	input := `#cloud-config
disk_setup:
  /dev/sdb:
    table_type: gpt
    layout: true
    overwrite: true
  /dev/disk/azure/scsi1/lun0:
    table_type: gpt
    layout: true
    overwrite: false
  /dev/nvme1n1:
    table_type: gpt
    layout: true
  /dev/sdc:
    layout: false
  /dev/sdd:
    table_type: mbr
    layout: false
    overwrite: true
`
	// overwrite never wipes the table, and layout: false leaves the device alone.
	want := `version: 1.1.0
variant: flatcar
storage:
  disks:
    - device: /dev/disk/azure/scsi1/lun0
      partitions:
        - number: 1
      wipe_table: false
    - device: /dev/nvme1n1
      partitions:
        - number: 1
      wipe_table: false
    - device: /dev/sdb
      partitions:
        - number: 1
      wipe_table: false
`
	got, err := transpile.Transpile([]byte(input))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != want {
		t.Fatalf("output mismatch\nwant:\n%s\ngot:\n%s", want, got)
	}
}

func TestTranspileDiskSetupPreservesDefaults(t *testing.T) {
	for _, tt := range []struct {
		name   string
		stanza string
	}{
		// Cluster API emits a bare key when only fs_setup is configured.
		{"bare stanza", "disk_setup:"},
		{"empty stanza", "disk_setup: {}"},
		{"only unpartitioned devices", "disk_setup: {/dev/sdb: {layout: false}}"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			out, err := transpile.Transpile([]byte("#cloud-config\n" + tt.stanza + "\n"))
			if err != nil {
				t.Fatal(err)
			}
			if string(out) != "version: 1.1.0\nvariant: flatcar\n" {
				t.Fatalf("expected no disks, got:\n%s", out)
			}
		})
	}
}

func TestTranspileRejectsInvalidDiskSetup(t *testing.T) {
	tests := []struct {
		name   string
		stanza string
		path   string
	}{
		{"stanza must be an object", "disk_setup: [/dev/sdb]", "disk_setup"},
		{"definition must be an object", "disk_setup: {/dev/sdb: true}", "disk_setup.'/dev/sdb'"},
		{"default mbr table", "disk_setup: {/dev/sdb: {layout: true}}", "disk_setup.'/dev/sdb'.table_type"},
		{"explicit mbr table", "disk_setup: {/dev/sdb: {table_type: mbr, layout: true}}", "disk_setup.'/dev/sdb'.table_type"},
		{"unknown table type", "disk_setup: {/dev/sdb: {table_type: GPT, layout: false}}", "disk_setup.'/dev/sdb'.table_type"},
		{"layout is required", "disk_setup: {/dev/sdb: {table_type: gpt}}", "disk_setup.'/dev/sdb'.layout"},
		{"layout must be boolean", "disk_setup: {/dev/sdb: {table_type: gpt, layout: 'true'}}", "disk_setup.'/dev/sdb'.layout"},
		{"partition list layout", "disk_setup: {/dev/sdb: {table_type: gpt, layout: [50, 50]}}", "disk_setup.'/dev/sdb'.layout"},
		{"overwrite must be boolean", "disk_setup: {/dev/sdb: {table_type: gpt, layout: true, overwrite: 'yes'}}", "disk_setup.'/dev/sdb'.overwrite"},
		{"device alias", "disk_setup: {ephemeral0: {table_type: gpt, layout: true}}", "disk_setup.'ephemeral0'"},
		{"unclean device path", "disk_setup: {/dev//sdb: {table_type: gpt, layout: true}}", "disk_setup.'/dev//sdb'"},
		{"unsupported field while unpartitioned", "disk_setup: {/dev/sdb: {layout: false, device: /dev/sdc}}", "disk_setup.'/dev/sdb'.device"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, err := transpile.Transpile([]byte("#cloud-config\n" + tt.stanza + "\n"))
			var problems transpile.ValidationErrors
			if !errors.As(err, &problems) || len(problems) != 1 || out != nil {
				t.Fatalf("expected one validation error and no output; got %v, %s", err, out)
			}
			if problems[0].Path != tt.path || problems[0].Line != 2 || problems[0].Column == 0 {
				t.Fatalf("expected located error for %s; got %+v", tt.path, problems[0])
			}
		})
	}
}
