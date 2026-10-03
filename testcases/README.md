# Cluster API cloud-config fixtures

These fixtures are adapted from Cluster API v1.11.1 at commit
`dc0fb87601071371044f60c59096ecad1b84397f`:

- The
  [cloud-init users template](https://github.com/kubernetes-sigs/cluster-api/blob/dc0fb87601071371044f60c59096ecad1b84397f/bootstrap/kubeadm/internal/cloudinit/users.go#L20-L57)
  defines the emitted field names and value shapes.
- The
  [populated user test](https://github.com/kubernetes-sigs/cluster-api/blob/dc0fb87601071371044f60c59096ecad1b84397f/bootstrap/kubeadm/internal/ignition/clc/clc_test.go#L85-L99)
  supplies representative values.
- The
  [cloud-init users and groups documentation](https://docs.cloud-init.io/en/24.2/reference/modules.html#users-and-groups)
  defines the input semantics.

Secrets and SSH keys are nonfunctional test values. The fixtures are deliberately
small excerpts of Cluster API output so each one isolates a transpilation
boundary.

| Fixture | Purpose | Current result |
| --- | --- | --- |
| `cluster-api-supported-user.yaml` | Cluster API's direct user-field mappings | Success |
| `cluster-api-groups.yaml` | Same group used as supplementary and primary | Rejected as contradictory |
| `cluster-api-deferred-fields.yaml` | Cluster API account-policy value shapes | Rejected because `inactive: true` cannot be represented |
