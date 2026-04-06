#!/bin/bash
# XML Utilities Loader
# Sources all XML modules for backward compatibility
#
# IMPORTANT: This file is now a loader only. New code should source
# individual modules directly from .aether/utils/ or .aether/exchange/
#
# Usage: source .aether/utils/xml-utils.sh
#
# Modules loaded:
#   - xml-core.sh      : Core operations (validate, format, escape)
#   - xml-query.sh     : XPath queries
#   - xml-convert.sh   : JSON/XML conversion
#   - xml-compose.sh   : XInclude composition
#   - pheromone-xml.sh : Pheromone exchange
#   - wisdom-xml.sh    : Queen wisdom exchange
#   - registry-xml.sh  : Colony registry exchange
#
# Deprecated functions (maintained for compatibility):
#   - pheromone-to-xml()  -> Use xml-pheromone-export()
#   - pheromone-from-xml() -> Use xml-pheromone-import()
#   - queen-wisdom-to-xml() -> Use xml-wisdom-export()
#   - queen-wisdom-from-xml() -> Use xml-wisdom-import()

set -euo pipefail

# Determine script directory for relative sourcing
# Handle case when sourced interactively (BASH_SOURCE[0] may be empty)
if [[ -n "${BASH_SOURCE[0]:-}" ]]; then
    SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
else
    # Fallback: derive from the sourced script's location
    SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]:-$0}")" && pwd)"
fi
EXCHANGE_DIR="$(cd "$SCRIPT_DIR/../exchange" && pwd)"

# ============================================================================
# Load Core Modules
# ============================================================================

# Core utilities (required by other modules)
source "$SCRIPT_DIR/xml-core.sh"

# Query functions
source "$SCRIPT_DIR/xml-query.sh"

# Conversion functions
source "$SCRIPT_DIR/xml-convert.sh"

# Composition functions
source "$SCRIPT_DIR/xml-compose.sh"

# ============================================================================
# Load Exchange Modules
# ============================================================================

# Pheromone exchange (export/import)
source "$EXCHANGE_DIR/pheromone-xml.sh"

# Queen wisdom exchange
source "$EXCHANGE_DIR/wisdom-xml.sh"

# Colony registry exchange
source "$EXCHANGE_DIR/registry-xml.sh"

# ============================================================================
# Backward Compatibility Aliases
# ============================================================================

# Map old function names to new ones for compatibility
pheromone-to-xml() { xml-pheromone-export "$@"; }
pheromone-from-xml() { xml-pheromone-import "$@"; }
queen-wisdom-to-xml() { xml-wisdom-export "$@"; }
queen-wisdom-from-xml() { xml-wisdom-import "$@"; }
registry-to-xml() { xml-registry-export "$@"; }
registry-from-xml() { xml-registry-import "$@"; }

# Export compatibility aliases
export -f pheromone-to-xml pheromone-from-xml
export -f queen-wisdom-to-xml queen-wisdom-from-xml
export -f registry-to-xml registry-from-xml

# ============================================================================
# Module Information
# ============================================================================

# xml-utils-info: Display module information
# Usage: xml-utils-info
xml-utils-info() {
    xml_json_ok '{
        "modules": [
            "xml-core.sh",
            "xml-query.sh",
            "xml-convert.sh",
            "xml-compose.sh",
            "pheromone-xml.sh",
            "wisdom-xml.sh",
            "registry-xml.sh"
        ],
        "note": "This loader provides backward compatibility. New code should source individual modules.",
        "tools": {
            "xmllint": '"$XMLLINT_AVAILABLE"',
            "xmlstarlet": '"$XMLSTARLET_AVAILABLE"',
            "xsltproc": '"$XSLTPROC_AVAILABLE"'
        }
    }'
}

export -f xml-utils-info

