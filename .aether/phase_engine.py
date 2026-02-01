"""
Queen Ant Colony - Phase Execution Engine

Implements phased autonomy:
- Queen sets intention via pheromones
- Colony creates phase structure
- Pure emergence within phases
- Checkpoints at phase boundaries
- Memory compression between phases

Based on research from:
- Phase 6: Integration & Synthesis
- Phase 7: Implementation Planning
- Phase 4: Anticipatory Systems
- Phase 5: Verification & Quality
"""

from typing import List, Dict, Any, Optional, Callable
from dataclasses import dataclass, field
from enum import Enum
from datetime import datetime, timedelta
import asyncio
import json

try:
    from .pheromone_system import (
        PheromoneType,
        PheromoneSignal,
        PheromoneLayer,
        SensitivityProfile,
        SENSITIVITY_PROFILES
    )
    from .worker_ants import (
        WorkerAnt,
        Colony,
        MapperAnt,
        PlannerAnt,
        ExecutorAnt,
        VerifierAnt,
        ResearcherAnt,
        SynthesizerAnt
    )
except ImportError:
    from pheromone_system import (
        PheromoneType,
        PheromoneSignal,
        PheromoneLayer,
        SensitivityProfile,
        SENSITIVITY_PROFILES
    )
    from worker_ants import (
        WorkerAnt,
        Colony,
        MapperAnt,
        PlannerAnt,
        ExecutorAnt,
        VerifierAnt,
        ResearcherAnt,
        SynthesizerAnt
    )


class PhaseStatus(Enum):
    """Status of a phase"""
    PENDING = "pending"
    PLANNING = "planning"
    IN_PROGRESS = "in_progress"
    AWAITING_REVIEW = "awaiting_review"
    APPROVED = "approved"
    COMPLETED = "completed"
    FAILED = "failed"


@dataclass
class Task:
    """A task within a phase"""
    id: str
    description: str
    status: str = "pending"
    assigned_to: Optional[str] = None
    spawned_for: Optional[str] = None
    dependencies: List[str] = field(default_factory=list)
    started_at: Optional[datetime] = None
    completed_at: Optional[datetime] = None
    metadata: Dict[str, Any] = field(default_factory=dict)

    def is_ready(self, completed_tasks: List[str]) -> bool:
        """Check if task is ready to start (dependencies met)"""
        return all(dep in completed_tasks for dep in self.dependencies)

    def to_dict(self) -> Dict[str, Any]:
        """Convert to dictionary"""
        return {
            "id": self.id,
            "description": self.description,
            "status": self.status,
            "assigned_to": self.assigned_to,
            "spawned_for": self.spawned_for,
            "dependencies": self.dependencies,
            "started_at": self.started_at.isoformat() if self.started_at else None,
            "completed_at": self.completed_at.isoformat() if self.completed_at else None,
            "metadata": self.metadata
        }


@dataclass
class Phase:
    """A phase in the project plan"""
    id: int
    name: str
    description: str
    tasks: List[Task] = field(default_factory=list)
    status: PhaseStatus = PhaseStatus.PENDING
    milestones: List[str] = field(default_factory=list)

    # Timing
    created_at: datetime = field(default_factory=datetime.now)
    started_at: Optional[datetime] = None
    completed_at: Optional[datetime] = None
    estimated_duration: Optional[timedelta] = None
    actual_duration: Optional[timedelta] = None

    # Queen interaction
    queen_approval: Optional[bool] = None
    queen_feedback: Optional[str] = None
    queen_adjustments: List[str] = field(default_factory=list)

    # Learning
    key_learnings: List[str] = field(default_factory=list)
    issues_found: List[Dict[str, Any]] = field(default_factory=list)
    patterns_extracted: List[str] = field(default_factory=list)

    # Agents
    agents_spawned: int = 0
    messages_exchanged: int = 0

    def to_dict(self) -> Dict[str, Any]:
        """Convert to dictionary"""
        return {
            "id": self.id,
            "name": self.name,
            "description": self.description,
            "status": self.status.value,
            "tasks": [t.to_dict() for t in self.tasks],
            "milestones": self.milestones,
            "created_at": self.created_at.isoformat(),
            "started_at": self.started_at.isoformat() if self.started_at else None,
            "completed_at": self.completed_at.isoformat() if self.completed_at else None,
            "estimated_duration": str(self.estimated_duration) if self.estimated_duration else None,
            "actual_duration": str(self.actual_duration) if self.actual_duration else None,
            "queen_approval": self.queen_approval,
            "queen_feedback": self.queen_feedback,
            "queen_adjustments": self.queen_adjustments,
            "key_learnings": self.key_learnings,
            "issues_found": self.issues_found,
            "patterns_extracted": self.patterns_extracted,
            "agents_spawned": self.agents_spawned,
            "messages_exchanged": self.messages_exchanged
        }

    def get_completion_percentage(self) -> float:
        """Get phase completion percentage"""
        if not self.tasks:
            return 0.0

        completed = sum(1 for t in self.tasks if t.status == "completed")
        return (completed / len(self.tasks)) * 100

    def get_pending_tasks(self) -> List[Task]:
        """Get pending tasks"""
        return [t for t in self.tasks if t.status == "pending"]

    def get_in_progress_tasks(self) -> List[Task]:
        """Get in-progress tasks"""
        return [t for t in self.tasks if t.status == "in_progress"]

    def get_completed_tasks(self) -> List[Task]:
        """Get completed tasks"""
        return [t for t in self.tasks if t.status == "completed"]


class PhaseEngine:
    """
    Manages phase lifecycle with phased autonomy.

    The Engine:
    1. Receives init pheromone from Queen
    2. Creates phase structure (via Planner)
    3. Executes phases with pure emergence
    4. Checks in with Queen at boundaries
    5. Adapts based on feedback
    6. Compresses memory between phases
    """

    def __init__(self, colony: Colony, pheromone_layer: PheromoneLayer):
        self.colony = colony
        self.pheromone_layer = pheromone_layer
        self.phases: List[Phase] = []
        self.current_phase: Optional[Phase] = None
        self.phase_history: List[Phase] = []

        # Worker Ants (from colony)
        self.mapper: MapperAnt = colony.worker_ants["mapper"]
        self.planner: PlannerAnt = colony.worker_ants["planner"]
        self.executor: ExecutorAnt = colony.worker_ants["executor"]
        self.verifier: VerifierAnt = colony.worker_ants["verifier"]
        self.researcher: ResearcherAnt = colony.worker_ants["researcher"]
        self.synthesizer: SynthesizerAnt = colony.worker_ants["synthesizer"]

    async def initiate_project(self, goal: str) -> Phase:
        """
        Initiate a new project from Queen's init pheromone.

        This triggers:
        1. Mapper explores codebase
        2. Planner creates phase structure
        3. Queen reviews phase plan
        """
        # Emit init pheromone
        await self.pheromone_layer.emit(
            PheromoneType.INIT,
            goal,
            strength=1.0,
            metadata={"goal": goal}
        )

        # Step 1: Mapper explores codebase
        await self.mapper.explore_codebase(goal)

        # Step 2: Planner creates phase structure
        phases = await self.planner.decompose_goal(goal)

        # Convert to Phase objects
        self.phases = [
            Phase(
                id=p["id"],
                name=p["name"],
                description=f"Phase {p['id']}: {p['name']}",
                tasks=[Task(**t) for t in p.get("tasks", [])],
                milestones=p.get("milestones", [])
            )
            for p in phases
        ]

        # Set first phase as current
        self.current_phase = self.phases[0]
        self.current_phase.status = PhaseStatus.PLANNING

        return self.current_phase

    async def execute_phase(self, phase: Phase) -> Phase:
        """
        Execute a phase with pure emergence.

        Within the phase:
        - Worker Ants self-organize
        - Subagents spawn as needed
        - Respond to pheromones in real-time
        - Coordinate peer-to-peer

        No Queen intervention during execution.
        """
        phase.status = PhaseStatus.IN_PROGRESS
        phase.started_at = datetime.now()

        # Notify colony of phase start
        await self._notify_phase_start(phase)

        # Pure emergence execution
        await self._execute_with_emergence(phase)

        # Phase completion detected
        phase.status = PhaseStatus.AWAITING_REVIEW
        phase.completed_at = datetime.now()
        phase.actual_duration = phase.completed_at - phase.started_at

        # Compress phase memory
        await self._compress_phase_memory(phase)

        # Check in with Queen
        await self._phase_checkin(phase)

        return phase

    async def _execute_with_emergence(self, phase: Phase):
        """
        Execute phase with pure emergence.

        This is where the magic happens.
        No central coordination. Worker Ants self-organize.
        """
        completed_tasks: List[str] = []

        # While there are pending tasks
        while phase.get_pending_tasks() or phase.get_in_progress_tasks():
            # Worker Ants detect and claim tasks
            await self._self_organize_tasks(phase, completed_tasks)

            # Worker Ants coordinate peer-to-peer
            await self._coordinate_ants(phase)

            # Check for phase completion
            if phase.get_completion_percentage() >= 100:
                break

            # Wait a bit before next cycle
            await asyncio.sleep(1)

        # Verify phase completion
        await self.verifier.verify_phase(phase.to_dict())

    async def _self_organize_tasks(self, phase: Phase, completed_tasks: List[str]):
        """
        Worker Ants self-organize to handle tasks.

        No assignment. They detect and respond.
        """
        pending = phase.get_pending_tasks()

        for task in pending:
            if task.is_ready(completed_tasks):
                # Worker Ants detect task based on capabilities
                await self._assign_task_emergent(task)

    async def _assign_task_emergent(self, task: Task):
        """
        Assign task through emergence, not command.

        Worker Ants detect task and respond if they have capability.
        """
        # Analyze task to determine which ant should handle it
        task_desc = task.description.lower()

        # Mapper handles exploration tasks
        if any(word in task_desc for word in ["explore", "map", "index", "understand"]):
            if not task.assigned_to:
                task.assigned_to = "mapper"
                task.status = "in_progress"
                task.started_at = datetime.now()
                await self.mapper.coordinate_with(self.planner)

        # Planner handles planning tasks
        elif any(word in task_desc for word in ["plan", "design", "structure"]):
            if not task.assigned_to:
                task.assigned_to = "planner"
                task.status = "in_progress"
                task.started_at = datetime.now()

        # Executor handles implementation tasks
        elif any(word in task_desc for word in ["implement", "write", "create", "build"]):
            if not task.assigned_to:
                task.assigned_to = "executor"
                task.status = "in_progress"
                task.started_at = datetime.now()
                await self.executor.implement_task(task.description)

        # Verifier handles testing tasks
        elif any(word in task_desc for word in ["test", "verify", "validate", "check"]):
            if not task.assigned_to:
                task.assigned_to = "verifier"
                task.status = "in_progress"
                task.started_at = datetime.now()
                await self.verifier.verify_phase(self.current_phase.to_dict())

        # Researcher handles research tasks
        elif any(word in task_desc for word in ["research", "find", "lookup", "search"]):
            if not task.assigned_to:
                task.assigned_to = "researcher"
                task.status = "in_progress"
                task.started_at = datetime.now()
                await self.researcher.research_topic(task.description)

    async def _coordinate_ants(self, phase: Phase):
        """
        Worker Ants coordinate peer-to-peer.

        No central coordinator. Direct communication.
        """
        # Executor coordinates with Verifier
        if phase.get_in_progress_tasks():
            await self.executor.coordinate_with(self.verifier)

        # Mapper coordinates with Planner
        if self.mapper.current_task:
            await self.mapper.coordinate_with(self.planner)

        # Researcher shares findings
        if self.researcher.current_task:
            await self.researcher.coordinate_with(self.executor)

    async def _notify_phase_start(self, phase: Phase):
        """Notify all Worker Ants that phase is starting"""
        for ant in self.colony.worker_ants.values():
            # Ants detect phase start and mobilize
            signals = await ant.detect_pheromones()
            if signals:
                await ant.respond_to_pheromones(signals)

    async def _compress_phase_memory(self, phase: Phase):
        """
        Compress phase memory after completion.

        Synthesizer Ant:
        - Extracts key learnings
        - Identifies patterns
        - Compresses to short-term memory
        """
        # Spawn compression agents
        await self.synthesizer.compress_phase_memory(phase.to_dict())

        # Extract patterns
        patterns = await self.synthesizer.extract_patterns(phase.to_dict())
        phase.patterns_extracted = patterns

        # Identify key learnings
        phase.key_learnings = [
            f"Completed {phase.name} in {phase.actual_duration}",
            f"Spawned {phase.agents_spawned} agents",
            f"{len(phase.issues_found)} issues found and resolved"
        ]

    async def _phase_checkin(self, phase: Phase):
        """
        Check in with Queen at phase boundary.

        Queen reviews:
        - What was done
        - How long it took
        - Issues found
        - Key learnings

        Queen can:
        - Approve next phase
        - Adjust direction
        - Add pheromones
        - Request changes
        """
        phase.status = PhaseStatus.AWAITING_REVIEW

        # Wait for Queen approval
        # (This is handled via /ant:phase command)

    async def queen_approve_phase(self, phase_id: int, feedback: Optional[str] = None):
        """
        Queen approves phase and allows continuation.

        Called via /ant:phase approve command.
        """
        phase = self.get_phase(phase_id)
        if phase:
            phase.status = PhaseStatus.APPROVED
            phase.queen_approval = True
            phase.queen_feedback = feedback

    async def queen_adjust_phase(
        self,
        phase_id: int,
        adjustments: List[str],
        focus_areas: Optional[List[str]] = None
    ):
        """
        Queen adjusts phase direction.

        Called via /ant:focus command.
        """
        phase = self.get_phase(phase_id)
        if phase:
            phase.queen_adjustments = adjustments
            phase.queen_feedback = f"Adjustments: {', '.join(adjustments)}"

            # Emit focus pheromones
            if focus_areas:
                for area in focus_areas:
                    await self.pheromone_layer.emit(
                        PheromoneType.FOCUS,
                        area,
                        strength=0.7
                    )

    async def execute_next_phase(self) -> Optional[Phase]:
        """
        Execute next phase after current phase is approved.

        Adapts based on Queen's feedback from previous phase.
        """
        if not self.current_phase:
            return None

        current_idx = self.phases.index(self.current_phase)

        # Move to next phase
        if current_idx + 1 < len(self.phases):
            next_phase = self.phases[current_idx + 1]

            # Adapt based on previous phase feedback
            if self.current_phase.queen_feedback:
                next_phase.metadata["previous_feedback"] = self.current_phase.queen_feedback

            # Adapt based on previous phase learnings
            if self.current_phase.key_learnings:
                next_phase.metadata["previous_learnings"] = self.current_phase.key_learnings

            # Execute next phase
            self.current_phase = next_phase
            return await self.execute_phase(next_phase)

        return None

    def get_phase(self, phase_id: int) -> Optional[Phase]:
        """Get phase by ID"""
        for phase in self.phases:
            if phase.id == phase_id:
                return phase
        return None

    def get_current_phase(self) -> Optional[Phase]:
        """Get current phase"""
        return self.current_phase

    def get_phase_summary(self) -> Dict[str, Any]:
        """Get summary of all phases"""
        return {
            "total_phases": len(self.phases),
            "current_phase": self.current_phase.to_dict() if self.current_phase else None,
            "phases": [p.to_dict() for p in self.phases],
            "progress": {
                "completed": sum(1 for p in self.phases if p.status == PhaseStatus.COMPLETED),
                "in_progress": sum(1 for p in self.phases if p.status == PhaseStatus.IN_PROGRESS),
                "pending": sum(1 for p in self.phases if p.status in [PhaseStatus.PENDING, PhaseStatus.PLANNING])
            }
        }

    async def respond_to_pheromones(self):
        """
        Respond to pheromones during phase execution.

        This allows real-time guidance from Queen.
        """
        # Get active pheromones
        active_signals = self.pheromone_layer.get_active_signals()

        # Respond to each signal
        for signal in active_signals:
            if signal.signal_type == PheromoneType.FOCUS:
                # Guide colony attention
                await self._handle_focus_pheromone(signal)

            elif signal.signal_type == PheromoneType.REDIRECT:
                # Warn away from approach
                await self._handle_redirect_pheromone(signal)

            elif signal.signal_type == PheromoneType.FEEDBACK:
                # Adjust behavior
                await self._handle_feedback_pheromone(signal)

    async def _handle_focus_pheromone(self, signal: PheromoneSignal):
        """Handle focus pheromone - guide attention"""
        # Executor prioritizes focused area
        if self.current_phase:
            await self.executor.prioritize_work(signal.content)

    async def _handle_redirect_pheromone(self, signal: PheromoneSignal):
        """Handle redirect pheromone - warn away"""
        # Executor avoids this pattern
        await self.executor.avoid_pattern(signal.content)

    async def _handle_feedback_pheromone(self, signal: PheromoneSignal):
        """Handle feedback pheromone - adjust behavior"""
        # Adjust based on feedback content
        if "bug" in signal.content.lower():
            await self.verifier.intensify_testing()
        elif "slow" in signal.content.lower():
            # Speed up execution
            pass


class PhaseCommands:
    """
    Command interface for phase management.

    This is what the /ant: commands use.
    """

    def __init__(self, phase_engine: PhaseEngine):
        self.phase_engine = phase_engine

    async def init(self, goal: str) -> Dict[str, Any]:
        """
        /ant:init <goal>

        Initialize new project with goal.
        """
        phase = await self.phase_engine.initiate_project(goal)
        return {"phase": phase.to_dict(), "status": "initiated"}

    async def phase(self, phase_id: Optional[int] = None) -> Dict[str, Any]:
        """
        /ant:phase

        Show phase status or specific phase details.
        """
        if phase_id is not None:
            phase = self.phase_engine.get_phase(phase_id)
            if phase:
                return {"phase": phase.to_dict()}
            else:
                return {"error": f"Phase {phase_id} not found"}

        return self.phase_engine.get_phase_summary()

    async def plan(self) -> Dict[str, Any]:
        """
        /ant:plan

        Show upcoming phases.
        """
        summary = self.phase_engine.get_phase_summary()
        return {
            "phases": [
                {
                    "id": p.id,
                    "name": p.name,
                    "status": p.status.value,
                    "tasks_count": len(p.tasks)
                }
                for p in self.phase_engine.phases
            ],
            "current": self.phase_engine.get_current_phase().to_dict() if self.phase_engine.get_current_phase() else None
        }

    async def approve(self, phase_id: int, feedback: Optional[str] = None) -> Dict[str, Any]:
        """
        /ant:phase approve <id>

        Approve phase and continue.
        """
        await self.phase_engine.queen_approve_phase(phase_id, feedback)
        return {"status": "approved", "phase_id": phase_id}

    async def continue_phase(self) -> Dict[str, Any]:
        """
        /ant:phase continue

        Continue to next phase.
        """
        next_phase = await self.phase_engine.execute_next_phase()
        if next_phase:
            return {"phase": next_phase.to_dict(), "status": "executing"}
        else:
            return {"status": "complete", "message": "All phases complete"}


# Factory function
def create_phase_engine(colony: Colony, pheromone_layer: PheromoneLayer) -> PhaseEngine:
    """Create a new phase engine"""
    return PhaseEngine(colony, pheromone_layer)
