# gobatfish

`gobatfish` is a Go client library for [Batfish](https://github.com/batfish/batfish)
that mirrors the public API of [pybatfish](https://github.com/batfish/pybatfish).

With `gobatfish` you can create snapshots, ask questions that are loaded
dynamically from the Batfish service, inspect table answers, run assertions,
and extract facts — all from idiomatic Go.

## Install

```bash
go get github.com/81ueman/gobatfish
```

## Quick start

```go
package main

import (
	"context"
	"log"

	"github.com/81ueman/gobatfish/client"
	"github.com/81ueman/gobatfish/question"
)

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

	q, err := bf.Q.Get("nodeProperties")
	if err != nil {
		log.Fatal(err)
	}
	q.Set("nodes", ".*")

	ans, err := q.Answer(ctx, question.AnswerOptions{Snapshot: bf.Snapshot})
	if err != nil {
		log.Fatal(err)
	}
	table, _ := ans.Table()
	log.Println(table.Frame().String())
}
```

## Where to go next

- [Getting started](getting_started.md) — connecting, snapshots, questions.
- [Questions and answers](questions.md) — the question registry and data frames.
- [Datamodel classes](datamodel.md) — HeaderConstraints, flows, routes, references.
- [Assertions](asserts.md) — network validation helpers.
- [Data frame](dataframe.md) — the pandas-like table API.
- [MCP server](mcp.md) — expose Batfish to AI agents over MCP.
- [API reference](https://pkg.go.dev/github.com/81ueman/gobatfish) — pkg.go.dev.

## Package layout

| pybatfish | gobatfish |
|---|---|
| `pybatfish.client` | `gobatfish/client` |
| `pybatfish.datamodel` | `gobatfish/datamodel` |
| `pybatfish.question` | `gobatfish/question` |
| `pybatfish.mcp` | `gobatfish/mcp` |
| `pandas.DataFrame` | `gobatfish/dataframe` |

## License

Apache License 2.0. `gobatfish` is a Go port of pybatfish and derives from the
Batfish data model, both licensed under Apache 2.0.
