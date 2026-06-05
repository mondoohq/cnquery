# Terraform Provider

The Terraform provider analyzes infrastructure-as-code statically — no cloud
credentials, nothing applied. Point it at:

- a **directory of HCL** (`.tf` / `.tf.json`), including any modules it contains,
- a **plan file** (`terraform show -json plan` output), or
- a **state file** (`terraform.tfstate` or `terraform show -json`).

```shell
cnspec shell terraform ./path/to/hcl
cnspec scan  terraform plan  plan.json
cnspec scan  terraform state terraform.tfstate
```

## One model across HCL, plan, and state

`terraform.managedResources` and `terraform.dataResources` are populated from
whichever source backs the connection, and every resource exposes a single
`values` view — the resolved configuration on HCL, the planned after-values on a
plan, the recorded values on state. **A policy written against `values` runs
unchanged on all three:**

```javascript
// identical on an HCL dir, a plan file, or a state file
terraform.managedResources.where(type == "aws_s3_bucket").all(values["acl"] != "public-read")
```

Each resource also has `type`, `name`, `address`, `mode`, `provider`, the typed
`references` it makes, the blocks that `referencedBy` it, and its `module`.

## How the typed model simplifies real policies

The patterns below are taken from the queries we ship in cnspec content
(`mondoo-aws-security` and friends). Each pair was verified to produce the same
verdict.

### Stop string-matching to join resources

A managed resource that decorates another (a flow log for a VPC, an encryption
config for a bucket) used to be joined by building a CSV of references and
substring-matching against a dot-padded label. The reverse reference index
replaces the whole dance:

```javascript
// before — VPC flow-log coverage
flowVpcs = terraform.resources("aws_flow_log")
  .where(arguments["traffic_type"] == "ALL" || arguments["traffic_type"] == "REJECT")
  .where(arguments["vpc_id"] != null).map(arguments["vpc_id"]).join(",")
terraform.resources("aws_vpc").all(flowVpcs.contains("." + labels[1] + "."))

// after
terraform.managedResources.where(type == "aws_vpc").all(
  referencedBy.where(resourceType == "aws_flow_log")
    .any(arguments["traffic_type"] == "ALL" || arguments["traffic_type"] == "REJECT"))
```

The same shape collapses the doubly-nested S3 encryption check (join **and**
`blocks.where(...).all(blocks.where(...))`):

```javascript
// before — every bucket encrypted with KMS
kmsBuckets = terraform.resources("aws_s3_bucket_server_side_encryption_configuration")
  .where(blocks.where(type == "rule").all(blocks.where(type == "apply_server_side_encryption_by_default")
    .all(arguments["sse_algorithm"] == "aws:kms" || arguments["sse_algorithm"] == "aws:kms:dsse")))
  .map(arguments["bucket"]).join(",")
terraform.resources("aws_s3_bucket").all(kmsBuckets.contains("." + labels[1] + "."))

// after
terraform.managedResources.where(type == "aws_s3_bucket").all(
  referencedBy.where(resourceType == "aws_s3_bucket_server_side_encryption_configuration").any(
    nestedBlocks["rule"].all(nestedBlocks["apply_server_side_encryption_by_default"].all(
      arguments["sse_algorithm"] == "aws:kms" || arguments["sse_algorithm"] == "aws:kms:dsse"))))
```

### Read policy documents without `.first`

`jsonencode({...})` arguments came back wrapped in a single-element list, so
every IAM-policy check carried a `.first`. `config` decodes them in place:

```javascript
// before
terraform.resources("aws_iam_policy").where(arguments["policy"].first["Statement"] != null).all(
  arguments["policy"].first["Statement"].all(
    _["Effect"].downcase == "deny" || [_["Resource"]].flat.any(_.contains("*")) == false))

// after
terraform.managedResources.where(type == "aws_iam_policy").all(
  config["policy"]["Statement"].all(
    _["Effect"].downcase == "deny" || [_["Resource"]].flat.any(_.contains("*")) == false))
```

### Select nested blocks by type

`blocks.where(type == "ingress")`, repeated and re-filtered, becomes a keyed
lookup with `nestedBlocks`:

```javascript
// before — SSH not open to the world
terraform.resources("aws_security_group")
  .where(blocks.where(type == "ingress").contains(arguments.to_port == 22))
  .none(blocks.where(type == "ingress") { arguments.cidr_blocks.contains("0.0.0.0/0") })

// after
terraform.managedResources.where(type == "aws_security_group").all(
  nestedBlocks["ingress"].where(arguments["to_port"] == 22)
    .all(arguments["cidr_blocks"].none(_ == "0.0.0.0/0")))
```

`tree` goes further, rendering a block as one walkable document:

```javascript
terraform.managedResources.where(type == "aws_instance")
  .all(tree["metadata_options"][0]["http_tokens"] == "required")
```

### Meta-arguments as first-class fields

The `lifecycle` block's flags are flattened onto the resource:

```javascript
// before
terraform.resources("aws_db_instance").all(
  blocks.where(type == "lifecycle").any(arguments["prevent_destroy"] == true))

// after
terraform.managedResources.where(type == "aws_db_instance").all(preventDestroy)
```

### Cross-references that weren't expressible before

A configured provider and its version requirement live in different blocks;
`requirement` resolves the join:

```javascript
// every configured provider is pinned to a version
terraform.providerConfigs.all(requirement.version != "")
```

## Effective values: see through variables, locals, and functions

`arguments` gives the literal source expression; `resolved` (and the
source-unified `values`) evaluate `var` defaults, `tfvars`, `local`s, and
Terraform functions into the value that will actually deploy:

```javascript
// resource "aws_instance" "web" {
//   instance_type = var.environment == "prod" ? "m5.large" : "t3.micro"
// }

terraform.managedResources.where(type == "aws_instance").all(arguments["instance_type"] == "m5.large")  // false — unresolved expression
terraform.managedResources.where(type == "aws_instance").all(values["instance_type"]    == "m5.large")  // true  — ternary + var.environment resolved
```

The four views, from raw to fully resolved:

| view | what it returns |
|------|-----------------|
| `arguments` | literal source; references as dotted strings; `jsonencode` list-wrapped |
| `config`    | `arguments` with `jsonencode` documents decoded in place |
| `resolved`  | `config` with var / tfvars / local / function values evaluated |
| `values`    | source-independent: `resolved` on HCL, after-values on a plan, recorded values on state |
| `tree`      | the whole block (arguments + nested blocks keyed by type) as one document |

## The dependency graph

```javascript
// blast radius and reachability
terraform.resource(type: "aws_vpc", name: "main").dependents       // everything that breaks if this changes
terraform.resource(type: "aws_instance", name: "web").dependencies // everything it transitively needs

// dead configuration
terraform.graph.orphans.where(type == "data")          // data sources nothing consumes
terraform.dataResources.where(referencedBy.length == 0)

// every reference relationship in the config
terraform.graph.edges { from to argument }
```

## Modules

Resources declared inside modules surface in `managedResources` with
fully-qualified addresses, and evaluation is per-module — a module's resource
resolves with the variable values the **call** passes:

```javascript
// module "logs" { source = "./modules/bucket"  bucket_name = "logs-${var.environment}" }
terraform.managedResources.where(type == "aws_s3_bucket") {
  address                       // "module.logs.aws_s3_bucket.this"
  module { key }                // "logs"
  values["bucket"]              // "logs-prod" — the call input flowed across the boundary and resolved
}
```

A module's source is navigable, and its inputs are typed (each input links to
the variable it sets and carries its resolved value):

```javascript
terraform.moduleCalls.where(key == "logs")[0] {
  sourceType                              // "local" | "registry" | "git" | "http"
  resources { address values }            // the module's own resources
  variables { name }
  inputs { name value variable { name } } // wiring: input -> module variable, with resolved value
}

// governance examples
terraform.moduleCalls.all(sourceType == "registry" && version != "")  // pinned registry modules only
terraform.moduleCalls.all(inputs.all(variable != null))               // no input sets an undeclared variable
```

## Compatibility

The typed model is additive. The lower-level surfaces it is built on —
`terraform.blocks`, `terraform.resources(...)`, `terraform.state.*`,
`terraform.plan.*`, and a block's raw `arguments` / `labels` — are unchanged and
remain available.
