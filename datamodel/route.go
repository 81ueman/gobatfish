package datamodel

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/81ueman/gobatfish/util"
)

// BgpRoute is a BGP routing advertisement.
type BgpRoute struct {
	Network         string  `json:"network"`
	OriginatorIP    string  `json:"originatorIp"`
	OriginType      string  `json:"originType"`
	Protocol        string  `json:"protocol"`
	AsPath          []any   `json:"asPath"`
	Communities     []any   `json:"communities"`
	LocalPreference int     `json:"localPreference"`
	Metric          int     `json:"metric"`
	NextHopIP       *string `json:"nextHopIp"`
	SourceProtocol  *string `json:"srcProtocol"`
	Tag             int     `json:"tag"`
	Weight          int     `json:"weight"`
}

// BgpRouteFromDict builds a BgpRoute from a dictionary.
func BgpRouteFromDict(d map[string]any) BgpRoute {
	return BgpRoute{
		Network:         strField(d, "network"),
		OriginatorIP:    strField(d, "originatorIp"),
		OriginType:      strField(d, "originType"),
		Protocol:        strField(d, "protocol"),
		AsPath:          sliceField(d, "asPath"),
		Communities:     sliceField(d, "communities"),
		LocalPreference: intField(d, "localPreference"),
		Metric:          intField(d, "metric"),
		NextHopIP:       optStrField(d, "nextHopIp"),
		SourceProtocol:  optStrField(d, "srcProtocol"),
		Tag:             intField(d, "tag"),
		Weight:          intField(d, "weight"),
	}
}

// Dict returns the dictionary representation of the route.
func (r BgpRoute) Dict() map[string]any {
	d := dictOf(r)
	d["class"] = "org.batfish.datamodel.questions.BgpRoute"
	return d
}

// HTMLLines returns the human readable lines used for HTML rendering.
func (r BgpRoute) HTMLLines() []string {
	return []string{
		fmt.Sprintf("Network: %s", r.Network),
		fmt.Sprintf("AS Path: %s", pyListRepr(r.AsPath)),
		"Communities: [" + joinStrings(r.Communities) + "]",
		fmt.Sprintf("Local Preference: %d", r.LocalPreference),
		fmt.Sprintf("Metric: %d", r.Metric),
		fmt.Sprintf("Next Hop IP: %s", optStrVal(r.NextHopIP)),
		fmt.Sprintf("Originator IP: %s", r.OriginatorIP),
		fmt.Sprintf("Origin Type: %s", r.OriginType),
		fmt.Sprintf("Protocol: %s", r.Protocol),
		fmt.Sprintf("Source Protocol: %s", optStrVal(r.SourceProtocol)),
		fmt.Sprintf("Tag: %d", r.Tag),
		fmt.Sprintf("Weight: %d", r.Weight),
	}
}

// HTML returns the HTML representation of the route.
func (r BgpRoute) HTML() string { return strings.Join(r.HTMLLines(), "<br>") }

func longspaceBRCConverter(value any) (any, error) {
	if value == nil {
		return nil, nil
	}
	if s, ok := value.(string); ok {
		return s, nil
	}
	if vals, ok := asList(value); ok {
		parts := make([]string, len(vals))
		for i, v := range vals {
			parts[i] = fmt.Sprint(v)
		}
		return strings.Join(parts, ","), nil
	}
	return nil, fmt.Errorf("invalid value %v", value)
}

func stringListBRCConverter(value any) (any, error) {
	if value == nil {
		return nil, nil
	}
	if s, ok := value.(string); ok {
		return []any{s}, nil
	}
	if vals, ok := asList(value); ok {
		return vals, nil
	}
	return nil, fmt.Errorf("invalid value %v", value)
}

func asList(v any) ([]any, bool) {
	if s, ok := v.([]any); ok {
		return s, true
	}
	if ss, ok := v.([]string); ok {
		out := make([]any, len(ss))
		for i, s := range ss {
			out[i] = s
		}
		return out, true
	}
	rv := reflect.ValueOf(v)
	if rv.Kind() == reflect.Slice || rv.Kind() == reflect.Array {
		out := make([]any, rv.Len())
		for i := 0; i < rv.Len(); i++ {
			out[i] = rv.Index(i).Interface()
		}
		return out, true
	}
	return nil, false
}

// BgpRouteConstraints represents constraints on a BGP route announcement.
type BgpRouteConstraints struct {
	Prefix           []any   `json:"prefix"`
	ComplementPrefix *bool   `json:"complementPrefix"`
	LocalPreference  *string `json:"localPreference"`
	Med              *string `json:"med"`
	Communities      []any   `json:"communities"`
	AsPath           []any   `json:"asPath"`
}

// BgpRouteConstraintsFromDict builds constraints from a dictionary.
func BgpRouteConstraintsFromDict(d map[string]any) (BgpRouteConstraints, error) {
	var out BgpRouteConstraints
	var err error
	if prefix, e := stringListBRCConverter(d["prefix"]); e != nil {
		return out, e
	} else if prefix != nil {
		if vals, ok := prefix.([]any); ok {
			out.Prefix = vals
		}
	}
	out.ComplementPrefix = optBoolField(d, "complementPrefix")
	if out.LocalPreference, err = optStringFromConverter(d["localPreference"], longspaceBRCConverter); err != nil {
		return out, err
	}
	if out.Med, err = optStringFromConverter(d["med"], longspaceBRCConverter); err != nil {
		return out, err
	}
	if communities, e := stringListBRCConverter(d["communities"]); e != nil {
		return out, e
	} else if communities != nil {
		if vals, ok := communities.([]any); ok {
			out.Communities = vals
		}
	}
	if asPath, e := stringListBRCConverter(d["asPath"]); e != nil {
		return out, e
	} else if asPath != nil {
		if vals, ok := asPath.([]any); ok {
			out.AsPath = vals
		}
	}
	return out, nil
}

func optStringFromConverter(value any, conv func(any) (any, error)) (*string, error) {
	res, err := conv(value)
	if err != nil {
		return nil, err
	}
	if res == nil {
		return nil, nil
	}
	s := fmt.Sprint(res)
	return &s, nil
}

// Dict returns the dictionary representation of the constraints.
func (c BgpRouteConstraints) Dict() map[string]any { return dictOf(c) }

// BgpRouteDiff is a difference between two BGP routes.
type BgpRouteDiff struct {
	FieldName string `json:"fieldName"`
	OldValue  string `json:"oldValue"`
	NewValue  string `json:"newValue"`
}

// BgpRouteDiffFromDict builds a BgpRouteDiff from a dictionary.
func BgpRouteDiffFromDict(d map[string]any) BgpRouteDiff {
	return BgpRouteDiff{
		FieldName: strField(d, "fieldName"),
		OldValue:  strField(d, "oldValue"),
		NewValue:  strField(d, "newValue"),
	}
}

// Dict returns the dictionary representation of the diff.
func (d BgpRouteDiff) Dict() map[string]any { return dictOf(d) }

// HTML returns the HTML representation of the diff.
func (d BgpRouteDiff) HTML() string {
	prettyNames := map[string]string{
		"asPath":          "AS Path",
		"localPreference": "Local Preference",
		"metric":          "Metric",
		"nextHopIp":       "Next Hop IP",
		"originatorIp":    "Originator IP",
		"originType":      "Origin Type",
		"sourceProtocol":  "Source Protocol",
		"tag":             "Tag",
		"weight":          "Weight",
	}
	name, ok := prettyNames[d.FieldName]
	if !ok {
		name = capitalize(d.FieldName)
	}
	return fmt.Sprintf("%s: %s --> %s", name, d.OldValue, d.NewValue)
}

// BgpRouteDiffs is a set of differences between two BGP routes.
type BgpRouteDiffs struct {
	Diffs []BgpRouteDiff `json:"diffs"`
}

// BgpRouteDiffsFromDict builds BgpRouteDiffs from a dictionary.
func BgpRouteDiffsFromDict(d map[string]any) BgpRouteDiffs {
	vals := sliceField(d, "diffs")
	out := make([]BgpRouteDiff, 0, len(vals))
	for _, v := range vals {
		out = append(out, BgpRouteDiffFromDict(toMap(v)))
	}
	return BgpRouteDiffs{Diffs: out}
}

// Dict returns the dictionary representation of the diffs.
func (d BgpRouteDiffs) Dict() map[string]any { return dictOf(d) }

// HTML returns the HTML representation of the diffs.
func (d BgpRouteDiffs) HTML() string {
	parts := make([]string, len(d.Diffs))
	for i, diff := range d.Diffs {
		parts[i] = diff.HTML()
	}
	return strings.Join(parts, "<br>")
}

// BgpSessionProperties are the properties of a BGP session.
type BgpSessionProperties struct {
	LocalAs  int    `json:"localAs"`
	RemoteAs int    `json:"remoteAs"`
	LocalIP  string `json:"localIp"`
	RemoteIP string `json:"remoteIp"`
}

// BgpSessionPropertiesFromDict builds session properties from a dictionary.
func BgpSessionPropertiesFromDict(d map[string]any) BgpSessionProperties {
	return BgpSessionProperties{
		LocalAs:  intField(d, "localAs"),
		RemoteAs: intField(d, "remoteAs"),
		LocalIP:  strField(d, "localIp"),
		RemoteIP: strField(d, "remoteIp"),
	}
}

// Dict returns the dictionary representation of the properties.
func (b BgpSessionProperties) Dict() map[string]any { return dictOf(b) }

// NextHop is a next-hop of a route.
type NextHop interface {
	String() string
	Dict() map[string]any
}

// NextHopFromDict builds the appropriate NextHop from a dictionary.
func NextHopFromDict(d map[string]any) (NextHop, error) {
	t, ok := d["type"]
	if !ok {
		return nil, fmt.Errorf("unknown type of NextHop, missing the type property in: %v", d)
	}
	switch t {
	case "discard":
		return NextHopDiscard{}, nil
	case "interface":
		return NextHopInterfaceFromDict(d), nil
	case "ip":
		return NextHopIPFromDict(d), nil
	case "vrf":
		return NextHopVrfFromDict(d), nil
	case "vtep":
		return NextHopVtepFromDict(d), nil
	default:
		return nil, fmt.Errorf("unhandled NextHop type: %v in: %v", t, d)
	}
}

// NextHopDiscard indicates the packet should be dropped.
type NextHopDiscard struct{}

// Dict returns the dictionary representation of the next hop.
func (NextHopDiscard) Dict() map[string]any { return map[string]any{"type": "discard"} }

// String returns "discard".
func (NextHopDiscard) String() string { return "discard" }

// NextHopInterface is a next hop with a fixed output interface and optional gateway IP.
type NextHopInterface struct {
	Interface string  `json:"interface"`
	IP        *string `json:"ip"`
}

// NextHopInterfaceFromDict builds a NextHopInterface from a dictionary.
func NextHopInterfaceFromDict(d map[string]any) NextHopInterface {
	return NextHopInterface{Interface: strField(d, "interface"), IP: optStrField(d, "ip")}
}

// Dict returns the dictionary representation of the next hop.
func (n NextHopInterface) Dict() map[string]any {
	return map[string]any{"type": "interface", "interface": n.Interface, "ip": jsonValue(n.IP)}
}

// String renders the next hop.
func (n NextHopInterface) String() string {
	if n.IP != nil {
		return fmt.Sprintf("interface %s ip %s", util.EscapeName(n.Interface), *n.IP)
	}
	return fmt.Sprintf("interface %s", util.EscapeName(n.Interface))
}

// NextHopIP is a next hop including the next gateway IP.
type NextHopIP struct {
	IP string `json:"ip"`
}

// NextHopIPFromDict builds a NextHopIP from a dictionary.
func NextHopIPFromDict(d map[string]any) NextHopIP { return NextHopIP{IP: strField(d, "ip")} }

// Dict returns the dictionary representation of the next hop.
func (n NextHopIP) Dict() map[string]any {
	return map[string]any{"type": "ip", "ip": n.IP}
}

// String renders the next hop.
func (n NextHopIP) String() string { return fmt.Sprintf("ip %s", n.IP) }

// NextHopVrf indicates the destination IP should be resolved in another VRF.
type NextHopVrf struct {
	Vrf string `json:"vrf"`
}

// NextHopVrfFromDict builds a NextHopVrf from a dictionary.
func NextHopVrfFromDict(d map[string]any) NextHopVrf { return NextHopVrf{Vrf: strField(d, "vrf")} }

// Dict returns the dictionary representation of the next hop.
func (n NextHopVrf) Dict() map[string]any {
	return map[string]any{"type": "vrf", "vrf": n.Vrf}
}

// String renders the next hop.
func (n NextHopVrf) String() string { return fmt.Sprintf("vrf %s", util.EscapeName(n.Vrf)) }

// NextHopVtep indicates the packet should be routed through a VXLAN tunnel.
type NextHopVtep struct {
	Vni  int    `json:"vni"`
	Vtep string `json:"vtep"`
}

// NextHopVtepFromDict builds a NextHopVtep from a dictionary.
func NextHopVtepFromDict(d map[string]any) NextHopVtep {
	return NextHopVtep{Vni: intField(d, "vni"), Vtep: strField(d, "vtep")}
}

// Dict returns the dictionary representation of the next hop.
func (n NextHopVtep) Dict() map[string]any {
	return map[string]any{"type": "vtep", "vni": n.Vni, "vtep": n.Vtep}
}

// String renders the next hop.
func (n NextHopVtep) String() string { return fmt.Sprintf("vni %d vtep %s", n.Vni, n.Vtep) }

func optBoolField(m map[string]any, key string) *bool {
	v, ok := mapGet(m, key)
	if !ok || v == nil {
		return nil
	}
	b := toBool(v)
	return &b
}

func optStrVal(s *string) string {
	if s == nil {
		return "None"
	}
	return *s
}

func pyListRepr(vals []any) string {
	parts := make([]string, len(vals))
	for i, v := range vals {
		parts[i] = fmt.Sprint(v)
	}
	return "[" + strings.Join(parts, ", ") + "]"
}

func joinStrings(vals []any) string {
	parts := make([]string, len(vals))
	for i, v := range vals {
		parts[i] = fmt.Sprint(v)
	}
	return strings.Join(parts, ", ")
}

func capitalize(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}
