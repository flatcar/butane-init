package transpile

import (
	_ "embed"
	"fmt"
	"strings"

	base "github.com/coreos/ignition/v2/butane/base/v0_5"
	"github.com/goccy/go-yaml/ast"
)

const (
	// Not /etc: Flatcar mounts /etc as a noexec confext overlay.
	commandsDir   = "/opt/butane-init"
	commandsState = "/var/lib/butane-init"
)

// commandReservedPaths are owned by the generated command units, even when a
// config does not use them, so write_files cannot interfere with them.
var commandReservedPaths = []string{
	commandsDir,
	commandsState,
	"/etc/systemd/system/butane-init-bootcmd.service",
	"/etc/systemd/system/butane-init-bootcmd.service.d",
	"/etc/systemd/system/butane-init-runcmd.service",
	"/etc/systemd/system/butane-init-runcmd.service.d",
	// Ignition appends enablement presets here; systemd then creates the
	// wants symlinks. Either can be blocked or overridden by a written file.
	"/etc/systemd/system-preset/20-ignition.preset",
	"/etc/systemd/system/multi-user.target.wants/butane-init-bootcmd.service",
	"/etc/systemd/system/multi-user.target.wants/butane-init-runcmd.service",
}

// The embedded files are written exactly as they land on disk. Their paths
// must stay within commandReservedPaths; TestCommandFilesUseReservedPaths
// checks this.
//
// Both units add /opt/bin, where Flatcar installations commonly place
// kubeadm, to PATH, with the cloud-init-per replacement first. bootcmd runs
// on every boot, like cloud-init's init stage: after basic networking and
// before network-online.target and runcmd. Its failure is logged without
// blocking runcmd. runcmd runs once per instance (machine ID); like
// cloud-init's semaphore, its marker is written before the commands run, so
// failed or interrupted commands are not retried on reboot.

//go:embed commands/butane-init-bootcmd.service
var bootcmdUnit string

//go:embed commands/butane-init-runcmd.service
var runcmdUnit string

// cloudInitPer replaces cloud-init's tools/cloud-init-per, which Flatcar does
// not ship. "instance" markers are stored per machine ID.
//
//go:embed commands/cloud-init-per
var cloudInitPer string

// parseCommands converts bootcmd and runcmd into scripts and oneshot units.
func parseCommands(document map[string]any, file *ast.File) ([]base.File, []base.Unit, ValidationErrors) {
	var files []base.File
	var units []base.Unit
	var problems ValidationErrors
	for _, stanza := range []struct{ key, unit string }{
		{"bootcmd", bootcmdUnit},
		{"runcmd", runcmdUnit},
	} {
		raw, exists := document[stanza.key]
		if !exists {
			continue
		}
		items, ok := raw.([]any)
		if !ok {
			problems = append(problems, problem(file, stanza.key, "must be a list"))
			continue
		}
		// Like cloud-init: one line per command, without set -e.
		var script strings.Builder
		script.WriteString("#!/bin/sh\n")
		for i, item := range items {
			command, ok := item.(string)
			if !ok {
				message := "must be a string"
				if _, isList := item.([]any); isList {
					message = "must be a string; argument lists are unsupported"
				}
				problems = append(problems, problem(file, fmt.Sprintf("%s[%d]", stanza.key, i), message))
				continue
			}
			script.WriteString(command + "\n")
		}
		if len(items) == 0 {
			continue
		}
		// Commands can embed bootstrap tokens, so only root may read them.
		files = append(files, commandFile(commandsDir+"/"+stanza.key+".sh", script.String(), 0o700))
		enabled := true
		contents := stanza.unit
		units = append(units, base.Unit{Name: "butane-init-" + stanza.key + ".service", Enabled: &enabled, Contents: &contents})
	}
	if len(problems) > 0 {
		return nil, nil, problems
	}
	if len(units) > 0 {
		files = append(files, commandFile(commandsDir+"/bin/cloud-init-per", cloudInitPer, 0o755))
	}
	return files, units, nil
}

func commandFile(path, contents string, mode int) base.File {
	file := generatedFile(path, contents)
	file.Mode = &mode
	return file
}
