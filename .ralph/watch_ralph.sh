#!/bin/bash
# Ralph Activity Monitor

while true; do
    clear
    echo "════════════════════════════════════════════════════════════════"
    echo "                    RALPH AUTONOMOUS RESEARCH AGENT"
    echo "════════════════════════════════════════════════════════════════"
    echo ""
    echo "Time: $(date '+%Y-%m-%d %H:%M:%S')"
    echo "Agent ID: a9978c7"
    echo ""

    # Extract recent text output
    echo "─ Recent Activity ───────────────────────────────────────────────"
    cat /Users/callumcowie/.claude/projects/-Users-callumcowie-repos-cosmic-dev-system/b1ee0af4-b83e-422d-8979-6a1ef79c17d9/subagents/agent-a9978c7.jsonl 2>/dev/null | \
        jq -r 'select(.message.content) | .message.content[] | select(.type == "text") | .text' 2>/dev/null | \
        tail -15

    echo ""
    echo "─ Status ────────────────────────────────────────────────────────"

    # Count total lines and tool uses
    local total_lines=$(wc -l < /Users/callumcowie/.claude/projects/-Users-callumcowie-repos-cosmic-dev-system/b1ee0af4-b83e-422d-8979-6a1ef79c17d9/subagents/agent-a9978c7.jsonl 2>/dev/null)
    local tool_uses=$(cat /Users/callumcowie/.claude/projects/-Users-callumcowie-repos-cosmic-dev-system/b1ee0af4-b83e-422d-8979-6a1ef79c17d9/subagents/agent-a9978c7.jsonl 2>/dev/null | jq -r 'select(.type=="progress")' | wc -l)

    echo "Total Output Lines: ${total_lines:-0}"
    echo "Tool Uses: ${tool_uses:-0}"

    # Check for created files
    echo ""
    echo "─ Research Documents ────────────────────────────────────────────"
    ls -lh .ralph/*.md 2>/dev/null | tail -5

    echo ""
    echo "════════════════════════════════════════════════════════════════"
    echo "  Press Ctrl+C to exit | Updates every 2 seconds"
    echo "════════════════════════════════════════════════════════════════"

    sleep 2
done
