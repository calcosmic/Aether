#!/usr/bin/env bash
# Create a scaffold regression test for an incident fix.
# Usage:
#   bash .aether/scripts/incident-test-add.sh <incident_id> <description> "<command>" "<expected_pattern>"

set -euo pipefail

incident_id="${1:-}"
description="${2:-}"
command_string="${3:-}"
expected_pattern="${4:-}"

if [[ -z "$incident_id" || -z "$description" || -z "$command_string" || -z "$expected_pattern" ]]; then
  echo "Usage: bash .aether/scripts/incident-test-add.sh <incident_id> <description> \"<command>\" \"<expected_pattern>\""
  exit 1
fi

tests_dir=".aether/tests"
mkdir -p "$tests_dir"
test_file="$tests_dir/incident-${incident_id}.sh"

cat > "$test_file" <<EOF
#!/usr/bin/env bash
# Regression test for incident ${incident_id}
# Description: ${description}

set -euo pipefail

result=\$(bash -lc '${command_string}' 2>&1 || true)

if echo "\$result" | grep -q '${expected_pattern}'; then
  echo "PASS: ${description}"
  exit 0
fi

echo "FAIL: Expected pattern '${expected_pattern}' not found"
echo ""
echo "Command:"
echo "  ${command_string}"
echo ""
echo "Output:"
echo "\$result"
exit 1
EOF

chmod +x "$test_file"
echo "Created: $test_file"
