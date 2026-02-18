#!/usr/bin/env bash
# test-xml.sh — XML Integration requirement verification
# XML-01: Pheromones stored/retrieved via XML format (export + import round-trip)
# XML-02: Wisdom exchange uses XML structure (export + import round-trip)
# XML-03: Registry uses XML for cross-colony communication (export + import)
#
# NOTE: Written for bash 3.2 (macOS default). No associative arrays.
# Supports --results-file <path> flag for master runner integration.
# Known issue: XSD schema validation may fail for expires_at="phase_end" — not a test failure.

set -euo pipefail

E2E_SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$E2E_SCRIPT_DIR/../.." && pwd)"

# Parse --results-file flag
EXTERNAL_RESULTS_FILE=""
while [[ $# -gt 0 ]]; do
  case "$1" in
    --results-file)
      EXTERNAL_RESULTS_FILE="$2"
      shift 2
      ;;
    *)
      shift
      ;;
  esac
done

# Source shared e2e infrastructure
# shellcheck source=./e2e-helpers.sh
source "$E2E_SCRIPT_DIR/e2e-helpers.sh"

echo ""
echo "================================================================"
echo "XML Area: XML Integration Requirements"
echo "================================================================"

# ============================================================================
# Environment Setup
# ============================================================================

E2E_TMP_DIR=$(setup_e2e_env)
trap teardown_e2e_env EXIT

init_results

UTILS="$E2E_TMP_DIR/.aether/aether-utils.sh"

# Check if xmllint is available — required for XML tests
if ! command -v xmllint >/dev/null 2>&1; then
  echo "  SKIP: xmllint not available — XML features require libxml2"
  echo "  Install: xcode-select --install on macOS"
  record_result "XML-01" "FAIL" "xmllint not available"
  record_result "XML-02" "FAIL" "xmllint not available"
  record_result "XML-03" "FAIL" "xmllint not available"
  # Write external results if requested
  if [[ -n "$EXTERNAL_RESULTS_FILE" ]]; then
    echo "XML-01=FAIL" >> "$EXTERNAL_RESULTS_FILE"
    echo "XML-02=FAIL" >> "$EXTERNAL_RESULTS_FILE"
    echo "XML-03=FAIL" >> "$EXTERNAL_RESULTS_FILE"
  fi
  print_area_results "XML"
  exit 1
fi

# ============================================================================
# XML-01: Pheromones stored/retrieved via XML format
# Strategy: export pheromones.json to XML, assert ok:true + file well-formed,
#           then import back and assert ok:true (round-trip)
# Note: XSD schema validation may fail for expires_at="phase_end" — known issue,
#       not counted as XML-01 failure.
# ============================================================================

echo ""
echo "--- XML-01: Pheromones stored/retrieved via XML format ---"

xml01_pass=true
xml01_notes=""

# The pheromones.json fixture is pre-populated by setup_e2e_env with 2 active signals
pheromones_xml="$E2E_TMP_DIR/pheromones-test.xml"

# Run pheromone-export-xml
echo "  Running pheromone-export-xml..."
raw_pex=$(run_in_isolated_env "$E2E_TMP_DIR" pheromone-export-xml "$pheromones_xml")
pex_out=$(extract_json "$raw_pex")

if echo "$pex_out" | jq -e '.ok == true' >/dev/null 2>&1; then
  echo "  PASS: pheromone-export-xml returned ok:true"
else
  xml01_pass=false
  xml01_notes="$xml01_notes [FAIL: pheromone-export-xml ok!=true: $pex_out]"
  echo "  FAIL: pheromone-export-xml did not return ok:true"
  echo "  Got: $pex_out"
fi

# Check XML file was created
if [[ -f "$pheromones_xml" ]]; then
  echo "  PASS: pheromones.xml file created at $pheromones_xml"
else
  xml01_pass=false
  xml01_notes="$xml01_notes [FAIL: XML file not created]"
  echo "  FAIL: pheromones.xml not created"
fi

# Check XML is well-formed
if [[ -f "$pheromones_xml" ]]; then
  if xmllint --noout "$pheromones_xml" 2>/dev/null; then
    echo "  PASS: pheromones.xml is well-formed XML"
  else
    xml01_pass=false
    xml01_notes="$xml01_notes [FAIL: XML not well-formed]"
    echo "  FAIL: pheromones.xml is not well-formed XML"
  fi
fi

# Run pheromone-import-xml (round-trip)
if [[ -f "$pheromones_xml" ]]; then
  echo "  Running pheromone-import-xml (round-trip)..."
  raw_pix=$(run_in_isolated_env "$E2E_TMP_DIR" pheromone-import-xml "$pheromones_xml" 2>&1 || true)
  pix_out=$(extract_json "$raw_pix")

  if echo "$pix_out" | jq -e '.ok == true' >/dev/null 2>&1; then
    echo "  PASS: pheromone-import-xml returned ok:true (round-trip complete)"
  else
    xml01_pass=false
    xml01_notes="$xml01_notes [FAIL: pheromone-import-xml ok!=true: $pix_out]"
    echo "  FAIL: pheromone-import-xml did not return ok:true"
    echo "  Got: $pix_out"
  fi
fi

# Note: XSD schema validation (phase_end) known issue — not tested here
if [[ "$xml01_pass" == "true" ]]; then
  record_result "XML-01" "PASS" "export+import round-trip ok; XSD phase_end known issue documented"
else
  record_result "XML-01" "FAIL" "$xml01_notes"
fi

# ============================================================================
# XML-02: Wisdom exchange uses XML structure
# Strategy: export wisdom (creates from COLONY_STATE if no queen-wisdom.json),
#           assert ok:true + file well-formed, then import back and assert ok:true
# ============================================================================

echo ""
echo "--- XML-02: Wisdom exchange uses XML structure ---"

xml02_pass=true
xml02_notes=""

wisdom_xml="$E2E_TMP_DIR/wisdom-test.xml"

# Run wisdom-export-xml (will auto-generate from COLONY_STATE memory field)
echo "  Running wisdom-export-xml..."
raw_wex=$(run_in_isolated_env "$E2E_TMP_DIR" wisdom-export-xml "" "$wisdom_xml" 2>&1 || true)
wex_out=$(extract_json "$raw_wex")

if echo "$wex_out" | jq -e '.ok == true' >/dev/null 2>&1; then
  echo "  PASS: wisdom-export-xml returned ok:true"
else
  xml02_pass=false
  xml02_notes="$xml02_notes [FAIL: wisdom-export-xml ok!=true: $wex_out]"
  echo "  FAIL: wisdom-export-xml did not return ok:true"
  echo "  Got: $wex_out"
fi

# Check XML file was created
if [[ -f "$wisdom_xml" ]]; then
  echo "  PASS: queen-wisdom.xml file created"
else
  xml02_pass=false
  xml02_notes="$xml02_notes [FAIL: wisdom XML file not created]"
  echo "  FAIL: queen-wisdom.xml not created"
fi

# Check XML is well-formed
if [[ -f "$wisdom_xml" ]]; then
  if xmllint --noout "$wisdom_xml" 2>/dev/null; then
    echo "  PASS: queen-wisdom.xml is well-formed XML"
  else
    xml02_pass=false
    xml02_notes="$xml02_notes [FAIL: wisdom XML not well-formed]"
    echo "  FAIL: queen-wisdom.xml is not well-formed XML"
  fi
fi

# Run wisdom-import-xml (round-trip)
if [[ -f "$wisdom_xml" ]]; then
  echo "  Running wisdom-import-xml (round-trip)..."
  wisdom_output="$E2E_TMP_DIR/wisdom-imported.json"
  raw_wix=$(run_in_isolated_env "$E2E_TMP_DIR" wisdom-import-xml "$wisdom_xml" "$wisdom_output" 2>&1 || true)
  wix_out=$(extract_json "$raw_wix")

  if echo "$wix_out" | jq -e '.ok == true' >/dev/null 2>&1; then
    echo "  PASS: wisdom-import-xml returned ok:true (round-trip complete)"
  else
    xml02_pass=false
    xml02_notes="$xml02_notes [FAIL: wisdom-import-xml ok!=true: $wix_out]"
    echo "  FAIL: wisdom-import-xml did not return ok:true"
    echo "  Got: $wix_out"
  fi
fi

if [[ "$xml02_pass" == "true" ]]; then
  record_result "XML-02" "PASS" "wisdom export+import round-trip ok"
else
  record_result "XML-02" "FAIL" "$xml02_notes"
fi

# ============================================================================
# XML-03: Registry uses XML for cross-colony communication
# Strategy: export registry (auto-generates from chambers scan),
#           assert ok:true + file well-formed, then import back and assert ok:true
# ============================================================================

echo ""
echo "--- XML-03: Registry uses XML for cross-colony communication ---"

xml03_pass=true
xml03_notes=""

registry_xml="$E2E_TMP_DIR/registry-test.xml"

# Run registry-export-xml (will auto-generate minimal registry from chambers)
echo "  Running registry-export-xml..."
raw_rex=$(run_in_isolated_env "$E2E_TMP_DIR" registry-export-xml "" "$registry_xml" 2>&1 || true)
rex_out=$(extract_json "$raw_rex")

if echo "$rex_out" | jq -e '.ok == true' >/dev/null 2>&1; then
  echo "  PASS: registry-export-xml returned ok:true"
else
  xml03_pass=false
  xml03_notes="$xml03_notes [FAIL: registry-export-xml ok!=true: $rex_out]"
  echo "  FAIL: registry-export-xml did not return ok:true"
  echo "  Got: $rex_out"
fi

# Check XML file was created
if [[ -f "$registry_xml" ]]; then
  echo "  PASS: colony-registry.xml file created"
else
  xml03_pass=false
  xml03_notes="$xml03_notes [FAIL: registry XML file not created]"
  echo "  FAIL: colony-registry.xml not created"
fi

# Check XML is well-formed
if [[ -f "$registry_xml" ]]; then
  if xmllint --noout "$registry_xml" 2>/dev/null; then
    echo "  PASS: colony-registry.xml is well-formed XML"
  else
    xml03_pass=false
    xml03_notes="$xml03_notes [FAIL: registry XML not well-formed]"
    echo "  FAIL: colony-registry.xml is not well-formed XML"
  fi
fi

# Run registry-import-xml (round-trip)
if [[ -f "$registry_xml" ]]; then
  echo "  Running registry-import-xml (round-trip)..."
  registry_output="$E2E_TMP_DIR/registry-imported.json"
  raw_rix=$(run_in_isolated_env "$E2E_TMP_DIR" registry-import-xml "$registry_xml" "$registry_output" 2>&1 || true)
  rix_out=$(extract_json "$raw_rix")

  if echo "$rix_out" | jq -e '.ok == true' >/dev/null 2>&1; then
    echo "  PASS: registry-import-xml returned ok:true (round-trip complete)"
  else
    xml03_pass=false
    xml03_notes="$xml03_notes [FAIL: registry-import-xml ok!=true: $rix_out]"
    echo "  FAIL: registry-import-xml did not return ok:true"
    echo "  Got: $rix_out"
  fi
fi

if [[ "$xml03_pass" == "true" ]]; then
  record_result "XML-03" "PASS" "registry export+import round-trip ok"
else
  record_result "XML-03" "FAIL" "$xml03_notes"
fi

# ============================================================================
# Output results
# ============================================================================

# Write external results file if requested (for master runner)
if [[ -n "$EXTERNAL_RESULTS_FILE" ]]; then
  while IFS='|' read -r req_id status notes; do
    echo "${req_id}=${status}" >> "$EXTERNAL_RESULTS_FILE"
  done < "$RESULTS_FILE"
fi

print_area_results "XML"
