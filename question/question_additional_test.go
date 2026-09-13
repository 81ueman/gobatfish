package question

// Port of pybatfish/tests/question/test_question_additional.py.
//
// The Python helpers return (bool, Optional[str]) tuples. The Go equivalents
// return (bool, string); an empty string plays the role of Python's None.

import (
	"math"
	"testing"

	"github.com/81ueman/gobatfish/datamodel"
)

// requireValid asserts that a (valid, errorMessage) pair reports success.
func requireValid(t *testing.T, valid bool, message string) {
	t.Helper()
	if !valid {
		t.Fatalf("expected valid, got error %q", message)
	}
	if message != "" {
		t.Fatalf("expected no error message, got %q", message)
	}
}

// requireInvalid asserts that a (valid, errorMessage) pair reports failure with
// the expected message.
func requireInvalid(t *testing.T, valid bool, message, want string) {
	t.Helper()
	if valid {
		t.Fatal("expected invalid, got valid")
	}
	if message != want {
		t.Fatalf("error = %q, want %q", message, want)
	}
}

// Tests for isSubRange

func TestInvalidSubRange(t *testing.T) {
	subRange := "100, 200"
	valid, message := isSubRange(subRange)
	requireInvalid(t, valid, message, "Invalid subRange: "+subRange)
}

func TestInvalidStartSubRange(t *testing.T) {
	subRange := "s100-200"
	valid, message := isSubRange(subRange)
	requireInvalid(t, valid, message, "Invalid subRange start: s100")
}

func TestInvalidEndSubRange(t *testing.T) {
	subRange := "100-s200"
	valid, message := isSubRange(subRange)
	requireInvalid(t, valid, message, "Invalid subRange end: s200")
}

func TestValidSubRange(t *testing.T) {
	valid, message := isSubRange("100-200")
	requireValid(t, valid, message)
}

// Tests for isIp

func TestInvalidIp(t *testing.T) {
	ip := "192.168.11"
	valid, message := isIP(ip)
	requireInvalid(t, valid, message, "Invalid ip string: '"+ip+"'")
}

func TestInvalidIpAddressWithIndicator(t *testing.T) {
	ip := "INVALID_IP(100)"
	valid, message := isIP(ip)
	requireInvalid(t, valid, message, "Invalid ip string: '"+ip+"'")
}

func TestValidIpAddressWithIndicator(t *testing.T) {
	valid, message := isIP("INVALID_IP(100l)")
	requireValid(t, valid, message)
}

func TestInvalidSegmentsIpAddress(t *testing.T) {
	ipAddress := "192.168.11.s"
	valid, message := isIP(ipAddress)
	requireInvalid(t, valid, message, "Ip segment is not a number: 's' in ip string: '"+ipAddress+"'")
}

func TestInvalidSegmentRangeIpAddress(t *testing.T) {
	ipAddress := "192.168.11.256"
	valid, message := isIP(ipAddress)
	requireInvalid(t, valid, message, "Ip segment is out of range 0-255: '256' in ip string: '"+ipAddress+"'")
}

func TestInvalidSegmentRangeIpAddress2(t *testing.T) {
	ipAddress := "192.168.11.-1"
	valid, message := isIP(ipAddress)
	requireInvalid(t, valid, message, "Ip segment is out of range 0-255: '-1' in ip string: '"+ipAddress+"'")
}

func TestValidIpAddress(t *testing.T) {
	valid, message := isIP("192.168.1.1")
	requireValid(t, valid, message)
}

// Tests for isPrefix

func TestInvalidIpInPrefix(t *testing.T) {
	prefix := "192.168.1.s/100"
	valid, message := isPrefix(prefix)
	requireInvalid(t, valid, message, "Ip segment is not a number: 's' in ip string: '192.168.1.s'")
}

func TestInvalidLengthInPrefix(t *testing.T) {
	prefix := "192.168.1.1/s"
	valid, message := isPrefix(prefix)
	requireInvalid(t, valid, message, "Prefix length must be an integer")
}

func TestValidPrefix(t *testing.T) {
	valid, message := isPrefix("192.168.1.1/100")
	requireValid(t, valid, message)
}

// Tests for isPrefixRange

func TestInvalidPrefixRangeInput(t *testing.T) {
	prefixRange := "192.168.1.s/100:100:100"
	valid, message := isPrefixRange(prefixRange)
	requireInvalid(t, valid, message, "Invalid PrefixRange string: '"+prefixRange+"'")
}

func TestInvalidPrefixInput(t *testing.T) {
	prefixRange := "192.168.1.s/100:100"
	valid, message := isPrefixRange(prefixRange)
	requireInvalid(t, valid, message, "Invalid prefix string: '192.168.1.s/100' in prefix range string: '"+prefixRange+"'")
}

func TestInvalidRangeInput(t *testing.T) {
	prefixRange := "192.168.1.1/100:100-s110"
	valid, message := isPrefixRange(prefixRange)
	requireInvalid(t, valid, message, "Invalid subRange end: s110")
}

func TestValidPrefixRange(t *testing.T) {
	valid, message := isPrefixRange("192.168.1.1/100:100-110")
	requireValid(t, valid, message)
}

// Tests for isIpWildcard

func TestInvalidIpWildcardWithColon(t *testing.T) {
	ipWildcard := "192.168.1.s:192.168.10.10:192"
	valid, message := isIPWildcard(ipWildcard)
	requireInvalid(t, valid, message, "Invalid IpWildcard string: '"+ipWildcard+"'")
}

func TestInvalidStartIpWildcardWithColon(t *testing.T) {
	ipWildcard := "192.168.1.s:192.168.1.1"
	valid, message := isIPWildcard(ipWildcard)
	requireInvalid(t, valid, message, "Invalid ip string: '192.168.1.s'")
}

func TestInvalidEndIpWildcardWithColon(t *testing.T) {
	ipWildcard := "192.168.1.1:192.168.10.s"
	valid, message := isIPWildcard(ipWildcard)
	requireInvalid(t, valid, message, "Ip segment is not a number: 's' in ip string: '192.168.10.s'")
}

func TestValidIpWildcardWithColon(t *testing.T) {
	valid, message := isIPWildcard("192.168.1.1:192.168.10.10")
	requireValid(t, valid, message)
}

func TestInvalidIpWildcardWithSlash(t *testing.T) {
	ipWildcard := "192.168.1.s/192.168.10.10/192"
	valid, message := isIPWildcard(ipWildcard)
	requireInvalid(t, valid, message, "Invalid IpWildcard string: '"+ipWildcard+"'")
}

func TestInvalidStartIpWildcardWithSlash(t *testing.T) {
	ipWildcard := "192.168.1.s/s"
	valid, message := isIPWildcard(ipWildcard)
	requireInvalid(t, valid, message, "Invalid ip string: '192.168.1.s'")
}

func TestInvalidEndIpWildcardWithSlash(t *testing.T) {
	ipWildcard := "192.168.1.1/s"
	valid, message := isIPWildcard(ipWildcard)
	requireInvalid(t, valid, message, "Invalid prefix length: 's' in IpWildcard string: '"+ipWildcard+"'")
}

func TestValidIpWildcardWithSlash(t *testing.T) {
	valid, message := isIPWildcard("192.168.1.1/100")
	requireValid(t, valid, message)
}

func TestInvalidIpAddressIpWildcard(t *testing.T) {
	ipWildcard := "192.168.11.s"
	valid, message := isIPWildcard(ipWildcard)
	requireInvalid(t, valid, message, "Ip segment is not a number: 's' in ip string: '"+ipWildcard+"'")
}

func TestValidIpAddressIpWildcard(t *testing.T) {
	valid, message := isIPWildcard("192.168.11.1")
	requireValid(t, valid, message)
}

// Tests for validateType

func TestInvalidBooleanValidateType(t *testing.T) {
	valid, _ := validateType(1.5, "boolean")
	if valid {
		t.Fatal("expected invalid")
	}
}

func TestValidBooleanValidateType(t *testing.T) {
	valid, _ := validateType(true, "boolean")
	if !valid {
		t.Fatal("expected valid")
	}
}

func TestInvalidIntegerValidateType(t *testing.T) {
	valid, _ := validateType(1.5, "integer")
	if valid {
		t.Fatal("expected invalid")
	}
}

func TestValidIntegerValidateType(t *testing.T) {
	valid, _ := validateType(10, "integer")
	if !valid {
		t.Fatal("expected valid")
	}
}

func TestInvalidComparatorValidateType(t *testing.T) {
	valid, message := validateType("<==", "comparator")
	want := "'<==' is not a known comparator. Valid options are: '<, <=, ==, >=, >, !='"
	requireInvalid(t, valid, message, want)
}

func TestValidComparatorValidateType(t *testing.T) {
	valid, _ := validateType("<=", "comparator")
	if !valid {
		t.Fatal("expected valid")
	}
}

func TestInvalidFloatValidateType(t *testing.T) {
	valid, _ := validateType(10, "float")
	if valid {
		t.Fatal("expected invalid")
	}
}

func TestValidFloatValidateType(t *testing.T) {
	valid, _ := validateType(10.0, "float")
	if !valid {
		t.Fatal("expected valid")
	}
}

func TestInvalidDoubleValidateType(t *testing.T) {
	valid, _ := validateType(10, "double")
	if valid {
		t.Fatal("expected invalid")
	}
}

func TestValidDoubleValidateType(t *testing.T) {
	valid, _ := validateType(10.0, "double")
	if !valid {
		t.Fatal("expected valid")
	}
}

func TestInvalidLongValidateType(t *testing.T) {
	valid, _ := validateType(5.3, "long")
	if valid {
		t.Fatal("expected invalid for fractional float")
	}
	// Python's 2**64 does not fit in an int64, so represent it as a float64.
	valid, _ = validateType(float64(math.Pow(2, 64)), "long")
	if valid {
		t.Fatal("expected invalid for 2**64")
	}
}

func TestValidLongValidateType(t *testing.T) {
	valid, _ := validateType(10, "long")
	if !valid {
		t.Fatal("expected valid for 10")
	}
	valid, _ = validateType(1<<40, "long")
	if !valid {
		t.Fatal("expected valid for 2**40")
	}
}

func TestInvalidJavaRegexValidateType(t *testing.T) {
	valid, message := validateType(10, "javaRegex")
	requireInvalid(t, valid, message, "A Batfish javaRegex must be a string")
}

func TestInvalidNonDictionaryJsonPathValidateType(t *testing.T) {
	valid, message := validateType(10, "jsonPath")
	requireInvalid(t, valid, message,
		"Expected a jsonPath dictionary with elements 'path' (string) and optional 'suffix' (boolean)")
}

func TestInvalidDictionaryJsonPathValidateType(t *testing.T) {
	valid, message := validateType(map[string]any{"value": 10}, "jsonPath")
	requireInvalid(t, valid, message, "Missing 'path' element of jsonPath")
}

func TestPathNonStringJsonPathValidateType(t *testing.T) {
	valid, message := validateType(map[string]any{"path": 10}, "jsonPath")
	requireInvalid(t, valid, message, "'path' element of jsonPath dictionary should be a string")
}

func TestSuffixNonBooleanJsonPathValidateType(t *testing.T) {
	valid, message := validateType(map[string]any{"path": "I am path", "suffix": "hi"}, "jsonPath")
	requireInvalid(t, valid, message, "'suffix' element of jsonPath dictionary should be a boolean")
}

func TestValidJsonPathValidateType(t *testing.T) {
	valid, message := validateType(map[string]any{"path": "I am path", "suffix": true}, "jsonPath")
	requireValid(t, valid, message)
}

func TestInvalidTypeSubRangeValidateType(t *testing.T) {
	// Adapted from the Python input 10.0: Go decodes all JSON numbers to
	// float64, so a whole float must be treated as an integer. A fractional
	// float exercises the same "not a string or integer" branch.
	valid, message := validateType(10.5, "subrange")
	requireInvalid(t, valid, message, "A Batfish subrange must either be a string or an integer")
}

func TestValidIntegerSubRangeValidateType(t *testing.T) {
	valid, message := validateType(10, "subrange")
	requireValid(t, valid, message)
}

func TestNonStringProtocolValidateType(t *testing.T) {
	valid, message := validateType(10.0, "protocol")
	requireInvalid(t, valid, message, "A Batfish protocol must be a string")
}

func TestInvalidProtocolValidateType(t *testing.T) {
	valid, message := validateType("TCPP", "protocol")
	requireInvalid(t, valid, message, "'TCPP' is not a valid protocols. Valid options are: 'dns, ssh, tcp, udp'")
}

func TestValidProtocolValidateType(t *testing.T) {
	valid, message := validateType("TCP", "protocol")
	requireValid(t, valid, message)
}

func TestNonStringIpProtocolValidateType(t *testing.T) {
	valid, message := validateType(10.0, "ipProtocol")
	requireInvalid(t, valid, message, "A Batfish ipProtocol must be a string")
}

func TestInvalidIntegerIpProtocolValidateType(t *testing.T) {
	valid, message := validateType("1000", "ipProtocol")
	requireInvalid(t, valid, message, "'1000' is not in valid ipProtocol range: 0-255")
}

func TestValidIntegerIpProtocolValidateType(t *testing.T) {
	valid, message := validateType("10", "ipProtocol")
	requireValid(t, valid, message)
}

// completionTypes mirrors COMPLETION_TYPES from tests/conftest.py plus
// VariableType.BGP_ROUTE_STATUS_SPEC, matching the union used by the Python
// tests.
var completionTypes = []datamodel.VariableType{
	datamodel.VariableAddressGroupName,
	datamodel.VariableApplicationSpec,
	datamodel.VariableBGPPeerPropertySpec,
	datamodel.VariableBGPProcessPropertySpec,
	datamodel.VariableBGPRouteStatusSpec,
	datamodel.VariableBGPSessionCompatStatusSpec,
	datamodel.VariableBGPSessionStatusSpec,
	datamodel.VariableBGPSessionTypeSpec,
	datamodel.VariableDispositionSpec,
	datamodel.VariableFilter,
	datamodel.VariableFilterSpec,
	datamodel.VariableInterface,
	datamodel.VariableInterfaceGroupName,
	datamodel.VariableInterfacePropertySpec,
	datamodel.VariableIP,
	datamodel.VariableIPProtocolSpec,
	datamodel.VariableIPSpaceSpec,
	datamodel.VariableIPsecSessionStatusSpec,
	datamodel.VariableLocationSpec,
	datamodel.VariableMlagIDSpec,
	datamodel.VariableNamedStructureSpec,
	datamodel.VariableNodePropertySpec,
	datamodel.VariableNodeRoleDimensionName,
	datamodel.VariableNodeRoleName,
	datamodel.VariableNodeSpec,
	datamodel.VariableOSPFInterfacePropertySpec,
	datamodel.VariableOSPFProcessPropertySpec,
	datamodel.VariableOSPFSessionStatusSpec,
	datamodel.VariablePrefix,
	datamodel.VariableProtocol,
	datamodel.VariableReferenceBookName,
	datamodel.VariableRoutingPolicySpec,
	datamodel.VariableRoutingProtocolSpec,
	datamodel.VariableStructureName,
	datamodel.VariableVrf,
	datamodel.VariableZone,
}

func TestInvalidCompletionTypes(t *testing.T) {
	for _, completionType := range completionTypes {
		valid, message := validateType(5, string(completionType))
		want := "A Batfish " + string(completionType) + " must be a string"
		if valid {
			t.Fatalf("%s: expected invalid", completionType)
		}
		if message != want {
			t.Fatalf("%s: error = %q, want %q", completionType, message, want)
		}
	}
}

func TestValidCompletionTypes(t *testing.T) {
	values := map[datamodel.VariableType]any{
		datamodel.VariableIP:       "1.2.3.4",
		datamodel.VariablePrefix:   "1.2.3.4/24",
		datamodel.VariableProtocol: "ssh",
	}
	for _, completionType := range completionTypes {
		value, ok := values[completionType]
		if !ok {
			value = ".*"
		}
		valid, message := validateType(value, string(completionType))
		if !valid {
			t.Fatalf("%s: expected valid, got error %q", completionType, message)
		}
		if message != "" {
			t.Fatalf("%s: expected no error message, got %q", completionType, message)
		}
	}
}
