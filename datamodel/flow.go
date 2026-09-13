package datamodel

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/81ueman/gobatfish/util"
)

var ipProtocolPattern = regexp.MustCompile(`(?i)^UNNAMED_([0-9]+)$`)

// Flow is a concrete IPv4 flow.
type Flow struct {
	Dscp             int     `json:"dscp"`
	DstIP            string  `json:"dstIp"`
	DstPort          *int    `json:"dstPort"`
	Ecn              int     `json:"ecn"`
	FragmentOffset   int     `json:"fragmentOffset"`
	IcmpCode         *int    `json:"icmpCode"`
	IcmpVar          *int    `json:"icmpVar"`
	IngressInterface *string `json:"ingressInterface"`
	IngressNode      *string `json:"ingressNode"`
	IngressVrf       *string `json:"ingressVrf"`
	IPProtocol       string  `json:"ipProtocol"`
	PacketLength     any     `json:"packetLength"`
	SrcIP            string  `json:"srcIp"`
	SrcPort          *int    `json:"srcPort"`
	TcpFlagsAck      *int    `json:"tcpFlagsAck"`
	TcpFlagsCwr      *int    `json:"tcpFlagsCwr"`
	TcpFlagsEce      *int    `json:"tcpFlagsEce"`
	TcpFlagsFin      *int    `json:"tcpFlagsFin"`
	TcpFlagsPsh      *int    `json:"tcpFlagsPsh"`
	TcpFlagsRst      *int    `json:"tcpFlagsRst"`
	TcpFlagsSyn      *int    `json:"tcpFlagsSyn"`
	TcpFlagsUrg      *int    `json:"tcpFlagsUrg"`
}

// FlowFromDict builds a Flow from a dictionary.
func FlowFromDict(d map[string]any) Flow {
	return Flow{
		Dscp:             intField(d, "dscp"),
		DstIP:            strField(d, "dstIp"),
		DstPort:          optIntField(d, "dstPort"),
		Ecn:              intField(d, "ecn"),
		FragmentOffset:   intField(d, "fragmentOffset"),
		IcmpCode:         optIntField(d, "icmpCode"),
		IcmpVar:          optIntField(d, "icmpVar"),
		IngressInterface: optStrField(d, "ingressInterface"),
		IngressNode:      optStrField(d, "ingressNode"),
		IngressVrf:       optStrField(d, "ingressVrf"),
		IPProtocol:       strField(d, "ipProtocol"),
		PacketLength:     d["packetLength"],
		SrcIP:            strField(d, "srcIp"),
		SrcPort:          optIntField(d, "srcPort"),
		TcpFlagsAck:      optIntField(d, "tcpFlagsAck"),
		TcpFlagsCwr:      optIntField(d, "tcpFlagsCwr"),
		TcpFlagsEce:      optIntField(d, "tcpFlagsEce"),
		TcpFlagsFin:      optIntField(d, "tcpFlagsFin"),
		TcpFlagsPsh:      optIntField(d, "tcpFlagsPsh"),
		TcpFlagsRst:      optIntField(d, "tcpFlagsRst"),
		TcpFlagsSyn:      optIntField(d, "tcpFlagsSyn"),
		TcpFlagsUrg:      optIntField(d, "tcpFlagsUrg"),
	}
}

// Dict returns the dictionary representation of the flow.
func (f Flow) Dict() map[string]any { return dictOf(f) }

// HasPorts reports whether the flow's protocol has ports.
func (f Flow) HasPorts() bool {
	switch f.IPProtocol {
	case "TCP", "UDP", "DCCP", "SCTP":
		return f.SrcPort != nil && f.DstPort != nil
	default:
		return false
	}
}

// GetFlagStr returns a print friendly version of the set TCP flags.
func (f Flow) GetFlagStr() string {
	var flags []string
	if f.TcpFlagsSyn != nil && *f.TcpFlagsSyn != 0 {
		flags = append(flags, "SYN")
	}
	if f.TcpFlagsFin != nil && *f.TcpFlagsFin != 0 {
		flags = append(flags, "FIN")
	}
	if f.TcpFlagsAck != nil && *f.TcpFlagsAck != 0 {
		flags = append(flags, "ACK")
	}
	if f.TcpFlagsRst != nil && *f.TcpFlagsRst != 0 {
		flags = append(flags, "RST")
	}
	if f.TcpFlagsCwr != nil && *f.TcpFlagsCwr != 0 {
		flags = append(flags, "CWR")
	}
	if f.TcpFlagsEce != nil && *f.TcpFlagsEce != 0 {
		flags = append(flags, "ECE")
	}
	if f.TcpFlagsPsh != nil && *f.TcpFlagsPsh != 0 {
		flags = append(flags, "PSH")
	}
	if f.TcpFlagsUrg != nil && *f.TcpFlagsUrg != 0 {
		flags = append(flags, "URG")
	}
	if len(flags) == 0 {
		return "no flags set"
	}
	return strings.Join(flags, "-")
}

// GetIPProtocolStr returns a print friendly protocol description.
func (f Flow) GetIPProtocolStr() string {
	if m := ipProtocolPattern.FindStringSubmatch(f.IPProtocol); m != nil {
		return "ipProtocol=" + m[1]
	}
	switch strings.ToLower(f.IPProtocol) {
	case "tcp":
		return fmt.Sprintf("TCP (%s)", f.GetFlagStr())
	case "icmp":
		return fmt.Sprintf("ICMP (type=%s, code=%s)", optIntVal(f.IcmpVar), optIntVal(f.IcmpCode))
	default:
		return f.IPProtocol
	}
}

// String renders the flow.
func (f Flow) String() string {
	return fmt.Sprintf("start=%s%s%s [%s->%s %s%s%s%s%s]",
		ptrStr(f.IngressNode),
		f.ifaceStr(),
		f.vrfStr(),
		f.ipPort(f.SrcIP, f.SrcPort),
		f.ipPort(f.DstIP, f.DstPort),
		f.GetIPProtocolStr(),
		f.dscpStr(),
		f.ecnStr(),
		f.offsetStr(),
		f.lengthStr(),
	)
}

func (f Flow) ifaceStr() string {
	if f.IngressInterface != nil {
		return " interface=" + *f.IngressInterface
	}
	return ""
}

func (f Flow) vrfStr() string {
	if f.IngressVrf == nil || *f.IngressVrf == "default" {
		return ""
	}
	return " vrf=" + *f.IngressVrf
}

func (f Flow) dscpStr() string {
	if f.Dscp != 0 {
		return fmt.Sprintf(" dscp=%d", f.Dscp)
	}
	return ""
}

func (f Flow) ecnStr() string {
	if f.Ecn != 0 {
		return fmt.Sprintf(" ecn=%d", f.Ecn)
	}
	return ""
}

func (f Flow) offsetStr() string {
	if f.FragmentOffset != 0 {
		return fmt.Sprintf(" fragmentOffset=%d", f.FragmentOffset)
	}
	return ""
}

func (f Flow) lengthStr() string {
	if isDefaultPacketLength(f.PacketLength) {
		return ""
	}
	return fmt.Sprintf(" length=%v", f.PacketLength)
}

func (f Flow) ipPort(ip string, port *int) string {
	if f.HasPorts() && port != nil {
		return fmt.Sprintf("%s:%d", ip, *port)
	}
	return ip
}

// HTMLLines returns the human readable lines used for HTML rendering.
func (f Flow) HTMLLines() []string {
	lines := []string{
		fmt.Sprintf("Start Location: %s%s%s", ptrStr(f.IngressNode), f.ifaceStr(), f.vrfStr()),
		fmt.Sprintf("Src IP: %s", f.SrcIP),
	}
	if f.HasPorts() {
		lines = append(lines, fmt.Sprintf("Src Port: %d", *f.SrcPort))
	}
	lines = append(lines, fmt.Sprintf("Dst IP: %s", f.DstIP))
	if f.HasPorts() {
		lines = append(lines, fmt.Sprintf("Dst Port: %d", *f.DstPort))
	}
	lines = append(lines, fmt.Sprintf("IP Protocol: %s", f.GetIPProtocolStr()))
	if f.Dscp != 0 {
		lines = append(lines, fmt.Sprintf("DSCP: %d", f.Dscp))
	}
	if f.Ecn != 0 {
		lines = append(lines, fmt.Sprintf("ECN: %d", f.Ecn))
	}
	if f.FragmentOffset != 0 {
		lines = append(lines, fmt.Sprintf("Fragment Offset: %d", f.FragmentOffset))
	}
	if !isDefaultPacketLength(f.PacketLength) {
		lines = append(lines, fmt.Sprintf("Packet Length: %v", f.PacketLength))
	}
	return lines
}

// HTML returns the HTML representation of the flow.
func (f Flow) HTML() string { return strings.Join(f.HTMLLines(), "<br>") }

func isDefaultPacketLength(v any) bool {
	switch n := v.(type) {
	case nil:
		return false
	case int:
		return n == 512
	case int64:
		return n == 512
	case float64:
		return n == 512
	default:
		return false
	}
}

// FlowDiff is a difference between two Flows.
type FlowDiff struct {
	FieldName string `json:"fieldName"`
	OldValue  string `json:"oldValue"`
	NewValue  string `json:"newValue"`
}

// FlowDiffFromDict builds a FlowDiff from a dictionary.
func FlowDiffFromDict(d map[string]any) FlowDiff {
	return FlowDiff{FieldName: strField(d, "fieldName"), OldValue: strField(d, "oldValue"), NewValue: strField(d, "newValue")}
}

// Dict returns the dictionary representation of the diff.
func (d FlowDiff) Dict() map[string]any { return dictOf(d) }

// String renders the diff.
func (d FlowDiff) String() string {
	return fmt.Sprintf("%s: %s -> %s", d.FieldName, d.OldValue, d.NewValue)
}

// FlowTrace is a trace of a flow through the network.
type FlowTrace struct {
	Disposition string         `json:"disposition"`
	Hops        []FlowTraceHop `json:"hops"`
	Notes       any            `json:"notes"`
}

// FlowTraceFromDict builds a FlowTrace from a dictionary.
func FlowTraceFromDict(d map[string]any) FlowTrace {
	hops := sliceField(d, "hops")
	out := make([]FlowTraceHop, 0, len(hops))
	for _, h := range hops {
		out = append(out, FlowTraceHopFromDict(toMap(h)))
	}
	return FlowTrace{Disposition: strField(d, "disposition"), Hops: out, Notes: d["notes"]}
}

// Dict returns the dictionary representation of the trace.
func (t FlowTrace) Dict() map[string]any { return dictOf(t) }

// Len returns the number of hops.
func (t FlowTrace) Len() int { return len(t.Hops) }

// Hop returns the i-th hop.
func (t FlowTrace) Hop(i int) FlowTraceHop { return t.Hops[i] }

// String renders the trace.
func (t FlowTrace) String() string {
	parts := make([]string, len(t.Hops))
	for i, hop := range t.Hops {
		parts[i] = fmt.Sprintf("%d %s", i+1, hop.String())
	}
	return strings.Join(parts, "\n") + "\n" + fmt.Sprint(t.Notes)
}

// HTML returns the HTML representation of the trace.
func (t FlowTrace) HTML() string {
	parts := make([]string, len(t.Hops))
	for i, hop := range t.Hops {
		parts[i] = fmt.Sprintf("<strong>%d</strong> %s", i+1, hop.HTML())
	}
	return fmt.Sprintf(`%s<br>%s`, t.NotesHTML(), strings.Join(parts, "<br><br>"))
}

// NotesHTML renders the notes with a disposition dependent color.
func (t FlowTrace) NotesHTML() string {
	return fmt.Sprintf(`<span style="color:%s; text-weight:bold;">%s</span>`,
		colorForDisposition(t.Disposition), util.EscapeHTML(fmt.Sprint(t.Notes)))
}

// FlowTraceHop is a single hop in a flow trace.
type FlowTraceHop struct {
	Edge            Edge  `json:"edge"`
	Routes          []any `json:"routes"`
	TransformedFlow *Flow `json:"transformedFlow"`
}

// FlowTraceHopFromDict builds a FlowTraceHop from a dictionary.
func FlowTraceHopFromDict(d map[string]any) FlowTraceHop {
	var transformed *Flow
	if tf := mapField(d, "transformedFlow"); tf != nil {
		f := FlowFromDict(tf)
		transformed = &f
	}
	return FlowTraceHop{
		Edge:            EdgeFromDict(mapField(d, "edge")),
		Routes:          sliceField(d, "routes"),
		TransformedFlow: transformed,
	}
}

// Dict returns the dictionary representation of the hop.
func (h FlowTraceHop) Dict() map[string]any { return dictOf(h) }

// String renders the hop.
func (h FlowTraceHop) String() string {
	ret := fmt.Sprintf("%s\n    Route(s):\n    %s", h.Edge.String(), joinAny(h.Routes, "\n    "))
	if h.TransformedFlow != nil {
		ret += fmt.Sprintf("\n    Transformed flow: %s", h.TransformedFlow.String())
	}
	return ret
}

// HTML returns the HTML representation of the hop.
func (h FlowTraceHop) HTML() string {
	indent := strings.Repeat("&nbsp;", 4)
	parts := make([]string, len(h.Routes))
	for i, r := range h.Routes {
		parts[i] = util.EscapeHTML(fmt.Sprint(r))
	}
	result := fmt.Sprintf("%s<br>Route(s):<br>%s%s", h.Edge.HTML(), indent, strings.Join(parts, "<br>"+indent))
	if h.TransformedFlow != nil {
		result += fmt.Sprintf("<br>Transformed flow: %s", h.TransformedFlow.HTML())
	}
	return result
}

// SessionAction is an action a firewall session takes for return traffic.
type SessionAction interface {
	String() string
	Dict() map[string]any
}

// SessionActionFromDict builds the appropriate SessionAction from a dictionary.
func SessionActionFromDict(d map[string]any) (SessionAction, error) {
	switch strField(d, "type") {
	case "Accept":
		return Accept{}, nil
	case "PreNatFibLookup":
		return PreNatFibLookup{}, nil
	case "PostNatFibLookup", "FibLookup":
		return PostNatFibLookup{}, nil
	case "ForwardOutInterface":
		return ForwardOutInterfaceFromDict(d), nil
	default:
		return nil, fmt.Errorf("invalid session action type: %v", d["type"])
	}
}

// Accept accepts return traffic at the originating node.
type Accept struct{}

// Dict returns the dictionary representation of the action.
func (Accept) Dict() map[string]any { return map[string]any{"type": "Accept"} }

// String renders the action.
func (Accept) String() string { return "Accept" }

// PreNatFibLookup forwards return traffic using a pre-NAT FIB lookup.
type PreNatFibLookup struct{}

// Dict returns the dictionary representation of the action.
func (PreNatFibLookup) Dict() map[string]any { return map[string]any{"type": "PreNatFibLookup"} }

// String renders the action.
func (PreNatFibLookup) String() string { return "PreNatFibLookup" }

// PostNatFibLookup forwards return traffic using a post-NAT FIB lookup.
type PostNatFibLookup struct{}

// Dict returns the dictionary representation of the action.
func (PostNatFibLookup) Dict() map[string]any { return map[string]any{"type": "PostNatFibLookup"} }

// String renders the action.
func (PostNatFibLookup) String() string { return "PostNatFibLookup" }

// ForwardOutInterface forwards a return flow out a specified interface.
type ForwardOutInterface struct {
	NextHopHostname   string `json:"nextHopHostname"`
	NextHopInterface  string `json:"nextHopInterface"`
	OutgoingInterface string `json:"outgoingInterface"`
}

// ForwardOutInterfaceFromDict builds a ForwardOutInterface from a dictionary.
func ForwardOutInterfaceFromDict(d map[string]any) ForwardOutInterface {
	nextHop := mapField(d, "nextHop")
	return ForwardOutInterface{
		NextHopHostname:   strField(nextHop, "hostname"),
		NextHopInterface:  strField(nextHop, "interface"),
		OutgoingInterface: strField(d, "outgoingInterface"),
	}
}

// Dict returns the dictionary representation of the action.
func (f ForwardOutInterface) Dict() map[string]any { return dictOf(f) }

// String renders the action.
func (f ForwardOutInterface) String() string {
	return fmt.Sprintf("ForwardOutInterface(Next Hop: %s, Next Hop Interface: %s, Outgoing Interface: %s)",
		f.NextHopHostname, f.NextHopInterface, f.OutgoingInterface)
}

// SessionMatchExpr represents match criteria for a firewall session.
type SessionMatchExpr struct {
	IPProtocol string `json:"ipProtocol"`
	SrcIP      string `json:"srcIp"`
	DstIP      string `json:"dstIp"`
	SrcPort    *int   `json:"srcPort"`
	DstPort    *int   `json:"dstPort"`
}

// SessionMatchExprFromDict builds a SessionMatchExpr from a dictionary.
func SessionMatchExprFromDict(d map[string]any) SessionMatchExpr {
	return SessionMatchExpr{
		IPProtocol: strField(d, "ipProtocol"),
		SrcIP:      strField(d, "srcIp"),
		DstIP:      strField(d, "dstIp"),
		SrcPort:    optIntField(d, "srcPort"),
		DstPort:    optIntField(d, "dstPort"),
	}
}

// Dict returns the dictionary representation of the match expression.
func (m SessionMatchExpr) Dict() map[string]any { return dictOf(m) }

// String renders the match expression.
func (m SessionMatchExpr) String() string {
	fields := []string{"ipProtocol", "srcIp", "dstIp"}
	if m.SrcPort != nil && m.DstPort != nil {
		fields = append(fields, "srcPort", "dstPort")
	}
	parts := make([]string, len(fields))
	for i, f := range fields {
		parts[i] = fmt.Sprintf("%s=%s", f, m.fieldValue(f))
	}
	return "[" + strings.Join(parts, ", ") + "]"
}

func (m SessionMatchExpr) fieldValue(field string) string {
	switch field {
	case "ipProtocol":
		return m.IPProtocol
	case "srcIp":
		return m.SrcIP
	case "dstIp":
		return m.DstIP
	case "srcPort":
		return fmt.Sprint(*m.SrcPort)
	case "dstPort":
		return fmt.Sprint(*m.DstPort)
	default:
		return ""
	}
}

// SessionScope represents the scope of a firewall session.
type SessionScope interface {
	String() string
	Dict() map[string]any
}

// SessionScopeFromDict builds the appropriate SessionScope from a dictionary.
func SessionScopeFromDict(d map[string]any) (SessionScope, error) {
	if _, ok := d["incomingInterfaces"]; ok {
		return IncomingSessionScopeFromDict(d), nil
	}
	if _, ok := d["originatingVrf"]; ok {
		return OriginatingSessionScopeFromDict(d), nil
	}
	return nil, fmt.Errorf("invalid session scope: %v", d)
}

// IncomingSessionScope is a session scope based on incoming interfaces.
type IncomingSessionScope struct {
	IncomingInterfaces []string `json:"incomingInterfaces"`
}

// IncomingSessionScopeFromDict builds an IncomingSessionScope from a dictionary.
func IncomingSessionScopeFromDict(d map[string]any) IncomingSessionScope {
	return IncomingSessionScope{IncomingInterfaces: stringSliceField(d, "incomingInterfaces")}
}

// Dict returns the dictionary representation of the scope.
func (s IncomingSessionScope) Dict() map[string]any { return dictOf(s) }

// String renders the scope.
func (s IncomingSessionScope) String() string {
	return fmt.Sprintf("Incoming Interfaces: [%s]", strings.Join(s.IncomingInterfaces, ", "))
}

// OriginatingSessionScope is a session scope based on an originating VRF.
type OriginatingSessionScope struct {
	OriginatingVrf string `json:"originatingVrf"`
}

// OriginatingSessionScopeFromDict builds an OriginatingSessionScope from a dictionary.
func OriginatingSessionScopeFromDict(d map[string]any) OriginatingSessionScope {
	return OriginatingSessionScope{OriginatingVrf: strField(d, "originatingVrf")}
}

// Dict returns the dictionary representation of the scope.
func (s OriginatingSessionScope) Dict() map[string]any { return dictOf(s) }

// String renders the scope.
func (s OriginatingSessionScope) String() string {
	return fmt.Sprintf("Originating VRF: %s", s.OriginatingVrf)
}

// ArpErrorStepDetail details an ARP error step.
type ArpErrorStepDetail struct {
	OutputInterface   *string `json:"outputInterface"`
	ResolvedNexthopIP *string `json:"resolvedNexthopIp"`
}

// ArpErrorStepDetailFromDict builds details from a dictionary.
func ArpErrorStepDetailFromDict(d map[string]any) ArpErrorStepDetail {
	return ArpErrorStepDetail{
		OutputInterface:   optStrField(mapField(d, "outputInterface"), "interface"),
		ResolvedNexthopIP: optStrField(d, "resolvedNexthopIp"),
	}
}

// Dict returns the dictionary representation of the details.
func (a ArpErrorStepDetail) Dict() map[string]any { return dictOf(a) }

// String renders the details.
func (a ArpErrorStepDetail) String() string {
	return joinDetail([]string{
		optionalDetail("Output Interface: ", a.OutputInterface),
		optionalDetail("Resolved Next Hop IP: ", a.ResolvedNexthopIP),
	})
}

// DeliveredStepDetail details a delivered step.
type DeliveredStepDetail struct {
	OutputInterface   *string `json:"outputInterface"`
	ResolvedNexthopIP *string `json:"resolvedNexthopIp"`
}

// DeliveredStepDetailFromDict builds details from a dictionary.
func DeliveredStepDetailFromDict(d map[string]any) DeliveredStepDetail {
	return DeliveredStepDetail{
		OutputInterface:   optStrField(mapField(d, "outputInterface"), "interface"),
		ResolvedNexthopIP: optStrField(d, "resolvedNexthopIp"),
	}
}

// Dict returns the dictionary representation of the details.
func (a DeliveredStepDetail) Dict() map[string]any { return dictOf(a) }

// String renders the details.
func (a DeliveredStepDetail) String() string {
	return joinDetail([]string{
		optionalDetail("Output Interface: ", a.OutputInterface),
		optionalDetail("Resolved Next Hop IP: ", a.ResolvedNexthopIP),
	})
}

// EnterInputIfaceStepDetail details entering a flow into a hop.
type EnterInputIfaceStepDetail struct {
	InputInterface string  `json:"inputInterface"`
	InputVrf       *string `json:"inputVrf"`
}

// EnterInputIfaceStepDetailFromDict builds details from a dictionary.
func EnterInputIfaceStepDetailFromDict(d map[string]any) EnterInputIfaceStepDetail {
	return EnterInputIfaceStepDetail{
		InputInterface: strField(mapField(d, "inputInterface"), "interface"),
		InputVrf:       optStrField(d, "inputVrf"),
	}
}

// Dict returns the dictionary representation of the details.
func (a EnterInputIfaceStepDetail) Dict() map[string]any { return dictOf(a) }

// String renders the details.
func (a EnterInputIfaceStepDetail) String() string { return a.InputInterface }

// ExitOutputIfaceStepDetail details exiting a flow from a hop.
type ExitOutputIfaceStepDetail struct {
	OutputInterface string `json:"outputInterface"`
	TransformedFlow any    `json:"transformedFlow"`
}

// ExitOutputIfaceStepDetailFromDict builds details from a dictionary.
func ExitOutputIfaceStepDetailFromDict(d map[string]any) ExitOutputIfaceStepDetail {
	return ExitOutputIfaceStepDetail{
		OutputInterface: strField(mapField(d, "outputInterface"), "interface"),
		TransformedFlow: d["transformedFlow"],
	}
}

// Dict returns the dictionary representation of the details.
func (a ExitOutputIfaceStepDetail) Dict() map[string]any { return dictOf(a) }

// String renders the details.
func (a ExitOutputIfaceStepDetail) String() string { return a.OutputInterface }

// InboundStepDetail details receiving a flow into a hop.
type InboundStepDetail struct {
	Interface string `json:"interface"`
}

// InboundStepDetailFromDict builds details from a dictionary.
func InboundStepDetailFromDict(d map[string]any) InboundStepDetail {
	return InboundStepDetail{Interface: strField(d, "interface")}
}

// Dict returns the dictionary representation of the details.
func (a InboundStepDetail) Dict() map[string]any { return dictOf(a) }

// String renders the details.
func (a InboundStepDetail) String() string { return a.Interface }

// LoopStepDetail details a forwarding loop.
type LoopStepDetail struct{}

// LoopStepDetailFromDict builds details from a dictionary.
func LoopStepDetailFromDict(map[string]any) LoopStepDetail { return LoopStepDetail{} }

// Dict returns the dictionary representation of the details.
func (LoopStepDetail) Dict() map[string]any { return map[string]any{} }

// String renders the details.
func (LoopStepDetail) String() string { return "" }

// MatchSessionStepDetail details matching a firewall session.
type MatchSessionStepDetail struct {
	SessionScope   SessionScope     `json:"sessionScope"`
	SessionAction  SessionAction    `json:"sessionAction"`
	MatchCriteria  SessionMatchExpr `json:"matchCriteria"`
	Transformation []FlowDiff       `json:"transformation"`
}

// MatchSessionStepDetailFromDict builds details from a dictionary.
func MatchSessionStepDetailFromDict(d map[string]any) MatchSessionStepDetail {
	return MatchSessionStepDetail{
		SessionScope:   sessionScopeFromDictCompat(d),
		SessionAction:  mustSessionAction(d),
		MatchCriteria:  SessionMatchExprFromDict(mapField(d, "matchCriteria")),
		Transformation: flowDiffsFrom(d),
	}
}

// Dict returns the dictionary representation of the details.
func (a MatchSessionStepDetail) Dict() map[string]any { return dictOf(a) }

// String renders the details.
func (a MatchSessionStepDetail) String() string {
	return sessionDetailString(a.SessionScope, a.SessionAction, a.MatchCriteria, a.Transformation)
}

// SetupSessionStepDetail details setting up a firewall session.
type SetupSessionStepDetail struct {
	SessionScope   SessionScope     `json:"sessionScope"`
	SessionAction  SessionAction    `json:"sessionAction"`
	MatchCriteria  SessionMatchExpr `json:"matchCriteria"`
	Transformation []FlowDiff       `json:"transformation"`
}

// SetupSessionStepDetailFromDict builds details from a dictionary.
func SetupSessionStepDetailFromDict(d map[string]any) SetupSessionStepDetail {
	return SetupSessionStepDetail{
		SessionScope:   sessionScopeFromDictCompat(d),
		SessionAction:  mustSessionAction(d),
		MatchCriteria:  SessionMatchExprFromDict(mapField(d, "matchCriteria")),
		Transformation: flowDiffsFrom(d),
	}
}

// Dict returns the dictionary representation of the details.
func (a SetupSessionStepDetail) Dict() map[string]any { return dictOf(a) }

// String renders the details.
func (a SetupSessionStepDetail) String() string {
	return sessionDetailString(a.SessionScope, a.SessionAction, a.MatchCriteria, a.Transformation)
}

// OriginateStepDetail details originating a flow in a hop.
type OriginateStepDetail struct {
	OriginatingVrf string `json:"originatingVrf"`
}

// OriginateStepDetailFromDict builds details from a dictionary.
func OriginateStepDetailFromDict(d map[string]any) OriginateStepDetail {
	return OriginateStepDetail{OriginatingVrf: strField(d, "originatingVrf")}
}

// Dict returns the dictionary representation of the details.
func (a OriginateStepDetail) Dict() map[string]any { return dictOf(a) }

// String renders the details.
func (a OriginateStepDetail) String() string { return a.OriginatingVrf }

// FilterStepDetail details a filter step.
type FilterStepDetail struct {
	Filter         string `json:"filter"`
	FilterType     string `json:"filterType"`
	InputInterface string `json:"inputInterface"`
	Flow           *Flow  `json:"flow"`
}

// FilterStepDetailFromDict builds details from a dictionary.
func FilterStepDetailFromDict(d map[string]any) FilterStepDetail {
	var flow *Flow
	if fm := mapField(d, "flow"); fm != nil {
		f := FlowFromDict(fm)
		flow = &f
	}
	return FilterStepDetail{
		Filter:         strField(d, "filter"),
		FilterType:     strField(d, "type"),
		InputInterface: strField(d, "inputInterface"),
		Flow:           flow,
	}
}

// Dict returns the dictionary representation of the details.
func (a FilterStepDetail) Dict() map[string]any { return dictOf(a) }

// String renders the details.
func (a FilterStepDetail) String() string { return fmt.Sprintf("%s (%s)", a.Filter, a.FilterType) }

// PolicyStepDetail details a generic policy processing step.
type PolicyStepDetail struct {
	Policy string `json:"policy"`
}

// PolicyStepDetailFromDict builds details from a dictionary.
func PolicyStepDetailFromDict(d map[string]any) PolicyStepDetail {
	return PolicyStepDetail{Policy: strField(d, "policy")}
}

// Dict returns the dictionary representation of the details.
func (a PolicyStepDetail) Dict() map[string]any { return dictOf(a) }

// String renders the details.
func (a PolicyStepDetail) String() string { return a.Policy }

// TransformationStepDetail details a packet transformation.
type TransformationStepDetail struct {
	TransformationType string     `json:"transformationType"`
	FlowDiffs          []FlowDiff `json:"flowDiffs"`
}

// TransformationStepDetailFromDict builds details from a dictionary.
func TransformationStepDetailFromDict(d map[string]any) TransformationStepDetail {
	return TransformationStepDetail{
		TransformationType: strField(d, "transformationType"),
		FlowDiffs:          flowDiffsFrom(d),
	}
}

// Dict returns the dictionary representation of the details.
func (a TransformationStepDetail) Dict() map[string]any { return dictOf(a) }

// String renders the details.
func (a TransformationStepDetail) String() string {
	if len(a.FlowDiffs) == 0 {
		return a.TransformationType
	}
	parts := make([]string, len(a.FlowDiffs))
	for i, d := range a.FlowDiffs {
		parts[i] = d.String()
	}
	return fmt.Sprintf("%s %s", a.TransformationType, strings.Join(parts, ", "))
}

// ForwardingDetail details how a flow is forwarded.
type ForwardingDetail interface {
	String() string
	Dict() map[string]any
}

// ForwardingDetailFromDict builds the appropriate ForwardingDetail.
func ForwardingDetailFromDict(d map[string]any) (ForwardingDetail, error) {
	switch strField(d, "type") {
	case "DelegatedToNextVrf":
		return DelegatedToNextVrfFromDict(d), nil
	case "ForwardedIntoVxlanTunnel":
		return ForwardedIntoVxlanTunnelFromDict(d), nil
	case "ForwardedOutInterface":
		return ForwardedOutInterfaceFromDict(d), nil
	case "Discarded":
		return Discarded{}, nil
	default:
		return nil, fmt.Errorf("unhandled ForwardingDetail type: %v in: %v", d["type"], d)
	}
}

// DelegatedToNextVrf delegates a flow to a different VRF.
type DelegatedToNextVrf struct {
	NextVrf string `json:"nextVrf"`
}

// DelegatedToNextVrfFromDict builds a DelegatedToNextVrf from a dictionary.
func DelegatedToNextVrfFromDict(d map[string]any) DelegatedToNextVrf {
	return DelegatedToNextVrf{NextVrf: strField(d, "nextVrf")}
}

// Dict returns the dictionary representation of the detail.
func (f DelegatedToNextVrf) Dict() map[string]any {
	return map[string]any{"type": "DelegatedToNextVrf", "nextVrf": f.NextVrf}
}

// String renders the detail.
func (f DelegatedToNextVrf) String() string {
	return fmt.Sprintf("Delegated to next VRF: %s", util.EscapeName(f.NextVrf))
}

// ForwardedIntoVxlanTunnel forwards a flow into a VXLAN tunnel.
type ForwardedIntoVxlanTunnel struct {
	Vni  int    `json:"vni"`
	Vtep string `json:"vtep"`
}

// ForwardedIntoVxlanTunnelFromDict builds a ForwardedIntoVxlanTunnel from a dictionary.
func ForwardedIntoVxlanTunnelFromDict(d map[string]any) ForwardedIntoVxlanTunnel {
	return ForwardedIntoVxlanTunnel{Vni: intField(d, "vni"), Vtep: strField(d, "vtep")}
}

// Dict returns the dictionary representation of the detail.
func (f ForwardedIntoVxlanTunnel) Dict() map[string]any {
	return map[string]any{"type": "ForwardedIntoVxlanTunnel", "vni": f.Vni, "vtep": f.Vtep}
}

// String renders the detail.
func (f ForwardedIntoVxlanTunnel) String() string {
	return fmt.Sprintf("Forwarded into VXLAN tunnel with VNI: %d and VTEP: %s", f.Vni, f.Vtep)
}

// ForwardedOutInterface forwards a flow out an interface.
type ForwardedOutInterface struct {
	OutputInterface   string  `json:"outputInterface"`
	ResolvedNextHopIP *string `json:"resolvedNextHopIp"`
}

// ForwardedOutInterfaceFromDict builds a ForwardedOutInterface from a dictionary.
func ForwardedOutInterfaceFromDict(d map[string]any) ForwardedOutInterface {
	return ForwardedOutInterface{
		OutputInterface:   strField(d, "outputInterface"),
		ResolvedNextHopIP: optStrField(d, "resolvedNextHopIp"),
	}
}

// Dict returns the dictionary representation of the detail.
func (f ForwardedOutInterface) Dict() map[string]any {
	return map[string]any{
		"type":              "ForwardedOutInterface",
		"outputInterface":   f.OutputInterface,
		"resolvedNextHopIp": jsonValue(f.ResolvedNextHopIP),
	}
}

// String renders the detail.
func (f ForwardedOutInterface) String() string {
	if f.ResolvedNextHopIP != nil {
		return fmt.Sprintf("Forwarded out interface: %s with resolved next-hop IP: %s",
			util.EscapeName(f.OutputInterface), *f.ResolvedNextHopIP)
	}
	return fmt.Sprintf("Forwarded out interface: %s", util.EscapeName(f.OutputInterface))
}

// Discarded discards a flow.
type Discarded struct{}

// Dict returns the dictionary representation of the detail.
func (Discarded) Dict() map[string]any { return map[string]any{"type": "Discarded"} }

// String renders the detail.
func (Discarded) String() string { return "Discarded" }

// RouteInfo contains information about a route that led to a forwarding action.
type RouteInfo struct {
	Protocol  string  `json:"protocol"`
	Network   string  `json:"network"`
	NextHop   NextHop `json:"nextHop"`
	NextHopIP *string `json:"nextHopIp"`
	Admin     int     `json:"admin"`
	Metric    int     `json:"metric"`
}

// RouteInfoFromDict builds a RouteInfo from a dictionary.
func RouteInfoFromDict(d map[string]any) RouteInfo {
	var nextHop NextHop
	if nh := mapField(d, "nextHop"); nh != nil {
		nextHop, _ = NextHopFromDict(nh)
	}
	return RouteInfo{
		Protocol:  strField(d, "protocol"),
		Network:   strField(d, "network"),
		NextHop:   nextHop,
		NextHopIP: optStrField(d, "nextHopIp"),
		Admin:     intField(d, "admin"),
		Metric:    intField(d, "metric"),
	}
}

// Dict returns the dictionary representation of the route info.
func (r RouteInfo) Dict() map[string]any { return dictOf(r) }

// String renders the route info.
func (r RouteInfo) String() string {
	if r.NextHop == nil {
		return fmt.Sprintf("%s (Network: %s, Next Hop IP:%s)", r.Protocol, r.Network, optStrVal(r.NextHopIP))
	}
	return fmt.Sprintf("%s (Network: %s, Next Hop: %s)", r.Protocol, r.Network, r.NextHop.String())
}

// RoutingStepDetail details routing from input to output interface.
type RoutingStepDetail struct {
	Routes           []RouteInfo      `json:"routes"`
	ForwardingDetail ForwardingDetail `json:"forwardingDetail"`
	ArpIP            *string          `json:"arpIp"`
	OutputInterface  *string          `json:"outputInterface"`
}

// RoutingStepDetailFromDict builds details from a dictionary.
func RoutingStepDetailFromDict(d map[string]any) RoutingStepDetail {
	vals := sliceField(d, "routes")
	routes := make([]RouteInfo, 0, len(vals))
	for _, v := range vals {
		routes = append(routes, RouteInfoFromDict(toMap(v)))
	}
	var fd ForwardingDetail
	if fdm := mapField(d, "forwardingDetail"); fdm != nil {
		fd, _ = ForwardingDetailFromDict(fdm)
	}
	return RoutingStepDetail{
		Routes:           routes,
		ForwardingDetail: fd,
		ArpIP:            optStrField(d, "arpIp"),
		OutputInterface:  optStrField(d, "outputInterface"),
	}
}

// Dict returns the dictionary representation of the details.
func (a RoutingStepDetail) Dict() map[string]any { return dictOf(a) }

// String renders the details.
func (a RoutingStepDetail) String() string {
	if a.ForwardingDetail == nil {
		var parts []string
		if a.ArpIP != nil {
			parts = append(parts, "ARP IP: "+*a.ArpIP)
		}
		if a.OutputInterface != nil {
			parts = append(parts, "Output Interface: "+*a.OutputInterface)
		}
		if len(a.Routes) > 0 {
			parts = append(parts, "Routes: ["+joinRouteInfo(a.Routes)+"]")
		}
		return strings.Join(parts, ", ")
	}
	parts := []string{a.ForwardingDetail.String()}
	if len(a.Routes) > 0 {
		parts = append(parts, "Routes: ["+joinRouteInfo(a.Routes)+"]")
	}
	return strings.Join(parts, ", ")
}

func joinRouteInfo(routes []RouteInfo) string {
	parts := make([]string, len(routes))
	for i, r := range routes {
		parts[i] = r.String()
	}
	return strings.Join(parts, ",")
}

// Step is a step in a hop.
type Step struct {
	Detail any    `json:"detail"`
	Action string `json:"action"`
}

// StepFromDict builds a Step from a dictionary. It returns nil for unknown step
// types, matching pybatfish.
func StepFromDict(d map[string]any) *Step {
	detail := mapField(d, "detail")
	action := strField(d, "action")
	var detailVal any
	switch strField(d, "type") {
	case "ArpError":
		detailVal = ArpErrorStepDetailFromDict(detail)
	case "Delivered":
		detailVal = DeliveredStepDetailFromDict(detail)
	case "EnterInputInterface":
		detailVal = EnterInputIfaceStepDetailFromDict(detail)
	case "ExitOutputInterface":
		detailVal = ExitOutputIfaceStepDetailFromDict(detail)
	case "Inbound":
		detailVal = InboundStepDetailFromDict(detail)
	case "Loop":
		detailVal = LoopStepDetailFromDict(detail)
	case "MatchSession":
		detailVal = MatchSessionStepDetailFromDict(detail)
	case "Originate":
		detailVal = OriginateStepDetailFromDict(detail)
	case "Routing":
		detailVal = RoutingStepDetailFromDict(detail)
	case "SetupSession":
		detailVal = SetupSessionStepDetailFromDict(detail)
	case "Transformation":
		detailVal = TransformationStepDetailFromDict(detail)
	case "Policy":
		detailVal = PolicyStepDetailFromDict(detail)
	case "Filter":
		detailVal = FilterStepDetailFromDict(detail)
	default:
		return nil
	}
	return &Step{Detail: detailVal, Action: action}
}

// Dict returns the dictionary representation of the step.
func (s Step) Dict() map[string]any { return dictOf(s) }

// String renders the step.
func (s Step) String() string {
	if s.Detail != nil {
		if ds := detailString(s.Detail); ds != "" {
			return fmt.Sprintf("%s(%s)", s.Action, ds)
		}
	}
	return s.Action
}

// HTML returns the HTML representation of the step.
func (s Step) HTML() string { return s.String() }

func detailString(detail any) string {
	switch d := detail.(type) {
	case interface{ String() string }:
		return d.String()
	default:
		return fmt.Sprint(d)
	}
}

// Hop is a single hop in a flow trace.
type Hop struct {
	Node  string `json:"node"`
	Steps []Step `json:"steps"`
}

// HopFromDict builds a Hop from a dictionary.
func HopFromDict(d map[string]any) Hop {
	stepVals := sliceField(d, "steps")
	steps := make([]Step, 0, len(stepVals))
	for _, sv := range stepVals {
		if s := StepFromDict(toMap(sv)); s != nil {
			steps = append(steps, *s)
		}
	}
	return Hop{Node: strField(mapField(d, "node"), "name"), Steps: steps}
}

// Dict returns the dictionary representation of the hop.
func (h Hop) Dict() map[string]any { return dictOf(h) }

// Len returns the number of steps.
func (h Hop) Len() int { return len(h.Steps) }

// Step returns the i-th step.
func (h Hop) Step(i int) Step { return h.Steps[i] }

// String renders the hop.
func (h Hop) String() string {
	parts := make([]string, len(h.Steps))
	for i, s := range h.Steps {
		parts[i] = s.String()
	}
	return fmt.Sprintf("node: %s\n  %s", h.Node, strings.Join(parts, "\n  "))
}

// HTML returns the HTML representation of the hop.
func (h Hop) HTML() string {
	parts := make([]string, len(h.Steps))
	for i, s := range h.Steps {
		parts[i] = s.HTML()
	}
	return fmt.Sprintf("node: %s<br>&nbsp;&nbsp;%s", h.Node, strings.Join(parts, "<br>&nbsp;&nbsp;"))
}

// Trace is a trace of a flow through the network.
type Trace struct {
	Disposition string `json:"disposition"`
	Hops        []Hop  `json:"hops"`
}

// TraceFromDict builds a Trace from a dictionary.
func TraceFromDict(d map[string]any) Trace {
	vals := sliceField(d, "hops")
	hops := make([]Hop, 0, len(vals))
	for _, v := range vals {
		hops = append(hops, HopFromDict(toMap(v)))
	}
	return Trace{Disposition: strField(d, "disposition"), Hops: hops}
}

// Dict returns the dictionary representation of the trace.
func (t Trace) Dict() map[string]any { return dictOf(t) }

// Len returns the number of hops.
func (t Trace) Len() int { return len(t.Hops) }

// Hop returns the i-th hop.
func (t Trace) Hop(i int) Hop { return t.Hops[i] }

// String renders the trace.
func (t Trace) String() string {
	parts := make([]string, len(t.Hops))
	for i, h := range t.Hops {
		parts[i] = fmt.Sprintf("%d. %s", i+1, h.String())
	}
	return fmt.Sprintf("%s\n%s", t.Disposition, strings.Join(parts, "\n"))
}

// HTML returns the HTML representation of the trace.
func (t Trace) HTML() string {
	span := fmt.Sprintf(`<span style="color:%s; text-weight:bold;">%s</span>`, colorForDisposition(t.Disposition), t.Disposition)
	parts := make([]string, len(t.Hops))
	for i, h := range t.Hops {
		parts[i] = fmt.Sprintf("<strong>%d</strong>. %s", i+1, h.HTML())
	}
	return fmt.Sprintf("%s<br>%s", span, strings.Join(parts, "<br>"))
}

// TcpFlags represents a set of TCP flags in a packet.
type TcpFlags struct {
	Ack bool `json:"ack"`
	Cwr bool `json:"cwr"`
	Ece bool `json:"ece"`
	Fin bool `json:"fin"`
	Psh bool `json:"psh"`
	Rst bool `json:"rst"`
	Syn bool `json:"syn"`
	Urg bool `json:"urg"`
}

// TcpFlagsFromDict builds TcpFlags from a dictionary.
func TcpFlagsFromDict(d map[string]any) TcpFlags {
	return TcpFlags{
		Ack: boolField(d, "ack"), Cwr: boolField(d, "cwr"), Ece: boolField(d, "ece"),
		Fin: boolField(d, "fin"), Psh: boolField(d, "psh"), Rst: boolField(d, "rst"),
		Syn: boolField(d, "syn"), Urg: boolField(d, "urg"),
	}
}

// Dict returns the dictionary representation of the flags.
func (t TcpFlags) Dict() map[string]any { return dictOf(t) }

// MatchTcpFlags matches a set of TCP flags.
type MatchTcpFlags struct {
	TcpFlags TcpFlags `json:"tcpFlags"`
	UseAck   bool     `json:"useAck"`
	UseCwr   bool     `json:"useCwr"`
	UseEce   bool     `json:"useEce"`
	UseFin   bool     `json:"useFin"`
	UsePsh   bool     `json:"usePsh"`
	UseRst   bool     `json:"useRst"`
	UseSyn   bool     `json:"useSyn"`
	UseUrg   bool     `json:"useUrg"`
}

// NewMatchTcpFlags creates MatchTcpFlags matching flags with every useX bit
// enabled, mirroring the attr defaults.
func NewMatchTcpFlags(flags TcpFlags) MatchTcpFlags {
	return MatchTcpFlags{
		TcpFlags: flags,
		UseAck:   true,
		UseCwr:   true,
		UseEce:   true,
		UseFin:   true,
		UsePsh:   true,
		UseRst:   true,
		UseSyn:   true,
		UseUrg:   true,
	}
}

// MatchTcpFlagsFromDict builds MatchTcpFlags from a dictionary.
func MatchTcpFlagsFromDict(d map[string]any) MatchTcpFlags {
	return MatchTcpFlags{
		TcpFlags: TcpFlagsFromDict(mapField(d, "tcpFlags")),
		UseAck:   boolField(d, "useAck"),
		UseCwr:   boolField(d, "useCwr"),
		UseEce:   boolField(d, "useEce"),
		UseFin:   boolField(d, "useFin"),
		UsePsh:   boolField(d, "usePsh"),
		UseRst:   boolField(d, "useRst"),
		UseSyn:   boolField(d, "useSyn"),
		UseUrg:   boolField(d, "useUrg"),
	}
}

// Dict returns the dictionary representation of the match flags.
func (m MatchTcpFlags) Dict() map[string]any { return dictOf(m) }

// MatchAck returns match conditions checking that the ACK bit is set.
func MatchAck() MatchTcpFlags { return NewMatchTcpFlags(TcpFlags{Ack: true}) }

// MatchRst returns match conditions checking that the RST bit is set.
func MatchRst() MatchTcpFlags { return NewMatchTcpFlags(TcpFlags{Rst: true}) }

// MatchSyn returns match conditions checking that the SYN bit is set.
func MatchSyn() MatchTcpFlags { return NewMatchTcpFlags(TcpFlags{Syn: true}) }

// MatchSynAck returns match conditions checking that SYN and ACK are set.
func MatchSynAck() MatchTcpFlags { return NewMatchTcpFlags(TcpFlags{Syn: true, Ack: true}) }

// MatchEstablished returns match conditions for an established flow.
func MatchEstablished() []MatchTcpFlags { return []MatchTcpFlags{MatchAck(), MatchRst()} }

// MatchNotEstablished returns match conditions for a non-established flow.
func MatchNotEstablished() []MatchTcpFlags {
	return []MatchTcpFlags{NewMatchTcpFlags(TcpFlags{Ack: false, Rst: false})}
}

func normalizePHCIntspace(value any) (any, error) {
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
	if isScalar(value) {
		return fmt.Sprint(value), nil
	}
	return nil, fmt.Errorf("invalid value %v", value)
}

func normalizePHCList(value any) (any, error) {
	if value == nil {
		return nil, nil
	}
	if vals, ok := asList(value); ok {
		return vals, nil
	}
	if s, ok := value.(string); ok {
		var out []any
		for _, part := range strings.Split(s, ",") {
			if trimmed := strings.TrimSpace(part); trimmed != "" {
				out = append(out, trimmed)
			}
		}
		if len(out) == 0 {
			return nil, fmt.Errorf("invalid value %v", value)
		}
		return out, nil
	}
	return nil, fmt.Errorf("invalid value %v", value)
}

func normalizePHCTcpflags(value any) (any, error) {
	if value == nil {
		return nil, nil
	}
	if vals, ok := asList(value); ok {
		return vals, nil
	}
	if m, ok := value.(MatchTcpFlags); ok {
		return []any{m}, nil
	}
	return nil, fmt.Errorf("invalid value %v", value)
}

// HeaderConstraints represents constraints on an IPv4 packet header space.
//
// Fields are `any` because pybatfish accepts polymorphic inputs (for example a
// single port, a list of ports or a comma separated string). Call Normalize to
// apply the same conversions pybatfish applies in its constructors.
type HeaderConstraints struct {
	SrcIps          any `json:"srcIps"`
	DstIps          any `json:"dstIps"`
	SrcPorts        any `json:"srcPorts"`
	DstPorts        any `json:"dstPorts"`
	IpProtocols     any `json:"ipProtocols"`
	Applications    any `json:"applications"`
	IcmpCodes       any `json:"icmpCodes"`
	IcmpTypes       any `json:"icmpTypes"`
	Ecns            any `json:"ecns"`
	Dscps           any `json:"dscps"`
	PacketLengths   any `json:"packetLengths"`
	FragmentOffsets any `json:"fragmentOffsets"`
	TcpFlags        any `json:"tcpFlags"`
}

// Normalize applies pybatfish's field converters and returns the normalized
// constraints. It returns an error for invalid values.
func (h HeaderConstraints) Normalize() (HeaderConstraints, error) {
	var out HeaderConstraints
	var err error
	if out.SrcIps, err = normalizePHCIntspace(h.SrcIps); err != nil {
		return h, err
	}
	if out.DstIps, err = normalizePHCIntspace(h.DstIps); err != nil {
		return h, err
	}
	if out.SrcPorts, err = normalizePHCIntspace(h.SrcPorts); err != nil {
		return h, err
	}
	if out.DstPorts, err = normalizePHCIntspace(h.DstPorts); err != nil {
		return h, err
	}
	if out.IpProtocols, err = normalizePHCList(h.IpProtocols); err != nil {
		return h, err
	}
	if out.Applications, err = normalizePHCList(h.Applications); err != nil {
		return h, err
	}
	if out.IcmpCodes, err = normalizePHCIntspace(h.IcmpCodes); err != nil {
		return h, err
	}
	if out.IcmpTypes, err = normalizePHCIntspace(h.IcmpTypes); err != nil {
		return h, err
	}
	if out.Ecns, err = normalizePHCIntspace(h.Ecns); err != nil {
		return h, err
	}
	if out.Dscps, err = normalizePHCIntspace(h.Dscps); err != nil {
		return h, err
	}
	if out.PacketLengths, err = normalizePHCIntspace(h.PacketLengths); err != nil {
		return h, err
	}
	if out.FragmentOffsets, err = normalizePHCIntspace(h.FragmentOffsets); err != nil {
		return h, err
	}
	if out.TcpFlags, err = normalizePHCTcpflags(h.TcpFlags); err != nil {
		return h, err
	}
	return out, nil
}

// HeaderConstraintsFromDict builds HeaderConstraints from a dictionary.
func HeaderConstraintsFromDict(d map[string]any) HeaderConstraints {
	return HeaderConstraints{
		SrcIps:          d["srcIps"],
		DstIps:          d["dstIps"],
		SrcPorts:        d["srcPorts"],
		DstPorts:        d["dstPorts"],
		IpProtocols:     d["ipProtocols"],
		Applications:    d["applications"],
		IcmpCodes:       d["icmpCodes"],
		IcmpTypes:       d["icmpTypes"],
		Ecns:            d["ecns"],
		Dscps:           d["dscps"],
		PacketLengths:   d["packetLengths"],
		FragmentOffsets: d["fragmentOffsets"],
	}
}

// Dict returns the normalized dictionary representation of the constraints.
func (h HeaderConstraints) Dict() map[string]any {
	normalized, err := h.Normalize()
	if err != nil {
		normalized = h
	}
	return dictOf(normalized)
}

// HeaderConstraintsOf creates constraints from an existing flow.
func HeaderConstraintsOf(flow Flow) HeaderConstraints {
	var srcPorts, dstPorts, icmpCodes, icmpTypes, tcpFlags any
	if flow.HasPorts() {
		srcPorts = fmt.Sprint(*flow.SrcPort)
		dstPorts = fmt.Sprint(*flow.DstPort)
	}
	switch strings.ToLower(flow.IPProtocol) {
	case "icmp":
		icmpCodes = optIntAny(flow.IcmpCode)
		icmpTypes = optIntAny(flow.IcmpVar)
	case "tcp":
		flags := TcpFlags{
			Ack: ptrNonZero(flow.TcpFlagsAck),
			Cwr: ptrNonZero(flow.TcpFlagsCwr),
			Ece: ptrNonZero(flow.TcpFlagsEce),
			Fin: ptrNonZero(flow.TcpFlagsFin),
			Psh: ptrNonZero(flow.TcpFlagsPsh),
			Rst: ptrNonZero(flow.TcpFlagsRst),
			Syn: ptrNonZero(flow.TcpFlagsSyn),
			Urg: ptrNonZero(flow.TcpFlagsUrg),
		}
		tcpFlags = NewMatchTcpFlags(flags)
	}
	h := HeaderConstraints{
		SrcIps:          flow.SrcIP,
		DstIps:          flow.DstIP,
		IpProtocols:     []any{flow.IPProtocol},
		SrcPorts:        srcPorts,
		DstPorts:        dstPorts,
		IcmpCodes:       icmpCodes,
		IcmpTypes:       icmpTypes,
		TcpFlags:        tcpFlags,
		FragmentOffsets: flow.FragmentOffset,
		PacketLengths:   flow.PacketLength,
	}
	normalized, err := h.Normalize()
	if err != nil {
		return h
	}
	return normalized
}

// PathConstraints represents constraints on the path of a flow.
type PathConstraints struct {
	StartLocation      *string `json:"startLocation"`
	EndLocation        *string `json:"endLocation"`
	TransitLocations   *string `json:"transitLocations"`
	ForbiddenLocations *string `json:"forbiddenLocations"`
}

// PathConstraintsFromDict builds PathConstraints from a dictionary.
func PathConstraintsFromDict(d map[string]any) PathConstraints {
	return PathConstraints{
		StartLocation:      optStrField(d, "startLocation"),
		EndLocation:        optStrField(d, "endLocation"),
		TransitLocations:   optStrField(d, "transitLocations"),
		ForbiddenLocations: optStrField(d, "forbiddenLocations"),
	}
}

// Dict returns the dictionary representation of the constraints.
func (p PathConstraints) Dict() map[string]any { return dictOf(p) }

// --- shared helpers ---------------------------------------------------------

func sessionScopeFromDictCompat(d map[string]any) SessionScope {
	if scope := mapField(d, "sessionScope"); scope != nil {
		s, _ := SessionScopeFromDict(scope)
		return s
	}
	return IncomingSessionScopeFromDict(d)
}

func mustSessionAction(d map[string]any) SessionAction {
	action, _ := SessionActionFromDict(mapField(d, "sessionAction"))
	return action
}

func flowDiffsFrom(d map[string]any) []FlowDiff {
	vals := sliceField(d, "transformation")
	out := make([]FlowDiff, 0, len(vals))
	for _, v := range vals {
		out = append(out, FlowDiffFromDict(toMap(v)))
	}
	return out
}

func sessionDetailString(scope SessionScope, action SessionAction, criteria SessionMatchExpr, transformation []FlowDiff) string {
	parts := []string{scope.String(), "Action: " + action.String(), "Match Criteria: " + criteria.String()}
	if len(transformation) > 0 {
		diffs := make([]string, len(transformation))
		for i, d := range transformation {
			diffs[i] = d.String()
		}
		parts = append(parts, "Transformation: ["+strings.Join(diffs, ", ")+"]")
	}
	return strings.Join(parts, ", ")
}

func colorForDisposition(disposition string) string {
	switch disposition {
	case "ACCEPTED", "DELIVERED_TO_SUBNET", "EXITS_NETWORK":
		return "#019612"
	default:
		return "#7c020e"
	}
}

func optionalDetail(prefix string, value *string) string {
	if value == nil {
		return ""
	}
	return prefix + *value
}

func joinDetail(parts []string) string {
	var out []string
	for _, p := range parts {
		if p != "" {
			out = append(out, p)
		}
	}
	return strings.Join(out, ", ")
}

func joinAny(vals []any, sep string) string {
	parts := make([]string, len(vals))
	for i, v := range vals {
		parts[i] = fmt.Sprint(v)
	}
	return strings.Join(parts, sep)
}

func optIntVal(v *int) string {
	if v == nil {
		return "None"
	}
	return fmt.Sprint(*v)
}

func optIntAny(v *int) any {
	if v == nil {
		return nil
	}
	return *v
}

func ptrStr(v *string) string {
	if v == nil {
		return "None"
	}
	return *v
}

func ptrNonZero(v *int) bool {
	return v != nil && *v != 0
}

func isScalar(v any) bool {
	switch v.(type) {
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64, bool:
		return true
	default:
		return false
	}
}
