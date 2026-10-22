package transpile_test

import (
	"errors"
	"os"
	"reflect"
	"regexp"
	"strings"
	"testing"

	schema "github.com/coreos/ignition/v2/butane/config/flatcar/v1_1"
	"github.com/flatcar/butane-init"
	"github.com/goccy/go-yaml"
)

func TestTranspileCommands(t *testing.T) {
	got, err := transpile.Transpile([]byte(readFixture(t, "cluster-api-commands.yaml")))
	if err != nil {
		t.Fatal(err)
	}
	var config schema.Config
	if err := yaml.Unmarshal(got, &config); err != nil {
		t.Fatal(err)
	}

	files := map[string]string{}
	modes := map[string]int{}
	for _, file := range config.Storage.Files {
		files[file.Path] = *file.Contents.Inline
		modes[file.Path] = *file.Mode
	}
	// Commands are preserved verbatim, one per line, without set -e.
	wantScripts := map[string]string{
		"/opt/butane-init/bootcmd.sh": "#!/bin/sh\ncloud-init-per once wipe-scratch echo \"first boot only\"\n",
		"/opt/butane-init/runcmd.sh": "#!/bin/sh\n" +
			"echo \"pre-kubeadm\" > /run/pre-kubeadm.log\n" +
			"kubeadm join --config /run/kubeadm/kubeadm-join-config.yaml  && echo success > /run/cluster-api/bootstrap-success.complete\n" +
			"\n" +
			"echo post-kubeadm\n",
	}
	for path, want := range wantScripts {
		if files[path] != want || modes[path] != 0o700 {
			t.Errorf("%s = %q (mode %o), want %q (mode 700)", path, files[path], modes[path], want)
		}
	}
	if !strings.Contains(files["/opt/butane-init/bin/cloud-init-per"], `"$@"`) || modes["/opt/butane-init/bin/cloud-init-per"] != 0o755 {
		t.Errorf("cloud-init-per replacement missing or not executable: %#v", config.Storage.Files)
	}

	units := map[string]string{}
	for _, unit := range config.Systemd.Units {
		if unit.Enabled == nil || !*unit.Enabled {
			t.Errorf("unit %s is not enabled", unit.Name)
		}
		units[unit.Name] = *unit.Contents
	}
	runcmd := units["butane-init-runcmd.service"]
	for _, want := range []string{
		"ConditionPathExists=!/var/lib/butane-init/sem/%m/runcmd",
		"After=network-online.target butane-init-bootcmd.service",
		"Environment=PATH=/opt/butane-init/bin:/opt/bin:",
		"Environment=INSTANCE_ID=%m",
		"ExecStart=/opt/butane-init/runcmd.sh",
	} {
		if !strings.Contains(runcmd, want) {
			t.Errorf("runcmd unit missing %q:\n%s", want, runcmd)
		}
	}
	// The marker is written before the commands run, so a failed or
	// interrupted runcmd is not retried on the next boot.
	marker := strings.Index(runcmd, "ExecStartPre=/usr/bin/touch /var/lib/butane-init/sem/%m/runcmd")
	if marker == -1 || marker > strings.Index(runcmd, "ExecStart=") {
		t.Errorf("runcmd marker is not written before the commands run:\n%s", runcmd)
	}
	bootcmd := units["butane-init-bootcmd.service"]
	if strings.Contains(bootcmd, "Condition") || !strings.Contains(bootcmd, "Before=network-online.target butane-init-runcmd.service") {
		t.Errorf("bootcmd unit must run on every boot before runcmd:\n%s", bootcmd)
	}
}

func TestTranspileCommandsOnlyGeneratesUsedStanzas(t *testing.T) {
	got, err := transpile.Transpile([]byte("#cloud-config\nbootcmd: []\nruncmd:\n  - \"echo hi\"\n"))
	if err != nil {
		t.Fatal(err)
	}
	var config schema.Config
	if err := yaml.Unmarshal(got, &config); err != nil {
		t.Fatal(err)
	}
	if len(config.Systemd.Units) != 1 || config.Systemd.Units[0].Name != "butane-init-runcmd.service" {
		t.Errorf("units = %#v, want only the runcmd unit", config.Systemd.Units)
	}

	got, err = transpile.Transpile([]byte("#cloud-config\nbootcmd: []\nruncmd: []\n"))
	if err != nil {
		t.Fatal(err)
	}
	if want := "version: 1.1.0\nvariant: flatcar\n"; string(got) != want {
		t.Errorf("empty command lists = %q, want %q", got, want)
	}
}

func TestTranspileCommandsRejectsInvalidInput(t *testing.T) {
	input := `#cloud-config
bootcmd:
runcmd:
  - [kubeadm, join]
  - 42
  - "echo ok"
write_files:
  - path: /opt/butane-init/runcmd.sh
  - path: /etc/systemd/system/butane-init-bootcmd.service.d/override.conf
  - path: /etc/systemd/system-preset/20-ignition.preset
  - path: /etc/systemd/system/multi-user.target.wants
`
	got, err := transpile.Transpile([]byte(input))
	if got != nil {
		t.Fatalf("Transpile() output = %q, want nil on error", got)
	}
	var problems transpile.ValidationErrors
	if !errors.As(err, &problems) {
		t.Fatalf("Transpile() error = %v, want validation errors", err)
	}
	var paths []string
	for _, problem := range problems {
		paths = append(paths, problem.Path+": "+problem.Message)
	}
	want := []string{
		"bootcmd: must be a list",
		"runcmd[0]: must be a string; argument lists are unsupported",
		"runcmd[1]: must be a string",
		`write_files[0].path: conflicts with file path "/opt/butane-init"`,
		`write_files[1].path: conflicts with file path "/etc/systemd/system/butane-init-bootcmd.service.d"`,
		`write_files[2].path: conflicts with file path "/etc/systemd/system-preset/20-ignition.preset"`,
		`write_files[3].path: conflicts with file path "/etc/systemd/system/multi-user.target.wants/butane-init-bootcmd.service"`,
	}
	if !reflect.DeepEqual(paths, want) {
		t.Errorf("problems = %q, want %q", paths, want)
	}
}

// The embedded unit and script files hard-code their paths. Every Butane Init
// path they use must stay reserved against write_files.
func TestCommandFilesUseReservedPaths(t *testing.T) {
	pathPattern := regexp.MustCompile(`/[^\s:"$]*butane-init[^\s:"$]*`)
	for _, name := range []string{"butane-init-bootcmd.service", "butane-init-runcmd.service", "cloud-init-per"} {
		contents, err := os.ReadFile("commands/" + name)
		if err != nil {
			t.Fatal(err)
		}
		paths := pathPattern.FindAllString(string(contents), -1)
		if len(paths) == 0 {
			t.Fatalf("%s: found no Butane Init paths", name)
		}
		for _, path := range paths {
			input := "#cloud-config\nwrite_files:\n  - path: " + path + "\n"
			if _, err := transpile.Transpile([]byte(input)); err == nil || !strings.Contains(err.Error(), "conflicts with file path") {
				t.Errorf("%s uses %s, which write_files can target: %v", name, path, err)
			}
		}
	}
}
