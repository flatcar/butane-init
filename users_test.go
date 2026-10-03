package transpile_test

import (
	"strings"
	"testing"

	"github.com/flatcar/butane-init"
)

func TestTranspileClusterAPIWorkerUser(t *testing.T) {
	input := readFixture(t, "cluster-api-supported-user.yaml")

	want := `version: 1.1.0
variant: flatcar
passwd:
  users:
    - gecos: Foo B. Bar
      home_dir: /home/foo
      name: foo
      password_hash: "!$6$REDACTED_TEST_HASH"
      ssh_authorized_keys:
        - ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIREDACTED fixture@example
      shell: /bin/false
`

	got, err := transpile.Transpile([]byte(input))
	if err != nil {
		t.Fatalf("Transpile() error = %v", err)
	}
	if string(got) != want {
		t.Fatalf("Transpile() output mismatch\nwant:\n%s\ngot:\n%s", want, got)
	}
}

func TestTranspileCreatesReferencedGroups(t *testing.T) {
	input := `#cloud-config
users:
  - name: alice
    groups: "docker, developers"
    primary_group: staff
  - name: bob
    groups: "developers, monitoring"
    primary_group: docker
`
	want := `version: 1.1.0
variant: flatcar
passwd:
  groups:
    - name: docker
    - name: developers
    - name: staff
    - name: monitoring
  users:
    - groups:
        - docker
        - developers
      name: alice
      primary_group: staff
    - groups:
        - developers
        - monitoring
      name: bob
      primary_group: docker
`

	got, err := transpile.Transpile([]byte(input))
	if err != nil {
		t.Fatalf("Transpile() error = %v", err)
	}
	if string(got) != want {
		t.Fatalf("Transpile() output mismatch\nwant:\n%s\ngot:\n%s", want, got)
	}
}

func TestTranspileHandlesInactive(t *testing.T) {
	falseInput := "#cloud-config\nusers:\n  - name: alice\n    inactive: false\n"
	want := "version: 1.1.0\nvariant: flatcar\npasswd:\n  users:\n    - name: alice\n"

	got, err := transpile.Transpile([]byte(falseInput))
	if err != nil {
		t.Fatalf("Transpile(inactive: false) error = %v", err)
	}
	if string(got) != want {
		t.Fatalf("Transpile(inactive: false) output mismatch\nwant:\n%s\ngot:\n%s", want, got)
	}

	trueInput := "#cloud-config\nusers:\n  - name: alice\n    inactive: true\n"
	got, err = transpile.Transpile([]byte(trueInput))
	if err == nil {
		t.Fatal("Transpile(inactive: true) error = nil, want an error")
	}
	if got != nil {
		t.Fatalf("Transpile(inactive: true) output = %q, want nil", got)
	}
	if !strings.Contains(err.Error(), "users[0].inactive: true is unsupported") {
		t.Fatalf("Transpile(inactive: true) error = %q, want unsupported inactivity error", err)
	}
}

func TestTranspileConfiguresPasswordLogin(t *testing.T) {
	input := `#cloud-config
users:
  - name: alice
    passwd: "$6$ALICE"
    lock_passwd: false
  - name: bob
    lock_passwd: false
  - name: carol
    passwd: "$6$CAROL"
    lock_passwd: true
`
	want := `version: 1.1.0
variant: flatcar
passwd:
  users:
    - name: alice
      password_hash: $6$ALICE
    - name: bob
    - name: carol
      password_hash: "!$6$CAROL"
storage:
  files:
    - overwrite: true
      path: /etc/ssh/sshd_config.d/20-butane-init-password-auth.conf
      contents:
        inline: |
          Match User alice,bob
              PasswordAuthentication yes
          Match all
      mode: 384
`

	got, err := transpile.Transpile([]byte(input))
	if err != nil {
		t.Fatalf("Transpile() error = %v", err)
	}
	if string(got) != want {
		t.Fatalf("Transpile() output mismatch\nwant:\n%s\ngot:\n%s", want, got)
	}
}

func TestTranspileCreatesSudoersFiles(t *testing.T) {
	input := `#cloud-config
users:
  - name: alice
    sudo: "ALL=(ALL) NOPASSWD:ALL"
  - name: bob
    sudo: "ALL=(root) /usr/bin/systemctl status kubelet"
`
	want := `version: 1.1.0
variant: flatcar
passwd:
  users:
    - name: alice
    - name: bob
storage:
  files:
    - overwrite: true
      path: /etc/sudoers.d/alice
      contents:
        inline: |
          alice ALL=(ALL) NOPASSWD:ALL
      mode: 384
    - overwrite: true
      path: /etc/sudoers.d/bob
      contents:
        inline: |
          bob ALL=(root) /usr/bin/systemctl status kubelet
      mode: 384
`

	got, err := transpile.Transpile([]byte(input))
	if err != nil {
		t.Fatalf("Transpile() error = %v", err)
	}
	if string(got) != want {
		t.Fatalf("Transpile() output mismatch\nwant:\n%s\ngot:\n%s", want, got)
	}
}

func TestTranspileRejectsInvalidUsers(t *testing.T) {
	tests := map[string]struct {
		input   string
		wantErr string
	}{
		"Cluster API groups fixture": {
			input:   readFixture(t, "cluster-api-groups.yaml"),
			wantErr: "primary_group: must not also be a supplementary group",
		},
		"Cluster API deferred fields fixture": {
			input:   readFixture(t, "cluster-api-deferred-fields.yaml"),
			wantErr: "inactive",
		},
		"empty supplementary group": {
			input:   "#cloud-config\nusers:\n  - name: alice\n    groups: \"docker,,wheel\"\n",
			wantErr: "empty group name",
		},
		"duplicate supplementary group": {
			input:   "#cloud-config\nusers:\n  - name: alice\n    groups: \"docker, docker\"\n",
			wantErr: "duplicate group",
		},
		"groups must be a scalar": {
			input:   "#cloud-config\nusers:\n  - name: alice\n    groups: [docker]\n",
			wantErr: "users[0].groups: must be a string",
		},
		"inactive must be a boolean": {
			input:   "#cloud-config\nusers:\n  - name: alice\n    inactive: \"false\"\n",
			wantErr: "users[0].inactive: must be a boolean",
		},
		"lock_passwd must be a boolean": {
			input:   "#cloud-config\nusers:\n  - name: alice\n    lock_passwd: \"false\"\n",
			wantErr: "users[0].lock_passwd: must be a boolean",
		},
		"sudo must be a scalar": {
			input:   "#cloud-config\nusers:\n  - name: alice\n    sudo: [\"ALL=(ALL) ALL\"]\n",
			wantErr: "users[0].sudo: must be a string",
		},
		"locked password with password login": {
			input:   "#cloud-config\nusers:\n  - name: alice\n    passwd: \"!$6$HASH\"\n    lock_passwd: false\n",
			wantErr: "must not be locked when lock_passwd is false",
		},
		"unsafe sudoers username": {
			input:   "#cloud-config\nusers:\n  - name: ALL\n    sudo: \"ALL=(ALL) NOPASSWD:ALL\"\n",
			wantErr: "users[0].name: is unsafe for generated configuration",
		},
		"ignored sudoers filename": {
			input:   "#cloud-config\nusers:\n  - name: alice.admin\n    sudo: \"ALL=(ALL) NOPASSWD:ALL\"\n",
			wantErr: "users[0].name: is unsafe for generated configuration",
		},
		"sudoers path traversal": {
			input:   "#cloud-config\nusers:\n  - name: ../alice\n    sudo: \"ALL=(ALL) NOPASSWD:ALL\"\n",
			wantErr: "users[0].name: is unsafe for generated configuration",
		},
		"SSH username pattern": {
			input:   "#cloud-config\nusers:\n  - name: alice,bob\n    lock_passwd: false\n",
			wantErr: "users[0].name: is unsafe for generated configuration",
		},
		"SSH username negation": {
			input:   "#cloud-config\nusers:\n  - name: '!alice'\n    lock_passwd: false\n",
			wantErr: "users[0].name: is unsafe for generated configuration",
		},
		"SSH username control character": {
			input:   "#cloud-config\nusers:\n  - name: \"alice\\nbob\"\n    lock_passwd: false\n",
			wantErr: "users[0].name: is unsafe for generated configuration",
		},
		"empty optional field": {
			input:   "#cloud-config\nusers:\n  - name: alice\n    shell: \"\"\n",
			wantErr: "users[0].shell",
		},
		"duplicate username": {
			input:   "#cloud-config\nusers:\n  - name: alice\n  - name: alice\n",
			wantErr: "duplicate user",
		},
		"missing username has source location": {
			input:   "#cloud-config\nusers:\n  - shell: /bin/bash\n",
			wantErr: "[3:10] users[0].name: is required",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := transpile.Transpile([]byte(tt.input))
			if err == nil {
				t.Fatalf("Transpile() error = nil, want an error containing %q", tt.wantErr)
			}
			if got != nil {
				t.Fatalf("Transpile() output = %q, want nil on error", got)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("Transpile() error = %q, want it to contain %q", err, tt.wantErr)
			}
		})
	}
}
