package transpile

import (
	"encoding/base64"
	"fmt"
	"path"
	"regexp"
	"strconv"
	"strings"
	"unicode"

	base "github.com/coreos/ignition/v2/butane/base/v0_5"
	"github.com/goccy/go-yaml/ast"
)

const writeFilesTmpfilesPath = "/etc/tmpfiles.d/60-butane-init-write-files.conf"

var fileOwnerName = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_.-]*\$?$`)
var filePermissions = regexp.MustCompile(`^[0-7]{3,4}$`)

func parseWriteFiles(document map[string]any, file *ast.File, generated []base.File) ([]base.File, ValidationErrors) {
	raw, exists := document["write_files"]
	if !exists {
		return nil, nil
	}
	items, ok := raw.([]any)
	if !ok {
		return nil, ValidationErrors{problem(file, "write_files", "must be a list")}
	}
	var files []base.File
	var runtimeRules strings.Builder
	var problems ValidationErrors
	paths := append([]string{writeFilesTmpfilesPath}, commandReservedPaths...)
	for _, generatedFile := range generated {
		paths = append(paths, generatedFile.Path)
	}
	for i, item := range items {
		fieldPath := fmt.Sprintf("write_files[%d]", i)
		fields, ok := item.(map[string]any)
		if !ok {
			problems = append(problems, problem(file, fieldPath, "must be an object"))
			continue
		}
		for _, key := range sortedKeys(fields) {
			switch key {
			case "path", "content", "owner", "permissions":
			default:
				problems = append(problems, problem(file, fieldPath+"."+key, "unsupported field"))
			}
		}
		filename := requiredString(fields, "path", fieldPath, file, &problems)
		if filename != "" && (!path.IsAbs(filename) || path.Clean(filename) != filename || filename == "/" || strings.ContainsFunc(filename, unicode.IsControl)) {
			problems = append(problems, problem(file, fieldPath+".path", "must be a clean absolute file path without control characters"))
		}
		if filename == "/run" || filename == "/var/run" {
			problems = append(problems, problem(file, fieldPath+".path", "must name a file beneath /run, not the runtime directory itself"))
		}
		// These are virtual/temporary mounts or aliases outside the supported
		// runtime-file subset. Use /run/lock instead of Flatcar's /var/lock alias.
		for _, mount := range []string{"/tmp", "/dev", "/proc", "/sys", "/media", "/var/lock"} {
			if filename == mount || strings.HasPrefix(filename, mount+"/") {
				problems = append(problems, problem(file, fieldPath+".path", "unsupported temporary or virtual filesystem path"))
				break
			}
		}
		// Flatcar's /var/run is a symlink to /run, not persistent storage.
		if strings.HasPrefix(filename, "/var/run/") {
			filename = strings.TrimPrefix(filename, "/var")
		}
		if filename != "" {
			for _, other := range paths {
				if filename == other || strings.HasPrefix(filename, other+"/") || strings.HasPrefix(other, filename+"/") {
					problems = append(problems, problem(file, fieldPath+".path", fmt.Sprintf("conflicts with file path %q", other)))
					break
				}
			}
			paths = append(paths, filename)
		}
		contents := ""
		if raw, exists := fields["content"]; exists {
			var ok bool
			contents, ok = raw.(string)
			if !ok {
				problems = append(problems, problem(file, fieldPath+".content", "must be a string"))
			}
		}
		user, group := "root", "root"
		if owner := optionalString(fields, "owner", fieldPath, file, &problems); owner != nil {
			var found bool
			user, group, found = strings.Cut(*owner, ":")
			if !found || !fileOwnerName.MatchString(user) || !fileOwnerName.MatchString(group) {
				problems = append(problems, problem(file, fieldPath+".owner", "must be user:group with non-numeric account names"))
			}
		}
		mode := 0o644
		if permissions := optionalString(fields, "permissions", fieldPath, file, &problems); permissions != nil {
			if !filePermissions.MatchString(*permissions) {
				problems = append(problems, problem(file, fieldPath+".permissions", "must be a three- or four-digit octal string"))
			} else {
				parsed, _ := strconv.ParseUint(*permissions, 8, 12)
				mode = int(parsed)
				if mode > 0o777 {
					problems = append(problems, problem(file, fieldPath+".permissions", "setuid, setgid, and sticky bits are unsupported by the target Ignition version"))
				}
			}
		}
		if strings.HasPrefix(filename, "/run/") {
			// f+ replaces existing contents; ! limits execution to --boot invocations.
			// ~ decodes base64 without interpreting content specifiers or escapes.
			escapedPath := strings.NewReplacer(`\`, `\\`, `"`, `\"`, `%`, `%%`).Replace(filename)
			rule := fmt.Sprintf("f+!~ \"%s\" %04o %s %s - %s\n", escapedPath, mode, user, group, base64.StdEncoding.EncodeToString([]byte(contents)))
			// systemd's config reader limits each line to 1 MiB. Count all
			// syntax and escaping, not just the decoded file contents.
			if len(rule) >= 1<<20 {
				problems = append(problems, problem(file, fieldPath+".content", "encoded tmpfiles rule must be smaller than 1 MiB"))
				continue
			}
			runtimeRules.WriteString(rule)
			continue
		}
		overwrite := true
		files = append(files, base.File{
			Path: filename, Contents: base.Resource{Inline: &contents}, Mode: &mode,
			User: base.NodeUser{Name: &user}, Group: base.NodeGroup{Name: &group}, Overwrite: &overwrite,
		})
	}
	if runtimeRules.Len() > 0 {
		// The rule contains file contents, potentially including bootstrap tokens.
		files = append(files, generatedFile(writeFilesTmpfilesPath, runtimeRules.String()))
	}
	return files, problems
}
