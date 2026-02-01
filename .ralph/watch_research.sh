#!/bin/bash
while true; do
    clear
    echo "════════════════════════════════════════════════════════════════"
    echo "                    AETHER RESEARCH DOCUMENTS"
    echo "════════════════════════════════════════════════════════════════"
    echo "Time: $(date '+%H:%M:%S')"
    echo ""
    ls -lh .ralph/*RESEARCH.md 2>/dev/null | awk '{printf "  %-50s %8s\n", $9, $5}'
    echo ""
    echo "Progress: $(ls .ralph/*RESEARCH.md 2>/dev/null | wc -l | tr -d ' ') documents"
    echo "════════════════════════════════════════════════════════════════"
    sleep 3
done
