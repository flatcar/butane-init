package transpile

import (
	"fmt"
	"regexp"
	"strings"

	base "github.com/coreos/ignition/v2/butane/base/v0_5"
	"github.com/goccy/go-yaml/ast"
)

var generatedConfigUsername = regexp.MustCompile(`^[a-z_][a-z0-9_-]*$`)

type usersConfig struct {
	Files  []base.File
	Groups []base.PasswdGroup
	Users  []base.PasswdUser
}

type parsedUser struct {
	PasswordLogin bool
	Sudo          *string
	User          base.PasswdUser
}

type userDeclaration struct {
	HasPrimaryGroup bool
	Path            string
}

func parseUsers(document map[string]any, file *ast.File) (usersConfig, ValidationErrors) {
	var config usersConfig
	var problems ValidationErrors
	rawUsers, exists := document["users"]
	if !exists {
		return config, problems
	}
	items, ok := rawUsers.([]any)
	if !ok {
		problems = append(problems, problem(file, "users", "must be a list"))
		return config, problems
	}

	seenGroups := map[string]struct{}{}
	declaredUsers := map[string]userDeclaration{}
	var passwordLoginUsers []string
	for i, item := range items {
		path := fmt.Sprintf("users[%d]", i)
		fields, ok := item.(map[string]any)
		if !ok {
			problems = append(problems, problem(file, path, "must be an object"))
			continue
		}

		parsed, userProblems := parseUser(fields, path, file)
		problems = append(problems, userProblems...)
		user := parsed.User
		if user.Name != "" {
			if _, duplicate := declaredUsers[user.Name]; duplicate {
				problems = append(problems, problem(file, path+".name", fmt.Sprintf("duplicate user %q", user.Name)))
			} else {
				declaredUsers[user.Name] = userDeclaration{
					HasPrimaryGroup: user.PrimaryGroup != nil,
					Path:            path,
				}
			}
		}
		for _, group := range user.Groups {
			config.Groups = appendGroup(config.Groups, seenGroups, string(group))
		}
		if user.PrimaryGroup != nil {
			config.Groups = appendGroup(config.Groups, seenGroups, *user.PrimaryGroup)
		}
		if parsed.PasswordLogin {
			passwordLoginUsers = append(passwordLoginUsers, user.Name)
		}
		if parsed.Sudo != nil && user.Name != "" {
			contents := fmt.Sprintf("%s %s\n", user.Name, *parsed.Sudo)
			config.Files = append(config.Files, generatedFile("/etc/sudoers.d/"+user.Name, contents))
		}
		config.Users = append(config.Users, user)
	}
	for _, group := range config.Groups {
		user, matchesUser := declaredUsers[group.Name]
		if matchesUser && !user.HasPrimaryGroup {
			message := fmt.Sprintf("is required because group %q is explicitly referenced", group.Name)
			problems = append(problems, problem(file, user.Path+".primary_group", message))
		}
	}
	if len(passwordLoginUsers) > 0 {
		contents := fmt.Sprintf("Match User %s\n    PasswordAuthentication yes\nMatch all\n", strings.Join(passwordLoginUsers, ","))
		config.Files = append(config.Files, generatedFile("/etc/ssh/sshd_config.d/20-butane-init-password-auth.conf", contents))
	}
	return config, problems
}

func parseUser(fields map[string]any, path string, file *ast.File) (parsedUser, ValidationErrors) {
	allowed := map[string]bool{
		"name":                true,
		"passwd":              true,
		"gecos":               true,
		"homedir":             true,
		"shell":               true,
		"ssh_authorized_keys": true,
		"groups":              true,
		"primary_group":       true,
		"inactive":            true,
		"lock_passwd":         true,
		"sudo":                true,
	}
	var problems ValidationErrors
	for _, key := range sortedKeys(fields) {
		if !allowed[key] {
			problems = append(problems, problem(file, path+"."+key, "unsupported field"))
		}
	}

	var result parsedUser
	user := &result.User
	user.Name = requiredString(fields, "name", path, file, &problems)
	user.Gecos = optionalString(fields, "gecos", path, file, &problems)
	user.Groups = supplementaryGroups(fields, path, file, &problems)
	user.HomeDir = optionalString(fields, "homedir", path, file, &problems)
	user.PrimaryGroup = optionalString(fields, "primary_group", path, file, &problems)
	user.Shell = optionalString(fields, "shell", path, file, &problems)
	result.Sudo = optionalString(fields, "sudo", path, file, &problems)
	if inactive := optionalBool(fields, "inactive", path, file, &problems); inactive != nil && *inactive {
		problems = append(problems, problem(file, path+".inactive", "true is unsupported"))
	}
	if user.PrimaryGroup != nil && containsGroup(user.Groups, *user.PrimaryGroup) {
		problems = append(problems, problem(file, path+".primary_group", "must not also be a supplementary group"))
	}
	if lockPasswd := optionalBool(fields, "lock_passwd", path, file, &problems); lockPasswd != nil {
		result.PasswordLogin = !*lockPasswd
	}
	if passwd := optionalString(fields, "passwd", path, file, &problems); passwd != nil {
		if result.PasswordLogin {
			if passwordLocked(*passwd) {
				problems = append(problems, problem(file, path+".passwd", "must not be locked when lock_passwd is false"))
			} else {
				user.PasswordHash = passwd
			}
		} else {
			locked := lockPassword(*passwd)
			user.PasswordHash = &locked
		}
	}
	user.SSHAuthorizedKeys = sshKeys(fields, path, file, &problems)
	if (result.PasswordLogin || result.Sudo != nil) && user.Name != "" && !generatedConfigUsername.MatchString(user.Name) {
		problems = append(problems, problem(file, path+".name", "is unsafe for generated configuration"))
	}
	return result, problems
}

func generatedFile(path, contents string) base.File {
	mode := 0o600
	overwrite := true
	return base.File{
		Overwrite: &overwrite,
		Path:      path,
		Contents:  base.Resource{Inline: &contents},
		Mode:      &mode,
	}
}

func supplementaryGroups(fields map[string]any, path string, file *ast.File, problems *ValidationErrors) []base.Group {
	raw := optionalString(fields, "groups", path, file, problems)
	if raw == nil {
		return nil
	}

	var groups []base.Group
	seen := map[string]struct{}{}
	for _, item := range strings.Split(*raw, ",") {
		name := strings.TrimSpace(item)
		if name == "" {
			*problems = append(*problems, problem(file, path+".groups", "must not contain an empty group name"))
			continue
		}
		if _, duplicate := seen[name]; duplicate {
			*problems = append(*problems, problem(file, path+".groups", fmt.Sprintf("duplicate group %q", name)))
			continue
		}
		seen[name] = struct{}{}
		groups = append(groups, base.Group(name))
	}
	return groups
}

func containsGroup(groups []base.Group, name string) bool {
	for _, group := range groups {
		if string(group) == name {
			return true
		}
	}
	return false
}

func appendGroup(groups []base.PasswdGroup, seen map[string]struct{}, name string) []base.PasswdGroup {
	if _, exists := seen[name]; exists {
		return groups
	}
	seen[name] = struct{}{}
	return append(groups, base.PasswdGroup{Name: name})
}

func requiredString(fields map[string]any, key, path string, file *ast.File, problems *ValidationErrors) string {
	value := optionalString(fields, key, path, file, problems)
	if value == nil {
		if _, exists := fields[key]; !exists {
			*problems = append(*problems, problem(file, path+"."+key, "is required"))
		}
		return ""
	}
	return *value
}

func optionalString(fields map[string]any, key, path string, file *ast.File, problems *ValidationErrors) *string {
	raw, exists := fields[key]
	if !exists {
		return nil
	}
	value, ok := raw.(string)
	if !ok {
		*problems = append(*problems, problem(file, path+"."+key, "must be a string"))
		return nil
	}
	if value == "" {
		*problems = append(*problems, problem(file, path+"."+key, "must not be empty"))
		return nil
	}
	return &value
}

func optionalBool(fields map[string]any, key, path string, file *ast.File, problems *ValidationErrors) *bool {
	raw, exists := fields[key]
	if !exists {
		return nil
	}
	value, ok := raw.(bool)
	if !ok {
		*problems = append(*problems, problem(file, path+"."+key, "must be a boolean"))
		return nil
	}
	return &value
}

func sshKeys(fields map[string]any, path string, file *ast.File, problems *ValidationErrors) []base.SSHAuthorizedKey {
	raw, exists := fields["ssh_authorized_keys"]
	if !exists {
		return nil
	}
	items, ok := raw.([]any)
	if !ok {
		*problems = append(*problems, problem(file, path+".ssh_authorized_keys", "must be a list"))
		return nil
	}
	if len(items) == 0 {
		*problems = append(*problems, problem(file, path+".ssh_authorized_keys", "must not be empty"))
		return nil
	}

	keys := make([]base.SSHAuthorizedKey, 0, len(items))
	seen := map[string]struct{}{}
	for i, rawKey := range items {
		keyPath := fmt.Sprintf("%s.ssh_authorized_keys[%d]", path, i)
		key, ok := rawKey.(string)
		if !ok || key == "" {
			*problems = append(*problems, problem(file, keyPath, "must be a non-empty string"))
			continue
		}
		if _, duplicate := seen[key]; duplicate {
			*problems = append(*problems, problem(file, keyPath, "duplicate SSH authorized key"))
			continue
		}
		seen[key] = struct{}{}
		keys = append(keys, base.SSHAuthorizedKey(key))
	}
	return keys
}

func lockPassword(hash string) string {
	if passwordLocked(hash) {
		return hash
	}
	return "!" + hash
}

func passwordLocked(hash string) bool {
	return strings.HasPrefix(hash, "!") || strings.HasPrefix(hash, "*")
}
