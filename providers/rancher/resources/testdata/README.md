# Rancher API fixtures

**These payloads were not captured from a live Rancher server.** No Rancher
instance was reachable while this provider was written: the container registry
CDN that serves the `rancher/rancher` image is blocked in the build environment,
so the image could not be pulled and no server could be started.

They are instead reconstructed field by field from the **generated Rancher v3
API client** in `rancher/rancher`, under
`pkg/client/generated/{management,project}/v3/zz_generated_*.go`. That client is
machine generated from the schema the running server serves, after every Norman
mapper has been applied, so it is the authoritative list of wire field names and
their types. It is a much stronger source than the API documentation, and it is
what caught the following, each of which a documentation-derived schema would
have got wrong:

| what the stored Kubernetes type says | what the v3 API actually puts on the wire |
|---|---|
| `Token.TTLMillis` | `ttl`, not `ttlMillis` |
| `Token.LastUsedAt *metav1.Time` | `lastUsedAt` as an RFC 3339 **string**, absent as `""` |
| `Token.UserPrincipal Principal` | `userPrincipal` as a **string** reference |
| `GlobalRoleBinding.GlobalRoleName` | `globalRoleId` (every `…Name` reference field is renamed to `…Id`) |
| `RoleTemplate.RoleTemplateNames` | `roleTemplateIds` |
| `Project.ClusterName` | `clusterId` |
| pod security defaults in camelCase | `enforce-version`, `audit-version`, `warn-version`, hyphenated |
| `Cluster.Status.ServiceAccountToken` | dropped by the schema, never returned |
| `Cluster.Spec.DisplayName` | `name`; the Kubernetes `metadata.name` becomes `id` |

The upstream revisions are now **pinned**, and the field inventory they declare
is carried in `resources/wireformat_test.go` so that the correspondence is
checked mechanically rather than asserted in prose:

| pinned revision | commit | used for |
|---|---|---|
| `v2.12.9` | `db2754edc35189187bb10c524601d3d62642ff9b` | every type except the two below |
| `v2.11.6` | `8e2eb63d5d2b4744ab3b4ab44de573106519d77d` | `ClusterTemplate`, `ClusterTemplateRevision`, which 2.12 deleted with RKE1 |

`docker_credentials.json` deliberately carries the `password`, `auth` and
`email` fields the real API declares, with obviously fake values, so that
`TestRegistryCredentialDropsSecrets` proves they do not survive decoding.

Anything asserted against these fixtures is therefore a statement about the
**schema**, not about a running server. Field values still need to be confirmed
against a live Rancher; `providers/rancher/TESTING-TODO.md` is the checklist for
that.
