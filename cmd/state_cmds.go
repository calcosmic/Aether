package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/spf13/cobra"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

var mutateArgJSON []string
var mutateArg []string

var stateMutateCmd = &cobra.Command{
	Use:          "state-mutate [expression]",
	Short:        "Atomically mutate COLONY_STATE.json (supports --field/--value and jq-like expressions)",
	Args:         cobra.ArbitraryArgs,
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}
		field, _ := cmd.Flags().GetString("field")
		if field != "" {
			return executeFieldMode(cmd, field)
		}
		vars := parseMutateVars(cmd, args)
		if len(vars.remaining) == 0 {
			outputError(1, "either an expression or --field/--value is required", nil)
			return nil
		}
		expr := vars.remaining[len(vars.remaining)-1]
		return executeExpression(expr, vars.vars)
	},
}

type mutateVars struct {
	vars      map[string]interface{}
	remaining []string
}

func parseMutateVars(cmd *cobra.Command, positionalArgs []string) mutateVars {
	result := mutateVars{vars: make(map[string]interface{})}
	argJSONChanged := cmd.Flags().Changed("argjson")
	argChanged := cmd.Flags().Changed("arg")
	var names []string
	if argJSONChanged {
		names = append(names, mutateArgJSON...)
	}
	if argChanged {
		names = append(names, mutateArg...)
	}
	argIdx := 0
	for _, p := range positionalArgs {
		if argIdx < len(names) {
			name := names[argIdx]
			if argIdx < len(mutateArgJSON) && argJSONChanged {
				var val interface{}
				if err := json.Unmarshal([]byte(p), &val); err != nil {
					result.vars[name] = p
				} else {
					result.vars[name] = val
				}
			} else {
				result.vars[name] = p
			}
			argIdx++
		} else {
			result.remaining = append(result.remaining, p)
		}
	}
	return result
}

func resetMutateFlags() {
	mutateArgJSON = nil
	mutateArg = nil
}

func executeFieldMode(cmd *cobra.Command, field string) error {
	value := mustGetString(cmd, "value")
	var state colony.ColonyState
	if err := store.LoadJSON("COLONY_STATE.json", &state); err != nil {
		outputError(1, "COLONY_STATE.json not found", nil)
		return nil
	}
	switch field {
	case "goal":
		state.Goal = &value
	case "state":
		newState := colony.State(value)
		if err := colony.Transition(state.State, newState); err != nil {
			outputError(1, fmt.Sprintf("invalid transition %s -> %s: %v", state.State, newState, err), nil)
			return nil
		}
		state.State = newState
	case "current_phase":
		phaseNum := 0
		if _, err := fmt.Sscanf(value, "%d", &phaseNum); err != nil {
			outputError(1, fmt.Sprintf("invalid phase number %q", value), nil)
			return nil
		}
		if phaseNum > 0 && phaseNum <= len(state.Plan.Phases) {
			state.Plan.Phases[phaseNum-1].Status = colony.PhaseInProgress
		}
		state.CurrentPhase = phaseNum
	case "milestone":
		state.Milestone = value
	case "colony_depth":
		d := colony.ColonyDepth(value)
		if !d.Valid() {
			outputError(1, fmt.Sprintf("invalid colony depth %q: must be light, standard, deep, or full", value), nil)
			return nil
		}
		state.ColonyDepth = d
	case "plan_granularity":
		g := colony.PlanGranularity(value)
		if !g.Valid() {
			outputError(1, fmt.Sprintf("invalid plan granularity %q: must be sprint, milestone, quarter, or major", value), nil)
			return nil
		}
		state.PlanGranularity = g
	case "colony_name":
		state.ColonyName = &value
	default:
		data, err := json.Marshal(state)
		if err != nil {
			outputError(2, fmt.Sprintf("marshal error: %v", err), nil)
			return nil
		}
		data, err = setNestedFieldJSON(data, field, value)
		if err != nil {
			outputError(1, fmt.Sprintf("unknown field %q", field), nil)
			return nil
		}
		if err := json.Unmarshal(data, &state); err != nil {
			outputError(2, fmt.Sprintf("failed to round-trip state: %v", err), nil)
			return nil
		}
	}
	if err := store.SaveJSON("COLONY_STATE.json", state); err != nil {
		outputError(2, fmt.Sprintf("failed to save state: %v", err), nil)
		return nil
	}
	outputOK(map[string]interface{}{"updated": true, "field": field, "value": value})
	return nil
}

func executeExpression(expr string, vars map[string]interface{}) error {
	data, err := store.ReadFile("COLONY_STATE.json")
	if err != nil {
		outputError(1, "COLONY_STATE.json not found", nil)
		return nil
	}
	for _, sub := range splitChainedAssignments(expr) {
		sub = strings.TrimSpace(sub)
		if sub == "" {
			continue
		}
		data, err = applySubExpression(data, sub, vars)
		if err != nil {
			outputError(1, fmt.Sprintf("expression error: %v", exprError(sub, err)), nil)
			return nil
		}
	}
	var pretty bytes.Buffer
	json.Indent(&pretty, data, "", "  ")
	pretty.WriteByte('\n')

	// Validate typed fields after expression mutation (per D-08)
	var validateState colony.ColonyState
	if err := json.Unmarshal(data, &validateState); err == nil {
		if validateState.ColonyDepth != "" && !validateState.ColonyDepth.Valid() {
			outputError(1, fmt.Sprintf("invalid colony depth %q: must be light, standard, deep, or full", validateState.ColonyDepth), nil)
			return nil
		}
		if validateState.PlanGranularity != "" && !validateState.PlanGranularity.Valid() {
			outputError(1, fmt.Sprintf("invalid plan_granularity %q: must be sprint, milestone, quarter, or major", validateState.PlanGranularity), nil)
			return nil
		}
	}

	if err := store.AtomicWrite("COLONY_STATE.json", pretty.Bytes()); err != nil {
		outputError(2, fmt.Sprintf("failed to save state: %v", err), nil)
		return nil
	}
	outputOK(map[string]interface{}{"updated": true, "expr": expr})
	return nil
}

func splitChainedAssignments(expr string) []string {
	var parts []string
	depth := 0
	current := strings.Builder{}
	hasAssignment := false
	runes := []rune(expr)
	for i, ch := range runes {
		switch ch {
		case '(', '[', '{':
			depth++
			current.WriteRune(ch)
		case ')', ']', '}':
			depth--
			current.WriteRune(ch)
		case '=':
			if depth == 0 && (i == 0 || runes[i-1] != '!' && runes[i-1] != '<' && runes[i-1] != '>') {
				hasAssignment = true
			}
			current.WriteRune(ch)
		case '|':
			if depth == 0 && i+1 < len(runes) && runes[i+1] != '=' && runes[i+1] != '|' {
				if hasAssignment {
					parts = append(parts, current.String())
					current.Reset()
					hasAssignment = false
					continue
				}
			}
			current.WriteRune(ch)
		default:
			current.WriteRune(ch)
		}
	}
	if current.Len() > 0 {
		parts = append(parts, current.String())
	}
	return parts
}

var reFieldSet = regexp.MustCompile(`^\.([\w.\[\]]+)\s*=\s*(.+)$`)
var reConditionalMap = regexp.MustCompile(
	`^\.([\w.]+)\s*\|=\s*map\(if \.([\w]+)\s*==\s*\$(\w+)\s+then \.([\w]+)\s*=\s*"([^"]+)"\s+else \.\s+end\)$`,
)
var reCappedAppendDoubleParen = regexp.MustCompile(`^\.([\w.]+)\s*=\s*\(\((\.([\w.]+)\[-\d+:\])\)\s*\+\s*\["([^"]+)"\]\)$`)
var reCappedAppend = regexp.MustCompile(`^\.([\w.]+)\s*=\s*\((\.([\w.]+)\[-\d+:\])\)\s*\+\s*\["([^"]+)"\]$`)
var reCappedAppendExpr = regexp.MustCompile(`^\.([\w.]+)\s*=\s*\((\.([\w.]+)\[-\d+:\])\)\s*\+\s*\[(.+)\]$`)
var reArrayAppend = regexp.MustCompile(`^\.([\w.]+)\s*\+=\s*\[(.+)\]$`)
var reSortBy = regexp.MustCompile(`^\.([\w.]+)\s*\|\s*sort_by\(([\w.]+)\)\s*\|\s*\.\[-(\d+):\]$`)
var reSortByOnly = regexp.MustCompile(`^\.([\w.]+)\s*\|\s*sort_by\(([\w.]+)\)$`)
var reSliceOnly = regexp.MustCompile(`^\.\[-(\d+):\]$`)
var reSortAssign = regexp.MustCompile(`^\.([\w.]+)\s*=\s*\(\.([\w.]+)\s*\|\s*sort_by\(([\w.]+)\)\s*\|\s*\.\[-(\d+):\]\)$`)
var reSliceAssign = regexp.MustCompile(`^\.([\w.]+)\s*=\s*\(\.([\w.]+)\[-(\d+):\]\)$`)

func applySubExpression(data []byte, expr string, vars map[string]interface{}) ([]byte, error) {
	if m := reSortAssign.FindStringSubmatch(expr); m != nil {
		return applySortAssign(data, m[1], m[2], m[3], m[4])
	}
	if m := reSliceAssign.FindStringSubmatch(expr); m != nil {
		return applySliceAssign(data, m[1], m[2], m[3])
	}
	if m := reCappedAppendDoubleParen.FindStringSubmatch(expr); m != nil {
		return applyCappedAppend(data, m[1], m[2], m[4])
	}
	if m := reSortBy.FindStringSubmatch(expr); m != nil {
		return applySortAndSlice(data, m[1], m[2], m[3])
	}
	if m := reSortByOnly.FindStringSubmatch(expr); m != nil {
		return applySortOnly(data, m[1], m[2])
	}
	if m := reConditionalMap.FindStringSubmatch(expr); m != nil {
		return applyConditionalMap(data, m[1], m[2], m[3], m[4], m[5], vars)
	}
	if m := reCappedAppend.FindStringSubmatch(expr); m != nil {
		return applyCappedAppend(data, m[1], m[2], m[4])
	}
	if m := reCappedAppendExpr.FindStringSubmatch(expr); m != nil {
		return applyCappedAppendExpr(data, m[1], m[2], m[3])
	}
	if m := reArrayAppend.FindStringSubmatch(expr); m != nil {
		return applyArrayAppend(data, m[1], m[2], vars)
	}
	if m := reFieldSet.FindStringSubmatch(expr); m != nil {
		return applyFieldSet(data, m[1], m[2], vars)
	}
	if m := reSliceOnly.FindStringSubmatch(expr); m != nil {
		return applySliceOnly(data, m[1], m[2])
	}
	return nil, fmt.Errorf("unrecognized expression pattern")
}

func exprError(expr string, err error) error {
	return fmt.Errorf("%s: %w", expr, err)
}

// normalizeBracketPath converts bracket notation (e.g. "phases[0]") to dot
// notation (e.g. "phases.0") so that sjson can handle array indexing.
func normalizeBracketPath(path string) string {
	return bracketRe.ReplaceAllString(path, ".$1")
}

var bracketRe = regexp.MustCompile(`\[(\d+)\]`)

func applyFieldSet(data []byte, path, valueExpr string, vars map[string]interface{}) ([]byte, error) {
	value := resolveValue(valueExpr, vars)
	if value == "" && strings.HasPrefix(valueExpr, "$") {
		return nil, fmt.Errorf("undeclared variable %s", valueExpr)
	}
	path = normalizeBracketPath(path)
	return sjson.SetRawBytes(data, path, []byte(value))
}

func applyConditionalMap(data []byte, arrayPath, matchField, varName, setField, setValue string, vars map[string]interface{}) ([]byte, error) {
	matchVal, ok := vars[varName]
	if !ok {
		return nil, fmt.Errorf("undeclared variable $%s", varName)
	}
	matchStr := toFloatOrString(matchVal)
	arr := gjson.GetBytes(data, arrayPath)
	if !arr.IsArray() {
		return nil, fmt.Errorf("path %q is not an array", arrayPath)
	}
	var newArr []json.RawMessage
	for _, item := range arr.Array() {
		var obj map[string]interface{}
		if err := json.Unmarshal([]byte(item.Raw), &obj); err != nil {
			return nil, fmt.Errorf("array element is not an object")
		}
		itemVal := gjson.Get(item.Raw, matchField)
		if compareValues(itemVal, matchStr) {
			obj[setField] = setValue
		}
		updated, err := json.Marshal(obj)
		if err != nil {
			return nil, fmt.Errorf("marshal element: %w", err)
		}
		newArr = append(newArr, updated)
	}
	arrBytes, err := json.Marshal(newArr)
	if err != nil {
		return nil, fmt.Errorf("marshal array: %w", err)
	}
	return sjson.SetRawBytes(data, arrayPath, arrBytes)
}

func applyCappedAppend(data []byte, targetPath, srcPath, value string) ([]byte, error) {
	basePath := extractSliceBasePath(strings.TrimPrefix(srcPath, "."))
	items := getArraySlice(data, basePath, extractSliceCount(srcPath))
	items = append(items, json.RawMessage(jsonString(value)))
	arrBytes, err := json.Marshal(items)
	if err != nil {
		return nil, fmt.Errorf("marshal: %w", err)
	}
	return sjson.SetRawBytes(data, targetPath, arrBytes)
}

func applyCappedAppendExpr(data []byte, targetPath, srcPath, valueExpr string) ([]byte, error) {
	basePath := extractSliceBasePath(strings.TrimPrefix(srcPath, "."))
	items := getArraySlice(data, basePath, extractSliceCount(srcPath))
	valueExpr = "[" + valueExpr + "]"
	var newItems []interface{}
	if err := json.Unmarshal([]byte(valueExpr), &newItems); err != nil {
		return nil, fmt.Errorf("invalid array expression %q: %w", valueExpr, err)
	}
	items = append(items, newItems...)
	arrBytes, err := json.Marshal(items)
	if err != nil {
		return nil, fmt.Errorf("marshal: %w", err)
	}
	return sjson.SetRawBytes(data, targetPath, arrBytes)
}

func applyArrayAppend(data []byte, path, valueExpr string, vars map[string]interface{}) ([]byte, error) {
	resolved := resolveValue(valueExpr, vars)
	var val interface{}
	if err := json.Unmarshal([]byte(resolved), &val); err != nil {
		val = resolved
	}
	arr := gjson.GetBytes(data, path)
	var items []interface{}
	if arr.IsArray() {
		for _, item := range arr.Array() {
			var v interface{}
			json.Unmarshal([]byte(item.Raw), &v)
			items = append(items, v)
		}
	}
	items = append(items, val)
	arrBytes, err := json.Marshal(items)
	if err != nil {
		return nil, fmt.Errorf("marshal: %w", err)
	}
	return sjson.SetRawBytes(data, path, arrBytes)
}

func applySortAndSlice(data []byte, path, sortField, countStr string) ([]byte, error) {
	count, err := strconv.Atoi(countStr)
	if err != nil {
		return nil, fmt.Errorf("invalid slice count %q: %w", countStr, err)
	}
	// Strip leading dot from sort field for gjson compatibility.
	if strings.HasPrefix(sortField, ".") {
		sortField = sortField[1:]
	}
	return sortArrayByField(data, path, sortField, count)
}

func applySortOnly(data []byte, path, sortField string) ([]byte, error) {
	if strings.HasPrefix(sortField, ".") {
		sortField = sortField[1:]
	}
	return sortArrayByField(data, path, sortField, 0)
}

func applySliceOnly(data []byte, _, countStr string) ([]byte, error) {
	count, err := strconv.Atoi(countStr)
	if err != nil {
		return nil, fmt.Errorf("invalid slice count %q: %w", countStr, err)
	}
	var arr []interface{}
	if err := json.Unmarshal(data, &arr); err != nil {
		return nil, fmt.Errorf("data is not a JSON array")
	}
	if count > 0 && len(arr) > count {
		arr = arr[len(arr)-count:]
	}
	return json.Marshal(arr)
}

func applySortAssign(data []byte, targetPath, basePath, sortField, countStr string) ([]byte, error) {
	count, err := strconv.Atoi(countStr)
	if err != nil {
		return nil, fmt.Errorf("invalid slice count %q: %w", countStr, err)
	}
	if strings.HasPrefix(sortField, ".") {
		sortField = sortField[1:]
	}
	sorted, err := sortArrayByField(data, basePath, sortField, count)
	if err != nil {
		return nil, err
	}
	arrResult := gjson.GetBytes(sorted, basePath)
	if !arrResult.IsArray() {
		return nil, fmt.Errorf("sort result for %q is not an array", basePath)
	}
	return sjson.SetRawBytes(data, targetPath, []byte(arrResult.Raw))
}

func applySliceAssign(data []byte, targetPath, basePath, countStr string) ([]byte, error) {
	count, err := strconv.Atoi(countStr)
	if err != nil {
		return nil, fmt.Errorf("invalid slice count %q: %w", countStr, err)
	}
	// Strip leading dot from basePath for gjson compatibility.
	basePath = strings.TrimPrefix(basePath, ".")
	items := getArraySlice(data, basePath, count)
	arrBytes, err := json.Marshal(items)
	if err != nil {
		return nil, fmt.Errorf("marshal: %w", err)
	}
	return sjson.SetRawBytes(data, targetPath, arrBytes)
}

func sortArrayByField(data []byte, path, sortField string, sliceCount int) ([]byte, error) {
	arr := gjson.GetBytes(data, path)
	if !arr.IsArray() {
		return nil, fmt.Errorf("path %q is not an array", path)
	}
	type kv struct {
		raw   json.RawMessage
		key   float64
		str   string
		isNum bool
	}
	var items []kv
	for _, item := range arr.Array() {
		fieldVal := gjson.Get(item.Raw, sortField)
		entry := kv{raw: json.RawMessage(item.Raw)}
		if fieldVal.Type == gjson.Number {
			entry.key = fieldVal.Float()
			entry.isNum = true
		} else {
			entry.str = fieldVal.String()
			entry.isNum = false
		}
		items = append(items, entry)
	}
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].isNum && items[j].isNum {
			return items[i].key < items[j].key
		}
		if items[i].isNum {
			return true
		}
		if items[j].isNum {
			return false
		}
		return items[i].str < items[j].str
	})
	if sliceCount > 0 && len(items) > sliceCount {
		items = items[len(items)-sliceCount:]
	}
	var result []json.RawMessage
	for _, item := range items {
		result = append(result, item.raw)
	}
	arrBytes, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("marshal sorted array: %w", err)
	}
	return sjson.SetRawBytes(data, path, arrBytes)
}

func getArraySlice(data []byte, basePath string, count int) []interface{} {
	arr := gjson.GetBytes(data, basePath)
	if !arr.IsArray() {
		return nil
	}
	all := arr.Array()
	if count <= 0 || count >= len(all) {
		count = len(all)
	}
	sliced := all[len(all)-count:]
	var items []interface{}
	for _, item := range sliced {
		items = append(items, json.RawMessage(item.Raw))
	}
	return items
}

func extractSliceBasePath(path string) string {
	if idx := strings.Index(path, "["); idx >= 0 {
		return path[:idx]
	}
	return path
}

func extractSliceCount(path string) int {
	start := strings.Index(path, "[-")
	if start < 0 {
		return 0
	}
	end := strings.Index(path[start:], ":]")
	if end < 0 {
		return 0
	}
	countStr := path[start+2 : start+end]
	count, err := strconv.Atoi(countStr)
	if err != nil {
		return 0
	}
	return count
}

func resolveValue(expr string, vars map[string]interface{}) string {
	expr = strings.TrimSpace(expr)
	if expr == "null" {
		return "null"
	}
	if expr == "true" {
		return "true"
	}
	if expr == "false" {
		return "false"
	}
	if len(expr) >= 2 && expr[0] == '"' && expr[len(expr)-1] == '"' {
		return expr
	}
	if _, err := strconv.ParseFloat(expr, 64); err == nil {
		return expr
	}
	if (expr[0] == '{' || expr[0] == '[') && json.Valid([]byte(expr)) {
		return expr
	}
	if strings.HasPrefix(expr, "$") {
		varName := expr[1:]
		val, ok := vars[varName]
		if !ok {
			return ""
		}
		switch v := val.(type) {
		case float64:
			return strconv.FormatFloat(v, 'f', -1, 64)
		case int:
			return strconv.Itoa(v)
		case int64:
			return strconv.FormatInt(v, 10)
		case string:
			return jsonString(v)
		case bool:
			if v {
				return "true"
			}
			return "false"
		default:
			b, _ := json.Marshal(v)
			return string(b)
		}
	}
	return jsonString(expr)
}

func jsonString(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}

func toFloatOrString(val interface{}) interface{} {
	switch v := val.(type) {
	case float64:
		return v
	case int:
		return float64(v)
	case int64:
		return float64(v)
	case string:
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f
		}
		return v
	default:
		return v
	}
}

func compareValues(itemVal gjson.Result, matchVal interface{}) bool {
	switch mv := matchVal.(type) {
	case float64:
		return itemVal.Type == gjson.Number && itemVal.Float() == mv
	default:
		return itemVal.String() == fmt.Sprint(mv)
	}
}

func setNestedFieldJSON(data []byte, fieldPath, value string) ([]byte, error) {
	// Try to parse value as raw JSON. If it's valid JSON (number, boolean, null,
	// object, array, or quoted string), use SetRawBytes to avoid double-encoding.
	// If it fails, treat it as a plain string and let SetBytes handle quoting.
	var raw json.RawMessage
	if json.Unmarshal([]byte(value), &raw) == nil {
		return sjson.SetRawBytes(data, fieldPath, []byte(value))
	}
	return sjson.SetBytes(data, fieldPath, value)
}

var loadStateCmd = &cobra.Command{
	Use:   "load-state",
	Short: "Load COLONY_STATE.json and return it",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}
		var state colony.ColonyState
		if err := store.LoadJSON("COLONY_STATE.json", &state); err != nil {
			outputError(1, "COLONY_STATE.json not found", nil)
			return nil
		}
		outputOK(state)
		return nil
	},
}

var unloadStateCmd = &cobra.Command{
	Use:   "unload-state",
	Short: "Release state lock (placeholder)",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		outputOK(map[string]interface{}{"unloaded": true})
		return nil
	},
}

var validateStateCmd = &cobra.Command{
	Use:   "validate-state",
	Short: "Validate COLONY_STATE.json structure",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}
		var state colony.ColonyState
		if err := store.LoadJSON("COLONY_STATE.json", &state); err != nil {
			outputError(1, fmt.Sprintf("failed to load state: %v", err), nil)
			return nil
		}
		issues := []string{}
		if state.Version == "" {
			issues = append(issues, "missing version")
		}
		if state.Goal == nil || *state.Goal == "" {
			issues = append(issues, "missing goal")
		}
		if state.State == "" {
			issues = append(issues, "missing state")
		}
		valid := len(issues) == 0
		outputOK(map[string]interface{}{"valid": valid, "issues": issues, "version": state.Version})
		return nil
	},
}

var stateReadCmd = &cobra.Command{
	Use:   "state-read",
	Short: "Read COLONY_STATE.json",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}
		var state colony.ColonyState
		if err := store.LoadJSON("COLONY_STATE.json", &state); err != nil {
			outputError(1, "COLONY_STATE.json not found", nil)
			return nil
		}
		outputOK(state)
		return nil
	},
}

var stateReadFieldCmd = &cobra.Command{
	Use:   "state-read-field",
	Short: "Read a specific field from COLONY_STATE.json",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}
		field := mustGetString(cmd, "field")
		if field == "" {
			return nil
		}
		var state colony.ColonyState
		if err := store.LoadJSON("COLONY_STATE.json", &state); err != nil {
			outputError(1, "COLONY_STATE.json not found", nil)
			return nil
		}
		var result interface{}
		switch field {
		case "goal":
			result = state.Goal
		case "state":
			result = string(state.State)
		case "current_phase":
			result = state.CurrentPhase
		case "milestone":
			result = state.Milestone
		case "colony_depth":
			result = string(state.ColonyDepth)
		case "colony_name":
			result = state.ColonyName
		case "session_id":
			result = state.SessionID
		default:
			outputError(1, fmt.Sprintf("unknown field %q", field), nil)
			return nil
		}
		outputOK(map[string]interface{}{"field": field, "value": result})
		return nil
	},
}

func init() {
	stateMutateCmd.Flags().String("field", "", "Field to mutate (legacy mode)")
	stateMutateCmd.Flags().String("value", "", "New value (legacy mode)")
	stateMutateCmd.Flags().StringArrayVar(&mutateArgJSON, "argjson", nil, "JSON variable (jq-style: --argjson name json)")
	stateMutateCmd.Flags().StringArrayVar(&mutateArg, "arg", nil, "String variable (jq-style: --arg name value)")
	stateReadFieldCmd.Flags().String("field", "", "Field to read (required)")
	rootCmd.AddCommand(stateMutateCmd)
	rootCmd.AddCommand(loadStateCmd)
	rootCmd.AddCommand(unloadStateCmd)
	rootCmd.AddCommand(validateStateCmd)
	rootCmd.AddCommand(stateReadCmd)
	rootCmd.AddCommand(stateReadFieldCmd)
}

func setNestedField(obj interface{}, fieldPath, value string) (interface{}, error) {
	data, err := json.Marshal(obj)
	if err != nil {
		return nil, err
	}
	result, err := setNestedFieldJSON(data, fieldPath, value)
	if err != nil {
		return nil, err
	}
	var m map[string]interface{}
	if err := json.Unmarshal(result, &m); err != nil {
		return nil, err
	}
	return m, nil
}
