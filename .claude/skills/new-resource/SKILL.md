---
name: new-resource
description: Add or change a resource in an existing mql provider — schema, codegen, implementation, tests, and verification against a real target. Use when the user wants to add a resource or fields to a provider (e.g. "add a bind9 resource", "expose the tomcat connectors", "add a resource for named.conf", "model the postfix config"), or when a resource returns null, empty, or wrong values and the cause is the schema rather than the API call.
argument-hint: "<provider>/<resource> (e.g. os/bind9, aws/aws.ec2.instance)"
---

# Add a resource to an mql provider

**CLAUDE.md is the rulebook** — doc-comment format, the typed-reference gate, `CreateResource` vs `NewResource`, `.lr.versions`, the sub-resource bar. Read it and follow it. This skill is the part that rulebooks do not carry: the traps that bite *even when you know the rule*, the commands that catch them mechanically, and the verification that proves the resource against a real target rather than against your reading of it.

Work through it in order. Every step ends in something you can run.

## 1. Ground the schema in a real artifact before writing it

Never design from documentation alone. Get the real thing first:

```bash
docker run -d --name src ubuntu:24.04 sleep 3600
docker exec src sh -c 'apt-get update -qq && apt-get install -y -qq <package>'
docker exec src sh -c 'cat /etc/<config>; ls /etc/<dir>/'
```

Read the distribution's actual layout. Debian splits configuration across fragments that the entry point includes; Red Hat usually does not. A schema designed from upstream docs models a file that no distribution ships.

**Validate every fixture with the format's own tool.** `named-checkconf`, `nginx -t`, `apachectl configtest`, `sshd -T`, `postfix check`. This is not politeness — it is what stops you pinning invented syntax as expected behavior:

> `named-checkconf` rejected two hand-written bind9 fixtures ("all zones must be in views", and a `recursion`/`allow-recursion` conflict). Both would have shipped as tests asserting configurations that BIND refuses to load.

## 2. Decide the shape, then check it against the identity dimensions

Follow CLAUDE.md's sub-resource bar (clear id, or nested typed refs — otherwise flatten). Then answer one more question the rulebook does not ask:

**Along which dimensions can this thing repeat?** Write them down and build `__id` from all of them.

Configuration formats nest, and the same name legitimately appears in two scopes:

| resource | dimensions | id must carry |
|---|---|---|
| `bind9.zone` | view, class, name | all three |
| `bind9.key` | view, name | both — two views may declare the same key name |
| `nginx.conf.server` | serverName, listen | both |

A missed dimension is not a cosmetic bug. `CreateResource` returns the **cached first instance** for a repeated id, so the second declaration silently reports the first one's values:

> `bind9.keys` reported both keys as `hmac-sha256` when the internet-facing view actually used `hmac-md5`. `bind9.keys.none(algorithm == "hmac-md5")` **passed** on a server using MD5 — the failure direction that matters.

Test it directly: two entries with the same name in different scopes, and assert the ids are distinct.

## 3. Traps that bite even though they are documented

Each of these is in CLAUDE.md. Each still gets walked into, because the failure looks like a bug in your Go code rather than a naming rule. Match the fingerprint:

| symptom | cause | fix |
|---|---|---|
| every field of a sub-resource reads `null`, plus `provider returned no data and no error for a field` with an empty `id=` | a field named `x` on a resource named `<parent>.x` — same dotted path, and the field loses | rename the resource, or flatten the block onto the parent |
| `is not a list type` | a list field whose path equals its element resource type | plural field over singular element resource (`zones` → `bind9.zone`) |
| `undefined: mql<Name>Internal` | an `Internal` struct added or removed after codegen | run `./mqlr generate` a second time |
| a generated accessor collides with an internal cache field | both named `directory` | prefix the cache field (`baseDir`, `cachePath`) |

**The init argument form is positional, not named.** This is not in CLAUDE.md and the `nginx.conf` doc comment gets it wrong:

```
bind9(path: "/etc/named.conf")   # failed to compile: resource bind9 does not have a field named path
bind9("/etc/named.conf")         # works
```

Write the positional form in the resource's doc comment, and use it for local verification.

## 4. Build and test without clobbering the installed provider

`PROVIDERS_PATH` replaces the provider search path entirely, so a build under test never touches `~/.config/mondoo/providers`:

```bash
make providers/build/<provider>
mkdir -p /tmp/pd/<provider> && cp providers/<provider>/dist/<provider>* /tmp/pd/<provider>/
PROVIDERS_PATH=/tmp/pd mql run <connection> -c "<query>"
```

Two builds in two directories give you an **A/B against `main`**, which is how you show a change is additive:

```bash
PROVIDERS_PATH=/tmp/pd-main mql run docker container c -c "$Q" -j > main.json
PROVIDERS_PATH=/tmp/pd-pr   mql run docker container c -c "$Q" -j > pr.json
cmp -s main.json pr.json && echo IDENTICAL
```

Build the baseline from the **current** `origin/main`. A stale baseline attributes someone else's merged change to your PR — or hides yours.

## 5. Verify against a real target, field by field

Automated tests do not prove a resource; they prove the parser. Run every new field against the real thing and read the values, then check these four, which are where resources actually fail:

- **The absent case.** On a host without the software, the check must **fail**, never pass vacuously. `mql run docker container <plain-ubuntu> -c '<resource>.<field> == false'` should error or report false. A resource that returns null and a check that reads null as satisfied is worse than no resource.
- **Null over invented values.** A field with no value in the file reports null or a documented zero — not a default you made up. If you *do* report an effective default (BIND recursion is on when unset), say so in the field's doc comment and pick the direction that fails safe.
- **Secrets.** If the format carries key material, sweep for it: `mql run ... -j | grep -c '<the secret>'` must be 0. Then state the guarantee precisely — "this field carries no secret" is true; "the secret is not exposed" is false while `file.content` exists.
- **Composability.** `<resource>.files { permissions { ... } }` — permission and ownership checks should reach every file that contributed, not just the entry point.

## 6. Before the PR

```bash
gofmt -l <changed .go files>
./mqlr generate providers/<p>/resources/<p>.lr --dist providers/<p>/resources
go build ./... && go test ./resources/...
git diff --stat providers/<p>/resources/<p>.lr.versions    # new entries at version+1
git diff -U0 -- '*.lr' | grep '^+' | grep -c "—"          # em dashes: check your lines, not the file
```

Check the diff, not the whole file: `os.lr` already carries 38 em dashes from before the rule, so grepping the file reports a problem you did not create.

Then write the PR body around what you **ran**, not what you wrote: the queries, the values they returned, and the cases that failed before the fix. A table of observed results is worth more than a description of the schema, and it is what lets a reviewer disagree with you about something real.

## Reference implementations

- **Brace-block config with includes:** `providers/os/resources/bind9/` + `bind9.go`, or `nginx/` + `nginx.go`. Include expansion, cycle detection, partial-parse survival.
- **Key/value config:** `providers/os/resources/sshd.go`.
- **Cloud API listing:** any `providers/aws/resources/aws_*.go` — different shape, same schema rules.

## Where this skill stops

This is the **build** path: design a resource, generate it, implement it, prove it against a real target.

- Auditing resources that already shipped — nil handling, pagination truncation, `__id` collisions across a whole provider — is the **`provider-bug-review`** skill. It reads code looking for defects; this one builds and verifies. If you are asking "what is wrong with the okta provider", you want that skill, not this one.
- Proving a PR against provisioned cloud infrastructure is **`provider-verification`**.
- Bootstrapping a provider that does not exist yet is **`new-provider`**, which hands off to this skill at its Step 7.
