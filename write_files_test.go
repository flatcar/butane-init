package transpile_test

import (
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"testing"

	schema "github.com/coreos/ignition/v2/butane/config/flatcar/v1_1"
	"github.com/flatcar/butane-init"
	"github.com/goccy/go-yaml"
)

func TestTranspileWriteFiles(t *testing.T) {
	input := `#cloud-config
write_files:
  - path: /etc/example.conf
    owner: core:root
    permissions: '0640'
    content: |-
      first line
      100% literal \n café
  - path: /etc/empty
  - path: /etc/explicit-empty
    permissions: '0000'
    content: ''
  - path: /runtime/persistent
    content: 'not under /run/'
`
	got, err := transpile.Transpile([]byte(input))
	if err != nil {
		t.Fatal(err)
	}
	var config schema.Config
	if err := yaml.Unmarshal(got, &config); err != nil {
		t.Fatal(err)
	}
	if len(config.Storage.Files) != 4 {
		t.Fatalf("files = %#v, want four files", config.Storage.Files)
	}
	for i, want := range []struct {
		path, content, user, group string
		mode                       int
	}{
		{"/etc/example.conf", "first line\n100% literal \\n café", "core", "root", 0o640},
		{"/etc/empty", "", "root", "root", 0o644},
		{"/etc/explicit-empty", "", "root", "root", 0},
		{"/runtime/persistent", "not under /run/", "root", "root", 0o644},
	} {
		file := config.Storage.Files[i]
		if file.Path != want.path || file.Contents.Inline == nil || *file.Contents.Inline != want.content ||
			file.Mode == nil || *file.Mode != want.mode || file.Overwrite == nil || !*file.Overwrite ||
			file.User.Name == nil || *file.User.Name != want.user || file.Group.Name == nil || *file.Group.Name != want.group {
			t.Errorf("file %d = %#v, want %+v", i, file, want)
		}
	}
}

func TestTranspileWriteFilesUnderRun(t *testing.T) {
	input := readFixture(t, "cluster-api-write-files.yaml") + `
  - path: '/run/kubeadm/a %m "quoted" \ file'
    owner: core:root
    permissions: '0640'
    content: |-
      first line
      literal %m \n café
  - path: /var/run/empty
    content: ''
`
	got, err := transpile.Transpile([]byte(input))
	if err != nil {
		t.Fatal(err)
	}
	var config schema.Config
	if err := yaml.Unmarshal(got, &config); err != nil {
		t.Fatal(err)
	}
	if len(config.Storage.Files) != 1 {
		t.Fatalf("want one persistent tmpfiles configuration, got %s", got)
	}
	file := config.Storage.Files[0]
	if file.Path != "/etc/tmpfiles.d/60-butane-init-write-files.conf" || file.Mode == nil || *file.Mode != 0o600 ||
		file.Overwrite == nil || !*file.Overwrite || file.Contents.Inline == nil {
		t.Fatalf("unprotected or missing tmpfiles configuration: %s", got)
	}
	lines := strings.Split(strings.TrimSuffix(*file.Contents.Inline, "\n"), "\n")
	const prefix = `f+!~ "/run/kubeadm/a %%m \"quoted\" \\ file" 0640 core root - `
	if len(lines) != 4 || !strings.HasPrefix(lines[2], prefix) || lines[3] != `f+!~ "/run/empty" 0644 root root - ` {
		t.Fatalf("unsafe or incorrect tmpfiles rules: %q", lines)
	}
	decoded, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(lines[2], prefix))
	if err != nil || string(decoded) != "first line\nliteral %m \\n café" {
		t.Fatalf("tmpfiles content = %q, error = %v", decoded, err)
	}
	const joinPrefix = `f+!~ "/run/kubeadm/kubeadm-join-config.yaml" 0640 root root - `
	decoded, err = base64.StdEncoding.DecodeString(strings.TrimPrefix(lines[0], joinPrefix))
	if err != nil || string(decoded) != "---\napiVersion: kubeadm.k8s.io/v1beta4\nkind: JoinConfiguration\nnodeRegistration:\n  name: worker-01\n" {
		t.Fatalf("Cluster API join configuration = %q, error = %v", decoded, err)
	}
	if !strings.HasPrefix(lines[1], `f+!~ "/run/cluster-api/placeholder" 0640 root root - `) {
		t.Fatalf("Cluster API placeholder rule = %q", lines[1])
	}
}

func TestTranspileWriteFilesRejectsUnsupportedValues(t *testing.T) {
	for _, fields := range []string{
		"content: null", "content: 42", "content: !!binary YWJj",
		"owner: root", "owner: ':root'", "owner: 'root:'", "owner: '1000:1000'",
		"owner: 'root:root:extra'", "owner: 'root:wheel - extra'",
		"permissions: 0644", "permissions: '0899'", "permissions: '4755'", "permissions: ''",
		"append: false", "encoding: text/plain", "defer: false", "source: {}",
	} {
		t.Run(fields, func(t *testing.T) {
			input := "#cloud-config\nwrite_files:\n  - path: /etc/example\n    " + fields + "\n"
			got, err := transpile.Transpile([]byte(input))
			var problems transpile.ValidationErrors
			if got != nil || !errors.As(err, &problems) || len(problems) == 0 {
				t.Fatalf("output = %s, error = %v; want structured validation errors", got, err)
			}
			if !strings.HasPrefix(problems[0].Path, "write_files[0].") || problems[0].Line == 0 {
				t.Fatalf("missing input field/location: %+v", problems)
			}
		})
	}
}

func TestTranspileWriteFilesRejectsConflictingPaths(t *testing.T) {
	for _, pair := range [][2]string{
		{"/etc/example", "/etc/example"},
		{"/run/kubeadm/join.yaml", "/run/kubeadm/join.yaml"},
		{"/run/kubeadm/join.yaml", "/var/run/kubeadm/join.yaml"},
		{"/run/kubeadm", "/run/kubeadm/join.yaml"},
		{"/run/kubeadm/join.yaml", "/run/kubeadm"},
		{"/etc/example", "/etc/sudoers.d/alice"},
		{"/etc/example", "/etc/ssh/sshd_config.d/20-butane-init-password-auth.conf"},
		{"/etc/example", "/etc/systemd/timesyncd.conf.d/60-butane-init.conf"},
		{"/run/example", "/etc/tmpfiles.d/60-butane-init-write-files.conf"},
	} {
		t.Run(pair[0]+"+"+pair[1], func(t *testing.T) {
			input := "#cloud-config\nusers:\n  - name: alice\n    sudo: ALL=(ALL) NOPASSWD:ALL\n    lock_passwd: false\nntp:\n  servers: [time.example.com]\nwrite_files:\n  - path: " + pair[0] + "\n  - path: " + pair[1] + "\n"
			got, err := transpile.Transpile([]byte(input))
			var problems transpile.ValidationErrors
			if got != nil || !errors.As(err, &problems) || !strings.Contains(err.Error(), "write_files[1].path") || !strings.Contains(err.Error(), "conflicts") {
				t.Fatalf("output = %s, error = %v; want a source-located path conflict", got, err)
			}
		})
	}
}

func TestTranspileWriteFilesRejectsInvalidPathsAndShapes(t *testing.T) {
	inputs := []string{"write_files: null", "write_files: {}", "write_files: [null]", "write_files: [{content: hello}]"}
	for _, path := range []string{"", "etc/example", "/etc/../run/example", "/run/./example", "/run/", "/run", "/var/run", "/tmp/example", "/proc/example", "/sys/example", "/dev/example", "/media", "/media/example", "/var/lock", "/var/lock/example", "/run/line\nbreak"} {
		inputs = append(inputs, fmt.Sprintf("write_files:\n  - path: %q", path))
	}
	for _, input := range inputs {
		t.Run(input, func(t *testing.T) {
			got, err := transpile.Transpile([]byte("#cloud-config\n" + input + "\n"))
			var problems transpile.ValidationErrors
			if got != nil || !errors.As(err, &problems) {
				t.Fatalf("output = %s, error = %v; want validation errors", got, err)
			}
		})
	}
	got, err := transpile.Transpile([]byte("#cloud-config\nwrite_files: []\n"))
	if err != nil || string(got) != "version: 1.1.0\nvariant: flatcar\n" {
		t.Fatalf("empty list: output = %s, error = %v", got, err)
	}
}

func TestTranspileWriteFilesRuntimeLineLimit(t *testing.T) {
	// 786402 bytes encode to 1048536 base64 bytes. With /run/large,
	// the 35-byte prefix and newline give a 1048572-byte tmpfiles line.
	payload := strings.Repeat("abc", 262134)
	for _, tt := range []struct {
		path    string
		content string
		wantErr bool
	}{
		{"/run/largeabc", payload, false}, // 1048575 bytes: just below 1 MiB.
		{"/run/largeabcd", payload, true}, // 1048576 bytes.
		{"/run/large%%", payload, true},   // Escaping the path also counts.
		{"/run/large", payload + "x", true},
		{"/etc/large", payload + "x", false}, // Only runtime rules have this limit.
	} {
		t.Run(tt.path, func(t *testing.T) {
			input := fmt.Sprintf("#cloud-config\nwrite_files:\n  - path: %s\n    content: %s\n", tt.path, tt.content)
			got, err := transpile.Transpile([]byte(input))
			if !tt.wantErr {
				if err != nil {
					t.Fatal(err)
				}
				var config schema.Config
				if err := yaml.Unmarshal(got, &config); err != nil {
					t.Fatal(err)
				}
				if len(config.Storage.Files) != 1 || config.Storage.Files[0].Contents.Inline == nil {
					t.Fatal("accepted content was not emitted")
				}
				content := *config.Storage.Files[0].Contents.Inline
				if strings.HasPrefix(tt.path, "/run/") {
					fields := strings.Fields(content)
					decoded, err := base64.StdEncoding.DecodeString(fields[len(fields)-1])
					if err != nil {
						t.Fatal(err)
					}
					content = string(decoded)
				}
				if content != tt.content {
					t.Fatal("accepted content was changed")
				}
				return
			}
			var problems transpile.ValidationErrors
			if got != nil || !errors.As(err, &problems) || len(problems) != 1 || problems[0].Path != "write_files[0].content" || !strings.Contains(problems[0].Message, "tmpfiles") {
				t.Fatalf("output length = %d, error = %v; want source-located tmpfiles size error", len(got), err)
			}
		})
	}
}
