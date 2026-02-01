#!/usr/bin/env python3
"""
Aether Queen Ant Colony - Progress Visualization

Visualization components for:
- Phase progress bars
- Agent activity dashboard
- Pheromone signal visualization
- Error pattern displays
"""

from typing import Dict, List, Any, Optional
from datetime import datetime
from dataclasses import dataclass
import math


@dataclass
class ProgressConfig:
    """Configuration for progress display"""
    width: int = 50
    fill_char: str = "█"
    empty_char: str = "░"
    show_percent: bool = True
    show_label: bool = True


class ProgressVisualizer:
    """
    Progress bar visualization for phases and tasks

    Examples:
        [████████░░░░░░] 60% (3/5 tasks)
    """

    def __init__(self, config: Optional[ProgressConfig] = None):
        self.config = config or ProgressConfig()

    def render_progress_bar(
        self,
        completed: int,
        total: int,
        label: Optional[str] = None
    ) -> str:
        """
        Render a progress bar

        Args:
            completed: Number of completed items
            total: Total number of items
            label: Optional label for the progress bar

        Returns:
            Formatted progress bar string
        """
        if total == 0:
            progress = 0
        else:
            progress = min(completed / total, 1.0)

        filled = int(self.config.width * progress)
        empty = self.config.width - filled

        bar = self.config.fill_char * filled + self.config.empty_char * empty

        parts = []
        if self.config.show_label and label:
            parts.append(f"{label}:")

        parts.append(f"[{bar}]")

        if self.config.show_percent:
            parts.append(f"{int(progress * 100)}%")

        parts.append(f"({completed}/{total})")

        return " ".join(parts)

    def render_phase_progress(self, phase: Dict[str, Any]) -> str:
        """
        Render phase progress with task breakdown

        Args:
            phase: Phase dictionary with tasks

        Returns:
            Multi-line progress display
        """
        lines = []

        # Phase header
        phase_id = phase.get("id", "?")
        phase_name = phase.get("name", "Unknown")
        status = phase.get("status", "pending").upper()

        lines.append(f"Phase {phase_id}: {phase_name} [{status}]")
        lines.append("")

        # Overall progress
        tasks = phase.get("tasks", [])
        completed = sum(1 for t in tasks if t.get("status") == "completed")
        total = len(tasks)

        lines.append(self.render_progress_bar(completed, total, "Progress"))
        lines.append("")

        # Task breakdown
        if tasks:
            lines.append("Tasks:")
            for i, task in enumerate(tasks, 1):
                status = task.get("status", "pending")
                desc = task.get("description", "Task")

                if status == "completed":
                    icon = "✓"
                elif status == "in_progress":
                    icon = "→"
                else:
                    icon = "⏳"

                lines.append(f"  {icon} {i}. {desc}")

        return "\n".join(lines)


class AgentDashboard:
    """
    Agent activity dashboard visualization

    Shows:
    - Active Worker Ants
    - Current tasks
    - Spawned subagents
    - Resource usage
    """

    def render_colony_status(self, status: Dict[str, Any]) -> str:
        """
        Render colony status dashboard

        Args:
            status: Colony status dictionary

        Returns:
            Formatted dashboard display
        """
        lines = []

        lines.append("╔══════════════════════════════════════════════════════════════╗")
        lines.append("║                     🐜 Colony Activity Dashboard            ║")
        lines.append("╠══════════════════════════════════════════════════════════════╣")
        lines.append("")

        # Worker Ants
        worker_ants = status.get("worker_ants", {})

        if worker_ants:
            lines.append("WORKER ANTS:")
            lines.append("")

            for ant_name, ant_info in worker_ants.items():
                current_task = ant_info.get("current_task", "Idle")
                subagents = ant_info.get("subagents_count", 0)

                # Determine activity indicator
                if current_task and current_task != "Idle":
                    indicator = "🟢 ACTIVE"
                else:
                    indicator = "⚪ IDLE"

                lines.append(f"  {ant_name.upper()} [{indicator}]")
                lines.append(f"    Current: {current_task}")

                if subagents > 0:
                    lines.append(f"    Spawned: {subagents} subagents")

                lines.append("")
        else:
            lines.append("No Worker Ants active")
            lines.append("")

        lines.append("╚══════════════════════════════════════════════════════════════╝")

        return "\n".join(lines)

    def render_agent_table(self, agents: List[Dict[str, Any]]) -> str:
        """
        Render agents as a table

        Args:
            agents: List of agent dictionaries

        Returns:
            Formatted table
        """
        if not agents:
            return "No agents to display"

        lines = []

        # Header
        lines.append("┌─────────────────────┬──────────────┬─────────────┬──────────┐")
        lines.append("│ Agent               │ Status       │ Task        │ Subagents│")
        lines.append("├─────────────────────┼──────────────┼─────────────┼──────────┤")

        # Rows
        for agent in agents:
            name = agent.get("name", "Unknown")[:20]
            status = agent.get("status", "Unknown")[:13]
            task = agent.get("current_task", "Idle")[:12]
            subagents = str(agent.get("subagents_count", 0))

            lines.append(f"│ {name:<20} │ {status:<13} │ {task:<12} │ {subagents:>8} │")

        # Footer
        lines.append("└─────────────────────┴──────────────┴─────────────┴──────────┘")

        return "\n".join(lines)


class PheromoneVisualizer:
    """
    Pheromone signal visualization

    Shows:
    - Active pheromone signals
    - Signal strength
    - Signal type
    - Decay over time
    """

    def render_pheromones(self, signals: List[Dict[str, Any]]) -> str:
        """
        Render active pheromone signals

        Args:
            signals: List of pheromone signal dictionaries

        Returns:
            Formatted pheromone display
        """
        if not signals:
            return "No active pheromone signals"

        lines = []

        lines.append("╔══════════════════════════════════════════════════════════════╗")
        lines.append("║                     🌸 Active Pheromones                  ║")
        lines.append("╠══════════════════════════════════════════════════════════════╣")
        lines.append("")

        for signal in signals:
            signal_type = signal.get("type", "UNKNOWN").upper()
            content = signal.get("content", "")
            strength = signal.get("strength", 0.5)

            # Type icons
            type_icons = {
                "INIT": "🎯",
                "FOCUS": "🔍",
                "REDIRECT": "🚫",
                "FEEDBACK": "💬"
            }
            icon = type_icons.get(signal_type, "🌸")

            # Strength bar
            strength_pct = int(strength * 100)
            strength_bar = "█" * int(strength * 10) + "░" * (10 - int(strength * 10))

            lines.append(f"{icon} [{signal_type}] {strength_pct}%")
            lines.append(f"   {content[:60]}")
            lines.append(f"   Strength: [{strength_bar}]")
            lines.append("")

        lines.append("╚══════════════════════════════════════════════════════════════╝")

        return "\n".join(lines)

    def render_signal_strength(self, strength: float) -> str:
        """
        Render signal strength as a bar

        Args:
            strength: Signal strength 0.0-1.0

        Returns:
            Strength bar string
        """
        clamped = max(0.0, min(1.0, strength))
        filled = int(clamped * 20)
        bar = "█" * filled + "░" * (20 - filled)

        return f"[{bar}] {int(clamped * 100)}%"


class ErrorDisplay:
    """
    Error pattern and ledger visualization

    Shows:
    - Error ledger entries
    - Flagged issues
    - Pattern occurrences
    """

    def render_error_ledger(self, errors: List[Dict[str, Any]]) -> str:
        """
        Render error ledger entries

        Args:
            errors: List of error records

        Returns:
            Formatted error display
        """
        if not errors:
            return "✅ No errors recorded"

        lines = []

        lines.append("╔══════════════════════════════════════════════════════════════╗")
        lines.append("║                      📋 Error Ledger                       ║")
        lines.append("╠══════════════════════════════════════════════════════════════╣")
        lines.append("")

        # Group by category
        by_category: Dict[str, List[Dict]] = {}
        for error in errors:
            category = error.get("category", "other")
            if category not in by_category:
                by_category[category] = []
            by_category[category].append(error)

        # Display by category
        for category, category_errors in by_category.items():
            count = len(category_errors)
            flag_status = "⚠️ FLAGGED" if count >= 3 else f"{count} occurrence(s)"

            lines.append(f"📌 {category.upper()}: {flag_status}")
            lines.append("")

            for error in category_errors[-3:]:  # Show last 3
                symptom = error.get("symptom", "Unknown error")[:60]
                lines.append(f"   • {symptom}")

            lines.append("")

        lines.append("╚══════════════════════════════════════════════════════════════╝")

        return "\n".join(lines)

    def render_flagged_issues(self, issues: List[Dict[str, Any]]) -> str:
        """
        Render flagged issues (3+ occurrences)

        Args:
            issues: List of flagged issue dictionaries

        Returns:
            Formatted flagged issues display
        """
        if not issues:
            return "✅ No flagged issues"

        lines = []

        lines.append("╔══════════════════════════════════════════════════════════════╗")
        lines.append("║                 ⚠️  Flagged Issues (3+ occurrences)        ║")
        lines.append("╠══════════════════════════════════════════════════════════════╣")
        lines.append("")

        for issue in issues:
            category = issue.get("category", "Unknown").upper()
            count = issue.get("count", 0)
            pattern = issue.get("pattern", "")[:60]

            lines.append(f"🚨 {category}: {count} occurrences")
            lines.append(f"   Pattern: {pattern}")
            lines.append(f"   Prevention: {issue.get('prevention', 'See error ledger')[:60]}")
            lines.append("")

        lines.append("╚══════════════════════════════════════════════════════════════╝")

        return "\n".join(lines)


class PhaseSummaryVisualizer:
    """
    Phase completion summary visualization

    Shows:
    - Phase overview
    - Completion stats
    - Key learnings
    - Issues resolved
    """

    def render_phase_summary(self, phase: Dict[str, Any]) -> str:
        """
        Render phase completion summary

        Args:
            phase: Phase dictionary

        Returns:
            Formatted summary display
        """
        lines = []

        # Header
        phase_id = phase.get("id", "?")
        phase_name = phase.get("name", "Unknown")

        lines.append("╔══════════════════════════════════════════════════════════════╗")
        lines.append(f"║           Phase {phase_id} Complete: {phase_name:<30} ║")
        lines.append("╠══════════════════════════════════════════════════════════════╣")
        lines.append("")

        # Stats
        tasks = phase.get("tasks", [])
        completed = sum(1 for t in tasks if t.get("status") == "completed")
        total = len(tasks)

        lines.append(f"✓ Tasks Completed: {completed}/{total}")

        if phase.get("duration"):
            lines.append(f"⏱️  Duration: {phase['duration']}")

        if phase.get("agents_spawned"):
            lines.append(f"🐜 Agents Spawned: {phase['agents_spawned']}")

        if phase.get("messages_exchanged"):
            lines.append(f"💬 Messages: {phase['messages_exchanged']}")

        lines.append("")

        # Key learnings
        learnings = phase.get("key_learnings", [])
        if learnings:
            lines.append("💡 KEY LEARNINGS:")
            for learning in learnings:
                lines.append(f"   • {learning}")
            lines.append("")

        # Issues resolved
        issues = phase.get("issues_found", [])
        if issues:
            lines.append(f"🔧 ISSUES RESOLVED: {len(issues)}")
            for issue in issues[:5]:
                lines.append(f"   • {issue.get('description', 'Issue')}")
            lines.append("")

        lines.append("╚══════════════════════════════════════════════════════════════╝")

        return "\n".join(lines)


class MemoryVisualizer:
    """
    Memory system visualization

    Shows:
    - All three memory layers
    - Token usage and compression stats
    - Memory contents
    """

    def render_memory_status(self, memory_status: Dict[str, Any]) -> str:
        """
        Render complete memory system status

        Args:
            memory_status: Status dictionary from TripleLayerMemory.get_status()

        Returns:
            Formatted memory status display
        """
        lines = []

        lines.append("╔══════════════════════════════════════════════════════════════╗")
        lines.append("║                  🧠 Memory System Status                    ║")
        lines.append("╠══════════════════════════════════════════════════════════════╣")
        lines.append("")

        tlm = memory_status.get("triple_layer_memory", {})

        # Working Memory
        working = tlm.get("working", {})
        lines.append("📝 WORKING MEMORY (Immediate Context)")
        lines.append(f"   Items: {working.get('item_count', 0)}")
        lines.append(f"   Tokens: {working.get('used_tokens', 0)}/{working.get('max_tokens', 200000)} "
                    f"({working.get('usage_percent', 0):.1f}%)")

        # Progress bar for token usage
        usage_percent = working.get('usage_percent', 0) / 100
        bar_width = 30
        filled = int(bar_width * usage_percent)
        bar = "█" * filled + "░" * (bar_width - filled)
        lines.append(f"   Usage: [{bar}]")
        lines.append("")

        # Short-Term Memory
        short_term = tlm.get("short_term", {})
        lines.append("📚 SHORT-TERM MEMORY (Compressed Sessions)")
        lines.append(f"   Sessions: {short_term.get('session_count', 0)}/{short_term.get('max_sessions', 10)}")
        lines.append(f"   Saved tokens: {short_term.get('total_saved_tokens', 0)}")
        lines.append(f"   Avg compression: {short_term.get('avg_compression_ratio', 0):.2f}x")

        # Session progress
        session_count = short_term.get('session_count', 0)
        max_sessions = short_term.get('max_sessions', 10)
        filled = int(bar_width * min(session_count / max_sessions, 1))
        bar = "█" * filled + "░" * (bar_width - filled)
        lines.append(f"   Capacity: [{bar}] {session_count}/{max_sessions}")
        lines.append("")

        # Long-Term Memory
        long_term = tlm.get("long_term", {})
        lines.append("💾 LONG-TERM MEMORY (Persistent Knowledge)")
        lines.append(f"   Total patterns: {long_term.get('total_patterns', 0)}")

        categories = long_term.get('categories', {})
        if categories:
            lines.append(f"   Categories:")
            for cat, count in categories.items():
                if count > 0:
                    lines.append(f"      {cat}: {count}")

        lines.append("")

        lines.append("╚══════════════════════════════════════════════════════════════╝")

        return "\n".join(lines)

    def render_working_memory(self, working_memory: Dict[str, Any]) -> str:
        """
        Render working memory contents

        Args:
            working_memory: Working memory dict

        Returns:
            Formatted working memory display
        """
        lines = []

        lines.append("╔══════════════════════════════════════════════════════════════╗")
        lines.append("║                    📝 Working Memory Contents               ║")
        lines.append("╠══════════════════════════════════════════════════════════════╣")
        lines.append("")

        items = working_memory.get('items', [])
        if not items:
            lines.append("  (empty)")
        else:
            for item in items[:20]:  # Show first 20
                item_type = item.get('metadata', {}).get('type', 'general')
                content = item.get('content', '')[:80]
                lines.append(f"  [{item_type}] {content}...")

            if len(items) > 20:
                lines.append(f"  ... and {len(items) - 20} more items")

        lines.append("")
        lines.append("╚══════════════════════════════════════════════════════════════╝")

        return "\n".join(lines)

    def render_short_term_sessions(self, sessions: List[Dict[str, Any]]) -> str:
        """
        Render short-term memory sessions

        Args:
            sessions: List of session summaries

        Returns:
            Formatted sessions display
        """
        lines = []

        lines.append("╔══════════════════════════════════════════════════════════════╗")
        lines.append("║                  📚 Short-Term Memory Sessions              ║")
        lines.append("╠══════════════════════════════════════════════════════════════╣")
        lines.append("")

        if not sessions:
            lines.append("  (no sessions)")
        else:
            for session in sessions:
                lines.append(f"📦 {session.get('session_id', 'unknown')}")
                lines.append(f"   Compression: {session.get('compression_ratio', 0):.2f}x "
                            f"({session.get('original_tokens', 0)} → {session.get('compressed_tokens', 0)} tokens)")
                lines.append(f"   Age: {session.get('age_hours', 0):.1f} hours")
                lines.append(f"   Content: {session.get('content', '')[:100]}...")
                lines.append("")

        lines.append("╚══════════════════════════════════════════════════════════════╝")

        return "\n".join(lines)

    def render_long_term_patterns(self, patterns: List[Dict[str, Any]]) -> str:
        """
        Render long-term memory patterns

        Args:
            patterns: List of knowledge patterns

        Returns:
            Formatted patterns display
        """
        lines = []

        lines.append("╔══════════════════════════════════════════════════════════════╗")
        lines.append("║                 💾 Long-Term Memory Patterns               ║")
        lines.append("╠══════════════════════════════════════════════════════════════╣")
        lines.append("")

        if not patterns:
            lines.append("  (no patterns)")
        else:
            # Group by category
            by_category: Dict[str, List[Dict]] = {}
            for pattern in patterns:
                cat = pattern.get('category', 'other')
                if cat not in by_category:
                    by_category[cat] = []
                by_category[cat].append(pattern)

            for category, cat_patterns in by_category.items():
                lines.append(f"📁 {category.upper()}: {len(cat_patterns)} patterns")
                for pattern in cat_patterns[:5]:  # Show first 5 per category
                    confidence = pattern.get('confidence', 0)
                    occurrences = pattern.get('occurrences', 1)
                    value = pattern.get('value', '')[:80]
                    lines.append(f"   [{confidence:.2f}★ x{occurrences}] {value}...")
                if len(cat_patterns) > 5:
                    lines.append(f"   ... and {len(cat_patterns) - 5} more")
                lines.append("")

        lines.append("╚══════════════════════════════════════════════════════════════╝")

        return "\n".join(lines)


# Convenience functions for quick rendering

def render_progress(completed: int, total: int, label: str = "") -> str:
    """Quick progress bar rendering"""
    viz = ProgressVisualizer()
    return viz.render_progress_bar(completed, total, label)


def render_colony_dashboard(status: Dict[str, Any]) -> str:
    """Quick colony status rendering"""
    dashboard = AgentDashboard()
    return dashboard.render_colony_status(status)


def render_pheromones(signals: List[Dict[str, Any]]) -> str:
    """Quick pheromone rendering"""
    viz = PheromoneVisualizer()
    return viz.render_pheromones(signals)


def render_errors(errors: List[Dict[str, Any]]) -> str:
    """Quick error ledger rendering"""
    display = ErrorDisplay()
    return display.render_error_ledger(errors)


def render_memory_status(memory_status: Dict[str, Any]) -> str:
    """Quick memory status rendering"""
    viz = MemoryVisualizer()
    return viz.render_memory_status(memory_status)


def render_working_memory(working_memory: Dict[str, Any]) -> str:
    """Quick working memory rendering"""
    viz = MemoryVisualizer()
    return viz.render_working_memory(working_memory)


def render_short_term_sessions(sessions: List[Dict[str, Any]]) -> str:
    """Quick short-term sessions rendering"""
    viz = MemoryVisualizer()
    return viz.render_short_term_sessions(sessions)


def render_long_term_patterns(patterns: List[Dict[str, Any]]) -> str:
    """Quick long-term patterns rendering"""
    viz = MemoryVisualizer()
    return viz.render_long_term_patterns(patterns)
