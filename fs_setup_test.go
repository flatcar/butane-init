package transpile_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/flatcar/butane-init"
)

func TestTranspileFSSetup(t *testing.T) {
	// Values are unquoted, as Cluster API's fs_setup template emits them.
	input := `#cloud-config
fs_setup:
  - label: etcd_disk
    filesystem: ext4
    device: /dev/nvme1n1
    partition: 1
    extra_opts:
      - -E
      - lazy_itable_init=1,lazy_journal_init=1
  - label: data
    filesystem: xfs
    device: /dev/sdb
    partition: "12"
    overwrite: false
  - label:
    filesystem: swap
    device: /dev/disk/by-id/scratch
    partition: None
    overwrite: true
`
	want := `version: 1.1.0
variant: flatcar
storage:
  filesystems:
    - device: /dev/nvme1n1p1
      format: ext4
      label: etcd_disk
      options:
        - -E
        - lazy_itable_init=1,lazy_journal_init=1
      wipe_filesystem: false
    - device: /dev/sdb12
      format: xfs
      label: data
      wipe_filesystem: false
    - device: /dev/disk/by-id/scratch
      format: swap
      wipe_filesystem: true
`
	got, err := transpile.Transpile([]byte(input))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != want {
		t.Fatalf("output mismatch\nwant:\n%s\ngot:\n%s", want, got)
	}
}

func TestTranspileFSSetupEmpty(t *testing.T) {
	// Cluster API emits a bare fs_setup: when disk_setup has only partitions.
	for _, stanza := range []string{"fs_setup:", "fs_setup: []"} {
		got, err := transpile.Transpile([]byte("#cloud-config\n" + stanza + "\n"))
		if err != nil || string(got) != "version: 1.1.0\nvariant: flatcar\n" {
			t.Fatalf("%s: expected empty config, got %v, %s", stanza, err, got)
		}
	}
}

func TestTranspileRejectsInvalidFSSetup(t *testing.T) {
	tests := []struct {
		name  string
		entry string
		path  string
	}{
		{"default any partition", "{device: /dev/sdb, filesystem: ext4}", "fs_setup[0].partition"},
		{"auto partition", "{device: /dev/sdb, filesystem: ext4, partition: auto}", "fs_setup[0].partition"},
		{"zero partition", "{device: /dev/sdb, filesystem: ext4, partition: 0}", "fs_setup[0].partition"},
		{"padded partition", "{device: /dev/sdb, filesystem: ext4, partition: '01'}", "fs_setup[0].partition"},
		{"numbered partition of symlink", "{device: /dev/disk/azure/scsi1/lun0, filesystem: ext4, partition: 1}", "fs_setup[0].device"},
		{"device alias", "{device: ephemeral0.1, filesystem: ext4, partition: none}", "fs_setup[0].device"},
		{"unclean device", "{device: /dev/../sdb, filesystem: ext4, partition: none}", "fs_setup[0].device"},
		{"missing device", "{filesystem: ext4, partition: none}", "fs_setup[0].device"},
		{"unsupported filesystem", "{device: /dev/sdb, filesystem: ext3, partition: none}", "fs_setup[0].filesystem"},
		{"label too long", "{device: /dev/sdb, filesystem: xfs, label: thirteen_char, partition: none}", "fs_setup[0].label"},
		{"empty label", "{device: /dev/sdb, filesystem: ext4, label: '', partition: none}", "fs_setup[0].label"},
		{"overwrite must be boolean", "{device: /dev/sdb, filesystem: ext4, partition: none, overwrite: 'true'}", "fs_setup[0].overwrite"},
		{"extra option must be a string", "{device: /dev/sdb, filesystem: ext4, partition: none, extra_opts: [4096]}", "fs_setup[0].extra_opts[0]"},
		{"replace_fs", "{device: /dev/sdb, filesystem: ext4, partition: none, replace_fs: ntfs}", "fs_setup[0].replace_fs"},
		{"custom command", "{device: /dev/sdb, filesystem: ext4, partition: none, cmd: mkfs}", "fs_setup[0].cmd"},
		{"entry must be an object", "/dev/sdb", "fs_setup[0]"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, err := transpile.Transpile([]byte("#cloud-config\nfs_setup:\n  - " + tt.entry + "\n"))
			var problems transpile.ValidationErrors
			if !errors.As(err, &problems) || len(problems) != 1 || out != nil {
				t.Fatalf("expected one validation error and no output; got %v, %s", err, out)
			}
			if problems[0].Path != tt.path || problems[0].Line != 3 {
				t.Fatalf("expected located error for %s; got %+v", tt.path, problems[0])
			}
		})
	}
}

func TestTranspileRejectsDuplicateFSSetupDevices(t *testing.T) {
	// /dev/sdb with partition 2 and /dev/sdb2 are the same block device.
	input := "#cloud-config\nfs_setup:\n  - {device: /dev/sdb, filesystem: ext4, partition: 2}\n  - {device: /dev/sdb2, filesystem: xfs, partition: none}\n"
	_, err := transpile.Transpile([]byte(input))
	if err == nil || !strings.Contains(err.Error(), `fs_setup[1].device: duplicates filesystem device "/dev/sdb2"`) {
		t.Fatalf("expected duplicate device error, got %v", err)
	}
}
