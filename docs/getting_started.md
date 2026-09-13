# Getting started

## Installing Batfish and gobatfish

Getting started with Batfish is easy. First, pull and run the latest `allinone`
Docker container:

```bash
docker pull batfish/allinone
docker run --name batfish -v batfish-data:/data -p 8888:8888 -p 9996:9996 batfish/allinone
```

Then install the Go client:

```bash
go get github.com/81ueman/gobatfish
```

`gobatfish` requires Go 1.25 or later.

## Upgrading

To upgrade to the latest Docker container, issue these commands on the Batfish
server:

```bash
docker stop batfish
docker rm batfish
docker pull batfish/allinone
docker run --name batfish -v batfish-data:/data -p 8888:8888 -p 9996:9996 batfish/allinone
```

To upgrade the Go client:

```bash
go get -u github.com/81ueman/gobatfish
```

We recommend that you upgrade Batfish and the client together.

## Connecting

```go
bf := client.NewSession(client.SessionConfig{Host: "localhost"})
defer bf.Close()
```

`SessionConfig` mirrors the pybatfish `Session` constructor:

| Field | Meaning | Default |
|---|---|---|
| `Host` | Batfish coordinator host | `localhost` |
| `Port` / `PortV2` | v2 coordinator port | `9996` |
| `SSL` | use HTTPS | `false` |
| `VerifySSLCerts` | verify TLS certificates | `true` |
| `APIKey` | API key | the Batfish default key |
| `Proxies` | per-scheme proxy URLs | none |
| `Timeout` | per-request timeout | `30s` (a zero value disables it) |
| `RequestKwargs` | extra request options | none |

## Your first snapshot

```go
ctx := context.Background()

if _, err := bf.SetNetwork(ctx, "example_dc"); err != nil {
	log.Fatal(err)
}

// Initialize a snapshot from a directory of configurations.
name, err := bf.InitSnapshot(ctx, "configs/", client.InitSnapshotOptions{
	Name:      "snapshot-2020-01-01",
	Overwrite: true,
})
if err != nil {
	log.Fatal(err)
}
fmt.Println("initialized", name)
```

`InitSnapshotFromText` creates a single-file snapshot in memory, which is handy
for tests:

```go
_, err := bf.InitSnapshotFromText(ctx, configText, client.InitSnapshotFromTextOptions{
	Platform: "cisco",
	Filename: "r1.cfg",
})
```

## Asking a question

Question definitions live on the Batfish server and are loaded at runtime:

```go
if err := bf.LoadQuestions(ctx); err != nil {
	log.Fatal(err)
}

q, err := bf.Q.Get("initIssues")
if err != nil {
	log.Fatal(err)
}
ans, err := q.Answer(ctx, question.AnswerOptions{Snapshot: bf.Snapshot})
if err != nil {
	log.Fatal(err)
}
table, _ := ans.Table()
fmt.Println(table.Frame().String())
```
