#!/bin/bash
# Aether v3.1 Recovery Script
# Run this after reading RECOVERY-PLAN.md

set -e

echo "╔══════════════════════════════════════════════════════════════╗"
echo "║     AETHER v3.1 RECOVERY - RUNNING SYNC                      ║"
echo "╚══════════════════════════════════════════════════════════════╝"
echo ""

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Step 1: Check if we're in the right place
if [ ! -f "package.json" ] || [ ! -d ".aether" ] || [ ! -d "runtime" ]; then
    echo -e "${RED}ERROR: Must run from Aether repo root${NC}"
    exit 1
fi

# Step 2: Commit any staged changes
echo "Step 1: Checking git status..."
if ! git diff --cached --quiet; then
    echo "Found staged changes. Committing..."
    git commit -m "recovery: staged changes before sync"
fi

# Step 3: Sync runtime/ from .aether/
echo ""
echo "Step 2: Syncing runtime/ from .aether/..."
echo "------------------------------------------"

# Core files
echo "  → workers.md"
cp .aether/workers.md runtime/workers.md

echo "  → aether-utils.sh"
cp .aether/aether-utils.sh runtime/aether-utils.sh

echo "  → verification-loop.md"
cp .aether/verification-loop.md runtime/verification-loop.md 2>/dev/null || echo "    (skipped - not in .aether)"

# Utils - sync all files
echo ""
echo "  → utilities/"
for file in .aether/utils/*.sh; do
    if [ -f "$file" ]; then
        filename=$(basename "$file")
        echo "     copying $filename"
        cp "$file" "runtime/utils/$filename"
    fi
done

# Docs - ensure runtime/docs/ exists and has files
if [ -d ".aether/docs" ]; then
    echo ""
    echo "  → docs/"
    mkdir -p runtime/docs
    for file in .aether/docs/*.md .aether/docs/*.json 2>/dev/null; do
        if [ -f "$file" ]; then
            filename=$(basename "$file")
            echo "     copying $filename"
            cp "$file" "runtime/docs/$filename"
        fi
    done
fi

# Step 4: Verify the sync
echo ""
echo "Step 3: Verifying sync..."
echo "------------------------------------------"

# Check emoji section
if grep -q "Caste Emoji Mapping:" runtime/workers.md; then
    echo -e "${GREEN}  ✓ workers.md has emoji section${NC}"
else
    echo -e "${RED}  ✗ workers.md MISSING emoji section${NC}"
fi

# Check get_caste_emoji
if grep -q "get_caste_emoji()" runtime/aether-utils.sh; then
    echo -e "${GREEN}  ✓ aether-utils.sh has get_caste_emoji${NC}"
else
    echo -e "${RED}  ✗ aether-utils.sh MISSING get_caste_emoji${NC}"
fi

# Count utils files
RUNTIME_UTILS=$(ls runtime/utils/*.sh 2>/dev/null | wc -l)
AETHER_UTILS=$(ls .aether/utils/*.sh 2>/dev/null | wc -l)
echo "  → runtime/utils: $RUNTIME_UTILS files"
echo "  → .aether/utils: $AETHER_UTILS files"

if [ "$RUNTIME_UTILS" -eq "$AETHER_UTILS" ]; then
    echo -e "${GREEN}  ✓ Utils counts match${NC}"
else
    echo -e "${YELLOW}  ⚠ Utils counts differ (may be okay if some are .aether-specific)${NC}"
fi

# Step 5: Stage the changes
echo ""
echo "Step 4: Staging changes..."
git add runtime/

# Show status
echo ""
echo "Git status after sync:"
git status --short runtime/

# Step 6: Instructions
echo ""
echo "╔══════════════════════════════════════════════════════════════╗"
echo "║     RECOVERY SYNC COMPLETE                                   ║"
echo "╚══════════════════════════════════════════════════════════════╝"
echo ""
echo "Next steps:"
echo ""
echo "  1. Review the changes:"
echo "     git diff --cached runtime/"
echo ""
echo "  2. Commit the sync:"
echo "     git commit -m 'sync: runtime/ updated from working .aether/'"
echo ""
echo "  3. Reinstall to update hub:"
echo "     npm install -g ."
echo ""
echo "  4. Verify hub updated:"
echo "     grep 'Caste Emoji Mapping:' ~/.aether/system/workers.md"
echo ""
echo "  5. Test in this repo:"
echo "     /ant:init 'Test recovery'"
echo ""
echo "Remember: runtime/ is the SOURCE, .aether/ is the WORKING COPY"
echo ""
