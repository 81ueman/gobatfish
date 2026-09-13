package datamodel

import (
	"testing"
)

// Ports of tests/datamodel/test_flow.py.

// flowTestDict mirrors the dictionary used by the Flow string/HTML tests.
func flowTestDict() map[string]any {
	return map[string]any{
		"dscp": 0, "dstIp": "2.1.1.1", "dstPort": 0, "ecn": 0, "fragmentOffset": 0,
		"icmpCode": 0, "icmpVar": 0, "ingressNode": "ingress", "ipProtocol": "UNNAMED_168",
		"packetLength": 512, "srcIp": "5.5.1.1", "srcPort": 0,
		"tcpFlagsAck": 0, "tcpFlagsCwr": 0, "tcpFlagsEce": 0, "tcpFlagsFin": 0,
		"tcpFlagsPsh": 0, "tcpFlagsRst": 0, "tcpFlagsSyn": 0, "tcpFlagsUrg": 0,
	}
}

func TestArpErrorStepDetailStr(t *testing.T) {
	detail := ArpErrorStepDetail{OutputInterface: StringPtr("iface"), ResolvedNexthopIP: StringPtr("1.1.1.1")}
	step := Step{Detail: detail, Action: "ACTION"}
	assertStr(t, step.String(), "ACTION(Output Interface: iface, Resolved Next Hop IP: 1.1.1.1)")
}

func TestArpErrorStepDetailDeserialization(t *testing.T) {
	json := map[string]any{"outputInterface": map[string]any{"interface": "iface"}, "resolvedNexthopIp": "1.1.1.1"}
	detail := ArpErrorStepDetailFromDict(json)
	assertDeepEqual(t, detail, ArpErrorStepDetail{OutputInterface: StringPtr("iface"), ResolvedNexthopIP: StringPtr("1.1.1.1")})
}

func TestDeliveredStepDetailStr(t *testing.T) {
	detail := DeliveredStepDetail{OutputInterface: StringPtr("iface"), ResolvedNexthopIP: StringPtr("1.1.1.1")}
	step := Step{Detail: detail, Action: "ACTION"}
	assertStr(t, step.String(), "ACTION(Output Interface: iface, Resolved Next Hop IP: 1.1.1.1)")
}

func TestDeliveredStepDetailDeserialization(t *testing.T) {
	json := map[string]any{"outputInterface": map[string]any{"interface": "iface"}, "resolvedNexthopIp": "1.1.1.1"}
	detail := DeliveredStepDetailFromDict(json)
	assertDeepEqual(t, detail, DeliveredStepDetail{OutputInterface: StringPtr("iface"), ResolvedNexthopIP: StringPtr("1.1.1.1")})
}

func TestExitOutputIfaceStepDetailStr(t *testing.T) {
	detail := ExitOutputIfaceStepDetail{OutputInterface: "iface"}
	step := Step{Detail: detail, Action: "ACTION"}
	assertStr(t, step.String(), "ACTION(iface)")
}

func TestTransformationStepDetailStr(t *testing.T) {
	noDiffs := TransformationStepDetail{TransformationType: "type", FlowDiffs: []FlowDiff{}}
	oneDiff := TransformationStepDetail{TransformationType: "type", FlowDiffs: []FlowDiff{{FieldName: "field", OldValue: "old", NewValue: "new"}}}
	twoDiffs := TransformationStepDetail{TransformationType: "type", FlowDiffs: []FlowDiff{
		{FieldName: "field1", OldValue: "old1", NewValue: "new1"},
		{FieldName: "field2", OldValue: "old2", NewValue: "new2"},
	}}

	assertStr(t, (Step{Detail: noDiffs, Action: "ACTION"}).String(), "ACTION(type)")
	assertStr(t, (Step{Detail: oneDiff, Action: "ACTION"}).String(), "ACTION(type field: old -> new)")
	assertStr(t, (Step{Detail: twoDiffs, Action: "ACTION"}).String(),
		"ACTION(type field1: old1 -> new1, field2: old2 -> new2)")
}

func TestFilterStepDetail(t *testing.T) {
	jsonDict := map[string]any{
		"filter":         "ACL",
		"type":           "ingressAcl",
		"inputInterface": "iface",
		"flow": map[string]any{
			"dscp": 0, "dstIp": "2.1.1.1", "dstPort": 0, "ecn": 0, "fragmentOffset": 0,
			"icmpCode": 255, "icmpVar": 255, "ingressInterface": "intface", "ingressNode": "ingress",
			"ingressVrf": "vrfAbc", "ipProtocol": "IP", "packetLength": 0, "srcIp": "5.5.1.1", "srcPort": 0,
			"tcpFlagsAck": 0, "tcpFlagsCwr": 0, "tcpFlagsEce": 0, "tcpFlagsFin": 0,
			"tcpFlagsPsh": 0, "tcpFlagsRst": 0, "tcpFlagsSyn": 0, "tcpFlagsUrg": 0,
		},
	}

	detail := FilterStepDetailFromDict(jsonDict)

	assertStr(t, detail.Filter, "ACL")
	assertStr(t, detail.FilterType, "ingressAcl")
	assertStr(t, detail.InputInterface, "iface")

	flow := detail.Flow
	assertStr(t, flow.SrcIP, "5.5.1.1")
	assertStr(t, *flow.IngressInterface, "intface")
	assertStr(t, *flow.IngressVrf, "vrfAbc")
}

func TestFlowDeserialization(t *testing.T) {
	hopDict := map[string]any{
		"dscp": 0, "dstIp": "2.1.1.1", "dstPort": 0, "ecn": 0, "fragmentOffset": 0,
		"icmpCode": 255, "icmpVar": 255, "ingressInterface": "intface", "ingressNode": "ingress",
		"ingressVrf": "vrfAbc", "ipProtocol": "TCP", "packetLength": 0, "srcIp": "5.5.1.1", "srcPort": 0,
		"tcpFlagsAck": 0, "tcpFlagsCwr": 0, "tcpFlagsEce": 0, "tcpFlagsFin": 0,
		"tcpFlagsPsh": 0, "tcpFlagsRst": 0, "tcpFlagsSyn": 0, "tcpFlagsUrg": 0,
	}

	// check deserialization
	flow := FlowFromDict(hopDict)
	assertStr(t, flow.SrcIP, "5.5.1.1")
	assertStr(t, *flow.IngressInterface, "intface")
	assertStr(t, *flow.IngressVrf, "vrfAbc")

	// check the string representation has the essential elements (without forcing a strict format)
	flowStr := flow.String()
	assertContains(t, flowStr, "5.5.1.1")
	assertContains(t, flowStr, "intface")
	assertContains(t, flowStr, "vrfAbc")
}

// test if a flow is deserialized properly when the optional fields are missing
func TestFlowDeserializationOptionalMissing(t *testing.T) {
	hopDict := map[string]any{
		"dscp": 0, "dstIp": "2.1.1.1", "dstPort": 0, "ecn": 0, "fragmentOffset": 0,
		"icmpCode": 255, "icmpVar": 255, "ingressNode": "ingress",
		"ipProtocol": "TCP", "packetLength": 0, "srcIp": "5.5.1.1", "srcPort": 0,
		"tcpFlagsAck": 0, "tcpFlagsCwr": 0, "tcpFlagsEce": 0, "tcpFlagsFin": 0,
		"tcpFlagsPsh": 0, "tcpFlagsRst": 0, "tcpFlagsSyn": 0, "tcpFlagsUrg": 0,
	}
	// check deserialization
	flow := FlowFromDict(hopDict)
	assertStr(t, flow.SrcIP, "5.5.1.1")

	// should convert to string without problems
	_ = flow.String()
}

func TestFlowTraceHopNoTransformedFlow(t *testing.T) {
	// Test we don't crash on missing or None values.
	FlowTraceHopFromDict(map[string]any{
		"edge": map[string]any{
			"node1": "dummy", "node1interface": "eth0",
			"node2": "router1", "node2interface": "Ethernet1",
		},
		"filterIn":        nil,
		"filterOut":       nil,
		"routes":          []any{"StaticRoute<0.0.0.0/0,nhip:10.192.48.1,nhint:eth0>_fnhip:10.192.48.1"},
		"transformedFlow": nil,
	})

	FlowTraceHopFromDict(map[string]any{
		"edge": map[string]any{
			"node1": "dummy", "node1interface": "eth0",
			"node2": "router1", "node2interface": "Ethernet1",
		},
		"filterIn":  nil,
		"filterOut": nil,
		"routes":    []any{"StaticRoute<0.0.0.0/0,nhip:10.192.48.1,nhint:eth0>_fnhip:10.192.48.1"},
	})
}

func TestGetIPProtocolStrNodetail(t *testing.T) {
	flowDict := flowTestDict()
	flowDict["ipProtocol"] = "UNNAMED_243"
	assertStr(t, FlowFromDict(flowDict).GetIPProtocolStr(), "ipProtocol=243")

	// named
	flowDict["ipProtocol"] = "UDP"
	assertStr(t, FlowFromDict(flowDict).GetIPProtocolStr(), "UDP")
}

func TestGetIPProtocolStrTCP(t *testing.T) {
	flowDict := flowTestDict()
	flowDict["ipProtocol"] = "TCP"
	flowDict["dstPort"] = 80
	flowDict["srcPort"] = 80
	assertStr(t, FlowFromDict(flowDict).GetIPProtocolStr(), "TCP (no flags set)")

	// with flags
	flowDict["tcpFlagsAck"] = 1
	assertStr(t, FlowFromDict(flowDict).GetIPProtocolStr(), "TCP (ACK)")
}

func TestGetIPProtocolStrICMP(t *testing.T) {
	flowDict := flowTestDict()
	flowDict["ipProtocol"] = "ICMP"
	flowDict["icmpCode"] = 0
	flowDict["icmpVar"] = 8
	assertStr(t, FlowFromDict(flowDict).GetIPProtocolStr(), "ICMP (type=8, code=0)")
}

func TestGetFlagStr(t *testing.T) {
	flowDict := flowTestDict()
	flowDict["ipProtocol"] = "ICMP"
	flowDict["icmpCode"] = 0
	flowDict["icmpVar"] = 8
	assertStr(t, FlowFromDict(flowDict).GetFlagStr(), "no flags set")

	flowDict["tcpFlagsAck"] = 1
	assertStr(t, FlowFromDict(flowDict).GetFlagStr(), "ACK")

	flowDict["tcpFlagsSyn"] = 1
	assertStr(t, FlowFromDict(flowDict).GetFlagStr(), "SYN-ACK")

	flowDict["tcpFlagsCwr"] = 1
	assertStr(t, FlowFromDict(flowDict).GetFlagStr(), "SYN-ACK-CWR")

	flowDict["tcpFlagsEce"] = 1
	assertStr(t, FlowFromDict(flowDict).GetFlagStr(), "SYN-ACK-CWR-ECE")

	flowDict["tcpFlagsFin"] = 1
	assertStr(t, FlowFromDict(flowDict).GetFlagStr(), "SYN-FIN-ACK-CWR-ECE")

	flowDict["tcpFlagsPsh"] = 1
	assertStr(t, FlowFromDict(flowDict).GetFlagStr(), "SYN-FIN-ACK-CWR-ECE-PSH")

	flowDict["tcpFlagsRst"] = 1
	assertStr(t, FlowFromDict(flowDict).GetFlagStr(), "SYN-FIN-ACK-RST-CWR-ECE-PSH")

	flowDict["tcpFlagsUrg"] = 1
	assertStr(t, FlowFromDict(flowDict).GetFlagStr(), "SYN-FIN-ACK-RST-CWR-ECE-PSH-URG")
}

func TestFlowStr(t *testing.T) {
	flowDict := flowTestDict()
	// no ports
	assertStr(t, FlowFromDict(flowDict).String(), "start=ingress [5.5.1.1->2.1.1.1 ipProtocol=168]")

	// with ports
	flowDict["ipProtocol"] = "TCP"
	flowDict["dstPort"] = "80"
	flowDict["srcPort"] = "800"
	assertStr(t, FlowFromDict(flowDict).String(), "start=ingress [5.5.1.1:800->2.1.1.1:80 TCP (no flags set)]")

	// uncommon dscp
	flowDict["dscp"] = "1"
	assertStr(t, FlowFromDict(flowDict).String(), "start=ingress [5.5.1.1:800->2.1.1.1:80 TCP (no flags set) dscp=1]")

	// uncommon ecn
	flowDict["ecn"] = "1"
	assertStr(t, FlowFromDict(flowDict).String(), "start=ingress [5.5.1.1:800->2.1.1.1:80 TCP (no flags set) dscp=1 ecn=1]")

	// uncommon fragment offset
	flowDict["fragmentOffset"] = "1501"
	assertStr(t, FlowFromDict(flowDict).String(),
		"start=ingress [5.5.1.1:800->2.1.1.1:80 TCP (no flags set) dscp=1 ecn=1 fragmentOffset=1501]")

	// uncommon packet length
	flowDict["packetLength"] = "100"
	assertStr(t, FlowFromDict(flowDict).String(),
		"start=ingress [5.5.1.1:800->2.1.1.1:80 TCP (no flags set) dscp=1 ecn=1 fragmentOffset=1501 length=100]")
}

func TestFlowHTMLLines(t *testing.T) {
	flowDict := flowTestDict()
	// no ports
	assertDeepEqual(t, FlowFromDict(flowDict).HTMLLines(), []string{
		"Start Location: ingress",
		"Src IP: 5.5.1.1",
		"Dst IP: 2.1.1.1",
		"IP Protocol: ipProtocol=168",
	})

	// with ports
	flowDict["ipProtocol"] = "TCP"
	flowDict["dstPort"] = "80"
	flowDict["srcPort"] = "800"
	assertDeepEqual(t, FlowFromDict(flowDict).HTMLLines(), []string{
		"Start Location: ingress",
		"Src IP: 5.5.1.1",
		"Src Port: 800",
		"Dst IP: 2.1.1.1",
		"Dst Port: 80",
		"IP Protocol: TCP (no flags set)",
	})

	// uncommon dscp
	flowDict["dscp"] = "1"
	assertDeepEqual(t, FlowFromDict(flowDict).HTMLLines(), []string{
		"Start Location: ingress",
		"Src IP: 5.5.1.1",
		"Src Port: 800",
		"Dst IP: 2.1.1.1",
		"Dst Port: 80",
		"IP Protocol: TCP (no flags set)",
		"DSCP: 1",
	})

	// uncommon ecn
	flowDict["ecn"] = "1"
	assertDeepEqual(t, FlowFromDict(flowDict).HTMLLines(), []string{
		"Start Location: ingress",
		"Src IP: 5.5.1.1",
		"Src Port: 800",
		"Dst IP: 2.1.1.1",
		"Dst Port: 80",
		"IP Protocol: TCP (no flags set)",
		"DSCP: 1",
		"ECN: 1",
	})

	// uncommon fragment offset
	flowDict["fragmentOffset"] = "1501"
	assertDeepEqual(t, FlowFromDict(flowDict).HTMLLines(), []string{
		"Start Location: ingress",
		"Src IP: 5.5.1.1",
		"Src Port: 800",
		"Dst IP: 2.1.1.1",
		"Dst Port: 80",
		"IP Protocol: TCP (no flags set)",
		"DSCP: 1",
		"ECN: 1",
		"Fragment Offset: 1501",
	})

	// uncommon packet length
	flowDict["packetLength"] = "100"
	assertDeepEqual(t, FlowFromDict(flowDict).HTMLLines(), []string{
		"Start Location: ingress",
		"Src IP: 5.5.1.1",
		"Src Port: 800",
		"Dst IP: 2.1.1.1",
		"Dst Port: 80",
		"IP Protocol: TCP (no flags set)",
		"DSCP: 1",
		"ECN: 1",
		"Fragment Offset: 1501",
		"Packet Length: 100",
	})
}

func TestHeaderConstraintsSerialization(t *testing.T) {
	hc := HeaderConstraints{}
	hcd := hc.Dict()
	fields := []string{
		"srcIps", "dstIps", "srcPorts", "dstPorts", "ipProtocols", "applications",
		"icmpCodes", "icmpTypes", "ecns", "dscps", "packetLengths", "fragmentOffsets", "tcpFlags",
	}
	for _, field := range fields {
		if hcd[field] != nil {
			t.Fatalf("expected nil for %s, got %v", field, hcd[field])
		}
	}

	hc = HeaderConstraints{SrcIps: "1.1.1.1"}
	assertDeepEqual(t, hc.Dict()["srcIps"], "1.1.1.1")

	hc = HeaderConstraints{DstPorts: []any{"10-20", "33-33"}}
	assertDeepEqual(t, hc.Dict()["dstPorts"], "10-20,33-33")

	hc = HeaderConstraints{DstPorts: "10-20,33"}
	assertDeepEqual(t, hc.Dict()["dstPorts"], "10-20,33")

	for _, dp := range []any{10, "10", []any{10}, []any{"10"}} {
		hc = HeaderConstraints{DstPorts: dp}
		assertDeepEqual(t, hc.Dict()["dstPorts"], "10")
	}

	hc = HeaderConstraints{Applications: "dns,ssh"}
	assertDeepEqual(t, hc.Dict()["applications"], []any{"dns", "ssh"})

	hc = HeaderConstraints{Applications: []any{"dns", "ssh"}}
	assertDeepEqual(t, hc.Dict()["applications"], []any{"dns", "ssh"})

	hc = HeaderConstraints{IpProtocols: "  ,  dns,"}
	assertDeepEqual(t, hc.Dict()["ipProtocols"], []any{"dns"})

	hc = HeaderConstraints{IpProtocols: []any{"tcp", "udp"}}
	assertDeepEqual(t, hc.Dict()["ipProtocols"], []any{"tcp", "udp"})

	matchSyn := NewMatchTcpFlags(TcpFlags{Syn: true})
	hc1 := HeaderConstraints{TcpFlags: matchSyn}
	hc2 := HeaderConstraints{TcpFlags: []any{matchSyn}}
	assertDeepEqual(t, hc1.Dict(), hc2.Dict())

	if _, err := (HeaderConstraints{Applications: ""}).Normalize(); err == nil {
		t.Fatal("expected ValueError for empty applications")
	}
}

func TestHopReprStr(t *testing.T) {
	hop := Hop{
		Node: "node1",
		Steps: []Step{
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
		},
	}

	assertStr(t, hop.String(), "node: node1\n  SENT_IN(in_iface1)\n"+
		"  FORWARDED(Forwarded out interface: iface1 with resolved next-hop IP: 12.123.1.2, Routes: [bgp (Network: 1.1.1.1/24, Next Hop: ip 1.2.3.4),static (Network: 1.1.1.2/24, Next Hop: ip 1.2.3.5)])\n  "+
		"PERMITTED(preSourceNat_filter (PRENAT))\n  SENT_OUT(out_iface1)")
}

// TODO: remove after sufficient period
func TestHopReprStrLegacy(t *testing.T) {
	hop := Hop{
		Node: "node1",
		Steps: []Step{
			{Detail: EnterInputIfaceStepDetail{InputInterface: "in_iface1", InputVrf: StringPtr("in_vrf1")}, Action: "SENT_IN"},
			{Detail: RoutingStepDetail{
				Routes: []RouteInfo{
					{Protocol: "bgp", Network: "1.1.1.1/24", NextHopIP: StringPtr("1.2.3.4"), Admin: 1, Metric: 1},
					{Protocol: "static", Network: "1.1.1.2/24", NextHopIP: StringPtr("1.2.3.5"), Admin: 1, Metric: 1},
				},
				ArpIP:           StringPtr("12.123.1.2"),
				OutputInterface: StringPtr("iface1"),
			}, Action: "FORWARDED"},
			{Detail: FilterStepDetail{Filter: "preSourceNat_filter", FilterType: "PRENAT"}, Action: "PERMITTED"},
			{Detail: ExitOutputIfaceStepDetail{OutputInterface: "out_iface1"}, Action: "SENT_OUT"},
		},
	}

	assertStr(t, hop.String(), "node: node1\n  SENT_IN(in_iface1)\n"+
		"  FORWARDED(ARP IP: 12.123.1.2, Output Interface: iface1, Routes: [bgp (Network: 1.1.1.1/24, Next Hop IP:1.2.3.4),static (Network: 1.1.1.2/24, Next Hop IP:1.2.3.5)])\n  "+
		"PERMITTED(preSourceNat_filter (PRENAT))\n  SENT_OUT(out_iface1)")
}

// TODO: remove after sufficient period
func TestOnlyRoutesStrLegacy(t *testing.T) {
	routingStepDetail := RoutingStepDetail{
		Routes: []RouteInfo{{Protocol: "bgp", Network: "1.1.1.1/24", NextHopIP: StringPtr("1.2.3.4"), Admin: 1, Metric: 1}},
	}
	assertStr(t, routingStepDetail.String(), "Routes: [bgp (Network: 1.1.1.1/24, Next Hop IP:1.2.3.4)]")
}

// TODO: remove after sufficient period
func TestNoOutputIfaceStrLegacy(t *testing.T) {
	routingStepDetail := RoutingStepDetail{
		Routes:          []RouteInfo{{Protocol: "bgp", Network: "1.1.1.1/24", NextHopIP: StringPtr("1.2.3.4"), Admin: 1, Metric: 1}},
		ArpIP:           StringPtr("1.2.3.4"),
		OutputInterface: nil,
	}
	assertStr(t, routingStepDetail.String(), "ARP IP: 1.2.3.4, Routes: [bgp (Network: 1.1.1.1/24, Next Hop IP:1.2.3.4)]")
}

// TODO: remove after sufficient period
func TestNoArpIPStrLegacy(t *testing.T) {
	routingStepDetail := RoutingStepDetail{
		Routes:          []RouteInfo{{Protocol: "bgp", Network: "1.1.1.1/24", NextHopIP: StringPtr("1.2.3.4"), Admin: 1, Metric: 1}},
		ArpIP:           nil,
		OutputInterface: StringPtr("iface1"),
	}
	assertStr(t, routingStepDetail.String(), "Output Interface: iface1, Routes: [bgp (Network: 1.1.1.1/24, Next Hop IP:1.2.3.4)]")
}

func TestNoRoute(t *testing.T) {
	step := Step{Detail: RoutingStepDetail{Routes: []RouteInfo{}, ForwardingDetail: Discarded{}}, Action: "NO_ROUTE"}
	assertStr(t, step.String(), "NO_ROUTE(Discarded)")
}

// TODO: remove after sufficient period
func TestNoRouteLegacy(t *testing.T) {
	step := Step{Detail: RoutingStepDetail{Routes: []RouteInfo{}}, Action: "NO_ROUTE"}
	assertStr(t, step.String(), "NO_ROUTE")
}

func TestMatchTCPGenerators(t *testing.T) {
	assertDeepEqual(t, MatchAck(), NewMatchTcpFlags(TcpFlags{Ack: true}))
	assertDeepEqual(t, MatchRst(), NewMatchTcpFlags(TcpFlags{Rst: true}))
	assertDeepEqual(t, MatchSyn(), NewMatchTcpFlags(TcpFlags{Syn: true}))
	assertDeepEqual(t, MatchSynAck(), NewMatchTcpFlags(TcpFlags{Ack: true, Syn: true}))
	if len(MatchEstablished()) != 2 {
		t.Fatalf("match_established length = %d", len(MatchEstablished()))
	}
	assertDeepEqual(t, MatchEstablished(), []MatchTcpFlags{MatchAck(), MatchRst()})
	assertDeepEqual(t, MatchNotEstablished(), []MatchTcpFlags{
		NewMatchTcpFlags(TcpFlags{Ack: false, Rst: false}),
	})
}

func TestFlowHTMLPorts(t *testing.T) {
	flowDict := flowTestDict()
	flowDict["ipProtocol"] = "ICMP"
	flowDict["icmpCode"] = 255
	flowDict["icmpVar"] = 255
	// ICMP flows do not have ports
	assertNotContains(t, FlowFromDict(flowDict).HTML(), "Port")

	// UDP
	flowDict["ipProtocol"] = "UDP"
	assertContains(t, FlowFromDict(flowDict).HTML(), "Port")
}

func TestFlowHTMLStartLocation(t *testing.T) {
	flowDict := flowTestDict()
	flowDict["ipProtocol"] = "ICMP"
	flowDict["icmpCode"] = 255
	flowDict["icmpVar"] = 255
	flowDict["ingressNode"] = "ingressNode"
	flowDict["ingressVrf"] = "default"

	assertContains(t, joinLines(FlowFromDict(flowDict).HTMLLines()), "Start Location: ingressNode")

	flowDict["ingressVrf"] = "ingressVrf"
	assertContains(t, joinLines(FlowFromDict(flowDict).HTMLLines()), "Start Location: ingressNode vrf=ingressVrf")

	delete(flowDict, "ingressVrf")
	flowDict["ingressInterface"] = "ingressIface"

	flow := FlowFromDict(flowDict)
	assertContains(t, joinLines(flow.HTMLLines()), "Start Location: ingressNode interface=ingressIface")
}

func TestFlowStrPorts(t *testing.T) {
	flowDict := flowTestDict()
	flowDict["ipProtocol"] = "ICMP"
	flowDict["icmpCode"] = 255
	flowDict["icmpVar"] = 255
	flowDict["dstPort"] = 1234
	flowDict["srcPort"] = 2345
	// ICMP flows do not have ports
	s := FlowFromDict(flowDict).String()
	assertNotContains(t, s, "2.1.1.1:1234")
	assertNotContains(t, s, "5.5.1.1:2345")

	// UDP flows do have ports, matching Python's __str__ representation
	flowDict["ipProtocol"] = "UDP"
	s = FlowFromDict(flowDict).String()
	assertContains(t, s, "2.1.1.1:1234")
	assertContains(t, s, "5.5.1.1:2345")
}

func TestInboundStepDetailFromDict(t *testing.T) {
	iface := "GigabitEthernet1/0"
	d := map[string]any{"type": "InboundStep", "interface": iface, "action": "ACCEPTED"}
	assertDeepEqual(t, InboundStepDetailFromDict(d), InboundStepDetail{Interface: iface})
}

func TestInboundStepDetailStr(t *testing.T) {
	iface := "GigabitEthernet1/0"
	assertStr(t, (InboundStepDetail{Interface: iface}).String(), iface)
}

func TestLoopStepFromDict(t *testing.T) {
	step := StepFromDict(map[string]any{"type": "Loop", "action": "LOOP", "detail": map[string]any{}})
	if step == nil {
		t.Fatal("expected a step")
	}
	assertStr(t, step.Action, "LOOP")
	assertDeepEqual(t, step.Detail, LoopStepDetail{})
}

func TestLoopStepDetailStr(t *testing.T) {
	assertStr(t, LoopStepDetail{}.String(), "")
}

func TestSetupSessionStepDetailFromDict(t *testing.T) {
	d := map[string]any{
		"sessionScope":   map[string]any{"incomingInterfaces": []any{"reth0.6"}},
		"sessionAction":  map[string]any{"type": "Accept"},
		"matchCriteria":  map[string]any{"ipProtocol": "ICMP", "srcIp": "2.2.2.2", "dstIp": "3.3.3.3"},
		"transformation": []any{map[string]any{"fieldName": "srcIp", "oldValue": "2.2.2.2", "newValue": "1.1.1.1"}},
	}
	assertDeepEqual(t, SetupSessionStepDetailFromDict(d), SetupSessionStepDetail{
		SessionScope:   IncomingSessionScope{IncomingInterfaces: []string{"reth0.6"}},
		SessionAction:  Accept{},
		MatchCriteria:  SessionMatchExpr{IPProtocol: "ICMP", SrcIP: "2.2.2.2", DstIP: "3.3.3.3"},
		Transformation: []FlowDiff{{FieldName: "srcIp", OldValue: "2.2.2.2", NewValue: "1.1.1.1"}},
	})
}

func TestSetupSessionStepDetailFromDictBwCompat(t *testing.T) {
	// For backward compatibility, allow "incomingInterfaces" instead of "sessionScope".
	d := map[string]any{
		"incomingInterfaces": []any{"reth0.6"},
		"sessionAction":      map[string]any{"type": "Accept"},
		"matchCriteria":      map[string]any{"ipProtocol": "ICMP", "srcIp": "2.2.2.2", "dstIp": "3.3.3.3"},
		"transformation":     []any{map[string]any{"fieldName": "srcIp", "oldValue": "2.2.2.2", "newValue": "1.1.1.1"}},
	}
	assertDeepEqual(t, SetupSessionStepDetailFromDict(d), SetupSessionStepDetail{
		SessionScope:   IncomingSessionScope{IncomingInterfaces: []string{"reth0.6"}},
		SessionAction:  Accept{},
		MatchCriteria:  SessionMatchExpr{IPProtocol: "ICMP", SrcIP: "2.2.2.2", DstIP: "3.3.3.3"},
		Transformation: []FlowDiff{{FieldName: "srcIp", OldValue: "2.2.2.2", NewValue: "1.1.1.1"}},
	})
}

func TestSetupSessionStepDetailStr(t *testing.T) {
	detail := SetupSessionStepDetail{
		SessionScope:   IncomingSessionScope{IncomingInterfaces: []string{"reth0.6"}},
		SessionAction:  Accept{},
		MatchCriteria:  SessionMatchExpr{IPProtocol: "ICMP", SrcIP: "2.2.2.2", DstIP: "3.3.3.3"},
		Transformation: []FlowDiff{{FieldName: "srcIp", OldValue: "2.2.2.2", NewValue: "1.1.1.1"}},
	}
	assertStr(t, detail.String(), "Incoming Interfaces: [reth0.6], "+
		"Action: Accept, "+
		"Match Criteria: [ipProtocol=ICMP, srcIp=2.2.2.2, dstIp=3.3.3.3], "+
		"Transformation: [srcIp: 2.2.2.2 -> 1.1.1.1]")

	detail = SetupSessionStepDetail{
		SessionScope:  IncomingSessionScope{IncomingInterfaces: []string{"reth0.6"}},
		SessionAction: Accept{},
		MatchCriteria: SessionMatchExpr{IPProtocol: "ICMP", SrcIP: "2.2.2.2", DstIP: "3.3.3.3"},
	}
	assertStr(t, detail.String(), "Incoming Interfaces: [reth0.6], "+
		"Action: Accept, "+
		"Match Criteria: [ipProtocol=ICMP, srcIp=2.2.2.2, dstIp=3.3.3.3]")
}

func TestMatchSessionStepDetailFromDict(t *testing.T) {
	d := map[string]any{
		"sessionScope":   map[string]any{"incomingInterfaces": []any{"reth0.6"}},
		"sessionAction":  map[string]any{"type": "Accept"},
		"matchCriteria":  map[string]any{"ipProtocol": "ICMP", "srcIp": "2.2.2.2", "dstIp": "3.3.3.3"},
		"transformation": []any{map[string]any{"fieldName": "srcIp", "oldValue": "2.2.2.2", "newValue": "1.1.1.1"}},
	}
	assertDeepEqual(t, MatchSessionStepDetailFromDict(d), MatchSessionStepDetail{
		SessionScope:   IncomingSessionScope{IncomingInterfaces: []string{"reth0.6"}},
		SessionAction:  Accept{},
		MatchCriteria:  SessionMatchExpr{IPProtocol: "ICMP", SrcIP: "2.2.2.2", DstIP: "3.3.3.3"},
		Transformation: []FlowDiff{{FieldName: "srcIp", OldValue: "2.2.2.2", NewValue: "1.1.1.1"}},
	})
}

func TestMatchSessionStepDetailFromDictBwCompat(t *testing.T) {
	// For backward compatibility, allow "incomingInterfaces" instead of "sessionScope".
	d := map[string]any{
		"incomingInterfaces": []any{"reth0.6"},
		"sessionAction":      map[string]any{"type": "Accept"},
		"matchCriteria":      map[string]any{"ipProtocol": "ICMP", "srcIp": "2.2.2.2", "dstIp": "3.3.3.3"},
		"transformation":     []any{map[string]any{"fieldName": "srcIp", "oldValue": "2.2.2.2", "newValue": "1.1.1.1"}},
	}
	assertDeepEqual(t, MatchSessionStepDetailFromDict(d), MatchSessionStepDetail{
		SessionScope:   IncomingSessionScope{IncomingInterfaces: []string{"reth0.6"}},
		SessionAction:  Accept{},
		MatchCriteria:  SessionMatchExpr{IPProtocol: "ICMP", SrcIP: "2.2.2.2", DstIP: "3.3.3.3"},
		Transformation: []FlowDiff{{FieldName: "srcIp", OldValue: "2.2.2.2", NewValue: "1.1.1.1"}},
	})
}

func TestMatchSessionStepDetailStr(t *testing.T) {
	detail := MatchSessionStepDetail{
		SessionScope:   IncomingSessionScope{IncomingInterfaces: []string{"reth0.6"}},
		SessionAction:  Accept{},
		MatchCriteria:  SessionMatchExpr{IPProtocol: "ICMP", SrcIP: "2.2.2.2", DstIP: "3.3.3.3"},
		Transformation: []FlowDiff{{FieldName: "srcIp", OldValue: "2.2.2.2", NewValue: "1.1.1.1"}},
	}
	assertStr(t, detail.String(), "Incoming Interfaces: [reth0.6], "+
		"Action: Accept, "+
		"Match Criteria: [ipProtocol=ICMP, srcIp=2.2.2.2, dstIp=3.3.3.3], "+
		"Transformation: [srcIp: 2.2.2.2 -> 1.1.1.1]")

	detail = MatchSessionStepDetail{
		SessionScope:  IncomingSessionScope{IncomingInterfaces: []string{"reth0.6"}},
		SessionAction: Accept{},
		MatchCriteria: SessionMatchExpr{IPProtocol: "ICMP", SrcIP: "2.2.2.2", DstIP: "3.3.3.3"},
	}
	assertStr(t, detail.String(), "Incoming Interfaces: [reth0.6], "+
		"Action: Accept, "+
		"Match Criteria: [ipProtocol=ICMP, srcIp=2.2.2.2, dstIp=3.3.3.3]")
}

func TestSessionMatchExprFromDict(t *testing.T) {
	d := map[string]any{"ipProtocol": "ICMP", "srcIp": "1.1.1.1", "dstIp": "2.2.2.2"}
	assertDeepEqual(t, SessionMatchExprFromDict(d), SessionMatchExpr{IPProtocol: "ICMP", SrcIP: "1.1.1.1", DstIP: "2.2.2.2"})

	d = map[string]any{
		"ipProtocol": "ICMP", "srcIp": "1.1.1.1", "dstIp": "2.2.2.2",
		"srcPort": 1111, "dstPort": 2222,
	}
	assertDeepEqual(t, SessionMatchExprFromDict(d), SessionMatchExpr{
		IPProtocol: "ICMP", SrcIP: "1.1.1.1", DstIP: "2.2.2.2", SrcPort: IntPtr(1111), DstPort: IntPtr(2222),
	})
}

func TestSessionMatchExprStr(t *testing.T) {
	match := SessionMatchExpr{IPProtocol: "ICMP", SrcIP: "1.1.1.1", DstIP: "2.2.2.2"}
	assertStr(t, match.String(), "[ipProtocol=ICMP, srcIp=1.1.1.1, dstIp=2.2.2.2]")
	match = SessionMatchExpr{IPProtocol: "ICMP", SrcIP: "1.1.1.1", DstIP: "2.2.2.2", SrcPort: IntPtr(1111), DstPort: IntPtr(2222)}
	assertStr(t, match.String(), "[ipProtocol=ICMP, srcIp=1.1.1.1, dstIp=2.2.2.2, srcPort=1111, dstPort=2222]")
}

func TestSessionScopeFromDict(t *testing.T) {
	d := map[string]any{"incomingInterfaces": []any{"iface"}}
	scope, err := SessionScopeFromDict(d)
	assertDeepEqual(t, err, nil)
	assertDeepEqual(t, scope, IncomingSessionScope{IncomingInterfaces: []string{"iface"}})

	d = map[string]any{"originatingVrf": "vrf"}
	scope, err = SessionScopeFromDict(d)
	assertDeepEqual(t, err, nil)
	assertDeepEqual(t, scope, OriginatingSessionScope{OriginatingVrf: "vrf"})
}

func TestIncomingSessionScopeStr(t *testing.T) {
	scope := IncomingSessionScope{IncomingInterfaces: []string{"iface1", "iface2"}}
	assertStr(t, scope.String(), "Incoming Interfaces: [iface1, iface2]")
}

func TestOriginatingSessionScopeStr(t *testing.T) {
	scope := OriginatingSessionScope{OriginatingVrf: "vrf"}
	assertStr(t, scope.String(), "Originating VRF: vrf")
}

func TestSessionActionFromDict(t *testing.T) {
	action, err := SessionActionFromDict(map[string]any{"type": "Accept"})
	assertDeepEqual(t, err, nil)
	assertDeepEqual(t, action, Accept{})

	action, _ = SessionActionFromDict(map[string]any{"type": "FibLookup"})
	assertDeepEqual(t, action, PostNatFibLookup{})
	action, _ = SessionActionFromDict(map[string]any{"type": "PostNatFibLookup"})
	assertDeepEqual(t, action, PostNatFibLookup{})
	action, _ = SessionActionFromDict(map[string]any{"type": "PreNatFibLookup"})
	assertDeepEqual(t, action, PreNatFibLookup{})

	d := map[string]any{
		"type":              "ForwardOutInterface",
		"nextHop":           map[string]any{"hostname": "1.1.1.1", "interface": "iface1"},
		"outgoingInterface": "iface2",
	}
	action, _ = SessionActionFromDict(d)
	assertDeepEqual(t, action, ForwardOutInterface{NextHopHostname: "1.1.1.1", NextHopInterface: "iface1", OutgoingInterface: "iface2"})

	if _, err := SessionActionFromDict(map[string]any{"type": "NotAType"}); err == nil {
		t.Fatal("expected ValueError")
	}
}

func TestSessionActionStr(t *testing.T) {
	assertStr(t, (Accept{}).String(), "Accept")
	assertStr(t, (PostNatFibLookup{}).String(), "PostNatFibLookup")
	assertStr(t, (PreNatFibLookup{}).String(), "PreNatFibLookup")
	assertStr(t, (ForwardOutInterface{NextHopHostname: "1.1.1.1", NextHopInterface: "iface1", OutgoingInterface: "iface2"}).String(),
		"ForwardOutInterface(Next Hop: 1.1.1.1, Next Hop Interface: iface1, Outgoing Interface: iface2)")
}

func TestHeaderConstraintsOf(t *testing.T) {
	hc := HeaderConstraintsOf(Flow{
		IPProtocol: "ICMP", IcmpCode: IntPtr(7), SrcIP: "1.1.1.1", DstIP: "2.2.2.2",
	})
	assertStr(t, hc.SrcIps.(string), "1.1.1.1")
	assertStr(t, hc.DstIps.(string), "2.2.2.2")
	assertDeepEqual(t, hc.IpProtocols, []any{"ICMP"})
	assertDeepEqual(t, hc.IcmpCodes, "7")
	if hc.SrcPorts != nil {
		t.Fatalf("srcPorts = %v", hc.SrcPorts)
	}
	if hc.DstPorts != nil {
		t.Fatalf("dstPorts = %v", hc.DstPorts)
	}
	if hc.TcpFlags != nil {
		t.Fatalf("tcpFlags = %v", hc.TcpFlags)
	}

	hc = HeaderConstraintsOf(Flow{
		IPProtocol: "TCP", SrcPort: IntPtr(1000), DstPort: IntPtr(2000),
		TcpFlagsAck: IntPtr(1), SrcIP: "1.1.1.1", DstIP: "2.2.2.2",
	})
	assertStr(t, hc.SrcIps.(string), "1.1.1.1")
	assertStr(t, hc.DstIps.(string), "2.2.2.2")
	assertDeepEqual(t, hc.IpProtocols, []any{"TCP"})
	if hc.IcmpCodes != nil {
		t.Fatalf("icmpCodes = %v", hc.IcmpCodes)
	}
	if hc.IcmpTypes != nil {
		t.Fatalf("icmpTypes = %v", hc.IcmpTypes)
	}
	assertStr(t, hc.SrcPorts.(string), "1000")
	assertStr(t, hc.DstPorts.(string), "2000")
	assertDeepEqual(t, hc.TcpFlags, []any{NewMatchTcpFlags(TcpFlags{Ack: true})})
}

func TestForwardingDetailCannotInstantiate(t *testing.T) {
	// ForwardingDetail is an interface in Go: the Python TypeError raised by
	// instantiating the abstract base is enforced at compile time. Verify the
	// zero value is nil.
	var detail ForwardingDetail
	if detail != nil {
		t.Fatal("zero ForwardingDetail should be nil")
	}
}

func TestForwardingDetailDeserializationInvalid(t *testing.T) {
	if _, err := ForwardingDetailFromDict(map[string]any{"type": "foo"}); err == nil {
		t.Fatal("expected ValueError")
	}
	if _, err := ForwardingDetailFromDict(map[string]any{}); err == nil {
		t.Fatal("expected ValueError")
	}
}

func TestDelegatedToNextVrfSerialization(t *testing.T) {
	assertDeepEqual(t, (DelegatedToNextVrf{NextVrf: "foo"}).Dict(), map[string]any{
		"type": "DelegatedToNextVrf", "nextVrf": "foo",
	})
}

func TestDelegatedToNextVrfDeserialization(t *testing.T) {
	d := map[string]any{"type": "DelegatedToNextVrf", "nextVrf": "foo"}
	detail, err := ForwardingDetailFromDict(d)
	assertDeepEqual(t, err, nil)
	assertDeepEqual(t, detail, DelegatedToNextVrf{NextVrf: "foo"})
	assertDeepEqual(t, DelegatedToNextVrfFromDict(d), DelegatedToNextVrf{NextVrf: "foo"})
}

func TestDelegatedToNextVrfStr(t *testing.T) {
	assertStr(t, (DelegatedToNextVrf{NextVrf: "foo"}).String(), "Delegated to next VRF: foo")
	assertStr(t, (DelegatedToNextVrf{NextVrf: "foo bar"}).String(), `Delegated to next VRF: "foo bar"`)
}

func TestForwardedOutInterfaceSerialization(t *testing.T) {
	assertDeepEqual(t, (ForwardedOutInterface{OutputInterface: "foo"}).Dict(), map[string]any{
		"type": "ForwardedOutInterface", "outputInterface": "foo", "resolvedNextHopIp": nil,
	})
	assertDeepEqual(t, (ForwardedOutInterface{OutputInterface: "foo", ResolvedNextHopIP: StringPtr("1.1.1.1")}).Dict(), map[string]any{
		"type": "ForwardedOutInterface", "outputInterface": "foo", "resolvedNextHopIp": "1.1.1.1",
	})
}

func TestForwardedOutInterfaceDeserialization(t *testing.T) {
	d := map[string]any{"type": "ForwardedOutInterface", "outputInterface": "foo"}
	detail, err := ForwardingDetailFromDict(d)
	assertDeepEqual(t, err, nil)
	assertDeepEqual(t, detail, ForwardedOutInterface{OutputInterface: "foo"})
	assertDeepEqual(t, ForwardedOutInterfaceFromDict(d), ForwardedOutInterface{OutputInterface: "foo"})
	assertDeepEqual(t, ForwardedOutInterfaceFromDict(map[string]any{
		"type": "ForwardedOutInterface", "outputInterface": "foo", "resolvedNextHopIp": nil,
	}), ForwardedOutInterface{OutputInterface: "foo"})
	assertDeepEqual(t, ForwardedOutInterfaceFromDict(map[string]any{
		"type": "ForwardedOutInterface", "outputInterface": "foo", "resolvedNextHopIp": "1.1.1.1",
	}), ForwardedOutInterface{OutputInterface: "foo", ResolvedNextHopIP: StringPtr("1.1.1.1")})
}

func TestForwardedOutInterfaceStr(t *testing.T) {
	assertStr(t, (ForwardedOutInterface{OutputInterface: "foo"}).String(), "Forwarded out interface: foo")
	assertStr(t, (ForwardedOutInterface{OutputInterface: "foo bar"}).String(), `Forwarded out interface: "foo bar"`)
	assertStr(t, (ForwardedOutInterface{OutputInterface: "foo bar", ResolvedNextHopIP: StringPtr("1.1.1.1")}).String(),
		`Forwarded out interface: "foo bar" with resolved next-hop IP: 1.1.1.1`)
}

func TestForwardedIntoVxlanTunnelVtepSerialization(t *testing.T) {
	assertDeepEqual(t, (ForwardedIntoVxlanTunnel{Vni: 5, Vtep: "1.1.1.1"}).Dict(), map[string]any{
		"type": "ForwardedIntoVxlanTunnel", "vni": 5, "vtep": "1.1.1.1",
	})
}

func TestForwardedIntoVxlanTunnelDeserialization(t *testing.T) {
	d := map[string]any{"type": "ForwardedIntoVxlanTunnel", "vni": 5, "vtep": "1.1.1.1"}
	detail, err := ForwardingDetailFromDict(d)
	assertDeepEqual(t, err, nil)
	assertDeepEqual(t, detail, ForwardedIntoVxlanTunnel{Vni: 5, Vtep: "1.1.1.1"})
	assertDeepEqual(t, ForwardedIntoVxlanTunnelFromDict(d), ForwardedIntoVxlanTunnel{Vni: 5, Vtep: "1.1.1.1"})
}

func TestForwardedIntoVxlanTunnelStr(t *testing.T) {
	assertStr(t, (ForwardedIntoVxlanTunnel{Vni: 5, Vtep: "1.1.1.1"}).String(),
		"Forwarded into VXLAN tunnel with VNI: 5 and VTEP: 1.1.1.1")
}

func TestDiscardedSerialization(t *testing.T) {
	assertDeepEqual(t, (Discarded{}).Dict(), map[string]any{"type": "Discarded"})
}

func TestDiscardedDeserialization(t *testing.T) {
	d := map[string]any{"type": "Discarded"}
	detail, err := ForwardingDetailFromDict(d)
	assertDeepEqual(t, err, nil)
	assertDeepEqual(t, detail, Discarded{})
}

func TestDiscardedStr(t *testing.T) {
	assertStr(t, (Discarded{}).String(), "Discarded")
}

func TestRouteInfoSerialization(t *testing.T) {
	assertDeepEqual(t, (RouteInfo{Protocol: "tcp", Network: "1.1.1.1/32", NextHop: NextHopDiscard{}, Admin: 1, Metric: 2}).Dict(), map[string]any{
		"protocol":  "tcp",
		"network":   "1.1.1.1/32",
		"nextHop":   map[string]any{"type": "discard"},
		"nextHopIp": nil,
		"admin":     1,
		"metric":    2,
	})
}

// TODO: remove after sufficient period
func TestRouteInfoSerializationLegacy(t *testing.T) {
	assertDeepEqual(t, (RouteInfo{Protocol: "tcp", Network: "1.1.1.1/32", NextHopIP: StringPtr("2.2.2.2"), Admin: 1, Metric: 2}).Dict(), map[string]any{
		"protocol":  "tcp",
		"network":   "1.1.1.1/32",
		"nextHop":   nil,
		"nextHopIp": "2.2.2.2",
		"admin":     1,
		"metric":    2,
	})
}

func TestRouteInfoDeserialization(t *testing.T) {
	assertDeepEqual(t, RouteInfoFromDict(map[string]any{
		"protocol": "tcp",
		"network":  "1.1.1.1/32",
		"nextHop":  map[string]any{"type": "discard"},
		"admin":    1,
		"metric":   2,
	}), RouteInfo{Protocol: "tcp", Network: "1.1.1.1/32", NextHop: NextHopDiscard{}, Admin: 1, Metric: 2})
}

// TODO: remove after sufficient period
func TestRouteInfoDeserializationLegacy(t *testing.T) {
	assertDeepEqual(t, RouteInfoFromDict(map[string]any{
		"protocol":  "tcp",
		"network":   "1.1.1.1/32",
		"nextHopIp": "2.2.2.2",
		"admin":     1,
		"metric":    2,
	}), RouteInfo{Protocol: "tcp", Network: "1.1.1.1/32", NextHopIP: StringPtr("2.2.2.2"), Admin: 1, Metric: 2})
}

func TestRouteInfoStr(t *testing.T) {
	assertStr(t, (RouteInfo{Protocol: "tcp", Network: "1.1.1.1/32", NextHop: NextHopDiscard{}, Admin: 1, Metric: 2}).String(),
		"tcp (Network: 1.1.1.1/32, Next Hop: discard)")
}

// TODO: remove after sufficient period
func TestRouteInfoStrLegacy(t *testing.T) {
	assertStr(t, (RouteInfo{Protocol: "tcp", Network: "1.1.1.1/32", NextHopIP: StringPtr("2.2.2.2"), Admin: 1, Metric: 2}).String(),
		"tcp (Network: 1.1.1.1/32, Next Hop IP:2.2.2.2)")
}

// joinLines joins HTML lines for substring checks.
func joinLines(lines []string) string {
	out := ""
	for i, l := range lines {
		if i > 0 {
			out += "\n"
		}
		out += l
	}
	return out
}
