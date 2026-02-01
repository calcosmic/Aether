#!/bin/bash
echo "════════════════════════════════════════════════════════════════"
echo "                    RALPH STATUS CHECK"
echo "════════════════════════════════════════════════════════════════"
echo ""
echo "Time: $(date '+%H:%M:%S')"
echo ""

# Get activity count
local total=$(wc -l < /Users/callumcowie/.claude/projects/-Users-callumcowie-repos-cosmic-dev-system/b1ee0af4-b83e-422d-8979-6a1ef79c17d9/subagents/agent-a9978c7.jsonl 2>/dev/null)
local progress=$(cat /Users/callumcowie/.claude/projects/-Users-callumcowie-repos-cosmic-dev-system/b1ee0af4-b83e-422d-8979-6a1ef79c17d9/subagents/agent-a9978c7.jsonl 2>/dev/null | jq -r 'select(.type=="progress")' | wc -l)

echo "📊 Research Activity:"
echo "   Total Output Lines: $total"
echo "   Research Events: $progress"
echo ""

# Get latest text
echo "📝 Latest Activity:"
cat /Users/callumcowie/.claude/projects/-Users/callumcowie-repos-cosmic-dev-system/b1ee0af4-b83e-422d-8979-6a1ef79c17d9/subagents/agent-a9978c7.jsonl 2>/dev/null | jq -r 'select(.message.content) | .message.content[] | select(.type == "text") | .text' 2>/dev/null | tail -3
echo ""

# Show documents
echo "📄 Research Documents:"
ls -lh .ralph/*.md 2>/dev/null | awk '{print "   " $9 " (" $5 ")"}'
echo ""

echo "════════════════════════════════════════════════════════════════"
