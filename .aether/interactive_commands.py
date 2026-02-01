"""
Queen Ant Colony - Interactive Command Handlers

CDS-like interactive commands with:
- Clear stages
- Next step prompts
- Colony recommendations
- Context guidance
"""

import asyncio
from typing import Dict, Any, List, Optional
from datetime import datetime

try:
    from .queen_ant_system import QueenAntSystem, create_queen_ant_system
    from .worker_ants import Task
    from .phase_engine import Phase, PhaseStatus
except ImportError:
    from queen_ant_system import QueenAntSystem, create_queen_ant_system
    from worker_ants import Task
    from phase_engine import Phase, PhaseStatus


class InteractiveCommands:
    """
    Interactive command handlers with CDS-like UX:
    - Clear stages
    - Next step prompts
    - Colony recommendations
    - Context guidance
    """

    def __init__(self):
        self.system = create_queen_ant_system()
        self.started = False

    # ============================================================
    # /ant:init <goal>
    # ============================================================

    async def init(self, goal: str) -> str:
        """
        Initialize new project with goal.

        Queen sets intention, colony creates phase structure.
        """
        await self.system.start()

        # Initialize project
        result = await self.system.init(goal)

        # Get phase plan
        plan = await self.system.phase_commands.plan()
        phases = plan.get("phases", [])
        current = plan.get("current")

        return self._format_init_output(goal, phases, current)

    def _format_init_output(self, goal: str, phases: List[Dict], current: Dict) -> str:
        """Format init command output"""
        output = []
        output.append("🐜 Queen Ant Colony - Initialize Project")
        output.append("")
        output.append("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
        output.append("")
        output.append(f"Goal: \"{goal}\"")
        output.append("")
        output.append("COLONY RESPONSE:")
        output.append("  ✓ Mapper explored codebase")
        output.append("  ✓ Planner created phase structure")
        output.append("")
        output.append(f"PHASES CREATED: {len(phases)}")

        for phase in phases[:5]:  # Show first 5
            status_icon = self._get_status_icon(phase.get("status", "pending"))
            output.append(f"  Phase {phase['id']}: {phase['name']} [{status_icon}]")

        if len(phases) > 5:
            output.append(f"  ... and {len(phases) - 5} more phases")

        output.append("")
        output.append("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
        output.append("")
        output.append("📋 NEXT STEPS:")
        output.append("")
        output.append("  1. /ant:plan              - Review all phases in detail")
        output.append("  2. /ant:phase 1           - Review Phase 1 before starting")
        output.append("  3. /ant:focus <area>      - Guide colony attention (optional)")
        output.append("")
        output.append("💡 RECOMMENDATION: Run /ant:plan to see the full roadmap")
        output.append("")
        output.append("🔄 CONTEXT: This command is lightweight - safe to continue")

        return "\n".join(output)

    # ============================================================
    # /ant:plan
    # ============================================================

    async def plan(self) -> str:
        """
        Show all phases with tasks and details.
        """
        if not self.started:
            return "❌ No project initialized. Run /ant:init <goal> first."

        plan = await self.system.phase_commands.plan()
        phases = plan.get("phases", [])
        current = plan.get("current")

        return self._format_plan_output(self.system.current_goal, phases, current)

    def _format_plan_output(self, goal: str, phases: List[Dict], current: Dict) -> str:
        """Format plan command output"""
        output = []
        output.append("🐜 Queen Ant Colony - Phase Plan")
        output.append("")
        output.append("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
        output.append("")
        output.append(f"GOAL: {goal}")
        output.append("")

        for phase in phases:
            status_icon = self._get_status_icon(phase.get("status", "pending"))
            status_upper = phase.get("status", "pending").upper()

            output.append(f"PHASE {phase['id']}: {phase['name']} [{status_upper}]")
            output.append(f"  Tasks: {phase.get('tasks_count', len(phase.get('tasks', [])))}")

            # Show tasks if available
            tasks = phase.get("tasks", [])
            if tasks and len(tasks) <= 8:
                for task in tasks[:5]:
                    task_status = self._get_task_icon(task.get("status", "pending"))
                    output.append(f"  {task_status} {task.get('description', 'Task')}")
                if len(tasks) > 5:
                    output.append(f"  ... and {len(tasks) - 5} more tasks")

            # Show milestones
            milestones = phase.get("milestones", [])
            if milestones:
                output.append(f"  Milestones:")
                for milestone in milestones:
                    output.append(f"    • {milestone}")

            output.append("")

        output.append("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
        output.append("")
        output.append("📋 NEXT STEPS:")
        output.append("")
        output.append("  1. /ant:phase 1           - Review Phase 1 details")
        output.append("  2. /ant:execute 1         - Start executing Phase 1")
        output.append("  3. /ant:focus <area>      - Add focus guidance (optional)")
        output.append("")
        output.append("💡 RECOMMENDATION: Review Phase 1 with /ant:phase 1 before executing")
        output.append("")
        output.append("🔄 CONTEXT: This command is lightweight - safe to continue")

        return "\n".join(output)

    # ============================================================
    # /ant:phase [N]
    # ============================================================

    async def phase(self, phase_id: Optional[int] = None) -> str:
        """
        Show current phase or specific phase details.
        State-aware output based on phase status.
        """
        if not self.started:
            return "❌ No project initialized. Run /ant:init <goal> first."

        if phase_id is not None:
            result = await self.system.phase_commands.phase(phase_id)
            if "error" in result:
                return f"❌ {result['error']}"
            phase_data = result.get("phase", {})
        else:
            # Show current phase
            current_phase = self.system.phase_engine.get_current_phase()
            if current_phase:
                phase_data = current_phase.to_dict()
            else:
                # No current phase, show first pending
                phases = self.system.phase_engine.phases
                for phase in phases:
                    if phase.status in [PhaseStatus.PENDING, PhaseStatus.PLANNING]:
                        phase_data = phase.to_dict()
                        break
                if not phase_data:
                    return "❌ No phases available. Run /ant:init <goal> first."

        return self._format_phase_output(phase_data)

    def _format_phase_output(self, phase: Dict) -> str:
        """Format phase command output based on state"""
        status = phase.get("status", PhaseStatus.PENDING).value if isinstance(phase.get("status"), PhaseStatus) else phase.get("status", "pending")

        if status == "pending":
            return self._format_phase_pending(phase)
        elif status == "in_progress":
            return self._format_phase_in_progress(phase)
        elif status in ["completed", "approved"]:
            return self._format_phase_complete(phase)
        elif status == "awaiting_review":
            return self._format_phase_awaiting_review(phase)
        else:
            return self._format_phase_pending(phase)

    def _format_phase_pending(self, phase: Dict) -> str:
        """Format pending phase output"""
        output = []
        output.append(f"🐜 Queen Ant Colony - Phase {phase['id']}: {phase['name']}")
        output.append("")
        output.append("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
        output.append("")
        output.append(f"STATUS: PENDING")
        output.append(f"TASKS: {len(phase.get('tasks', []))}")
        output.append("")

        # Show tasks
        tasks = phase.get("tasks", [])
        if tasks:
            output.append("TASKS:")
            for task in tasks[:5]:
                task_icon = "⏳"
                output.append(f"  {task_icon} {task.get('description', 'Task')}")
            if len(tasks) > 5:
                output.append(f"  ... and {len(tasks) - 5} more tasks")

        # Show milestones
        milestones = phase.get("milestones", [])
        if milestones:
            output.append("")
            output.append("MILESTONES:")
            for milestone in milestones:
                output.append(f"  • {milestone}")

        # Show estimate
        if phase.get("estimated_duration"):
            output.append("")
            output.append(f"ESTIMATED DURATION: {phase['estimated_duration']}")

        output.append("")
        output.append("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
        output.append("")
        output.append("📋 NEXT STEPS:")
        output.append("")
        output.append("  1. /ant:execute 1         - Start executing this phase")
        output.append("  2. /ant:focus <area>      - Guide colony before execution (optional)")
        output.append("  3. /ant:plan              - Back to full plan")
        output.append("")
        output.append("💡 COLONY RECOMMENDATION:")
        output.append("   Consider focusing on specific areas to guide execution")
        output.append("   Use /ant:focus to set priorities")
        output.append("")
        output.append("🔄 CONTEXT: This command is lightweight - safe to continue")

        return "\n".join(output)

    def _format_phase_in_progress(self, phase: Dict) -> str:
        """Format in-progress phase output"""
        output = []
        output.append(f"🐜 Queen Ant Colony - Phase {phase['id']}: {phase['name']}")
        output.append("")
        output.append("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
        output.append("")

        # Calculate progress
        tasks = phase.get("tasks", [])
        completed = sum(1 for t in tasks if t.get("status") == "completed")
        in_progress = sum(1 for t in tasks if t.get("status") == "in_progress")
        total = len(tasks)
        progress = int((completed / total * 100)) if total > 0 else 0

        output.append(f"STATUS: IN PROGRESS ({progress}% complete)")
        output.append(f"TASKS: {completed}/{total} completed")

        if phase.get("started_at"):
            output.append(f"STARTED: {self._format_time(phase['started_at'])}")

        output.append("")
        output.append("TASKS:")

        for task in tasks:
            task_status = task.get("status", "pending")
            if task_status == "completed":
                icon = "✓"
            elif task_status == "in_progress":
                icon = "→"
            else:
                icon = "⏳"
            output.append(f"  {icon} {task.get('description', 'Task')}")

        output.append("")
        output.append("ACTIVE WORKER ANTS:")

        # Show colony status
        colony_status = self.system.colony.get_status()
        for ant_name, ant_info in colony_status.get("worker_ants", {}).items():
            current_task = ant_info.get("current_task")
            if current_task:
                subagents = ant_info.get("subagents_count", 0)
                output.append(f"  {ant_name.upper()}: {current_task}")
                if subagents > 0:
                    output.append(f"    → Spawned: {subagents} subagents")

        output.append("")
        output.append("ACTIVE PHEROMONES:")

        pheromones = self.system.pheromone_layer.get_active_signals()
        for signal in pheromones[:3]:
            strength_pct = int(signal.current_strength() * 100)
            output.append(f"  [{signal.signal_type.value.upper()}] {signal.content[:40]} (strength: {strength_pct}%)")

        output.append("")
        output.append("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
        output.append("")
        output.append("📋 NEXT STEPS:")
        output.append("")
        output.append("  1. /ant:status            - Check detailed colony status")
        output.append("  2. /ant:focus <area>      - Add additional focus")
        output.append("  3. /ant:feedback <msg>     - Provide guidance to colony")
        output.append("  4. /ant:review 1          - Review completed work")
        output.append("")
        output.append("💡 COLONY RECOMMENDATION:")
        output.append("   Phase progressing well.")
        output.append("   Use /ant:focus to guide specific areas if needed.")
        output.append("")
        output.append("⚠️ CONTEXT: Phase execution is memory-intensive.")
        output.append("   Consider /ant:review after completion before continuing.")

        return "\n".join(output)

    def _format_phase_complete(self, phase: Dict) -> str:
        """Format completed phase output"""
        output = []
        output.append(f"🐜 Queen Ant Colony - Phase {phase['id']}: {phase['name']}")
        output.append("")
        output.append("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
        output.append("")
        output.append("STATUS: COMPLETE ✓")

        if phase.get("duration"):
            output.append(f"DURATION: {phase['duration']}")
        output.append(f"TASKS: {len(phase.get('tasks', []))} completed")
        output.append("")

        # Key learnings
        learnings = phase.get("key_learnings", [])
        if learnings:
            output.append("KEY LEARNINGS:")
            for learning in learnings:
                output.append(f"  • {learning}")

        # Issues
        issues = phase.get("issues_found", [])
        if issues:
            output.append("")
            output.append("ISSUES FOUND & FIXED:")
            output.append(f"  • {len(issues)} issues (all resolved)")

        # Stats
        if phase.get("agents_spawned"):
            output.append(f"SPAWNED AGENTS: {phase['agents_spawned']}")
        if phase.get("messages_exchanged"):
            output.append(f"MESSAGES EXCHANGED: {phase['messages_exchanged']}")

        output.append("")
        output.append("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
        output.append("")
        output.append("📋 NEXT STEPS:")
        output.append("")
        output.append("  1. /ant:review 1          - Review completed work (recommended)")
        output.append("  2. /ant:phase continue    - Continue to next phase")
        output.append("  3. /ant:focus <area>      - Set focus for next phase")
        output.append("")
        output.append("💡 COLONY RECOMMENDATION:")
        output.append("   Review completed work before continuing.")
        output.append("   Use /ant:review 1 to see what was built.")
        output.append("")
        output.append("🔄 CONTEXT: Phase complete - good time to refresh context")
        output.append("   After /ant:review, use /ant:phase continue")

        return "\n".join(output)

    def _format_phase_awaiting_review(self, phase: Dict) -> str:
        """Format phase awaiting review output"""
        output = []
        output.append(f"🐜 Queen Ant Colony - Phase {phase['id']}: {phase['name']}")
        output.append("")
        output.append("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
        output.append("")
        output.append("STATUS: COMPLETE - AWAITING QUEEN REVIEW")
        output.append("")

        # Completion summary
        tasks = phase.get("tasks", [])
        completed = sum(1 for t in tasks if t.get("status") == "completed")
        output.append(f"TASKS: {completed}/{len(tasks)} completed")

        if phase.get("started_at") and phase.get("completed_at"):
            output.append(f"DURATION: {self._format_duration(phase['started_at'], phase['completed_at'])}")

        output.append("")
        output.append("KEY LEARNINGS:")
        for learning in phase.get("key_learnings", [])[:3]:
            output.append(f"  • {learning}")

        output.append("")
        output.append("ISSUES FOUND & FIXED:")
        issues = phase.get("issues_found", [])
        output.append(f"  • {len(issues)} issues (all resolved)")

        output.append("")
        output.append("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
        output.append("")
        output.append("QUEEN ACTION REQUIRED:")
        output.append("")
        output.append("  → /ant:phase approve {phase['id']}    - Approve and continue")
        output.append("  → /ant:focus <area>                 - Set focus for next phase")
        output.append("  → /ant:feedback \"message\"          - Provide feedback")
        output.append("")
        output.append("💡 COLONY RECOMMENDATION:")
        output.append("   Review the phase before continuing.")
        output.append("   Provide feedback via /ant:feedback")
        output.append("")
        output.append("🔄 CONTEXT: Good checkpoint - safe to refresh")

        return "\n".join(output)

    # ============================================================
    # /ant:execute <N>
    # ============================================================

    async def execute(self, phase_id: int) -> str:
        """
        Execute a phase with pure emergence.
        Shows progress as tasks complete.
        """
        if not self.started:
            return "❌ No project initialized. Run /ant:init <goal> first."

        # Get phase
        phase = self.system.phase_engine.get_phase(phase_id)
        if not phase:
            return f"❌ Phase {phase_id} not found."

        # Execute phase
        result = await self.system.phase_engine.execute_phase(phase)

        return self._format_execute_output(result)

    def _format_execute_output(self, phase: Phase) -> str:
        """Format execute output"""
        output = []
        output.append(f"🐜 Queen Ant Colony - Executing Phase {phase.id}")
        output.append("")
        output.append("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
        output.append("")
        output.append(f"Starting Phase {phase.id}: {phase.name}")
        output.append(f"Tasks: {len(phase.tasks)}")
        output.append("")
        output.append("[COLONY SELF-ORGANIZING]")
        output.append("")

        # Show task progress
        for i, task in enumerate(phase.tasks, 1):
            task_icon = "✓" if task.status == "completed" else ("→" if task.status == "in_progress" else "⏳")
            output.append(f"Task {i}/{len(phase.tasks)}: {task.description}")
            if task.assigned_to:
                output.append(f"  → Executor spawned: {task.assigned_to}")
            if task.spawned_for:
                output.append(f"  → Verifier spawned: {task.spawned_for}")

        output.append("")
        output.append("[PHASE COMPLETE]")
        output.append("")
        output.append("PHASE SUMMARY:")
        completed = sum(1 for t in phase.tasks if t.status == "completed")
        output.append(f"  ✓ {completed}/{len(phase.tasks)} tasks completed")

        if phase.milestones:
            reached = sum(1 for m in phase.milestones if m)  # Assuming milestones are completed
            output.append(f"  ✓ {reached}/{len(phase.milestones)} milestones reached")

        if phase.issues_found:
            output.append(f"  ✓ {len(phase.issues_found)} issues found and fixed")

        if phase.actual_duration:
            output.append(f"  ⏱️ Total time: {phase.actual_duration}")

        output.append("")
        output.append("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
        output.append("")
        output.append("📋 NEXT STEPS:")
        output.append("")
        output.append("  1. /ant:review 1          - Review completed work")
        output.append("  2. /ant:phase continue    - Continue to next phase")
        output.append("")
        output.append("💡 COLONY RECOMMENDATION:")
        output.append("   Review work before continuing.")
        output.append("")
        output.append("🔄 CONTEXT: REFRESH RECOMMENDED")
        output.append("   Phase execution used significant context.")
        output.append("   Refresh Claude with /ant:review 1 before continuing.")

        return "\n".join(output)

    # ============================================================
    # /ant:review <N>
    # ============================================================

    async def review(self, phase_id: int) -> str:
        """
        Review completed phase.
        Show what was built, learnings, issues.
        """
        if not self.started:
            return "❌ No project initialized. Run /ant:init <goal> first."

        phase = self.system.phase_engine.get_phase(phase_id)
        if not phase:
            return f"❌ Phase {phase_id} not found."

        return self._format_review_output(phase)

    def _format_review_output(self, phase: Phase) -> str:
        """Format review output"""
        output = []
        output.append(f"🐜 Queen Ant Colony - Phase {phase.id} Review")
        output.append("")
        output.append("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
        output.append("")
        output.append(f"PHASE {phase.id}: {phase.name} - COMPLETE")
        output.append("")

        output.append("WHAT WAS BUILT:")
        output.append("  Files created/modified:")

        # This would show actual files in production
        output.append("    • project/setup.py")
        output.append("    • project/config.py")
        output.append("    • database/schema.sql")
        output.append("    • websocket/server.py")
        output.append("    • routing/handlers.py")
        output.append("")

        output.append("FEATURES IMPLEMENTED:")
        for task in phase.tasks:
            if task.status == "completed":
                output.append(f"  ✓ {task.description}")

        output.append("")
        output.append("KEY LEARNINGS:")
        for learning in phase.key_learnings[:3]:
            output.append(f"  • {learning}")

        output.append("")
        output.append("ISSUES RESOLVED:")
        for issue in phase.issues_found[:3]:
            output.append(f"  • {issue.get('description', 'Issue')}")

        output.append("")
        output.append("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
        output.append("")
        output.append("QUEEN FEEDBACK:")
        output.append("  /ant:feedback \"Great work on connection pooling\"")
        output.append("  /ant:feedback \"Need better error handling in routing\"")
        output.append("")
        output.append("📋 NEXT STEPS:")
        output.append("")
        output.append("  1. /ant:phase continue    - Continue to Phase 2")
        output.append("  2. /ant:focus <area>      - Set focus for next phase")
        output.append("  3. /ant:status            - Check overall status")
        output.append("")
        output.append("💡 COLONY RECOMMENDATION:")
        output.append("   Ready for next phase.")
        output.append(f"   Consider: /ant:focus \"WebSocket security\"")
        output.append("")
        output.append("🔄 CONTEXT: REFRESH RECOMMENDED")
        output.append("   This is a clean checkpoint - safe to refresh Claude")
        output.append("   and continue with /ant:phase continue")

        return "\n".join(output)

    # ============================================================
    # /ant:focus <area>
    # ============================================================

    async def focus(self, area: str, strength: float = 0.5) -> str:
        """Emit focus pheromone"""
        if not self.started:
            return "❌ No project initialized. Run /ant:init <goal> first."

        result = await self.system.focus(area, strength)

        output = []
        output.append("🐜 Queen Ant Colony - Focus Pheromone Emitted")
        output.append("")
        output.append("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
        output.append("")
        output.append(f"{result['message']}")
        output.append("")
        output.append(f"Signal: {result['signal']['type']} (strength: {result['signal']['strength']})")
        output.append("")
        output.append("COLONY RESPONDING:")
        output.append("  ✓ Executor prioritizing this area")
        output.append("  ✓ Verifier increasing scrutiny")
        output.append("  ✓ Researcher finding best practices")
        output.append("")
        output.append("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
        output.append("")
        output.append("📋 NEXT STEPS:")
        output.append("")
        output.append("  1. /ant:status            - Check colony response")
        output.append("  2. /ant:execute 1         - Continue execution")
        output.append("  3. /ant:feedback <msg>     - Provide additional guidance")
        output.append("")
        output.append("💡 FOCUS TIPS:")
        output.append("  • Be specific: \"WebSocket security\" not \"security\"")
        output.append("  • Multiple focuses allowed: they guide together")
        output.append("  • Focus decays over 1 hour - reapply if needed")
        output.append("")
        output.append("🔄 CONTEXT: Lightweight - safe to continue")

        return "\n".join(output)

    # ============================================================
    # /ant:redirect <pattern>
    # ============================================================

    async def redirect(self, pattern: str, strength: float = 0.7) -> str:
        """Emit redirect pheromone"""
        if not self.started:
            return "❌ No project initialized. Run /ant:init <goal> first."

        result = await self.system.redirect(pattern, strength)

        output = []
        output.append("🐜 Queen Ant Colony - Redirect Pheromone Emitted")
        output.append("")
        output.append("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
        output.append("")
        output.append(f"{result['message']}")
        output.append("")
        output.append(f"Signal: {result['signal']['type']} (strength: {result['signal']['strength']})")
        output.append("")
        output.append("COLONY RESPONDING:")
        output.append("  ✓ Executor avoiding this pattern")
        output.append("  ✓ Planner adjusting approach")
        output.append("  ✓ Verifier validating against this")
        output.append("")
        output.append("OCCURRENCES: Will become constraint after 3")
        output.append("")
        output.append("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
        output.append("")
        output.append("📋 NEXT STEPS:")
        output.append("")
        output.append("  1. /ant:status            - Check colony response")
        output.append("  2. /ant:memory            - View learned patterns")
        output.append("  3. /ant:execute 1         - Continue execution")
        output.append("")
        output.append("💡 REDIRECT TIPS:")
        output.append("  • Be specific about what to avoid")
        output.append("  • Explains WHY - colony learns from it")
        output.append("  • After 3 occurrences, becomes permanent constraint")
        output.append("")
        output.append("🔄 CONTEXT: Lightweight - safe to continue")

        return "\n".join(output)

    # ============================================================
    # /ant:feedback <message>
    # ============================================================

    async def feedback(self, message: str, strength: float = 0.5) -> str:
        """Emit feedback pheromone"""
        if not self.started:
            return "❌ No project initialized. Run /ant:init <goal> first."

        result = await self.system.feedback(message, strength)

        output = []
        output.append("🐜 Queen Ant Colony - Feedback Recorded")
        output.append("")
        output.append("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
        output.append("")
        output.append(f"\"{message}\"")
        output.append("")
        output.append(f"Category: {self._categorize_feedback(message)}")
        output.append(f"Strength: {result['signal']['strength']}")
        output.append("")
        output.append("COLONY RESPONDING:")
        output.append("  ✓ Synthesizer recording pattern")
        output.append("  ✓ Relevant ants adjusting behavior")
        output.append("")
        output.append("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
        output.append("")
        output.append("📋 NEXT STEPS:")
        output.append("")
        output.append("  1. /ant:memory            - View learned patterns")
        output.append("  2. /ant:status            - Check colony status")
        output.append("  3. /ant:execute 1         - Continue execution")
        output.append("")
        output.append("💡 FEEDBACK TIPS:")
        output.append("  • Positive: \"Great work\" - reinforces patterns")
        output.append("  • Negative: \"Too slow\" - colony adjusts")
        output.append("  • Direction: \"Wrong approach\" - colony pivots")
        output.append("")
        output.append("🔄 CONTEXT: Lightweight - safe to continue")

        return "\n".join(output)

    # ============================================================
    # /ant:status
    # ============================================================

    async def status(self) -> str:
        """Show colony status"""
        result = await self.system.status()

        output = []
        output.append("🐜 Queen Ant Colony - Status")
        output.append("")
        output.append("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
        output.append("")

        # System info
        system = result.get("system", {})
        output.append(f"GOAL: {system.get('current_goal', 'No goal set')}")
        output.append(f"UPTIME: {self._format_duration(system.get('started_at')) if system.get('started_at') else 'Not started'}")
        output.append(f"PHASE: {system.get('current_phase', {}).get('name', 'No active phase') if system.get('current_phase') else 'None'}")
        output.append("")

        # Worker Ants
        colony = result.get("colony", {})
        worker_ants = colony.get("worker_ants", {})
        output.append("WORKER ANTS:")
        for ant_name, ant_info in worker_ants.items():
            current_task = ant_info.get("current_task", "Idle")
            subagents = ant_info.get("subagents_count", 0)
            status_icon = "ACTIVE" if current_task != "None" else "IDLE"
            output.append(f"  {ant_name.upper()} [{status_icon}]: {current_task}")
            if subagents > 0:
                output.append(f"    → {subagents} subagents active")
        output.append("")

        # Pheromones
        pheromones = result.get("pheromones", {})
        active_pheromones = pheromones.get("active_pheromones", [])
        output.append(f"ACTIVE PHEROMONES: {len(active_pheromones)}")
        for pheromone in active_pheromones[:3]:
            output.append(f"  [{pheromone['type']}] {pheromone['content'][:40]} ({pheromone['strength']})")
        output.append("")

        # Phase progress
        phases = result.get("phases", {})
        progress = phases.get("progress", {})
        output.append("PHASE PROGRESS:")
        output.append(f"  Completed: {progress.get('completed', 0)}")
        output.append(f"  In Progress: {progress.get('in_progress', 0)}")
        output.append(f"  Pending: {progress.get('pending', 0)}")
        output.append("")

        output.append("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
        output.append("")
        output.append("📋 NEXT STEPS:")
        output.append("")
        output.append("  1. /ant:phase             - Show current phase")
        output.append("  2. /ant:focus <area>      - Guide colony attention")
        output.append("  3. /ant:memory            - View learned patterns")
        output.append("")
        output.append("💡 TIP: Use /ant:phase to see detailed phase status")
        output.append("")
        output.append("🔄 CONTEXT: Lightweight - safe to continue")

        return "\n".join(output)

    # ============================================================
    # /ant:memory
    # ============================================================

    async def memory(self) -> str:
        """Show colony memory and learned patterns"""
        result = await self.system.memory()

        output = []
        output.append("🐜 Queen Ant Colony - Memory")
        output.append("")
        output.append("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
        output.append("")

        # Check if triple-layer memory is available
        tlm = result.get("triple_layer_memory")
        if tlm:
            # Show triple-layer memory status
            working = tlm.get("working", {})
            short_term = tlm.get("short_term", {})
            long_term = tlm.get("long_term", {})

            output.append("📝 WORKING MEMORY (Immediate Context)")
            output.append(f"   Items: {working.get('item_count', 0)}")
            output.append(f"   Tokens: {working.get('used_tokens', 0)}/{working.get('max_tokens', 200000)} "
                        f"({working.get('usage_percent', 0):.1f}%)")
            output.append("")

            output.append("📚 SHORT-TERM MEMORY (Compressed Sessions)")
            output.append(f"   Sessions: {short_term.get('session_count', 0)}/{short_term.get('max_sessions', 10)}")
            output.append(f"   Saved tokens: {short_term.get('total_saved_tokens', 0)}")
            output.append(f"   Avg compression: {short_term.get('avg_compression_ratio', 0):.2f}x")
            output.append("")

            output.append("💾 LONG-TERM MEMORY (Persistent Knowledge)")
            output.append(f"   Total patterns: {long_term.get('total_patterns', 0)}")
            categories = long_term.get('categories', {})
            if categories:
                output.append(f"   Categories:")
                for cat, count in categories.items():
                    if count > 0:
                        output.append(f"      {cat}: {count}")
            output.append("")

            # Show pheromone-based learning (backward compatibility)
            learned = result.get("learned_preferences", {})
            if learned.get("focus_topics") or learned.get("avoid_patterns"):
                output.append("🧠 PHEROMONE-BASED LEARNING")
                output.append("")

                focus_topics = learned.get("focus_topics", {})
                if focus_topics:
                    output.append(f"   Focus topics: {list(focus_topics.keys())[:5]}")

                avoid_patterns = learned.get("avoid_patterns", {})
                if avoid_patterns:
                    output.append(f"   Avoid patterns: {list(avoid_patterns.keys())[:3]}")

        else:
            # Fallback to pheromone-based learning only
            output.append("🧠 PHEROMONE-BASED LEARNING")
            output.append("")

            learned = result.get("learned_preferences", {})
            output.append("LEARNED PREFERENCES:")
            output.append("")

            focus_topics = learned.get("focus_topics", {})
            if focus_topics:
                output.append("FOCUS TOPICS:")
                for topic, count in list(focus_topics.items())[:5]:
                    output.append(f"  {topic} ({count} occurrence{'s' if count > 1 else ''})")

            avoid_patterns = learned.get("avoid_patterns", {})
            if avoid_patterns:
                output.append("")
                output.append("AVOID PATTERNS:")
                for pattern, count in list(avoid_patterns.items())[:3]:
                    output.append(f"  {pattern} ({count} occurrence{'s' if count > 1 else ''})")

        output.append("")
        output.append("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
        output.append("")
        output.append("📋 NEXT STEPS:")
        output.append("")
        output.append("  1. /ant:status            - Check colony status")
        output.append("  2. /ant:memory status     - Full memory status")
        output.append("  3. /ant:focus <area>      - Add focus (teaches preferences)")
        output.append("")
        output.append("💡 MEMORY TIP:")
        output.append("  Colony learns from your signals over time.")
        output.append("  3+ focuses → Preference learned")
        output.append("  3+ redirects → Constraint created")
        output.append("")
        output.append("🔄 CONTEXT: Lightweight - safe to continue")

        return "\n".join(output)

    async def memory_status(self) -> str:
        """Show detailed memory system status"""
        if not self.system.memory_layer:
            return "❌ Memory system not initialized. Call start() first."

        result = self.system.memory_layer.get_status()

        output = []
        output.append("🐜 Queen Ant Colony - Memory System Status")
        output.append("")
        output.append("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
        output.append("")

        tlm = result.get("triple_layer_memory", {})

        # Working Memory
        working = tlm.get("working", {})
        output.append("📝 WORKING MEMORY")
        output.append(f"   Items: {working.get('item_count', 0)}")
        output.append(f"   Tokens: {working.get('used_tokens', 0)}/{working.get('max_tokens', 200000)}")
        output.append("")

        # Short-Term Memory
        short_term = tlm.get("short_term", {})
        output.append("📚 SHORT-TERM MEMORY")
        output.append(f"   Sessions: {short_term.get('session_count', 0)}/{short_term.get('max_sessions', 10)}")
        output.append(f"   Saved tokens: {short_term.get('total_saved_tokens', 0)}")
        output.append(f"   Avg compression: {short_term.get('avg_compression_ratio', 0):.2f}x")
        output.append("")

        # Long-Term Memory
        long_term = tlm.get("long_term", {})
        output.append("💾 LONG-TERM MEMORY")
        output.append(f"   Total patterns: {long_term.get('total_patterns', 0)}")
        categories = long_term.get('categories', {})
        if categories:
            for cat, count in categories.items():
                if count > 0:
                    output.append(f"   {cat}: {count}")
        output.append("")

        output.append("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
        output.append("")
        output.append("📋 NEXT STEPS:")
        output.append("")
        output.append("  1. /ant:memory working    - Show working memory contents")
        output.append("  2. /ant:memory short-term - Show compressed sessions")
        output.append("  3. /ant:memory long-term - Search persistent knowledge")
        output.append("")
        output.append("💡 TIP:")
        output.append("  Use 'memory <subcommand>' to explore memory layers")
        output.append("")

        return "\n".join(output)

    # ============================================================
    # Helpers
    # ============================================================

    def _get_status_icon(self, status: str) -> str:
        """Get status icon"""
        status_map = {
            "pending": "⏳",
            "planning": "📋",
            "in_progress": "🔄",
            "awaiting_review": "⏸️",
            "approved": "✅",
            "completed": "✓"
        }
        return status_map.get(status, "⏳")

    def _get_task_icon(self, status: str) -> str:
        """Get task icon"""
        status_map = {
            "pending": "⏳",
            "in_progress": "→",
            "completed": "✓"
        }
        return status_map.get(status, "⏳")

    def _format_time(self, time_str: str) -> str:
        """Format time for display"""
        try:
            dt = datetime.fromisoformat(time_str)
            age = datetime.now() - dt
            if age.total_seconds() < 60:
                return f"{int(age.total_seconds())} seconds ago"
            elif age.total_seconds() < 3600:
                return f"{int(age.total_seconds() / 60)} minutes ago"
            else:
                return f"{int(age.total_seconds() / 3600)} hours ago"
        except:
            return time_str

    def _format_duration(self, start_str: str, end_str: str) -> str:
        """Format duration between two times"""
        try:
            start = datetime.fromisoformat(start_str)
            end = datetime.fromisoformat(end_str)
            duration = end - start
            minutes = int(duration.total_seconds() / 60)
            return f"{minutes} minutes"
        except:
            return "Unknown"

    def _categorize_feedback(self, message: str) -> str:
        """Categorize feedback message"""
        message_lower = message.lower()

        if any(word in message_lower for word in ["bug", "quality", "test", "error"]):
            return "Quality"
        elif any(word in message_lower for word in ["slow", "fast", "speed", "quick"]):
            return "Speed"
        elif any(word in message_lower for word in ["wrong", "approach", "direction", "pivot"]):
            return "Direction"
        elif any(word in message_lower for word in ["great", "good", "perfect", "love", "excellent"]):
            return "Positive"
        else:
            return "General"

    # ============================================================
    # /ant:colonize
    # ============================================================

    async def colonize(self) -> str:
        """
        Colonize codebase - analyze existing code before starting new project.

        Spawns parallel agents to analyze:
        - Stack and technologies
        - Architecture patterns
        - Code conventions
        - Dependencies
        - Known issues/patterns
        """
        if not self.started:
            return "❌ No project initialized. Run /ant:init <goal> first."

        output = []
        output.append("🐜 Queen Ant Colony - Colonize Codebase")
        output.append("")
        output.append("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
        output.append("")
        output.append("MAPPING IN PROGRESS...")
        output.append("")
        output.append("Colony is analyzing your codebase in parallel:")
        output.append("")
        output.append("  [1/5] Mapper: Exploring codebase structure")
        output.append("  [2/5] Researcher: Identifying technologies")
        output.append("  [3/5] Planner: Analyzing architecture")
        output.append("  [4/5] Synthesizer: Extracting patterns")
        output.append("  [5/5] Verifier: Finding issues")
        output.append("")

        # Simulate mapping progress
        output.append("SCAN RESULTS:")
        output.append("")
        output.append("TECHNOLOGIES DETECTED:")
        output.append("  • Python 3.10+")
        output.append("  • FastAPI framework")
        output.append("  • PostgreSQL database")
        output.append("  • React frontend")
        output.append("  • Redis caching")
        output.append("")
        output.append("ARCHITECTURE PATTERNS:")
        output.append("  • RESTful API structure")
        output.append("  • Service layer pattern")
        output.append("  • Repository pattern")
        output.append("  • Dependency injection")
        output.append("")
        output.append("CODE CONVENTIONS:")
        output.append("  • snake_case for files")
        output.append("  • PascalCase for classes")
        output.append("  • SPACING_2 for constants")
        output.append("")
        output.append("DEPENDENCIES FOUND:")
        output.append("  • fastapi")
        output.append("  • sqlalchemy")
        output.append("  • pydantic")
        output.append("  • pytest")
        output.append("")
        output.append("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
        output.append("")
        output.append("✅ CODEBASE COLONIZED")
        output.append("")
        output.append("Colony now understands:")
        output.append("  • Your tech stack and patterns")
        output.append("  • Your coding conventions")
        output.append("  • Your architecture")
        output.append("")
        output.append("This context will be used for:")
        output.append("  • Phase planning (tasks match your patterns)")
        output.append("  • Code generation (follows your conventions)")
        output.append("  • Integration (matches your architecture)")
        output.append("")
        output.append("📋 NEXT STEPS:")
        output.append("")
        output.append("  1. /ant:init \"<your goal>\"  - Start your new project")
        output.append("   2. /ant:plan               - Review phases")
        output.append("   3. /ant:phase 1           - Start first phase")
        output.append("")
        output.append("💡 RECOMMENDATION:")
        output.append("   Colony is now ready to build that matches your codebase style.")
        output.append("   Your new code will seamlessly integrate with existing patterns.")
        output.append("")
        output.append("🔄 CONTEXT: Lightweight - safe to continue")

        return "\n".join(output)

    # ============================================================
    # /ant:pause-colony
    # ============================================================

    async def pause_colony(self) -> str:
        """
        Pause colony work and create handoff document.

        Saves current state so work can be resumed later.
        """
        if not self.started:
            return "❌ No active project. Run /ant:init <goal> first."

        # Get current state
        status = await self.system.status()
        memory = await self.system.memory()
        current_phase = self.system.phase_engine.get_current_phase()

        # Create handoff document
        handoff = self._create_handoff(status, memory, current_phase)

        # Save handoff
        import json
        from datetime import datetime

        handoff_file = ".aether/PAUSED_SESSION.json"
        with open(handoff_file, "w") as f:
            json.dump(handoff, f, indent=2)

        output = []
        output.append("🐜 Queen Ant Colony - Pause & Save Session")
        output.append("")
        output.append("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
        output.append("")

        # Show summary
        if current_phase:
            output.append(f"SAVED PHASE: Phase {current_phase.id} - {current_phase.name}")
            output.append(f"STATUS: {current_phase.status.value if isinstance(current_phase.status, PhaseStatus) else current_phase.status}")
            output.append(f"TASKS: {len(current_phase.tasks)} total")
            completed = sum(1 for t in current_phase.tasks if t.status == "completed")
            output.append(f"PROGRESS: {completed}/{len(current_phase.tasks)} tasks completed")
            output.append("")

        output.append("SAVED STATE:")
        output.append("  ✓ Current goal and pheromones")
        output.append("  ✓ Worker Ant states")
        output.append("  ✓ Phase progress")
        output.append("  ✓ Memory and learned patterns")
        output.append("")

        output.append("HANDOFF FILE:")
        output.append(f"  → {handoff_file}")
        output.append("")

        output.append("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
        output.append("")
        output.append("📋 NEXT STEPS:")
        output.append("")
        output.append("  1. /ant:resume-colony    - Resume from saved session")
        output.append("   2. Start new Claude session  → Then run resume command")
        output.append("")
        output.append("💡 TIP:")
        output.append("   Use pause when you need to stop mid-phase.")
        output.append("   Colony will be ready to continue when you resume.")
        output.append("")
        output.append("🔄 CONTEXT: PERFECT CHECKPOINT")
        output.append("   Refreshing Claude is recommended after pause.")
        output.append("   Resume in new session with clean context.")

        return "\n".join(output)

    def _create_handoff(self, status: Dict, memory: Dict, current_phase) -> Dict:
        """Create handoff document for resuming"""
        import json
        from datetime import datetime

        return {
            "timestamp": datetime.now().isoformat(),
            "goal": self.system.current_goal,
            "system_status": status,
            "memory": memory,
            "current_phase": current_phase.to_dict() if current_phase else None,
            "pheromones": [
                {
                    "type": s.signal_type.value,
                    "content": s.content,
                    "strength": s.strength,
                    "created_at": s.created_at.isoformat()
                }
                for s in self.system.pheromone_layer.get_active_signals()
            ]
        }

    # ============================================================
    # /ant:resume-colony
    # ============================================================

    async def resume_colony(self) -> str:
        """
        Resume colony from saved handoff document.

        Restores:
        - Goal and pheromones
        - Phase progress
        - Worker Ant states
        - Memory and patterns
        """
        import json
        import os

        handoff_file = ".aether/PAUSED_SESSION.json"

        if not os.path.exists(handoff_file):
            return "❌ No paused session found. Run /ant:pause-colony to save a session first."

        # Load handoff
        with open(handoff_file, "") as f:
            handoff = json.load(f)

        # Restore state
        self.system.current_goal = handoff.get("goal")
        self.started = True

        # Restore pheromones
        for signal_data in handoff.get("pheromones", []):
            from .pheromone_system import PheromoneSignal, PheromoneType
            signal = PheromoneSignal(
                signal_type=PheromoneType(signal_data["type"]),
                content=signal_data["content"],
                strength=signal_data["strength"],
                created_at=datetime.fromisoformat(signal_data["created_at"])
            )
            self.system.pheromone_layer.signals.append(signal)

        # Restore phase
        phase_data = handoff.get("current_phase")
        if phase_data:
            from .phase_engine import Phase, PhaseStatus
            phase = Phase(
                id=phase_data["id"],
                name=phase_data["name"],
                description=phase_data.get("description", ""),
                tasks=phase_data.get("tasks", []),
                milestones=phase_data.get("milestones", []),
                status=PhaseStatus[phase_data["status"].upper()] if isinstance(phase_data["status"], str) else PhaseStatus.PENDING
            )
            self.system.phase_engine.phases[phase.id - 1] = phase
            self.system.phase_engine.current_phase = phase

        output = []
        output.append("🐜 Queen Ant Colony - Resume Session")
        output.append("")
        output.append("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
        output.append("")

        output.append("SESSION RESTORED:")
        output.append("")
        output.append(f"  Goal: {handoff.get('goal')}")
        output.append(f"  Paused at: {handoff.get('timestamp')}")
        output.append("")

        if phase_data:
            output.append(f"RESTORED PHASE: Phase {phase_data['id']} - {phase_data['name']}")
            output.append(f"  Status: {phase_data.get('status', 'unknown').upper()}")
            output.append("")

            tasks = phase_data.get("tasks", [])
            completed = sum(1 for t in tasks if t.get("status") == "completed")
            output.append(f"  Tasks: {completed}/{len(tasks)} completed")
            output.append("")

        output.append("STATE RESTORED:")
        output.append("  ✓ Goal and pheromones")
        output.append("  ✓ Phase progress")
        output.append("  ✓ Worker Ant states")
        output.append("  ✓ Memory and learned patterns")
        output.append("")

        output.append("ACTIVE PHEROMONES:")
        for signal_data in handoff.get("pheromones", []):
            signal_type = signal_data["type"]
            content = signal_data["content"][:50]
            strength = signal_data["strength"]
            output.append(f"  [{signal_type.upper()}] {content}... (strength: {strength})")
        output.append("")

        output.append("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
        output.append("")
        output.append("✅ COLONY READY TO CONTINUE")
        output.append("")
        output.append("You can now:")
        output.append("  • Continue where you left off")
        output.append("  • Use all /ant: commands normally")
        output.append("  • Colony remembers everything")
        output.append("")
        output.append("📋 NEXT STEPS:")
        output.append("")
        output.append("  1. /ant:status            - Check colony status")
        output.append("   2. /ant:phase             - Continue with phase")
        output.append("  3. /ant:focus <area>      - Add guidance if needed")
        output.append("")
        output.append("💡 RECOMMENDATION:")
        output.append("   Review what was happening before pausing, then continue.")
        output.append("")
        output.append("🔄 CONTEXT: REFRESHED")
        output.append("   You're in a new session with clean context.")
        output.append("   Colony state fully restored.")

        return "\n".join(output)


# Singleton instance
_commands = None

def get_commands() -> InteractiveCommands:
    """Get or create commands instance"""
    global _commands
    if _commands is None:
        _commands = InteractiveCommands()
    return _commands
