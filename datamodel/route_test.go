package datamodel

import "testing"

// Ports of tests/datamodel/test_route.py.

func TestBgpRouteDeserialization(t *testing.T) {
	network := "0.0.0.0/0"
	asPath := []any{1, 2, 3}
	communities := []any{4, 5, 6}
	localPreference := 1
	metric := 2
	nextHopIP := "2.2.2.2"
	originType := "egp"
	originatorIP := "1.1.1.1"
	protocol := "bgp"
	srcProtocol := "connected"
	tag := 23
	weight := 42

	dct := map[string]any{
		"network":         network,
		"asPath":          asPath,
		"communities":     communities,
		"localPreference": localPreference,
		"metric":          metric,
		"nextHopIp":       nextHopIP,
		"originatorIp":    originatorIP,
		"originType":      originType,
		"protocol":        protocol,
		"srcProtocol":     srcProtocol,
		"tag":             tag,
		"weight":          weight,
	}
	bgpRoute := BgpRouteFromDict(dct)
	assertStr(t, bgpRoute.Network, network)
	assertDeepEqual(t, bgpRoute.AsPath, asPath)
	assertDeepEqual(t, bgpRoute.Communities, communities)
	if bgpRoute.LocalPreference != localPreference {
		t.Fatalf("localPreference = %d", bgpRoute.LocalPreference)
	}
	if bgpRoute.Metric != metric {
		t.Fatalf("metric = %d", bgpRoute.Metric)
	}
	assertStr(t, *bgpRoute.NextHopIP, nextHopIP)
	assertStr(t, bgpRoute.OriginType, originType)
	assertStr(t, bgpRoute.OriginatorIP, originatorIP)
	assertStr(t, bgpRoute.Protocol, protocol)
	assertStr(t, *bgpRoute.SourceProtocol, srcProtocol)
	if bgpRoute.Tag != tag {
		t.Fatalf("tag = %d", bgpRoute.Tag)
	}
	if bgpRoute.Weight != weight {
		t.Fatalf("weight = %d", bgpRoute.Weight)
	}
}

func TestBgpRouteSerialization(t *testing.T) {
	network := "0.0.0.0/0"
	asPath := []any{1, 2, 3}
	communities := []any{4, 5, 6}
	localPreference := 1
	metric := 2
	nextHopIP := "2.2.2.2"
	originType := "egp"
	originatorIP := "1.1.1.1"
	protocol := "bgp"
	srcProtocol := "connected"
	tag := 23
	weight := 42

	bgpRoute := BgpRoute{
		Network:         network,
		AsPath:          asPath,
		Communities:     communities,
		LocalPreference: localPreference,
		Metric:          metric,
		NextHopIP:       StringPtr(nextHopIP),
		OriginatorIP:    originatorIP,
		OriginType:      originType,
		Protocol:        protocol,
		SourceProtocol:  StringPtr(srcProtocol),
		Tag:             tag,
		Weight:          weight,
	}

	dct := bgpRoute.Dict()

	assertStr(t, dct["class"].(string), "org.batfish.datamodel.questions.BgpRoute")
	assertStr(t, dct["network"].(string), network)
	assertDeepEqual(t, dct["asPath"], asPath)
	assertDeepEqual(t, dct["communities"], communities)
	assertDeepEqual(t, dct["localPreference"], localPreference)
	assertDeepEqual(t, dct["metric"], metric)
	assertDeepEqual(t, dct["nextHopIp"], nextHopIP)
	assertDeepEqual(t, dct["originatorIp"], originatorIP)
	assertDeepEqual(t, dct["originType"], originType)
	assertDeepEqual(t, dct["protocol"], protocol)
	assertDeepEqual(t, dct["srcProtocol"], srcProtocol)
	assertDeepEqual(t, dct["tag"], tag)
	assertDeepEqual(t, dct["weight"], weight)
}

func TestBgpRouteStr(t *testing.T) {
	bgpRoute := BgpRoute{
		Network: "A", AsPath: []any{1, 2}, Communities: []any{1, 2, 3},
		LocalPreference: 4, Metric: 5, NextHopIP: StringPtr("2.2.2.2"), OriginatorIP: "1.1.1.1",
		OriginType: "egp", Protocol: "bgp", SourceProtocol: StringPtr("connected"), Tag: 23, Weight: 42,
	}
	lines := bgpRoute.HTMLLines()
	assertDeepEqual(t, lines, []string{
		"Network: A",
		"AS Path: [1, 2]",
		"Communities: [1, 2, 3]",
		"Local Preference: 4",
		"Metric: 5",
		"Next Hop IP: 2.2.2.2",
		"Originator IP: 1.1.1.1",
		"Origin Type: egp",
		"Protocol: bgp",
		"Source Protocol: connected",
		"Tag: 23",
		"Weight: 42",
	})
}

func TestBgpRouteConstraintsDeserialization(t *testing.T) {
	prefix := []any{"1.2.3.4/5:6-7", "8.8.8.8:8/8-8"}
	complementPrefix := true
	localPreference := "1-2, 3-4, !5-6"
	med := "0-255, !50-55"
	communities := []any{"/.*/", "!/4[0-9]:2.+/"}
	asPath := []any{"/.*/", "40"}

	dct := map[string]any{
		"prefix":           prefix,
		"complementPrefix": complementPrefix,
		"localPreference":  localPreference,
		"med":              med,
		"communities":      communities,
		"asPath":           asPath,
	}
	bgpRouteConstraints, err := BgpRouteConstraintsFromDict(dct)
	assertDeepEqual(t, err, nil)
	assertDeepEqual(t, bgpRouteConstraints.Prefix, prefix)
	if bgpRouteConstraints.ComplementPrefix == nil || *bgpRouteConstraints.ComplementPrefix != complementPrefix {
		t.Fatalf("complementPrefix = %v", bgpRouteConstraints.ComplementPrefix)
	}
	assertStr(t, *bgpRouteConstraints.LocalPreference, localPreference)
	assertStr(t, *bgpRouteConstraints.Med, med)
	assertDeepEqual(t, bgpRouteConstraints.Communities, communities)
	assertDeepEqual(t, bgpRouteConstraints.AsPath, asPath)
}

func TestBgpRouteConstraintsConversions(t *testing.T) {
	prefix := "1.2.3.4/5:6-7"
	communities := []any{"20:30"}
	med := []any{"1-2", "3-4", "!5-6"}
	localPreference := []any{}

	got, err := stringListBRCConverter(prefix)
	assertDeepEqual(t, err, nil)
	assertDeepEqual(t, got, []any{prefix})

	got, err = stringListBRCConverter(communities)
	assertDeepEqual(t, err, nil)
	assertDeepEqual(t, got, communities)

	got, err = longspaceBRCConverter(med)
	assertDeepEqual(t, err, nil)
	assertDeepEqual(t, got, "1-2,3-4,!5-6")

	got, err = longspaceBRCConverter(localPreference)
	assertDeepEqual(t, err, nil)
	assertDeepEqual(t, got, "")
}

func TestBgpRouteDiffDeserialization(t *testing.T) {
	name := "communities"
	oldValue := "old"
	newValue := "new"
	dct := map[string]any{"fieldName": name, "oldValue": oldValue, "newValue": newValue}
	routeDiff := BgpRouteDiffFromDict(dct)
	assertStr(t, routeDiff.FieldName, name)
	assertStr(t, routeDiff.OldValue, oldValue)
	assertStr(t, routeDiff.NewValue, newValue)
}

func TestBgpRouteDiffStr(t *testing.T) {
	diff1 := BgpRouteDiff{FieldName: "nm", OldValue: "old", NewValue: "new"}
	diff2 := BgpRouteDiff{FieldName: "localPreference", OldValue: "old", NewValue: "new"}
	assertStr(t, diff1.HTML(), "Nm: old --> new")
	assertStr(t, diff2.HTML(), "Local Preference: old --> new")
}

func TestBgpSessionPropertiesDeserialization(t *testing.T) {
	localAs := 22
	remoteAs := 33
	localIP := "2.2.2.2"
	remoteIP := "3.3.3.3"
	dct := map[string]any{
		"localAs":  localAs,
		"remoteAs": remoteAs,
		"localIp":  localIP,
		"remoteIp": remoteIP,
	}
	properties := BgpSessionPropertiesFromDict(dct)
	if properties.LocalAs != localAs {
		t.Fatalf("localAs = %d", properties.LocalAs)
	}
	if properties.RemoteAs != remoteAs {
		t.Fatalf("remoteAs = %d", properties.RemoteAs)
	}
	assertStr(t, properties.LocalIP, localIP)
	assertStr(t, properties.RemoteIP, remoteIP)
}

func TestNextHopCannotInstantiate(t *testing.T) {
	// NextHop is an interface in Go: the Python TypeError raised by
	// instantiating the abstract base is enforced at compile time. Verify the
	// zero value is nil.
	var nextHop NextHop
	if nextHop != nil {
		t.Fatal("zero NextHop should be nil")
	}
}

func TestNextHopDeserializationInvalid(t *testing.T) {
	if _, err := NextHopFromDict(map[string]any{"type": "foo"}); err == nil {
		t.Fatal("expected ValueError")
	}
	if _, err := NextHopFromDict(map[string]any{}); err == nil {
		t.Fatal("expected ValueError")
	}
}

func TestNextHopDiscardSerialization(t *testing.T) {
	assertDeepEqual(t, (NextHopDiscard{}).Dict(), map[string]any{"type": "discard"})
}

func TestNextHopDiscardDeserialization(t *testing.T) {
	nh, err := NextHopFromDict(map[string]any{"type": "discard"})
	assertDeepEqual(t, err, nil)
	assertDeepEqual(t, nh, NextHopDiscard{})
}

func TestNextHopDiscardStr(t *testing.T) {
	assertStr(t, (NextHopDiscard{}).String(), "discard")
}

func TestNextHopInterfaceSerialization(t *testing.T) {
	assertDeepEqual(t, (NextHopInterface{Interface: "foo"}).Dict(), map[string]any{
		"type": "interface", "interface": "foo", "ip": nil,
	})
	assertDeepEqual(t, (NextHopInterface{Interface: "foo", IP: StringPtr("1.1.1.1")}).Dict(), map[string]any{
		"type": "interface", "interface": "foo", "ip": "1.1.1.1",
	})
}

func TestNextHopInterfaceDeserialization(t *testing.T) {
	nh, err := NextHopFromDict(map[string]any{"type": "interface", "interface": "foo"})
	assertDeepEqual(t, err, nil)
	assertDeepEqual(t, nh, NextHopInterface{Interface: "foo"})
	assertDeepEqual(t, NextHopInterfaceFromDict(map[string]any{"type": "interface", "interface": "foo"}), NextHopInterface{Interface: "foo"})
	assertDeepEqual(t, NextHopInterfaceFromDict(map[string]any{"type": "interface", "interface": "foo", "ip": nil}), NextHopInterface{Interface: "foo"})
	assertDeepEqual(t, NextHopInterfaceFromDict(map[string]any{"type": "interface", "interface": "foo", "ip": "1.1.1.1"}),
		NextHopInterface{Interface: "foo", IP: StringPtr("1.1.1.1")})
}

func TestNextHopInterfaceStr(t *testing.T) {
	assertStr(t, (NextHopInterface{Interface: "foo"}).String(), "interface foo")
	assertStr(t, (NextHopInterface{Interface: "foo bar"}).String(), `interface "foo bar"`)
	assertStr(t, (NextHopInterface{Interface: "foo bar", IP: StringPtr("1.1.1.1")}).String(), `interface "foo bar" ip 1.1.1.1`)
}

func TestNextHopIPSerialization(t *testing.T) {
	assertDeepEqual(t, (NextHopIP{IP: "1.1.1.1"}).Dict(), map[string]any{"type": "ip", "ip": "1.1.1.1"})
}

func TestNextHopIPDeserialization(t *testing.T) {
	nh, err := NextHopFromDict(map[string]any{"type": "ip", "ip": "1.1.1.1"})
	assertDeepEqual(t, err, nil)
	assertDeepEqual(t, nh, NextHopIP{IP: "1.1.1.1"})
	assertDeepEqual(t, NextHopIPFromDict(map[string]any{"type": "ip", "ip": "1.1.1.1"}), NextHopIP{IP: "1.1.1.1"})
}

func TestNextHopIPStr(t *testing.T) {
	assertStr(t, (NextHopIP{IP: "1.1.1.1"}).String(), "ip 1.1.1.1")
}

func TestNextHopVrfSerialization(t *testing.T) {
	assertDeepEqual(t, (NextHopVrf{Vrf: "foo"}).Dict(), map[string]any{"type": "vrf", "vrf": "foo"})
}

func TestNextHopVrfDeserialization(t *testing.T) {
	nh, err := NextHopFromDict(map[string]any{"type": "vrf", "vrf": "foo"})
	assertDeepEqual(t, err, nil)
	assertDeepEqual(t, nh, NextHopVrf{Vrf: "foo"})
	assertDeepEqual(t, NextHopVrfFromDict(map[string]any{"type": "vrf", "vrf": "foo"}), NextHopVrf{Vrf: "foo"})
}

func TestNextHopVrfStr(t *testing.T) {
	assertStr(t, (NextHopVrf{Vrf: "foo"}).String(), "vrf foo")
	assertStr(t, (NextHopVrf{Vrf: "foo bar"}).String(), `vrf "foo bar"`)
}

func TestNextHopVtepSerialization(t *testing.T) {
	assertDeepEqual(t, (NextHopVtep{Vni: 5, Vtep: "1.1.1.1"}).Dict(), map[string]any{
		"type": "vtep", "vni": 5, "vtep": "1.1.1.1",
	})
}

func TestNextHopVtepDeserialization(t *testing.T) {
	nh, err := NextHopFromDict(map[string]any{"type": "vtep", "vni": 5, "vtep": "1.1.1.1"})
	assertDeepEqual(t, err, nil)
	assertDeepEqual(t, nh, NextHopVtep{Vni: 5, Vtep: "1.1.1.1"})
	assertDeepEqual(t, NextHopVtepFromDict(map[string]any{"type": "vtep", "vni": 5, "vtep": "1.1.1.1"}),
		NextHopVtep{Vni: 5, Vtep: "1.1.1.1"})
}

func TestNextHopVtepStr(t *testing.T) {
	assertStr(t, (NextHopVtep{Vni: 5, Vtep: "1.1.1.1"}).String(), "vni 5 vtep 1.1.1.1")
}
