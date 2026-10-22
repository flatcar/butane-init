<div style="text-align: center">

[![Flatcar OS](https://img.shields.io/badge/Flatcar-Website-blue?logo=data:image/svg+xml;base64,PD94bWwgdmVyc2lvbj0iMS4wIiBlbmNvZGluZz0idXRmLTgiPz4NCjwhLS0gR2VuZXJhdG9yOiBBZG9iZSBJbGx1c3RyYXRvciAyNi4wLjMsIFNWRyBFeHBvcnQgUGx1Zy1JbiAuIFNWRyBWZXJzaW9uOiA2LjAwIEJ1aWxkIDApICAtLT4NCjxzdmcgdmVyc2lvbj0iMS4wIiBpZD0ia2F0bWFuXzEiIHhtbG5zPSJodHRwOi8vd3d3LnczLm9yZy8yMDAwL3N2ZyIgeG1sbnM6eGxpbms9Imh0dHA6Ly93d3cudzMub3JnLzE5OTkveGxpbmsiIHg9IjBweCIgeT0iMHB4Ig0KCSB2aWV3Qm94PSIwIDAgODAwIDYwMCIgc3R5bGU9ImVuYWJsZS1iYWNrZ3JvdW5kOm5ldyAwIDAgODAwIDYwMDsiIHhtbDpzcGFjZT0icHJlc2VydmUiPg0KPHN0eWxlIHR5cGU9InRleHQvY3NzIj4NCgkuc3Qwe2ZpbGw6IzA5QkFDODt9DQo8L3N0eWxlPg0KPHBhdGggY2xhc3M9InN0MCIgZD0iTTQ0MCwxODIuOGgtMTUuOXYxNS45SDQ0MFYxODIuOHoiLz4NCjxwYXRoIGNsYXNzPSJzdDAiIGQ9Ik00MDAuNSwzMTcuOWgtMzEuOXYxNS45aDMxLjlWMzE3Ljl6Ii8+DQo8cGF0aCBjbGFzcz0ic3QwIiBkPSJNNTQzLjgsMzE3LjlINTEydjE1LjloMzEuOVYzMTcuOXoiLz4NCjxwYXRoIGNsYXNzPSJzdDAiIGQ9Ik02NTUuMiw0MjAuOXYtOTUuNGgtMTUuOXY5NS40aC0xNS45VjI2MmgtMzEuOVYxMzQuOEgyMDkuNFYyNjJoLTMxLjl2MTU5aC0xNS45di05NS40aC0xNnY5NS40aC0xNS45djMxLjINCgloMzEuOXYxNS44aDQ3Ljh2LTE1LjhoMTUuOXYxNS44SDI3M3YtMTUuOGgyNTQuOHYxNS44aDQ3Ljh2LTE1LjhoMTUuOXYxNS44aDQ3Ljh2LTE1LjhoMzEuOXYtMzEuMkg2NTUuMnogTTQ4Ny44LDE1MWg3OS42djMxLjgNCgloLTIzLjZ2NjMuNkg1MTJ2LTYzLjZoLTI0LjJMNDg3LjgsMTUxTDQ4Ny44LDE1MXogTTIzMywyMTQuNlYxNTFoNjMuN3YyMy41aC0zMS45djE1LjhoMzEuOXYyNC4yaC0zMS45djMxLjhIMjMzVjIxNC42eiBNMzA1LDMxNy45DQoJdjE1LjhoLTQ3Ljh2MzEuOEgzMDV2NDcuN2gtOTUuNVYyODYuMUgzMDVMMzA1LDMxNy45eiBNMzEyLjYsMjQ2LjRWMTUxaDMxLjl2NjMuNmgzMS45djMxLjhMMzEyLjYsMjQ2LjRMMzEyLjYsMjQ2LjRMMzEyLjYsMjQ2LjR6DQoJIE00NDguMywzMTcuOXY5NS40aC00Ny44di00Ny43aC0zMS45djQ3LjdoLTQ3LjhWMzAyaDE1Ljl2LTE1LjhoOTUuNVYzMDJoMTUuOUw0NDguMywzMTcuOXogTTQ0MCwyNDYuNHYtMzEuOGgtMTUuOXYzMS44aC0zMS45DQoJdi03OS41aDE1Ljl2LTE1LjhoNDcuOHYxNS44aDE1Ljl2NzkuNUg0NDB6IE01OTEuNiwzMTcuOXY0Ny43aC0xNS45djE1LjhoMTUuOXYzMS44aC00Ny44di0zMS43SDUyOHYtMTUuOGgtMTUuOXY0Ny43aC00Ny44VjI4Ni4xDQoJaDEyNy4zVjMxNy45eiIvPg0KPC9zdmc+DQo=)](https://www.flatcar.org/)
[![Discord](https://img.shields.io/badge/Discord-Chat%20with%20us!-5865F2?logo=discord)](https://discord.gg/PMYjFUsJyq)
[![Slack](https://img.shields.io/badge/Slack-Chat%20with%20us!-4A154B?logo=slack)](https://kubernetes.slack.com/archives/C03GQ8B5XNJ)
[![Twitter Follow](https://img.shields.io/twitter/follow/flatcar?style=social)](https://x.com/flatcar)
[![Mastodon Follow](https://img.shields.io/badge/Mastodon-Follow-6364FF?logo=mastodon)](https://hachyderm.io/@flatcar)
[![Bluesky](https://img.shields.io/badge/Bluesky-Follow-0285FF?logo=bluesky)](https://bsky.app/profile/flatcar.org)
[![OpenSSF Best Practices](https://www.bestpractices.dev/projects/10926/badge)](https://www.bestpractices.dev/projects/10926)



</div>



# butane-init: a Cloud-Init to Butane Config Transpiler

### Background

Flatcar uses Butane and Ignition for provisioning. Much of the wider ecosystem still produces cloud-config (cloud-init) YAML, which makes it harder to adopt Flatcar in environments where cloud-config is already the norm.

### The Project

A transpiler in Go that converts cloud-config YAML into Butane YAML for Flatcar
Container Linux. Generated configurations always use `variant: flatcar`; other
distributions, including Fedora CoreOS, are not currently supported.
The transpiler implements the minimum functionality needed to support ClusterAPI worker node provisioning, and to replace [coreos-cloudinit](https://github.com/flatcar/coreos-cloudinit)
In practice that limits the transpiler to basic Butane features such as users, groups, certificates, files and systemd units.

#### Goals and Non-Goals
- A working Go transpiler covering a documented subset of cloud-config.
- Test coverage for each supported feature, running in CI.
- End to end integration with Flatcar and ClusterAPI.
- Documentation covering supported fields and known limitations.

### Usage

Build `bt`, then pass a cloud-config file or pipe one on standard input:

```sh
go build -o bt ./cmd/bt
bt cloud-config.yaml > butane.yaml
# or: cat cloud-config.yaml | bt -o butane.yaml
```

`bt` currently converts the Cluster API `users`, `ntp`, `bootcmd`, `runcmd`, and plain `write_files` subsets. A config
is either converted completely or rejected; unsupported fields are never
silently discarded.

| cloud-config stanza | Status | Butane output |
| ------------------- | ------ | ------------- |
| `users` | Supported Cluster API subset | `passwd`, plus generated `storage.files` |
| `write_files` | Supported plain-file subset | `storage.files`, including tmpfiles rules for `/run` |
| `runcmd` | Supported Cluster API subset | A generated script and once-per-instance systemd unit |
| `bootcmd` | Supported Cluster API subset | A generated script and every-boot systemd unit |
| `ntp` | Supported Cluster API subset | A generated timesyncd configuration file |
| `disk_setup` | Unsupported | Planned storage configuration |
| `fs_setup` | Unsupported | Planned `storage.filesystems` |
| `mounts` | Unsupported | Planned filesystems or systemd mount units |

| cloud-config field    | Butane output |
| --------------------- | ------------- |
| `name`                | `passwd.users[].name` |
| `passwd`              | `passwd.users[].password_hash` |
| `gecos`               | `passwd.users[].gecos` |
| `homedir`             | `passwd.users[].home_dir` |
| `shell`               | `passwd.users[].shell` |
| `ssh_authorized_keys` | `passwd.users[].ssh_authorized_keys` |
| `groups`              | `passwd.users[].groups` and synthesized `passwd.groups` |
| `primary_group`       | `passwd.users[].primary_group` and synthesized `passwd.groups` |
| `inactive`            | `false` is a no-op; `true` is rejected |
| `lock_passwd`         | Password locking and a generated sshd drop-in when `false` |
| `sudo`                | A generated `/etc/sudoers.d/<username>` file |

The new fields accept only the shapes emitted by Cluster API: `groups` is a
comma-separated string, `primary_group` and `sudo` are non-empty strings, and
`inactive` and `lock_passwd` are booleans. Referenced groups are created before
users. Empty or duplicate supplementary groups are rejected, and a primary
group must not also be listed as a supplementary group. If a referenced group
has the same name as a configured user, that user must set `primary_group`
explicitly to avoid colliding with Linux's implicit private-group creation.

Password hashes are locked when `lock_passwd` is omitted or `true`. When it is
`false`, the hash remains unlocked and `bt` enables SSH password authentication
for that user with a Flatcar sshd drop-in. An already-locked hash is rejected
when `lock_passwd` is `false`. Users that request password login or sudo must
have names matching `^[a-z_][a-z0-9_-]*$` so names cannot become SSH, sudoers,
or path syntax.

Generated sshd and sudoers files are root-owned, mode 0600, and replace any
existing file at the generated path. Sudo rules are preserved verbatim and are
not checked for valid sudoers syntax. No sshd restart is needed because
Ignition writes the drop-in before sshd starts.

### Write files

`write_files` accepts a list of objects with these fields, using the Cluster API
v1.11.1 cloud-config vocabulary:

| Field | Supported values |
| ----- | ---------------- |
| `path` | Required clean absolute file path: no `.`/`..` components, repeated slashes, trailing slash, or control characters |
| `content` | Plain string, preserved exactly; omitted or `''` creates an empty file; null and binary values are rejected |
| `owner` | Explicit `user:group` names, default `root:root`; no partial forms or numeric IDs |
| `permissions` | Quoted three- or four-digit octal string, default `'0644'`; only `0000`–`0777` |

Owner names must match `[a-zA-Z_][a-zA-Z0-9_.-]*` with an optional trailing `$`.
Ownership does not create accounts or groups: they must exist on the target image
or be declared in `users`. `encoding`, `append`, `defer`, `source`, and all other
fields are rejected, even when they specify a no-op. An omitted stanza or empty
list generates nothing; a null stanza is rejected.

Persistent files use Ignition's overwrite behavior. Targets must be regular files
or absent, with writable parent directories on the target image; following
existing destination symlinks is not part of this subset. Parent-directory
ownership is left to Ignition/systemd defaults, not taken from `owner`.
`/tmp`, `/dev`, `/proc`, `/sys`, `/media`, and `/var/lock` paths are unsupported
(use `/run/lock/...` instead of `/var/lock/...`). Duplicate paths,
file/directory conflicts, and collisions with files generated by `users` or
`ntp` are rejected. `/etc/tmpfiles.d/60-butane-init-write-files.conf` is reserved.

Files beneath `/run` (including the `/var/run/` alias) instead generate `f+!~`
rules in that reserved tmpfiles configuration. The rules create or truncate files
after `/run` is mounted, preserve literal content using base64, and apply the
requested mode and ownership. Each complete encoded rule must be smaller than
1 MiB, including its escaped path and metadata; oversized runtime content is
rejected before producing output. This internal encoding does not enable the input
`encoding` field. The rule file is root-owned, mode 0600, and retains the content
on persistent storage: base64 is **not encryption**, and bootstrap credentials
are not automatically removed.

Runtime destinations and their parents must be trusted. Tmpfiles truncates an
existing regular file in place, writing before it applies the requested mode and
ownership. Do not use it to replace an untrusted or world-readable file with
sensitive content: existing readers could see that content during replacement.

**Unlike cloud-init's once-per-instance `write_files`, these `/run` files are
recreated on every boot.** `!` restricts rules to tmpfiles invocations using
`--boot`, not just the first boot. Such invocations also replace existing content.
Consumers must run after `systemd-tmpfiles-setup.service`; conflicting tmpfiles
rules installed outside this cloud-config remain the operator's responsibility.
Creating a kubeadm configuration does not execute `kubeadm join`; use `runcmd`
for that. Boot and reboot behavior is verified on Flatcar 4757.2.1.

### NTP

`ntp` accepts only `enabled` (a boolean) and `servers` (a list of ASCII hostnames,
IPv4 addresses, or bare IPv6 addresses without zone identifiers or ports).
Server names are validated without DNS lookups; whitespace, URLs, and configuration
syntax are rejected. Unsupported fields such as `pools`, `ntp_client`, and `config`
are rejected even when `enabled: false`.

With `enabled` omitted or `true`, non-empty servers produce a root-owned, mode 0644
`/etc/systemd/timesyncd.conf.d/60-butane-init.conf` file, replacing any file at that
path. It resets earlier global `NTP=` settings before assigning the supplied list;
later drop-ins can still override it. DHCP-provided sources remain allowed.
`enabled: false` skips NTP configuration after validation; it does not disable time
synchronization. Omitted, null, or empty server lists, an absent `ntp` stanza, and
`ntp: {}` retain Flatcar defaults without generating files or units.

This relies on Flatcar's native timesyncd service, does not switch clients on images
using another daemon, and does not make kubeadm wait for synchronization.
Fedora CoreOS [uses chrony](https://docs.fedoraproject.org/en-US/fedora-coreos/time-zone/#_time_synchronization),
so this timesyncd drop-in would not configure its default NTP client. Supporting it
would require both the appropriate Butane variant and chrony configuration, not
just a change to the variant.
### Commands

`bootcmd` and `runcmd` accept only the shape Cluster API emits: a list of strings.
Argument-list items (`- [ls, -l]`), other non-string items, and a null stanza are
rejected. An omitted stanza or empty list generates nothing.

Each stanza becomes a root-owned, mode 0700 `/bin/sh` script at
`/opt/butane-init/<stanza>.sh`, with one command per line, preserved verbatim, as
cloud-init does. The scripts are not run with `set -e`: a failing command does not
stop later ones, and the unit's result is the last command's exit status. Commands
may contain bootstrap tokens, so the scripts are readable only by root. They live in
`/opt` because Flatcar mounts `/etc` `noexec`.

| | `bootcmd` | `runcmd` |
| --- | --- | --- |
| Unit | `butane-init-bootcmd.service` | `butane-init-runcmd.service` |
| Frequency | Every boot | Once per instance, even if the commands fail |
| Ordering | After `network.target`, before `network-online.target` and `runcmd` | After `network-online.target` and `bootcmd` |

Both are oneshot units without a start timeout. A failing `bootcmd` is logged but
does not prevent `runcmd`. An *instance* is identified by the Flatcar machine ID:
`runcmd` writes `/var/lib/butane-init/sem/<machine-id>/runcmd` before running its
commands, so it reruns only after the machine ID changes, for example after
`flatcar-reset`. Output goes to the journal; `/var/log/cloud-init-output.log` is
not emulated. Unlike cloud-init's first boot, `write_files` content already exists
when `bootcmd` runs.

The units, and only the units, run with `HOME=/root`, `INSTANCE_ID` set to the
machine ID, and
`PATH=/opt/butane-init/bin:/opt/bin:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin`.
Flatcar does not ship cloud-init, so `/opt/butane-init/bin/cloud-init-per`
replaces its `cloud-init-per` tool:
`cloud-init-per once|instance|always <name> <command> [args...]`, with markers
under `/var/lib/butane-init/sem/` and `instance` markers stored per machine ID.
Commands are never rewritten; other cloud-init tools are not provided.

`/opt/butane-init`, `/var/lib/butane-init`, the two units and their drop-in
directories and enablement symlinks, and `/etc/systemd/system-preset/20-ignition.preset`
are reserved: `write_files` entries that target them are rejected.

All stanzas other than `users`, `ntp`, `write_files`, `bootcmd`, and `runcmd`
remain unsupported. Jinja templates are
not evaluated and are rejected; render them before passing cloud-config to `bt`.

### A Flatcar Container Linux project

Flatcar Container Linux is a fully open source, minimal-footprint, secure by default and always up-to-date Linux distribution for running containers at scale.

This repository is maintained by the Flatcar community and contributors. Development and design decisions are guided by the principles of transparency, modularity, and collaboration.

Please find information on:

- [Contribution Guide](https://github.com/flatcar/Flatcar/blob/main/CONTRIBUTING.md) – how to get involved and submit patches.
- [Repository Codeowners](./CODEOWNERS) – who's assigned for contribution reviews in this repository.
- [Repository Maintainers](./MAINTAINERS.md) – who’s responsible for what in this repository.
- [Flatcar Project Maintainers](https://github.com/flatcar/Flatcar/blob/main/MAINTAINERS.md) – who’s responsible for what at Flatcar.
- [Governance](https://github.com/flatcar/Flatcar/blob/main/governance.md) – how decisions are made and who the stakeholders are.
- [Code of Conduct](./code-of-conduct.md) – our commitment to a welcoming and inclusive community.
- [LICENSE](./LICENSE) - License under which the software is released.

---

## Community & Project Documentation

- [Contributing Guidelines](CONTRIBUTING.md) — How to contribute, find issues, and submit pull requests
- [Code of Conduct](CODE_OF_CONDUCT.md) — Standards for respectful and inclusive community participation
- [Security Policy](SECURITY.md) — How to report vulnerabilities and security-related information
- [Maintainers](MAINTAINERS.md) — Current project maintainers and their responsibilities
- [Governance](GOVERNANCE.md) — Project governance model, decision-making process, and roles
