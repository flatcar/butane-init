package transpile

import (
	"fmt"
	"net/netip"
	"regexp"
	"strings"

	base "github.com/coreos/ignition/v2/butane/base/v0_5"
	"github.com/goccy/go-yaml/ast"
)

// A hostname label is 1–63 ASCII letters, digits, or hyphens.
// The first and last characters must be letters or digits.
// RFC 1035 §2.3.1 defines the label syntax and length limit:
// https://www.rfc-editor.org/rfc/rfc1035.html#section-2.3.1
// RFC 1123 §2.1 relaxes the first-character rule to also allow digits:
// https://www.rfc-editor.org/rfc/rfc1123.html#section-2.1
var ntpHostnameLabel = regexp.MustCompile(
	`^[a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?$`,
)

func parseNTP(document map[string]any, file *ast.File) ([]base.File, ValidationErrors) {
	raw, exists := document["ntp"]
	if !exists {
		return nil, nil
	}
	fields, ok := raw.(map[string]any)
	if !ok {
		return nil, ValidationErrors{problem(file, "ntp", "must be an object")}
	}
	var problems ValidationErrors
	for _, key := range sortedKeys(fields) {
		if key != "enabled" && key != "servers" {
			problems = append(problems, problem(file, "ntp."+key, "unsupported field"))
		}
	}
	enabled := optionalBool(fields, "enabled", "ntp", file, &problems)
	var servers []string
	// Cluster API emits a bare servers: key when no servers are configured.
	if raw := fields["servers"]; raw != nil {
		items, ok := raw.([]any)
		if !ok {
			problems = append(problems, problem(file, "ntp.servers", "must be a list"))
		} else {
			for i, item := range items {
				server, ok := item.(string)
				if !ok || server == "" {
					path := fmt.Sprintf("ntp.servers[%d]", i)
					problems = append(problems, problem(file, path, "must be a non-empty string"))
					continue
				}
				if !validNTPServer(server) {
					path := fmt.Sprintf("ntp.servers[%d]", i)
					problems = append(problems, problem(file, path, "must be a hostname or an unscoped IP address without a port"))
					continue
				}
				servers = append(servers, server)
			}
		}
	}
	if len(problems) > 0 || (enabled != nil && !*enabled) || len(servers) == 0 {
		return nil, problems
	}
	// Reset earlier global servers, leaving per-link/DHCP sources untouched.
	contents := fmt.Sprintf(`[Time]
NTP=
NTP=%s
`, strings.Join(servers, " "))
	mode := 0o644
	overwrite := true
	return []base.File{{
		Path:      "/etc/systemd/timesyncd.conf.d/60-butane-init.conf",
		Mode:      &mode,
		Overwrite: &overwrite,
		Contents:  base.Resource{Inline: &contents},
	}}, nil
}

func validNTPServer(server string) bool {
	if address, err := netip.ParseAddr(server); err == nil {
		return address.Zone() == ""
	}
	hostname := strings.TrimSuffix(server, ".")
	if len(hostname) > 253 {
		return false
	}
	for _, label := range strings.Split(hostname, ".") {
		if !ntpHostnameLabel.MatchString(label) {
			return false
		}
	}
	return true
}
