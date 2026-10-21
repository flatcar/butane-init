package transpile_test

import (
	"errors"
	"strings"
	"testing"

	schema "github.com/coreos/ignition/v2/butane/config/flatcar/v1_1"
	"github.com/flatcar/butane-init"
	"github.com/goccy/go-yaml"
)

func TestTranspileNTPServers(t *testing.T) {
	input := `#cloud-config
users:
  - name: alice
    sudo: "ALL=(ALL) NOPASSWD:ALL"
ntp:
  servers:
    - time.example.net
    - 192.0.2.123
    - 2001:db8::123
`
	out, err := transpile.Transpile([]byte(input))
	if err != nil {
		t.Fatal(err)
	}
	// Explicitly enabling NTP must produce the same output as omitting enabled.
	explicit := strings.Replace(input, "ntp:\n", "ntp:\n  enabled: true\n", 1)
	explicitOut, err := transpile.Transpile([]byte(explicit))
	if err != nil || string(explicitOut) != string(out) {
		t.Fatalf("enabled: true must match the omitted default: %v, %s", err, explicitOut)
	}
	var config schema.Config
	if err := yaml.Unmarshal(out, &config); err != nil {
		t.Fatal(err)
	}
	if len(config.Passwd.Users) != 1 || config.Passwd.Users[0].Name != "alice" {
		t.Fatalf("NTP must preserve users: %+v", config.Passwd)
	}
	if len(config.Storage.Files) != 2 || config.Storage.Files[0].Path != "/etc/sudoers.d/alice" {
		t.Fatalf("NTP must preserve generated user files: %+v", config.Storage.Files)
	}
	file := config.Storage.Files[1]
	if file.Path != "/etc/systemd/timesyncd.conf.d/60-butane-init.conf" || file.Mode == nil || *file.Mode != 0o644 || file.Overwrite == nil || !*file.Overwrite {
		t.Fatalf("unexpected NTP file: %+v", file)
	}
	want := "[Time]\nNTP=\nNTP=time.example.net 192.0.2.123 2001:db8::123\n"
	if file.Contents.Inline == nil || *file.Contents.Inline != want {
		t.Fatalf("NTP must replace global sources without changing DHCP or fallbacks: %+v", file.Contents)
	}
	if len(config.Systemd.Units) != 0 {
		t.Fatalf("NTP must preserve service defaults: %+v", config.Systemd)
	}
}

func TestTranspileNTPPreservesDefaults(t *testing.T) {
	for _, tt := range []struct {
		name   string
		stanza string
	}{
		{"missing stanza", ""},
		{"empty stanza", "ntp: {}"},
		{"enabled without servers", "ntp: {enabled: true}"},
		{"disabled without servers", "ntp: {enabled: false}"},
		{"bare server list", "ntp: {servers:}"},
		{"null server list", "ntp: {servers: null}"},
		{"empty server list", "ntp: {servers: []}"},
		{"enabled with bare server list", "ntp: {enabled: true, servers:}"},
		{"enabled with empty server list", "ntp: {enabled: true, servers: []}"},
		{"disabled with servers", "ntp: {enabled: false, servers: [time.example.net]}"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			out, err := transpile.Transpile([]byte("#cloud-config\n" + tt.stanza + "\n"))
			if err != nil {
				t.Fatal(err)
			}
			if string(out) != "version: 1.1.0\nvariant: flatcar\n" {
				t.Fatalf("expected no NTP or service changes, got:\n%s", out)
			}
		})
	}
}

func TestTranspileRejectsInvalidNTP(t *testing.T) {
	tests := []struct {
		name   string
		stanza string
		path   string
	}{
		{"null stanza", "ntp: null", "ntp"},
		{"stanza must be an object", "ntp: []", "ntp"},
		{"enabled must be boolean", "ntp: {enabled: 'false'}", "ntp.enabled"},
		{"enabled must not be null", "ntp: {enabled: null}", "ntp.enabled"},
		{"servers must be a list", "ntp: {servers: time.example.net}", "ntp.servers"},
		{"server must be a string", "ntp: {servers: [123]}", "ntp.servers[0]"},
		{"empty server", "ntp: {servers: ['']}", "ntp.servers[0]"},
		{"server contains whitespace", "ntp: {servers: ['time.example.net other.example.net']}", "ntp.servers[0]"},
		{"server contains configuration injection", `ntp: {servers: ["time.example.net\nFallbackNTP=evil.example.net"]}`, "ntp.servers[0]"},
		{"server is a URL", "ntp: {servers: ['ntp://time.example.net']}", "ntp.servers[0]"},
		{"server contains a port", "ntp: {servers: ['time.example.net:123']}", "ntp.servers[0]"},
		{"bracketed IPv6 address", "ntp: {servers: ['[2001:db8::123]']}", "ntp.servers[0]"},
		{"scoped IPv6 address", "ntp: {servers: ['fe80::1%eth0']}", "ntp.servers[0]"},
		{"label starts with hyphen", "ntp: {servers: ['-time.example.net']}", "ntp.servers[0]"},
		{"empty hostname label", "ntp: {servers: ['time..example.net']}", "ntp.servers[0]"},
		{"label exceeds maximum length", "ntp: {servers: ['" + strings.Repeat("a", 64) + ".net']}", "ntp.servers[0]"},
		{"hostname exceeds maximum length", "ntp: {servers: ['" + strings.Repeat("abc.", 63) + "ab']}", "ntp.servers[0]"},
		{"invalid server while disabled", "ntp: {enabled: false, servers: ['bad server']}", "ntp.servers[0]"},
		{"unsupported pools while disabled", "ntp: {enabled: false, pools: []}", "ntp.pools"},
		{"unsupported NTP client", "ntp: {ntp_client: chrony}", "ntp.ntp_client"},
		{"unsupported custom configuration", "ntp: {config: {}}", "ntp.config"},
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

func TestTranspileNTPHostnameForms(t *testing.T) {
	for _, tt := range []struct {
		name   string
		server string
	}{
		{"single-label hostname", "ntp"},
		{"mixed case with trailing dot", "Time-1.example.net."},
		{"punycode", "xn--bcher-kva.example"},
		{"maximum label length", strings.Repeat("a", 63) + ".net"},
		{"maximum hostname length", strings.Repeat("abc.", 63) + "a."},
	} {
		t.Run(tt.name, func(t *testing.T) {
			input := "#cloud-config\nntp:\n  servers: ['" + tt.server + "']\n"
			out, err := transpile.Transpile([]byte(input))
			if err != nil {
				t.Fatal(err)
			}
			var config schema.Config
			if err := yaml.Unmarshal(out, &config); err != nil {
				t.Fatal(err)
			}
			if len(config.Storage.Files) != 1 || config.Storage.Files[0].Contents.Inline == nil ||
				*config.Storage.Files[0].Contents.Inline != "[Time]\nNTP=\nNTP="+tt.server+"\n" {
				t.Fatalf("server must be preserved verbatim: %s", out)
			}
		})
	}
}
