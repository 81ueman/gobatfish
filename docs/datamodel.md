# Datamodel classes

Here we describe the Go types used in answers and their attributes, which may
help you filter your answers as desired. They mirror
`pybatfish.datamodel`.

Every element is a plain Go struct with exported fields whose `json` tags match
the Batfish wire format. Each type provides:

- `XxxFromDict(map[string]any) Xxx` — the analogue of `from_dict`.
- `Dict() map[string]any` — the analogue of `dict()`.
- `String()` and, where applicable, `HTML()` — the analogue of `__str__` and
  `_repr_html_`.

Because the types are JSON-native, you can also use `encoding/json` directly.

## Base types

```go
iface := datamodel.Interface{Hostname: "node", Interface: "iface"}
fmt.Println(iface.String())          // node[iface]
fmt.Println(iface.Dict())            // map[hostname:node interface:iface]

edge := datamodel.NewEdge("r1", "iface1", "r2", iface)
```

`Edge` accepts either a string or an `Interface` for its interface fields, just
like the Python converter.

`ListWrapper` holds an ordered list of answer values.

## ACL traces

`AclTrace`, `AclTraceEvent`, `TextFragment`, `LinkFragment`, `TraceElement`,
`TraceTree`, and `TraceTreeList` render ACL traces.

```go
tree := datamodel.TraceTreeFromDict(dict)
fmt.Println(tree.String())
fmt.Println(tree.HTML())
```

## Flows and packets

`Flow`, `FlowTrace`, `Hop`, `Step`, `HeaderConstraints`, `PathConstraints`,
`TcpFlags`, `MatchTcpFlags`, and the various step-detail types are available.

```go
flow := datamodel.FlowFromDict(dict)
fmt.Println(flow.String())
fmt.Println(flow.GetFlagStr())

hc := datamodel.HeaderConstraints{
	SrcIps:       "1.1.1.1",
	IpProtocols:  []any{"TCP"},
	DstPorts:     "10-20,33-33",
}
normalized, err := hc.Normalize()
```

`HeaderConstraints` fields are `any` because pybatfish accepts polymorphic
inputs (a single port, a list, or a comma separated string). Call `Normalize`
(or `Dict`, which normalizes) to apply the same conversions.

## Reference library

`AddressGroup`, `InterfaceGroup`, `RoleMapping`, `NodeRolesData`,
`ReferenceBook`, and `ReferenceLibrary` model the reference library. Validated
constructors mirror the Python converters:

```go
group, err := datamodel.NewAddressGroup("g1", "ag", []any{"child"})
book, err := datamodel.NewReferenceBook("book1", group, nil)
library, err := datamodel.NewReferenceLibrary(book)
```

## Routes

`BgpRoute`, `BgpRouteConstraints`, `BgpRouteDiff`, `BgpRouteDiffs`,
`BgpSessionProperties`, and the `NextHop` family (`NextHopDiscard`,
`NextHopInterface`, `NextHopIP`, `NextHopVrf`, `NextHopVtep`) are available.

```go
nextHop, err := datamodel.NextHopFromDict(map[string]any{"type": "ip", "ip": "1.1.1.1"})
route := datamodel.RouteInfo{
	Protocol: "bgp",
	Network:  "1.1.1.1/32",
	NextHop:  nextHop,
	Admin:    1,
	Metric:   2,
}
fmt.Println(route.String())
```

## Misc question outputs

`FileLines` represents a set of lines in a file, and `AutoCompleteSuggestion`
represents one autocomplete suggestion.

## Assertions

`Assertion` and `AssertionType` are used to turn a question into a check:

```go
q.MakeCheck() // asserts that there are no results
q.SetAssertion(datamodel.Assertion{Type: datamodel.AssertionCountEquals, Expect: 0})
```

## Enums

`VariableType` enumerates the question variable types used for autocompletion.
