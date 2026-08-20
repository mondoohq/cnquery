# Trivy vs. cnspec: comparison and gap plan

**Date:** 2026-08-20
**Sources read:** mql `main` @ `7bcf6fd`, cnspec `v13.35.2` @ `8009b20f`, Trivy `main` @ `dcbadb7` (2026-08-17, latest stable v0.71.x).
This is a source-level comparison, not a marketing comparison. Every claim below points at a
file or a doc in one of those three trees.

---

## 1. Verdict

cnspec and Trivy are not the same kind of tool, and the places each one wins follow directly
from that.

* **cnspec wins on breadth of target and depth of assertion.** ~90 providers against Trivy's
  five targets, a real query language against Rego, compliance frameworks against a handful of
  CIS bundles, and live cloud/SaaS/identity/network posture that Trivy OSS no longer attempts
  at all.
* **Trivy wins on the container/SDLC scanner core and on friction.** It answers "what CVEs are
  in this image" in one command with no account, no network dependency on a vendor API, and
  materially higher package-detection recall inside a container image than cnspec has today.

Three gaps decide whether a Trivy user can even *try* cnspec on their own laptop:

1. `cnspec vuln` **hard-fails without a Mondoo Platform login** — there is no offline path.
2. cnspec's container package inventory has **lower recall by construction** (fixed path globs,
   top-level only, no compiled-binary parsing).
3. cnspec has **no secret scanner** — a whole Trivy scanner category with no counterpart.

Everything else on the list is smaller. Fixing those three is what converts "cnspec is a
different, broader product" into "cnspec is a strictly better default."

---

## 2. The architectural difference, because it explains every row below

**Trivy** is an analyzer→matcher pipeline. A fixed set of analyzers walks an artifact and emits
a package list; the list is matched against a pre-built database shipped as an OCI artifact
(`trivy-db`, `trivy-java-db`, `trivy-checks`). Misconfiguration is Rego over parsed IaC. Every
finding is a signature match or a rule evaluation. The whole thing is a static pipeline over a
static artifact.

**cnspec** is a resource graph plus policy-as-code. mql models the target as typed, queryable
resources; policies are MQL queries with assertions over them. Nothing is a signature; findings
are the result of evaluating a predicate against modeled state.

The consequence cuts both ways. Trivy cannot ask "does this IAM role permit privilege
escalation" because it has no live cloud model. cnspec cannot answer "what CVEs are in this
distroless image" as well as Trivy because it never built the artifact-walking analyzers that
question needs. Most gaps below are cases where **mql's model can already express the answer and
the product just doesn't ship the plumbing** — which is why they are tractable.

---

## 3. Side by side

| Capability | Trivy (OSS) | cnspec | Who wins |
|---|---|---|---|
| Targets | Container image, filesystem, git repo, VM image, K8s, SBOM, rootfs | ~90 providers: clouds, SaaS, identity, K8s, containers, hosts, network gear, databases, IaC, AI/MCP | **cnspec**, decisively |
| Vulnerability DB | Local, OCI-distributed, air-gap supported, `--db-repository` mirroring | Mondoo Platform API only; `cnspec vuln` exits if not logged in (`apps/cnspec/cmd/vuln.go:93`) | **Trivy** |
| Offline / air-gapped | Documented and supported | Not possible for vuln scanning | **Trivy** |
| OS package coverage | ~22 distros incl. Wolfi, Chainguard, Bottlerocket, MinimOS, Echo, Photon, Azure Linux | apt/dpkg, rpm/dnf/yum, apk, zypper, plus macOS/Windows/BSD | Comparable; Trivy names more niche distros |
| Language ecosystems modeled | 13 (Ruby, Python, PHP, Node, .NET, Java, Go, Rust, C/C++, Elixir, Dart, Swift, Julia) | 25+ — all of Trivy's plus Haskell, Erlang, Prolog, R, Lua, OPAM, vcpkg, Conda, GitHub Actions, Terraform providers, Jenkins plugins, WordPress plugins, Homebrew, Chocolatey | **cnspec** on paper |
| Language detection *recall in a container image* | Full filesystem walk; post-build artifacts (JAR hash lookup, Go build info, cargo-auditable, `.dist-info`, gemspec, `.deps.json`) | Fixed default-path globs, top-level only, lockfiles only (`providers/os/resources/go.go:24`, `java.go:28`) | **Trivy**, by a lot |
| SBOM output | CycloneDX, SPDX, both encodings | CycloneDX JSON/XML, SPDX JSON/tag-value, cnquery JSON (`sbom/formats.go`) | Tie |
| SBOM *content* out of the box | Everything the analyzers find | OS + Python + npm + kernel + Windows print drivers only (`internal/sbom/pack/sbom.mql.yaml`) | **Trivy** |
| SBOM as an input | `trivy sbom file.json`, `trivy convert` | Platform API path only (`upload/sbomscan.go`), no local CLI target | **Trivy** |
| Secret scanning | Built in, on by default, image/fs/repo, incl. `.pyc` | None | **Trivy** |
| License scanning | First-class scanner, SPDX expressions, risk classification | `license` fields exist on some package resources; no scanner, no policy surface | **Trivy** |
| VEX | Consumes 4 ways: repo, local file, OCI attestation, SBOM ref | Produces VEX for the platform; consumes third-party findings (SARIF/DefectDojo/Burp/ZAP/JUnit) but does not filter its own findings by VEX | **Trivy** |
| IaC / misconfiguration | K8s, Docker, Terraform (+plan), CloudFormation, ARM, Helm, Ansible, raw YAML/JSON, via Rego | Same list plus Bicep and Kustomize, via MQL, with cross-resource assertions | **cnspec** slightly |
| Live cloud posture (CSPM) | **None in OSS** — `trivy aws` was moved out to a plugin; no cloud target remains in `docs/guide/target/` | AWS, Azure, GCP, OCI, Alibaba, DigitalOcean, Hetzner, Proxmox, OpenStack, Nutanix, VMware, StackIt… | **cnspec**, uncontested |
| SaaS / identity posture | None | GitHub, GitLab, Okta, Slack, M365, Google Workspace, Atlassian, JumpCloud, Auth0, Keycloak, Zoom… | **cnspec**, uncontested |
| Compliance frameworks | CIS, NSA, PSS via YAML | Framework engine with evidence, CIS/NIST/ISO mapping, 80+ shipped policy bundles | **cnspec** |
| Policy authoring | Rego | MQL + YAML bundles, `bundle lint`/`fmt`/`docs`, LSP | **cnspec** |
| Report formats | table, JSON, SARIF, CycloneDX, SPDX, template, GitHub dep-snapshot | table, JSON, SARIF, JUnit, CSV, proto, plus S3/SQS/Service Bus sinks | Tie |
| Client/server split | `trivy server` — DB stays server-side, thin clients | `cnspec serve` is a scan *scheduler*, not a DB server | **Trivy** |
| Continuous monitoring | trivy-operator (in-cluster, CRDs, Prometheus) | mondoo-operator + `cnspec serve` + platform | Tie |
| Ecosystem | Plugin system, VS Code extension, 20+ documented CI integrations | GitHub Action, K8s operator, agent skills, LSP | **Trivy** |
| Risk scoring / prioritization | Severity only | Risk factors, exploitability, asset context (platform side) | **cnspec** |

---

## 4. Where cnspec is already ahead, and should be said louder

1. **Trivy OSS has no cloud posture scanning left.** The `aws` target was pulled out into a
   plugin; `docs/guide/target/` lists only container_image, filesystem, kubernetes, repository,
   rootfs, sbom, vm. Aqua's own comparison page reserves "Cloud scanning" for the commercial
   product. Anyone evaluating Trivy for CSPM is evaluating the wrong tool, and cnspec should say
   so with the receipt.
2. **Ecosystem coverage Trivy simply doesn't have.** Jenkins plugins, WordPress plugins, GitHub
   Actions pins, Terraform provider pins, Homebrew, Chocolatey, R, Lua, OPAM, vcpkg, Haskell,
   Erlang. Jenkins and WordPress plugin CVEs are a large, real, under-served attack surface.
   This is a differentiator that costs nothing to promote because the resources already exist.
3. **Assertions across resources.** "This security group is open *and* attached to an instance
   with a public IP *and* that instance has an unpatched kernel" is one MQL query and is not
   expressible in Trivy at all.
4. **cnspec can already eat Trivy's output.** `upload/report_conversion/` ships SARIF,
   DefectDojo, Burp, ZAP and JUnit converters into Mondoo FEX/VEX. The "aggregate everything,
   including Trivy" position is available today and is barely marketed.

---

## 5. The gaps, in priority order

### G1 — No offline vulnerability database *(blocker)*

`cnspec vuln` builds an SBOM locally, then requires `upstreamConf` and calls
`mvd.NewAdvisoryScannerClient`; with no login it calls `log.Fatal` (`apps/cnspec/cmd/vuln.go:90-103`).
Trivy pulls `trivy-db` from any of five registries, supports `--db-repository` mirroring, and
documents air-gapped operation.

This is not only a feature gap, it is an **evaluation gap**: the Trivy user's first command
fails, and they never see anything else cnspec does. It also disqualifies cnspec from regulated
and disconnected environments, which are exactly the accounts that pay.

There is a real business tension here — the advisory feed is Mondoo's moat. But Aqua faced the
identical tension and resolved it by giving away the DB, which is *why* Trivy won distribution
and why Aqua gets to sell on top of it. The recommended middle path:

* Ship an offline DB built from open feeds (OSV, distro security trackers, GHSA) as an OCI
  artifact, matched locally by purl and CPE.
* Keep the curated/commercial feed, risk factors, exploitability and asset context upstream.
* `cnspec vuln` works logged out, and prints what the logged-in version would add.

Work: DB build + publish pipeline; a local purl/CPE matcher in cnspec; `--db-repository` and
`--skip-db-update`; air-gap docs. Non-trivial — call it a quarter — but nothing else on this
list matters as much.

### G2 — Container package inventory recall *(blocker)*

mql's language inventories search **fixed default paths, top level only, lockfiles only**:

```go
// providers/os/resources/go.go:24
var defaultGoPaths = []string{"/app", "/home/*/go/src", "/root/go/src",
    "/usr/local/go/src", "/usr/src/app", "/workspace"}
// java.go:26 — "Only top-level files in these directories are scanned"
```

Against a real image this misses: a Go binary at `/usr/local/bin/app` (no `go.mod` in the image
at all — the single most common Go container layout, and the entire content of a distroless
image); a Spring Boot fat JAR one directory below `/opt`; any app installed somewhere the list
doesn't name. Trivy walks the whole layer filesystem and identifies post-build artifacts by
content.

Work, roughly in value order:

* Layer-aware full filesystem walk with an extension/filename index, replacing the glob lists.
* Go build-info parsing from ELF/PE/Mach-O (`debug/buildinfo`) — biggest single recall win.
* JAR identification by hash for JARs with no usable manifest (needs an artifact index, same
  shape as `trivy-java-db`).
* `cargo-auditable` binaries; `.deps.json` for .NET; `.dist-info`/`.egg-info` for Python;
  `gemspec` for Ruby; `installed.json` for PHP — i.e. the post-build metadata column of Trivy's
  coverage table, which mql currently has none of.

### G3 — No secret scanning *(blocker)*

Trivy scans every plaintext file (and `.pyc`) against built-in rules, on by default, on images,
filesystems and repos. cnspec has nothing: no rules, no entropy analysis, no allowlist model.
There is no partial credit available in a comparison table — the row is empty.

Work: a `secrets` resource in the os provider (rule set + entropy + allow rules + path
skipping), a shipped policy bundle, and results in the standard reporters. Self-contained, and
a strong candidate to ship before G1 and G2 because it is far smaller and closes a visible row.

### G4 — The SBOM query pack under-collects *(cheapest fix on the list)*

`sbom/generator/report_collection.go:52-74` already has fields for Go, Java, Rust, .NET, PHP,
Ruby, Dart, Haskell, Elixir, Erlang, Prolog, Julia, Conda, R, Lua, Swift, Terraform, Jenkins,
WordPress and GitHub Actions packages. The generator handles them all. And then
`internal/sbom/pack/sbom.mql.yaml` asks for **five things**: asset, OS packages, Python, npm,
Windows print drivers, kernel.

Everything else is dropped before it reaches the generator, for both `cnspec sbom` and
`cnspec vuln`. Adding ~20 query blocks to one YAML file immediately widens both. It does not
fix G2 — recall is still bounded by the default path globs — but it is a one-file change that
should not wait for anything.

### G5 — No license scanning

`license` exists on `package`, `python.package`, `npm.package`, `chocolatey.package`,
`wordpress.package`, and on container image labels. There is no scanner, no SPDX expression
evaluation, no allow/deny policy, no report section. The data is most of the way there; the
product surface is missing. A shipped license-policy bundle plus a reporter section would close
the row.

### G6 — No VEX consumption

Trivy filters its own findings through VEX from four sources. cnspec *produces*
`fex.VulnerabilityExchange` and ingests other tools' findings, but cannot be handed an OpenVEX
or CSAF document and told to suppress accordingly. Given that cnspec already speaks the format
in both directions internally, `--vex <file|repo|oci>` is a comparatively small addition, and it
is table stakes for anyone shipping images with a VEX pipeline.

### G7 — SBOM is not a local target

`upload/sbomscan.go` scans an uploaded SBOM, but only via the platform. There is no
`cnspec scan sbom ./bom.json`. Once G1 exists this falls out nearly free, and it is how many
teams want to use a scanner in the first place (build once, scan the SBOM everywhere).

### G8 — Distribution and friction

Trivy has a plugin system, a VS Code extension, and 20+ documented CI integrations. cnspec has
a GitHub Action, an operator, an LSP and agent skills. Less urgent than G1–G3, but it is why
Trivy shows up in a pipeline by default and cnspec has to be argued for.

---

## 6. Recommended plan

**Tier 0 — required to be a Trivy replacement rather than a Trivy complement**

1. **G1 offline vulnerability DB.** The one that unblocks evaluation, air-gapped accounts, and
   every comparison table. Decide the open-feed / commercial-feed split first; it is a business
   decision, not an engineering one, and it gates the design.
2. **G2 container inventory recall.** Filesystem walk + Go build info first; the rest of the
   post-build analyzers after.
3. **G3 secret scanner.** Smallest of the three, closes an empty row, ship it first if capacity
   is tight.

**Tier 1 — parity items that remove specific objections**

4. **G4 SBOM pack**, this week, it is one file.
5. **G6 VEX consumption** — `--vex` with OpenVEX and CSAF.
6. **G5 license scanning** as a policy surface.
7. **G7 `cnspec scan sbom`** once G1 lands.

**Tier 2 — press the advantage instead of only closing gaps**

8. **Lead with what Trivy structurally cannot do.** Trivy OSS has *no* cloud posture scanning.
   That should be the first line of every comparison, not a footnote.
9. **Promote the ecosystem coverage that already exists.** Jenkins and WordPress plugin
   inventories, GitHub Actions pins, Terraform provider pins. Ship policies against them so the
   coverage is usable, not just modeled.
10. **Adopt, don't replace.** Document `trivy image -f sarif | cnspec upload` as a supported
    path. Being the aggregation layer over Trivy is a better opening move than asking teams to
    rip Trivy out, and it works today.

**Tier 3 — make "superior" measurable**

11. **Build a recall benchmark.** A corpus of ~30 public images and repos (distroless Go, Spring
    Boot fat JAR, Rails, Django, .NET, Rust, a Jenkins controller, a WordPress install), run
    both tools, diff package counts and CVE counts per ecosystem, and track it in CI. Without
    this, every claim in this document — including mine — is unfalsifiable, and G2 in particular
    cannot be shown to be fixed.

---

## 7. Suggested first PR

Wire the missing ecosystems into `internal/sbom/pack/sbom.mql.yaml` (G4), and add the benchmark
corpus skeleton (Tier 3, item 11) so the improvement is visible as a number. Together they are
small, they are independent of the G1 business decision, and they produce the baseline that
every later item is measured against.
