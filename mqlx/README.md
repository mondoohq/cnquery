# mqlx

A high-level Go API for embedding MQL — both as a standalone expression engine
and for querying connected infrastructure.

> [!WARNING]
> **Experimental.** This package is under active development. Its API may
> change — or be removed — at any time, without notice and without a
> deprecation cycle, including in patch releases. Do not depend on it in
> production until it is marked stable.

## Two modes

**Expression mode** evaluates MQL over values you pass in. It needs no
providers, no asset, and no subprocesses:

```go
env, err := mqlx.NewEnv()
q, err := env.Compile(`props.name == /admin-.*/ && props.count > 3`,
    mqlx.WithProps(map[string]any{"name": "", "count": 0}))
res, err := q.Eval(ctx,
    mqlx.WithPropValues(map[string]any{"name": "admin-x", "count": 5}))
fmt.Println(res.Value()) // true
```

**Asset mode** runs queries against connected infrastructure. Connect once,
compile once, and evaluate against as many assets as needed:

```go
conn, err := env.ConnectLocal(ctx)
res, err := conn.Query(ctx, "asset { name platform }")

var info struct {
    Name     string `mql:"name"`
    Platform string `mql:"platform"`
}
err = res.Decode(&info)
```

## Custom resources

When your data is not a plain value but a resource with fields computed lazily
in Go — only when an expression reads them — register a caller-supplied,
in-process provider with `WithPrivateProvider`. Its resources compile and
evaluate without a subprocess, and you never implement `llx.Runtime` by hand:

```go
env, err := mqlx.NewEnv(
    mqlx.WithPrivateProvider(myprovider.Config, myprovider.Schema(), myprovider.Init()),
)
res, err := env.Compile("secret.entropy > 4.0").Eval(ctx)
```

## Design

See [ADR 027](../docs/adr/027-mqlx-go-embedding-api.md) for the motivation,
the full API contract, and the roadmap of planned follow-ups.

See [ADR 027](../docs/adr/027-mqlx-go-embedding-api.md) for the motivation,
the full API contract, and the roadmap of planned follow-ups.
