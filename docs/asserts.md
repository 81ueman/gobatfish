# Assertion helpers

Utility assertions for writing network tests (or policies). They mirror
`pybatfish.client.asserts`.

All assertion functions return `(bool, error)`. A hard failure returns
`false` and a `*exception.BatfishAssertError`; a soft failure logs a warning and
returns `(false, nil)`. Successful assertions return `(true, nil)`.

Every assertion is available both as a package-level function (taking a session)
and as a method on `Session.Asserts`.

| Function | Method | Checks |
|---|---|---|
| `AssertFilterDenies` | `AssertFilterDenies` | a filter denies a set of flows |
| `AssertFilterHasNoUnreachableLines` | `AssertFilterHasNoUnreachableLines` | a filter has no unreachable lines |
| `AssertFilterPermits` | `AssertFilterPermits` | a filter permits a set of flows |
| `AssertFlowsFail` | `AssertFlowsFail` | flows fail |
| `AssertFlowsSucceed` | `AssertFlowsSucceed` | flows succeed |
| `AssertHasRoute` / `AssertHasNoRoute` | — | a route is present / absent |
| `AssertNoDuplicateRouterIDs` | `AssertNoDuplicateRouterIDs` | no duplicate router IDs |
| `AssertNoForwardingLoops` | `AssertNoForwardingLoops` | no forwarding loops |
| `AssertNoIncompatibleBGPSessions` | `AssertNoIncompatibleBGPSessions` | no incompatible BGP sessions |
| `AssertNoIncompatibleOSPFSessions` | `AssertNoIncompatibleOSPFSessions` | no incompatible OSPF sessions |
| `AssertNoUnestablishedBGPSessions` | `AssertNoUnestablishedBGPSessions` | no unestablished BGP sessions |
| `AssertNoUndefinedReferences` | `AssertNoUndefinedReferences` | no undefined references |
| `AssertNumResults` / `AssertZeroResults` | — | an exact number of results |

## Examples

```go
ok, err := bf.Asserts.AssertNoUndefinedReferences(ctx, false, nil, "table")
if err != nil {
	log.Fatal(err)
}
```

```go
headers := datamodel.HeaderConstraints{SrcIps: "10.0.0.0/8"}
_ = headers
ok, err := bf.Asserts.AssertFilterDenies(ctx, "acl_in", headers, nil, false, nil, "table")
```

### Soft assertions

Soft assertions do not return an error; they log a warning and return
`(false, nil)`:

```go
pass, err := bf.Asserts.AssertNoForwardingLoops(ctx, nil, true, "records")
if err != nil {
	log.Fatal(err)
}
if !pass {
	log.Println("forwarding loops detected (soft)")
}
```

### Routes

`AssertHasRoute` and `AssertHasNoRoute` accept either a data frame or a
multilevel map (`node -> vrf -> []route`):

```go
routes := map[string]map[string][]map[string]any{
	"n1": {"vrf1": {{"Network": "10.10.10.0/24"}}},
}
if _, err := client.AssertHasRoute(routes, map[string]any{"Network": "10.10.10.0/24"}, "n1", "vrf1", false); err != nil {
	log.Fatal(err)
}
```

### Output format

The `df_format` argument controls how found rows are rendered in the message:
`"table"` (default) or `"records"`.
