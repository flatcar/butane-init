package transpile_test

import (
	"errors"
	"strings"
	"testing"

	schema "github.com/coreos/ignition/v2/butane/config/flatcar/v1_1"
	"github.com/flatcar/butane-init"
	"github.com/goccy/go-yaml"
)

func TestTranspileMounts(t *testing.T) {
	// Rendered by Cluster API's mounts template, which leaves every value unquoted.
	input := `#cloud-config
mounts:
  - - LABEL=etcd_disk
    - /var/lib/etcddisk
    
  - - /dev/nvme1n1p1
    - /mnt/data-1
    - xfs
    - noatime,x-systemd.after=data-ready.service,nofail,fail
    - 0
    - 0
    
  - - UUID=0a1b2c3d-0000-4000-8000-00000000abcd
    - /srv/.cache
    - ext4
    - defaults
    
`
	out, err := transpile.Transpile([]byte(input))
	if err != nil {
		t.Fatal(err)
	}
	var config schema.Config
	if err := yaml.Unmarshal(out, &config); err != nil {
		t.Fatal(err)
	}
	want := []struct{ name, contents string }{
		// cloud-init's default fields: auto type, nofail and _netdev options, and fsck.
		{`var-lib-etcddisk.mount`, `[Unit]
Before=butane-init-runcmd.service
After=cloud-init-network.service
Requires=systemd-fsck@dev-disk-by\x2dlabel-etcd_disk.service
After=systemd-fsck@dev-disk-by\x2dlabel-etcd_disk.service

[Mount]
What=/dev/disk/by-label/etcd_disk
Where=/var/lib/etcddisk
Options=defaults,nofail,x-systemd.after=cloud-init-network.service,_netdev

[Install]
WantedBy=remote-fs.target
`},
		// The later fail overrides nofail; pass 0 skips fsck.
		{`mnt-data\x2d1.mount`, `[Unit]
Before=butane-init-runcmd.service
After=data-ready.service

[Mount]
What=/dev/nvme1n1p1
Where=/mnt/data-1
Type=xfs
Options=noatime,x-systemd.after=data-ready.service,nofail,fail

[Install]
RequiredBy=local-fs.target
`},
		// systemd only escapes a dot at the start of the whole unit name.
		{`srv-.cache.mount`, `[Unit]
Before=butane-init-runcmd.service
Requires=systemd-fsck@dev-disk-by\x2duuid-0a1b2c3d\x2d0000\x2d4000\x2d8000\x2d00000000abcd.service
After=systemd-fsck@dev-disk-by\x2duuid-0a1b2c3d\x2d0000\x2d4000\x2d8000\x2d00000000abcd.service

[Mount]
What=/dev/disk/by-uuid/0a1b2c3d-0000-4000-8000-00000000abcd
Where=/srv/.cache
Type=ext4

[Install]
RequiredBy=local-fs.target
`},
	}
	if len(config.Systemd.Units) != len(want) || len(config.Storage.Files) != 0 || len(config.Storage.Filesystems) != 0 {
		t.Fatalf("expected only %d mount units, got:\n%s", len(want), out)
	}
	for i, unit := range config.Systemd.Units {
		if unit.Name != want[i].name || unit.Enabled == nil || !*unit.Enabled || unit.Contents == nil || *unit.Contents != want[i].contents {
			t.Errorf("unit %d = %s enabled=%v:\n%v\nwant %s:\n%s", i, unit.Name, unit.Enabled, unit.Contents, want[i].name, want[i].contents)
		}
	}

	// Omitted fields must match cloud-init's spelled-out defaults exactly.
	short, err := transpile.Transpile([]byte("#cloud-config\nmounts: [[LABEL=etcd_disk, /var/lib/etcddisk]]\n"))
	if err != nil {
		t.Fatal(err)
	}
	explicit, err := transpile.Transpile([]byte("#cloud-config\nmounts: [[LABEL=etcd_disk, /var/lib/etcddisk, auto, 'defaults,nofail,x-systemd.after=cloud-init-network.service,_netdev', '0', '2']]\n"))
	if err != nil || string(short) != string(explicit) {
		t.Fatalf("default fields differ: %v\n%s\nwant:\n%s", err, explicit, short)
	}
}

func TestTranspileEmptyMounts(t *testing.T) {
	out, err := transpile.Transpile([]byte("#cloud-config\nmounts: []\n"))
	if err != nil || string(out) != "version: 1.1.0\nvariant: flatcar\n" {
		t.Fatalf("expected no units: %v\n%s", err, out)
	}
}

func TestTranspileRejectsInvalidMounts(t *testing.T) {
	tests := []struct {
		name   string
		stanza string
		path   string
	}{
		{"null stanza", "mounts: null", "mounts"},
		{"stanza must be a list", "mounts: {}", "mounts"},
		{"entry must be a list", "mounts: [LABEL=data]", "mounts[0]"},
		{"device without mount point", "mounts: [[LABEL=data]]", "mounts[0]"},
		{"more than six fields", "mounts: [[LABEL=data, /data, auto, defaults, 0, 2, extra]]", "mounts[0]"},
		{"null field", "mounts: [[LABEL=data, null]]", "mounts[0][1]"},
		{"numeric device", "mounts: [[0, /data]]", "mounts[0][0]"},
		{"numeric options", "mounts: [[LABEL=data, /data, auto, 0]]", "mounts[0][3]"},
		{"datasource alias", "mounts: [[ephemeral0, /mnt]]", "mounts[0][0]"},
		{"short device name", "mounts: [[sdb1, /data]]", "mounts[0][0]"},
		{"whole disk", "mounts: [[/dev/sdb, /data]]", "mounts[0][0]"},
		{"whole NVMe namespace", "mounts: [[/dev/nvme1n1, /data]]", "mounts[0][0]"},
		{"symlink that can name a whole disk", "mounts: [[/dev/disk/by-id/virtio-data, /data]]", "mounts[0][0]"},
		{"network filesystem", "mounts: [['server:/export', /data]]", "mounts[0][0]"},
		{"label needing escapes", "mounts: [['LABEL=my disk', /data]]", "mounts[0][0]"},
		{"relative mount point", "mounts: [[LABEL=data, data]]", "mounts[0][1]"},
		{"root mount point", "mounts: [[LABEL=data, /]]", "mounts[0][1]"},
		{"trailing slash", "mounts: [[LABEL=data, /data/]]", "mounts[0][1]"},
		{"parent component", "mounts: [[LABEL=data, /data/../srv]]", "mounts[0][1]"},
		{"mount point needing escapes", "mounts: [[LABEL=data, '/data%n']]", "mounts[0][1]"},
		{"read-only usr", "mounts: [[LABEL=data, /usr/local]]", "mounts[0][1]"},
		{"symlinked runtime path", "mounts: [[LABEL=data, /var/run/data]]", "mounts[0][1]"},
		{"var itself", "mounts: [[LABEL=data, /var]]", "mounts[0][1]"},
		{"hides default home", "mounts: [[LABEL=data, /home]]\nusers: [{name: alice}]", "mounts[0][1]"},
		{"hides explicit home", "mounts: [[LABEL=data, /srv]]\nusers: [{name: alice, homedir: /srv/alice}]", "mounts[0][1]"},
		{"duplicate mount point", "mounts: [[LABEL=a, /data], [LABEL=b, /data]]", "mounts[1][1]"},
		{"swap", "mounts: [[LABEL=swap, /swap, swap]]", "mounts[0][2]"},
		{"network type", "mounts: [[LABEL=data, /data, nfs]]", "mounts[0][2]"},
		{"noauto", "mounts: [[LABEL=data, /data, auto, 'defaults,noauto']]", "mounts[0][3]"},
		{"other systemd option", "mounts: [[LABEL=data, /data, auto, 'x-systemd.requires=foo.service']]", "mounts[0][3]"},
		{"after without unit suffix", "mounts: [[LABEL=data, /data, auto, 'x-systemd.after=foo']]", "mounts[0][3]"},
		{"option needing escapes", "mounts: [[LABEL=data, /data, auto, 'defaults,x%n']]", "mounts[0][3]"},
		{"empty option", "mounts: [[LABEL=data, /data, auto, 'defaults,,nofail']]", "mounts[0][3]"},
		{"dump", "mounts: [[LABEL=data, /data, auto, defaults, 1]]", "mounts[0][4]"},
		{"pass out of range", "mounts: [[LABEL=data, /data, auto, defaults, 0, 3]]", "mounts[0][5]"},
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

func TestTranspileMountsAllowHidingWriteFiles(t *testing.T) {
	// cloud-init writes files before mounting too, so the mount hides them in both.
	input := "#cloud-config\nwrite_files: [{path: /data/seed, content: x}]\nmounts: [[LABEL=data, /data]]\n"
	out, err := transpile.Transpile([]byte(input))
	if err != nil || !strings.Contains(string(out), "data.mount") {
		t.Fatalf("expected a mount over a written file: %v\n%s", err, out)
	}
}
