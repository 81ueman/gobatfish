# gobatfish

`gobatfish` is a Go client library for [Batfish](https://github.com/batfish/batfish),
mirroring the public API of [pybatfish](https://github.com/batfish/pybatfish).

The goal is that anything you can do with pybatfish you can do with `gobatfish`
in an idiomatic Go way: create snapshots, ask questions loaded dynamically from
the Batfish service, inspect table answers, run assertions, and extract facts.

## Install

```bash
go get github.com/81ueman/gobatfish
```

## Quick start

```go
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/81ueman/gobatfish/client"
	"github.com/81ueman/gobatfish/question"
)

const config = `hostname r1
!
interface GigabitEthernet0/0
 ip address 10.0.0.1 255.255.255.0
 no shutdown
!
`

func main() {
	ctx := context.Background()
	bf := client.NewSession(client.SessionConfig{Host: "localhost"})
	defer bf.Close()

	if _, err := bf.SetNetwork(ctx, "mynet"); err != nil {
		log.Fatal(err)
	}
	if _, err := bf.InitSnapshotFromText(ctx, config, client.InitSnapshotFromTextOptions{
		Platform: "cisco",
		Filename: "r1.cfg",
	}); err != nil {
		log.Fatal(err)
	}
	if err := bf.LoadQuestions(ctx); err != nil {
		log.Fatal(err)
	}

	// Questions are loaded from the Batfish service at runtime; Go has no
	// dynamic attributes, so they are addressed through a registry.
	q, err := bf.Q.Get("nodeProperties")
	if err != nil {
		log.Fatal(err)
	}
	q.Set("nodes", ".*")

	ans, err := q.Answer(ctx, question.AnswerOptions{Snapshot: bf.Snapshot})
	if err != nil {
		log.Fatal(err)
	}
	table, ok := ans.Table()
	if !ok {
		log.Fatal("nodeProperties did not return a table")
	}

	// Table answers expose a pandas-like data frame.
	df := table.Frame()
	for row := range df.Rows() {
		fmt.Println(row.Str("Node"), row.Get("Configuration_Format"))
	}

	if _, err := bf.Asserts.AssertNoUndefinedReferences(ctx, false, nil, "table"); err != nil {
		log.Fatal(err)
	}
}
```

## Mapping from pybatfish

Go has no operator overloading and no dynamic attributes, so a few Python
constructs are expressed as methods or registries. Everything else keeps the
pybatfish names and semantics.

| pybatfish | gobatfish |
|---|---|
| `Session(host=..., port=...)` | `client.NewSession(client.SessionConfig{...})` |
| `session.set_network("n")` | `bf.SetNetwork(ctx, "n")` |
| `session.init_snapshot(path, name=...)` | `bf.InitSnapshot(ctx, path, client.InitSnapshotOptions{...})` |
| `session.init_snapshot_from_text(text)` | `bf.InitSnapshotFromText(ctx, text, ...)` |
| `session.q.bgpSessionStatus(nodes="n")` | `bf.Q.Get("bgpSessionStatus")` then `q.Set("nodes", "n")` |
| `question.answer(snapshot=...)` | `q.Answer(ctx, question.AnswerOptions{Snapshot: ...})` |
| `answer.frame()` | `ans.Table()` then `.Frame()` |
| `df["Node"]` | `df.Col("Node")` |
| `df[df["Node"] == "n1"]` | `df.Filter(df.Col("Node").Eq("n1"))` |
| `df.loc[0]["col"]` | `df.At(0, "col")` |
| `len(df)` / `df.empty` | `df.Len()` / `df.Empty()` |
| `df.columns` | `df.Columns()` |
| `df.to_dict(orient="records")` | `df.Records()` |
| `df.to_string()` | `df.String()` |
| `df.groupby("x").filter(f)` | `df.GroupBy("x").Filter(f)` |
| `df.duplicated(["x"], keep=False)` | `df.Duplicated([]string{"x"}, false)` |
| `series.apply(f)` / `series.nunique()` | `series.Map(f)` / `series.NUnique()` |
| `assert_filter_denies(...)` | `bf.Asserts.AssertFilterDenies(ctx, ...)` |
| `BatfishException` | `error` (`errors.As` against `*exception.BatfishError`) |
| `from_dict(d)` / `dict()` | `datamodel.XxxFromDict(d)` / `x.Dict()` |

## Packages

| pybatfish | gobatfish |
|---|---|
| `pybatfish` | `gobatfish` (version metadata) |
| `pybatfish.client` | `gobatfish/client` |
| `pybatfish.datamodel` | `gobatfish/datamodel` |
| `pybatfish.datamodel.answer` | `gobatfish/datamodel/answer` |
| `pybatfish.question` | `gobatfish/question` |
| `pybatfish.exception` | `gobatfish/exception` |
| `pybatfish.util` | `gobatfish/util` |
| `pandas.DataFrame` | `gobatfish/dataframe` |

`gobatfish/dataframe` is a small, dependency-free, object-typed data frame that
implements the subset of pandas semantics pybatfish relies on. Its columns are
`dtype=object`: every cell may hold `nil`, a primitive, a slice, or a datamodel
struct.

## Documentation

- [Getting started](docs/getting_started.md)
- [Questions and answers](docs/questions.md)
- [Datamodel classes](docs/datamodel.md)
- [Assertions](docs/asserts.md)
- [Data frame](docs/dataframe.md)

## Testing

Unit tests do not need a server:

```bash
go test ./...
```

Integration tests run against a real Batfish `allinone` container:

```bash
docker run --name batfish -p 9996:9996 -p 9997:9997 batfish/allinone
BATFISH_INTEGRATION=1 go test ./client/ -run Integration -v
```

Set `BATFISH_HOST` to point at a remote server.

## Differences from pybatfish

- **Errors instead of exceptions.** Failures are returned as `error` values.
  Use `errors.As` with `*exception.BatfishError`, `*exception.BatfishAssertError`,
  or `*exception.QuestionValidationError`. Soft assertions log a warning and
  return `(false, nil)`.
- **`context.Context`** is the first argument of every network call, so requests
  can be cancelled and timeouts controlled per call.
- **Options structs** replace Python keyword arguments.
- **Question access** goes through `session.Q.Get(name)`; there are no dynamic
  attributes.
- **Capirca**: Capirca is a Python library. `gobatfish` exposes
  `CreateReferenceBookFromDefinitions` for an already-parsed definitions model
  and `InitSnapshotFromACL` for already-rendered ACL text.
- **MCP server**: not ported.

## License

Apache License 2.0. This library is a Go port of
[pybatfish](https://github.com/batfish/pybatfish) and derives from the
[Batfish](https://github.com/batfish/batfish) data model, both of which are
licensed under Apache 2.0. See [LICENSE](LICENSE) and [NOTICE](NOTICE).

The Go implementation was written from scratch against the pybatfish public
API. The Python source was used as the reference for behavior, naming and
documentation, but was not copied verbatim.
