package datamodel

import (
	"reflect"
	"testing"
)

func TestInterfaceStrAndHTML(t *testing.T) {
	if got := (Interface{Hostname: "node", Interface: "iface"}).String(); got != "node[iface]" {
		t.Fatalf("str = %q", got)
	}
	if got := (Interface{Hostname: "0node", Interface: "iface"}).String(); got != `"0node"[iface]` {
		t.Fatalf("str = %q", got)
	}
	if got := (Interface{Hostname: "node", Interface: "/iface"}).String(); got != `node["/iface"]` {
		t.Fatalf("str = %q", got)
	}
	if got := (Interface{Hostname: "host", Interface: "special&"}).HTML(); got != "host[&quot;special&amp;&quot;]" {
		t.Fatalf("html = %q", got)
	}
	if got := (Interface{Hostname: "host", Interface: "normal:0/0.0"}).HTML(); got != "host[normal:0/0.0]" {
		t.Fatalf("html = %q", got)
	}
}

func TestEdgeDictConvertsInterface(t *testing.T) {
	e := NewEdge("r1", "iface1", "r2", Interface{Hostname: "r2", Interface: "iface2"})
	want := map[string]any{
		"node1": "r1", "node1interface": "iface1", "node2": "r2", "node2interface": "iface2",
	}
	if got := e.Dict(); !reflect.DeepEqual(got, want) {
		t.Fatalf("edge dict = %v, want %v", got, want)
	}
}

func TestFileLines(t *testing.T) {
	fl := FileLinesFromDict(map[string]any{"filename": "myfile", "lines": []any{2, 3}})
	if fl.Filename != "myfile" || !reflect.DeepEqual(fl.Lines, []int{2, 3}) {
		t.Fatalf("filelines = %+v", fl)
	}
	empty := FileLinesFromDict(map[string]any{"filename": "myfile"})
	if len(empty.Lines) != 0 {
		t.Fatalf("expected no lines, got %v", empty.Lines)
	}
}

func TestAutoCompleteSuggestion(t *testing.T) {
	s := AutoCompleteSuggestionFromDict(map[string]any{
		"description": "desc", "hint": "hint", "insertionIndex": 16, "isPartial": true, "rank": 42, "text": "suggestion",
	})
	if s.Description == nil || *s.Description != "desc" || s.InsertionIndex != 16 || !s.IsPartial || s.Rank != 42 {
		t.Fatalf("suggestion = %+v", s)
	}
	s = AutoCompleteSuggestionFromDict(map[string]any{"isPartial": true, "rank": 42, "text": "x"})
	if s.Description != nil || s.Hint != nil || s.InsertionIndex != 0 {
		t.Fatalf("optional fields not defaulted: %+v", s)
	}
}

func flowDict() map[string]any {
	return map[string]any{
		"dscp": 0, "dstIp": "2.1.1.1", "dstPort": 0, "ecn": 0, "fragmentOffset": 0,
		"icmpCode": 255, "icmpVar": 255, "ingressNode": "ingress", "ipProtocol": "UNNAMED_168",
		"packetLength": 512, "srcIp": "5.5.1.1", "srcPort": 0,
		"tcpFlagsAck": 0, "tcpFlagsCwr": 0, "tcpFlagsEce": 0, "tcpFlagsFin": 0,
		"tcpFlagsPsh": 0, "tcpFlagsRst": 0, "tcpFlagsSyn": 0, "tcpFlagsUrg": 0,
	}
}

func TestFlowString(t *testing.T) {
	f := FlowFromDict(flowDict())
	if got := f.String(); got != "start=ingress [5.5.1.1->2.1.1.1 ipProtocol=168]" {
		t.Fatalf("flow str = %q", got)
	}
	d := flowDict()
	d["ipProtocol"] = "TCP"
	d["dstPort"] = "80"
	d["srcPort"] = "800"
	f = FlowFromDict(d)
	if got := f.String(); got != "start=ingress [5.5.1.1:800->2.1.1.1:80 TCP (no flags set)]" {
		t.Fatalf("flow str = %q", got)
	}
	d["dscp"] = "1"
	d["ecn"] = "1"
	d["fragmentOffset"] = "1501"
	d["packetLength"] = "100"
	f = FlowFromDict(d)
	want := "start=ingress [5.5.1.1:800->2.1.1.1:80 TCP (no flags set) dscp=1 ecn=1 fragmentOffset=1501 length=100]"
	if got := f.String(); got != want {
		t.Fatalf("flow str = %q, want %q", got, want)
	}
}

func TestFlowFlagsAndProtocol(t *testing.T) {
	d := flowDict()
	d["ipProtocol"] = "ICMP"
	d["icmpCode"] = 0
	d["icmpVar"] = 8
	f := FlowFromDict(d)
	if got := f.GetFlagStr(); got != "no flags set" {
		t.Fatalf("flags = %q", got)
	}
	if got := f.GetIPProtocolStr(); got != "ICMP (type=8, code=0)" {
		t.Fatalf("proto = %q", got)
	}
	d["tcpFlagsAck"] = 1
	d["tcpFlagsSyn"] = 1
	f = FlowFromDict(d)
	if got := f.GetFlagStr(); got != "SYN-ACK" {
		t.Fatalf("flags = %q", got)
	}
}

func TestHeaderConstraints(t *testing.T) {
	hc := HeaderConstraints{}
	d := hc.Dict()
	for _, key := range []string{"srcIps", "dstIps", "srcPorts", "dstPorts", "ipProtocols", "applications",
		"icmpCodes", "icmpTypes", "ecns", "dscps", "packetLengths", "fragmentOffsets", "tcpFlags"} {
		if d[key] != nil {
			t.Fatalf("expected nil for %s, got %v", key, d[key])
		}
	}
	dst := HeaderConstraints{DstPorts: []any{"10-20", "33-33"}}
	if got := dst.Dict()["dstPorts"]; got != "10-20,33-33" {
		t.Fatalf("dstPorts = %v", got)
	}
	apps := HeaderConstraints{Applications: "dns,ssh"}
	if got := apps.Dict()["applications"]; !reflect.DeepEqual(got, []any{"dns", "ssh"}) {
		t.Fatalf("applications = %v", got)
	}
	protos := HeaderConstraints{IpProtocols: "  ,  dns,"}
	if got := protos.Dict()["ipProtocols"]; !reflect.DeepEqual(got, []any{"dns"}) {
		t.Fatalf("ipProtocols = %v", got)
	}
	if _, err := (HeaderConstraints{Applications: ""}).Normalize(); err == nil {
		t.Fatal("expected error for empty applications")
	}
}

func TestHeaderConstraintsOf(t *testing.T) {
	f := Flow{IPProtocol: "ICMP", IcmpCode: IntPtr(7), SrcIP: "1.1.1.1", DstIP: "2.2.2.2", PacketLength: 0}
	hc := HeaderConstraintsOf(f)
	if hc.SrcIps != "1.1.1.1" || hc.DstIps != "2.2.2.2" {
		t.Fatalf("ips = %v %v", hc.SrcIps, hc.DstIps)
	}
	if !reflect.DeepEqual(hc.IpProtocols, []any{"ICMP"}) {
		t.Fatalf("ipProtocols = %v", hc.IpProtocols)
	}
	if hc.IcmpCodes != "7" {
		t.Fatalf("icmpCodes = %v", hc.IcmpCodes)
	}
	if hc.SrcPorts != nil || hc.DstPorts != nil || hc.TcpFlags != nil {
		t.Fatalf("unexpected ports/flags: %+v", hc)
	}
}

func TestRouteInfoAndForwarding(t *testing.T) {
	ri := RouteInfo{Protocol: "tcp", Network: "1.1.1.1/32", NextHop: NextHopDiscard{}, Admin: 1, Metric: 2}
	if got := ri.String(); got != "tcp (Network: 1.1.1.1/32, Next Hop: discard)" {
		t.Fatalf("routeinfo = %q", got)
	}
	want := map[string]any{
		"protocol": "tcp", "network": "1.1.1.1/32", "nextHop": map[string]any{"type": "discard"},
		"nextHopIp": nil, "admin": 1, "metric": 2,
	}
	if got := ri.Dict(); !reflect.DeepEqual(got, want) {
		t.Fatalf("routeinfo dict = %v, want %v", got, want)
	}
	legacy := RouteInfo{Protocol: "tcp", Network: "1.1.1.1/32", NextHopIP: StringPtr("2.2.2.2"), Admin: 1, Metric: 2}
	if got := legacy.String(); got != "tcp (Network: 1.1.1.1/32, Next Hop IP:2.2.2.2)" {
		t.Fatalf("legacy routeinfo = %q", got)
	}
}

func TestNextHop(t *testing.T) {
	nh, err := NextHopFromDict(map[string]any{"type": "discard"})
	if err != nil || !reflect.DeepEqual(nh, NextHopDiscard{}) {
		t.Fatalf("discard = %v, %v", nh, err)
	}
	nh, err = NextHopFromDict(map[string]any{"type": "interface", "interface": "foo"})
	if err != nil {
		t.Fatal(err)
	}
	if got := nh.String(); got != "interface foo" {
		t.Fatalf("nh str = %q", got)
	}
	nh, _ = NextHopFromDict(map[string]any{"type": "interface", "interface": "foo bar"})
	if got := nh.String(); got != `interface "foo bar"` {
		t.Fatalf("nh str = %q", got)
	}
	if _, err := NextHopFromDict(map[string]any{"type": "foo"}); err == nil {
		t.Fatal("expected error for unknown type")
	}
}

func TestStepAndHop(t *testing.T) {
	step := Step{Detail: EnterInputIfaceStepDetail{InputInterface: "in_iface1", InputVrf: StringPtr("in_vrf1")}, Action: "SENT_IN"}
	if got := step.String(); got != "SENT_IN(in_iface1)" {
		t.Fatalf("step = %q", got)
	}
	hop := Hop{Node: "node1", Steps: []Step{
		{Detail: EnterInputIfaceStepDetail{InputInterface: "in_iface1", InputVrf: StringPtr("in_vrf1")}, Action: "SENT_IN"},
		{Detail: RoutingStepDetail{
			Routes: []RouteInfo{
				{Protocol: "bgp", Network: "1.1.1.1/24", NextHop: NextHopIP{IP: "1.2.3.4"}, Admin: 1, Metric: 1},
				{Protocol: "static", Network: "1.1.1.2/24", NextHop: NextHopIP{IP: "1.2.3.5"}, Admin: 1, Metric: 1},
			},
			ForwardingDetail: ForwardedOutInterface{OutputInterface: "iface1", ResolvedNextHopIP: StringPtr("12.123.1.2")},
			ArpIP:            StringPtr("12.123.1.2"),
			OutputInterface:  StringPtr("iface1"),
		}, Action: "FORWARDED"},
		{Detail: FilterStepDetail{Filter: "preSourceNat_filter", FilterType: "PRENAT"}, Action: "PERMITTED"},
		{Detail: ExitOutputIfaceStepDetail{OutputInterface: "out_iface1"}, Action: "SENT_OUT"},
	}}
	want := "node: node1\n  SENT_IN(in_iface1)\n" +
		"  FORWARDED(Forwarded out interface: iface1 with resolved next-hop IP: 12.123.1.2, Routes: [bgp (Network: 1.1.1.1/24, Next Hop: ip 1.2.3.4),static (Network: 1.1.1.2/24, Next Hop: ip 1.2.3.5)])\n  " +
		"PERMITTED(preSourceNat_filter (PRENAT))\n  SENT_OUT(out_iface1)"
	if got := hop.String(); got != want {
		t.Fatalf("hop =\n%q\nwant\n%q", got, want)
	}
}

func TestStepFromDictLoop(t *testing.T) {
	step := StepFromDict(map[string]any{"type": "Loop", "action": "LOOP", "detail": map[string]any{}})
	if step == nil || step.Action != "LOOP" || !reflect.DeepEqual(step.Detail, LoopStepDetail{}) {
		t.Fatalf("loop step = %+v", step)
	}
	if got := StepFromDict(map[string]any{"type": "Unknown"}); got != nil {
		t.Fatal("expected nil for unknown type")
	}
}

func TestSetupSessionStepDetail(t *testing.T) {
	d := map[string]any{
		"sessionScope":   map[string]any{"incomingInterfaces": []any{"reth0.6"}},
		"sessionAction":  map[string]any{"type": "Accept"},
		"matchCriteria":  map[string]any{"ipProtocol": "ICMP", "srcIp": "2.2.2.2", "dstIp": "3.3.3.3"},
		"transformation": []any{map[string]any{"fieldName": "srcIp", "oldValue": "2.2.2.2", "newValue": "1.1.1.1"}},
	}
	got := SetupSessionStepDetailFromDict(d)
	want := SetupSessionStepDetail{
		SessionScope:   IncomingSessionScope{IncomingInterfaces: []string{"reth0.6"}},
		SessionAction:  Accept{},
		MatchCriteria:  SessionMatchExpr{IPProtocol: "ICMP", SrcIP: "2.2.2.2", DstIP: "3.3.3.3"},
		Transformation: []FlowDiff{{FieldName: "srcIp", OldValue: "2.2.2.2", NewValue: "1.1.1.1"}},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("setup detail = %+v, want %+v", got, want)
	}
	wantStr := "Incoming Interfaces: [reth0.6], Action: Accept, " +
		"Match Criteria: [ipProtocol=ICMP, srcIp=2.2.2.2, dstIp=3.3.3.3], Transformation: [srcIp: 2.2.2.2 -> 1.1.1.1]"
	if gotStr := got.String(); gotStr != wantStr {
		t.Fatalf("setup str = %q, want %q", gotStr, wantStr)
	}
}

func TestTraceTree(t *testing.T) {
	textFragment := "org.batfish.datamodel.TraceElement$TextFragment"
	noChildren := TraceTreeFromDict(map[string]any{
		"traceElement": map[string]any{"fragments": []any{map[string]any{"class": textFragment, "text": "aaa"}}},
	})
	if len(noChildren.Children) != 0 || noChildren.String() != "aaa" || noChildren.HTML() != "aaa" {
		t.Fatalf("no children tree = %+v", noChildren)
	}
	tree := TraceTreeFromDict(map[string]any{
		"traceElement": map[string]any{"fragments": []any{map[string]any{"class": textFragment, "text": "aaa"}}},
		"children": []any{
			map[string]any{"traceElement": map[string]any{"fragments": []any{map[string]any{"class": textFragment, "text": "bbb"}}}},
			map[string]any{"traceElement": map[string]any{"fragments": []any{map[string]any{"class": textFragment, "text": "ccc"}}}},
		},
	})
	want := "aaa\n  - bbb\n  - ccc"
	if got := tree.String(); got != want {
		t.Fatalf("tree str = %q, want %q", got, want)
	}
	if html := tree.HTML(); html == "" {
		t.Fatal("empty html")
	}
}

func TestFlowTraceHopNoTransformed(t *testing.T) {
	h := FlowTraceHopFromDict(map[string]any{
		"edge":   map[string]any{"node1": "dummy", "node1interface": "eth0", "node2": "router1", "node2interface": "Ethernet1"},
		"routes": []any{"r1"}, "transformedFlow": nil,
	})
	if h.TransformedFlow != nil || len(h.Routes) != 1 {
		t.Fatalf("hop = %+v", h)
	}
}

func TestBgpRoute(t *testing.T) {
	r := BgpRoute{
		Network: "0.0.0.0/0", AsPath: []any{1, 2, 3}, Communities: []any{4, 5, 6},
		LocalPreference: 1, Metric: 2, NextHopIP: StringPtr("2.2.2.2"), OriginatorIP: "1.1.1.1",
		OriginType: "egp", Protocol: "bgp", SourceProtocol: StringPtr("connected"), Tag: 23, Weight: 42,
	}
	d := r.Dict()
	if d["class"] != "org.batfish.datamodel.questions.BgpRoute" || d["srcProtocol"] != "connected" {
		t.Fatalf("bgp route dict = %v", d)
	}
	lines := r.HTMLLines()
	if lines[0] != "Network: 0.0.0.0/0" || lines[1] != "AS Path: [1, 2, 3]" || lines[2] != "Communities: [4, 5, 6]" {
		t.Fatalf("bgp lines = %v", lines)
	}
}

func TestReferenceLibrary(t *testing.T) {
	if _, err := NewAddressGroup("g1", AddressGroup{Name: "g1"}, nil); err == nil {
		t.Fatal("expected bad type error")
	}
	ag, err := NewAddressGroup("g1", "ag", nil)
	if err != nil || !reflect.DeepEqual(ag.Addresses, []string{"ag"}) {
		t.Fatalf("address group = %+v, %v", ag, err)
	}
	ig, err := NewInterfaceGroup("g1", []any{Interface{Hostname: "h1", Interface: "i1"}})
	if err != nil || len(ig.Interfaces) != 1 {
		t.Fatalf("interface group = %+v, %v", ig, err)
	}
	book, err := NewReferenceBook("b1", []any{AddressGroup{Name: "ag"}}, nil)
	if err != nil || len(book.AddressGroups) != 1 {
		t.Fatalf("reference book = %+v, %v", book, err)
	}
	lib, err := NewReferenceLibrary(book)
	if err != nil || len(lib.Books) != 1 {
		t.Fatalf("reference library = %+v, %v", lib, err)
	}
	rm := RoleMappingFromDict(map[string]any{"regex": "re", "roleDimensionGroups": map[string]any{"dim1": []any{1}}})
	if rm.Name != nil || !reflect.DeepEqual(rm.RoleDimensionGroups, map[string][]int{"dim1": {1}}) {
		t.Fatalf("role mapping = %+v", rm)
	}
}
