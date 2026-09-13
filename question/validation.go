package question

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/81ueman/gobatfish/datamodel"
	"github.com/81ueman/gobatfish/exception"
)

var validVariableNameRegex = regexp.MustCompile(`^\w+$`)

func processVariables(questionName string, variables map[string]any, ordered []string) ([]string, error) {
	if len(variables) == 0 {
		return []string{}, nil
	}
	for varName, varData := range variables {
		if err := validateVariableName(questionName, varName); err != nil {
			return nil, err
		}
		if err := validateVariableData(questionName, varName, mapField(variables, varName)); err != nil {
			return nil, err
		}
		_ = varData
	}
	if hasValidOrderedVariableNames(ordered, variables) {
		return ordered, nil
	}
	names := make([]string, 0, len(variables))
	for name := range variables {
		names = append(names, name)
	}
	sort.SliceStable(names, func(i, j int) bool {
		oi := boolValue(mapField(variables, names[i])["optional"])
		oj := boolValue(mapField(variables, names[j])["optional"])
		if oi != oj {
			return !oi
		}
		return names[i] < names[j]
	})
	return names, nil
}

func validateVariableName(questionName, varName string) error {
	if !validVariableNameRegex.MatchString(varName) {
		return exception.NewQuestionValidationErrorf(
			"Question %s has invalid variable name: %s. Only alphanumeric characters are allowed", questionName, varName)
	}
	return nil
}

func validateVariableData(questionName, varName string, varData map[string]any) error {
	varType := strings.TrimSpace(fmt.Sprint(varData["type"]))
	if varType == "" || varData["type"] == nil {
		return exception.NewQuestionValidationErrorf("Question %s is missing type for variable %s", questionName, varName)
	}
	varData["type"] = varType

	varDesc := strings.TrimSpace(fmt.Sprint(varData["description"]))
	if varDesc == "" || varData["description"] == nil {
		return exception.NewQuestionValidationErrorf("Question %s is missing description for variable %s", questionName, varName)
	}
	if !strings.HasSuffix(varDesc, ".") {
		varDesc += "."
	}
	varData["description"] = varDesc
	return nil
}

func hasValidOrderedVariableNames(ordered []string, variables map[string]any) bool {
	if len(ordered) == 0 {
		return false
	}
	if len(ordered) != len(variables) {
		return false
	}
	seen := make(map[string]bool, len(ordered))
	for _, name := range ordered {
		if _, ok := variables[name]; !ok {
			return false
		}
		if seen[name] {
			return false
		}
		seen[name] = true
	}
	return true
}

func computeDocstring(baseDocstring string, varNames []string, variables map[string]any) string {
	if len(variables) == 0 {
		return baseDocstring
	}
	parts := []string{baseDocstring, ""}
	for _, name := range varNames {
		parts = append(parts, computeVarHelp(name, mapField(variables, name)))
	}
	return strings.Join(parts, "\n")
}

func computeVarHelp(varName string, varData map[string]any) string {
	required := ""
	if !boolValue(varData["optional"]) {
		required = "*Required.* "
	}
	desc := fmt.Sprint(varData["description"])
	paramLine := fmt.Sprintf(":param %s: %s%s\n", varName, required, desc)

	if allowed := buildAllowedValues(varData); len(allowed) > 0 {
		parts := make([]string, len(allowed))
		for i, v := range allowed {
			parts[i] = v.String()
		}
		paramLine += "    Allowed values:\n\n    * " + strings.Join(parts, "\n    * ") + "\n"
	}
	if defaultValue, ok := varData["value"]; ok && defaultValue != nil {
		paramLine += fmt.Sprintf("\n    Default value: ``%s``\n", pyRepr(defaultValue))
	}
	typeLine := fmt.Sprintf(":type %s: %s", varName, varData["type"])
	return paramLine + typeLine
}

func buildAllowedValues(varData map[string]any) []AllowedValue {
	if values, ok := varData["values"].([]any); ok && len(values) > 0 {
		out := make([]AllowedValue, 0, len(values))
		for _, v := range values {
			out = append(out, AllowedValueFromDict(asMap(v)))
		}
		return out
	}
	if oldValues, ok := varData["allowedValues"].([]any); ok && len(oldValues) > 0 {
		out := make([]AllowedValue, 0, len(oldValues))
		for _, v := range oldValues {
			out = append(out, AllowedValue{Name: fmt.Sprint(v)})
		}
		return out
	}
	return nil
}

// validate validates a question dictionary, mirroring pybatfish's _validate.
func validate(questionJSON map[string]any) error {
	valid := true
	errorMessage := "\n"
	instanceData := mapField(questionJSON, "instance")
	if instanceData == nil {
		return exception.NewQuestionValidationError(errorMessage)
	}
	variables := mapField(instanceData, "variables")
	for variableName, rawVariable := range variables {
		variable := asMap(rawVariable)
		optional := boolValue(variable["optional"])
		if !optional {
			if _, ok := variable["value"]; !ok {
				valid = false
				errorMessage += "   Missing value for mandatory parameter: '" + variableName + "'\n"
			}
		}
		allowedValues := buildAllowedValues(variable)
		if _, ok := variable["value"]; !ok {
			continue
		}
		value := variable["value"]
		variableType := fmt.Sprint(variable["type"])
		var minLength *int
		if ml, ok := variable["minLength"]; ok && ml != nil {
			v := toInt(ml)
			minLength = &v
		}
		_, isArray := variable["minElements"]

		if isArray {
			valueList, ok := value.([]any)
			if !ok {
				valid = false
				errorMessage += "   Expected a list for parameter: '" + variableName + "'\n"
				continue
			}
			minElements := toInt(variable["minElements"])
			if len(valueList) < minElements {
				valid = false
				errorMessage += "   Number of elements provided for parameter: '" + variableName +
					"' less than the minimum: " + strconv.Itoa(minElements) + "\n"
				continue
			}
			for i, element := range valueList {
				typeValid, _ := validateType(element, variableType)
				if !typeValid {
					valid = false
					errorMessage += "   Expected type: '" + variableType + "' for element: " + strconv.Itoa(i) + " of parameter: " + variableName + "\n"
				} else if minLength != nil && lengthOf(element) < *minLength {
					valid = false
					errorMessage += "   Length of value: '" + fmt.Sprint(element) + "' for element : " + strconv.Itoa(i) +
						" of parameter: '" + variableName + "' below minimum length: " + strconv.Itoa(*minLength) + "\n"
				} else if allowedValues != nil && !allowedValueContains(allowedValues, element) {
					valid = false
					errorMessage += fmt.Sprintf("   Value: '%v' is not among allowed values %s of parameter: '%s'\n",
						element, pyStringListRepr(allowedValueNames(allowedValues)), variableName)
				}
			}
		} else {
			typeValid, typeValidErrorMessage := validateType(value, variableType)
			if !typeValid {
				valid = false
				if typeValidErrorMessage != "" {
					errorMessage += "   Expected type: '" + variableType + "' for parameter: '" + variableName +
						"'. Got error: '" + typeValidErrorMessage + "'\n"
				} else {
					errorMessage += "   Expected type: '" + variableType + "' for parameter: '" + variableName + "'\n"
				}
			} else if minLength != nil && lengthOf(value) < *minLength {
				valid = false
				errorMessage += "   Length of value: '" + fmt.Sprint(value) + "' for parameter: '" + variableName +
					"' below minimum length: " + strconv.Itoa(*minLength) + "\n"
			} else if allowedValues != nil && !allowedValueContains(allowedValues, value) {
				valid = false
				errorMessage += fmt.Sprintf("   Value: '%v' is not among allowed values %s of parameter: '%s'\n",
					value, pyStringListRepr(allowedValueNames(allowedValues)), variableName)
			}
		}
	}
	if !valid {
		return exception.NewQuestionValidationError(errorMessage)
	}
	return nil
}

func validateType(value any, expectedType string) (bool, string) {
	switch datamodel.VariableType(expectedType) {
	case datamodel.VariableBoolean:
		_, ok := value.(bool)
		return ok, ""
	case datamodel.VariableComparator:
		valid := []string{"<", "<=", "==", ">=", ">", "!="}
		for _, c := range valid {
			if fmt.Sprint(value) == c {
				return true, ""
			}
		}
		return false, fmt.Sprintf("'%v' is not a known comparator. Valid options are: '%s'", value, strings.Join(valid, ", "))
	case datamodel.VariableInteger:
		n, ok := asInt(value)
		if !ok {
			return false, ""
		}
		const int32Min = -(1 << 32)
		const int32Max = (1 << 32) - 1
		return n >= int32Min && n <= int32Max, ""
	case datamodel.VariableFloat, datamodel.VariableDouble:
		_, ok := value.(float64)
		return ok, ""
	case datamodel.VariableAddressGroupName, datamodel.VariableApplicationSpec,
		datamodel.VariableBGPPeerPropertySpec, datamodel.VariableBGPProcessPropertySpec,
		datamodel.VariableBGPRouteStatusSpec, datamodel.VariableBGPSessionCompatStatusSpec,
		datamodel.VariableBGPSessionStatusSpec, datamodel.VariableBGPSessionTypeSpec,
		datamodel.VariableDispositionSpec, datamodel.VariableFilter, datamodel.VariableFilterSpec,
		datamodel.VariableIntegerSpace, datamodel.VariableInterface, datamodel.VariableInterfaceGroupName,
		datamodel.VariableInterfacePropertySpec, datamodel.VariableInterfacesSpec,
		datamodel.VariableIPProtocolSpec, datamodel.VariableIPSpaceSpec,
		datamodel.VariableIPsecSessionStatusSpec, datamodel.VariableJavaRegex,
		datamodel.VariableJSONPathRegex, datamodel.VariableLocationSpec, datamodel.VariableMlagID,
		datamodel.VariableMlagIDSpec, datamodel.VariableNamedStructureSpec,
		datamodel.VariableNodePropertySpec, datamodel.VariableNodeRoleDimensionName,
		datamodel.VariableNodeRoleName, datamodel.VariableNodeSpec,
		datamodel.VariableOSPFInterfacePropertySpec, datamodel.VariableOSPFProcessPropertySpec,
		datamodel.VariableOSPFSessionStatusSpec, datamodel.VariableReferenceBookName,
		datamodel.VariableRoutingPolicySpec, datamodel.VariableRoutingProtocolSpec,
		datamodel.VariableStructureName, datamodel.VariableVrf,
		datamodel.VariableVxlanVniPropertySpec, datamodel.VariableZone:
		if _, ok := value.(string); !ok {
			return false, fmt.Sprintf("A Batfish %s must be a string", expectedType)
		}
		return true, ""
	case datamodel.VariableIP:
		if _, ok := value.(string); !ok {
			return false, fmt.Sprintf("A Batfish %s must be a string", expectedType)
		}
		return isIP(value.(string))
	case datamodel.VariableIPWildcard:
		if _, ok := value.(string); !ok {
			return false, fmt.Sprintf("A Batfish %s must be a string", expectedType)
		}
		return isIPWildcard(value.(string))
	case datamodel.VariableJSONPath:
		return isJSONPath(value)
	case datamodel.VariableLong:
		_, ok := asInt(value)
		return ok, ""
	case datamodel.VariablePrefix:
		if _, ok := value.(string); !ok {
			return false, fmt.Sprintf("A Batfish %s must be a string", expectedType)
		}
		return isPrefix(value.(string))
	case datamodel.VariablePrefixRange:
		if _, ok := value.(string); !ok {
			return false, fmt.Sprintf("A Batfish %s must be a string", expectedType)
		}
		return isPrefixRange(value.(string))
	case datamodel.VariableQuestion:
		_, ok := value.(*Question)
		return ok, ""
	case datamodel.VariableBGPRoutes:
		list, ok := value.([]any)
		if !ok {
			return false, fmt.Sprintf("A Batfish %s must be a list of BgpRoute", expectedType)
		}
		for _, item := range list {
			if _, ok := item.(datamodel.BgpRoute); !ok {
				return false, fmt.Sprintf("A Batfish %s must be a list of BgpRoute", expectedType)
			}
		}
		return true, ""
	case datamodel.VariableString:
		_, ok := value.(string)
		return ok, ""
	case datamodel.VariableSubrange:
		if _, ok := asInt(value); ok {
			return true, ""
		}
		if s, ok := value.(string); ok {
			return isSubRange(s)
		}
		return false, fmt.Sprintf("A Batfish %s must either be a string or an integer", expectedType)
	case datamodel.VariableProtocol:
		s, ok := value.(string)
		if !ok {
			return false, fmt.Sprintf("A Batfish %s must be a string", expectedType)
		}
		validProtocols := []string{"dns", "ssh", "tcp", "udp"}
		for _, p := range validProtocols {
			if strings.ToLower(s) == p {
				return true, ""
			}
		}
		return false, fmt.Sprintf("'%s' is not a valid protocols. Valid options are: '%s'", s, strings.Join(validProtocols, ", "))
	case datamodel.VariableIPProtocol:
		s, ok := value.(string)
		if !ok {
			return false, fmt.Sprintf("A Batfish %s must be a string", expectedType)
		}
		intValue, err := strconv.Atoi(s)
		if err != nil {
			return true, ""
		}
		if intValue < 0 || intValue >= 256 {
			return false, fmt.Sprintf("'%d' is not in valid ipProtocol range: 0-255", intValue)
		}
		return true, ""
	case datamodel.VariableAnswerElement, datamodel.VariableBGPRouteConstraints,
		datamodel.VariableBGPSessionProperties, datamodel.VariableHeaderConstraint,
		datamodel.VariablePathConstraint:
		return true, ""
	default:
		return true, ""
	}
}

func isJSONPath(value any) (bool, string) {
	m, ok := value.(map[string]any)
	if !ok {
		return false, "Expected a jsonPath dictionary with elements 'path' (string) and optional 'suffix' (boolean)"
	}
	path, ok := m["path"]
	if !ok {
		return false, "Missing 'path' element of jsonPath"
	}
	if _, ok := path.(string); !ok {
		return false, "'path' element of jsonPath dictionary should be a string"
	}
	if suffix, ok := m["suffix"]; ok {
		if _, ok := suffix.(bool); !ok {
			return false, "'suffix' element of jsonPath dictionary should be a boolean"
		}
	}
	return true, ""
}

func isIP(value string) (bool, string) {
	segments := strings.Split(value, ".")
	if len(segments) != 4 {
		if strings.HasPrefix(value, "INVALID_IP") || strings.HasPrefix(value, "AUTO/NONE") {
			tail := strings.SplitN(value, "(", 2)
			if len(tail) == 2 {
				longParts := strings.SplitN(tail[1], "l", 2)
				if len(longParts) == 2 {
					if _, err := strconv.Atoi(longParts[0]); err == nil {
						return true, ""
					}
				}
			}
		}
		return false, fmt.Sprintf("Invalid ip string: '%s'", value)
	}
	for _, segment := range segments {
		segmentVal, err := strconv.Atoi(segment)
		if err != nil {
			return false, fmt.Sprintf("Ip segment is not a number: '%s' in ip string: '%s'", segment, value)
		}
		if segmentVal < 0 || segmentVal > 255 {
			return false, fmt.Sprintf("Ip segment is out of range 0-255: '%s' in ip string: '%s'", segment, value)
		}
	}
	return true, ""
}

func isSubRange(value string) (bool, string) {
	contents := strings.Split(value, "-")
	if len(contents) != 2 {
		return false, fmt.Sprintf("Invalid subRange: %s", value)
	}
	if _, err := strconv.Atoi(contents[0]); err != nil {
		return false, fmt.Sprintf("Invalid subRange start: %s", contents[0])
	}
	if _, err := strconv.Atoi(contents[1]); err != nil {
		return false, fmt.Sprintf("Invalid subRange end: %s", contents[1])
	}
	return true, ""
}

func isPrefix(value string) (bool, string) {
	contents := strings.Split(value, "/")
	if len(contents) != 2 {
		return false, fmt.Sprintf("Invalid prefix string: '%s'", value)
	}
	if _, err := strconv.Atoi(contents[1]); err != nil {
		return false, "Prefix length must be an integer"
	}
	return isIP(contents[0])
}

func isPrefixRange(value string) (bool, string) {
	contents := strings.Split(value, ":")
	if len(contents) < 1 || len(contents) > 2 {
		return false, fmt.Sprintf("Invalid PrefixRange string: '%s'", value)
	}
	if ok, _ := isPrefix(contents[0]); !ok {
		return false, fmt.Sprintf("Invalid prefix string: '%s' in prefix range string: '%s'", contents[0], value)
	}
	if len(contents) == 2 {
		return isSubRange(contents[1])
	}
	return true, ""
}

func isIPWildcard(value string) (bool, string) {
	if strings.Contains(value, ":") {
		contents := strings.Split(value, ":")
		if len(contents) != 2 {
			return false, fmt.Sprintf("Invalid IpWildcard string: '%s'", value)
		}
		if ok, _ := isIP(contents[0]); !ok {
			return false, fmt.Sprintf("Invalid ip string: '%s'", contents[0])
		}
		return isIP(contents[1])
	} else if strings.Contains(value, "/") {
		contents := strings.Split(value, "/")
		if len(contents) != 2 {
			return false, fmt.Sprintf("Invalid IpWildcard string: '%s'", value)
		}
		if ok, _ := isIP(contents[0]); !ok {
			return false, fmt.Sprintf("Invalid ip string: '%s'", contents[0])
		}
		if _, err := strconv.Atoi(contents[1]); err != nil {
			return false, fmt.Sprintf("Invalid prefix length: '%s' in IpWildcard string: '%s'", contents[1], value)
		}
		return true, ""
	}
	return isIP(value)
}

// --- small helpers ----------------------------------------------------------

func asMap(v any) map[string]any {
	if m, ok := v.(map[string]any); ok {
		return m
	}
	return nil
}

func toInt(v any) int {
	switch n := v.(type) {
	case int:
		return n
	case int64:
		return int(n)
	case float64:
		return int(n)
	case string:
		i, _ := strconv.Atoi(n)
		return i
	default:
		return 0
	}
}

func asInt(v any) (int64, bool) {
	switch n := v.(type) {
	case int:
		return int64(n), true
	case int64:
		return n, true
	case int32:
		return int64(n), true
	case float64:
		return int64(n), true
	default:
		return 0, false
	}
}

func lengthOf(v any) int {
	switch t := v.(type) {
	case string:
		return len(t)
	case []any:
		return len(t)
	default:
		return 0
	}
}

func allowedValueContains(values []AllowedValue, target any) bool {
	for _, v := range values {
		if v.Name == fmt.Sprint(target) {
			return true
		}
	}
	return false
}

func allowedValueNames(values []AllowedValue) []string {
	out := make([]string, len(values))
	for i, v := range values {
		out[i] = v.Name
	}
	return out
}

func pyStringListRepr(values []string) string {
	parts := make([]string, len(values))
	for i, v := range values {
		parts[i] = "'" + v + "'"
	}
	return "[" + strings.Join(parts, ", ") + "]"
}

// pyRepr renders a value the way Python's str() would for common types.
func pyRepr(v any) string {
	switch t := v.(type) {
	case bool:
		if t {
			return "True"
		}
		return "False"
	case nil:
		return "None"
	case string:
		return t
	default:
		return fmt.Sprint(v)
	}
}
