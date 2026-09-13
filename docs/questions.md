# Questions and answers

This page is the Go counterpart of the pybatfish *interacting*, *snapshot*, and
*filters* notebooks.

## Loading questions

Question templates are loaded from the Batfish service. Go has no dynamic
attributes, so questions are addressed through `Session.Q`:

```go
bf := client.NewSession(client.SessionConfig{Host: "localhost"})
if err := bf.LoadQuestions(ctx); err != nil {
	log.Fatal(err)
}

fmt.Println(bf.Q.Names())            // all question names
fmt.Println(bf.Q.ListTags())         // all tags
fmt.Println(bf.Q.List("bgp"))        // questions tagged "bgp"
```

You can also load templates from a directory:

```go
if err := bf.LoadQuestionsFromDir(ctx, "questions/"); err != nil {
	log.Fatal(err)
}
```

## Asking questions

A question is configured with `Set` (variables) and answered with `Answer`.
`Answer` returns an `answer.Result`. Table answers expose a data frame.

```go
q, err := bf.Q.Get("initIssues")
if err != nil {
	log.Fatal(err)
}

ans, err := q.Answer(ctx, question.AnswerOptions{Snapshot: bf.Snapshot})
if err != nil {
	log.Fatal(err)
}

table, ok := ans.Table()
if !ok {
	log.Fatal("not a table answer")
}
result := table.Frame()
fmt.Println(result.String())
```

Question variables accept the same values as pybatfish, including datamodel
objects:

```go
q, _ := bf.Q.Get("searchFilters")
q.Set("headers", datamodel.HeaderConstraints{
	SrcIps:      "10.10.10.0/24",
	DstIps:      "218.8.104.58",
	Applications: []any{"dns"},
})
q.Set("action", "deny")
q.Set("filters", "acl_in")

ans, err := q.Answer(ctx, question.AnswerOptions{Snapshot: datamodel.StringPtr("filters")})
```

## Filtering answers

The data frame mirrors the pandas operations used by the pybatfish notebooks:

```go
df := table.Frame()

// issues[issues['Details'].apply(lambda x: "foo" not in x)]
filtered := df.Filter(df.Col("Details").Map(func(v any) any {
	return !strings.Contains(fmt.Sprint(v), "foo")
}))

// result.iloc[0]
first := df.Row(0)
fmt.Println(first.Get("Details"))
```

## Differential questions

```go
q, _ := bf.Q.Get("routes")
ans, err := q.Answer(ctx, question.AnswerOptions{
	Snapshot:          datamodel.StringPtr("new"),
	ReferenceSnapshot: datamodel.StringPtr("old"),
})
```

## Asynchronous questions

```go
ans, err := q.Answer(ctx, question.AnswerOptions{Snapshot: bf.Snapshot, Background: true})
// ans is an Answer holding the work item id under "workItemId".
```

## Available questions

The set of available questions and their variables depends on the Batfish
version. List them at runtime:

```go
for _, q := range bf.Q.List() {
	fmt.Println(q["name"], q["description"])
}
```
