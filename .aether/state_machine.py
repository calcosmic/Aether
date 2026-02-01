"""
Queen Ant Colony - State Machine Orchestration

Implements LangGraph-style state machine for production-grade orchestration:
- Explicit state transitions
- Checkpointing before/after each transition
- State recovery from checkpoints
- State history tracking
- Observability into all transitions

Based on research from:
- MULTI_AGENT_ORCHESTRATION_RESEARCH.md (LangGraph patterns)
- Ralph's Review Recommendation #4
"""

from typing import List, Dict, Any, Optional, Literal, TypedDict, TYPE_CHECKING
from dataclasses import dataclass, field
from enum import Enum
from datetime import datetime
import asyncio
import json
from pathlib import Path

if TYPE_CHECKING:
    from .phase_engine import Phase, PhaseStatus, Task
    from .worker_ants import Colony, WorkerAnt
else:
    # Runtime imports - avoid circular dependency
    Phase = None
    PhaseStatus = None
    Task = None
try:
    from .worker_ants import Colony, WorkerAnt
except ImportError:
    from worker_ants import Colony, WorkerAnt


# ============================================================
# STATE SCHEMA
# ============================================================

class SystemState(Enum):
    """
    Explicit states for AETHER orchestration.

    State Machine Flow:
        IDLE → ANALYZING → PLANNING → EXECUTING → VERIFYING → COMPLETED
                                  ↘          ↘
                                   FAILED ←─────┘
    """
    IDLE = "idle"
    ANALYZING = "analyzing"           # Mapper exploring codebase
    PLANNING = "planning"             # Planner creating structure
    EXECUTING = "executing"           # Colony working on tasks
    VERIFYING = "verifying"           # Verifier checking work
    COMPLETED = "completed"           # Phase finished successfully
    FAILED = "failed"                 # Error occurred


class EventType(Enum):
    """Events that trigger state transitions"""
    TASK_RECEIVED = "task_received"
    ANALYSIS_COMPLETE = "analysis_complete"
    PLAN_READY = "plan_ready"
    EXECUTION_STARTED = "execution_started"
    EXECUTION_COMPLETE = "execution_complete"
    VERIFICATION_STARTED = "verification_started"
    VERIFICATION_COMPLETE = "verification_complete"
    ERROR = "error"
    PHASE_APPROVED = "phase_approved"
    QUEEN_ADJUSTMENT = "queen_adjustment"
    CANCEL = "cancel"


@dataclass
class Event:
    """Event that triggers state transition"""
    type: EventType
    timestamp: datetime = field(default_factory=datetime.now)
    data: Dict[str, Any] = field(default_factory=dict)
    source: Optional[str] = None  # Which agent/component emitted

    def to_dict(self) -> Dict[str, Any]:
        return {
            "type": self.type.value,
            "timestamp": self.timestamp.isoformat(),
            "data": self.data,
            "source": self.source
        }


@dataclass
class AgentState(TypedDict):
    """
    Complete state for AETHER orchestration.

    This schema captures all relevant state for checkpointing and recovery.
    """
    # Phase info
    phase: Literal["IDLE", "ANALYZING", "PLANNING", "EXECUTING", "VERIFYING", "COMPLETED", "FAILED"]

    # Context
    semantic_context: Dict[str, Any]

    # Current task
    current_task: Optional[Dict[str, Any]]

    # Agent assignments
    agent_assignments: Dict[str, Any]

    # Checkpoint data
    checkpoint_data: Dict[str, Any]

    # Pheromone signals
    pheromone_signals: List[Dict[str, Any]]

    # Metadata
    timestamp: str
    state_id: str


@dataclass
class StateTransition:
    """Record of a state transition"""
    from_state: SystemState
    to_state: SystemState
    event: Event
    timestamp: datetime
    transition_id: str
    decision_reason: Optional[str] = None
    context: Dict[str, Any] = field(default_factory=dict)

    def to_dict(self) -> Dict[str, Any]:
        return {
            "from_state": self.from_state.value,
            "to_state": self.to_state.value,
            "event": self.event.to_dict(),
            "timestamp": self.timestamp.isoformat(),
            "transition_id": self.transition_id,
            "decision_reason": self.decision_reason,
            "context": self.context
        }


@dataclass
class Checkpoint:
    """
    Checkpoint for state recovery.

    Contains all information needed to restore system state.
    """
    id: str
    timestamp: datetime
    state: AgentState
    agents: Dict[str, Any]  # Serialized agent states
    phase_snapshot: Optional[Dict[str, Any]] = None

    def to_dict(self) -> Dict[str, Any]:
        return {
            "id": self.id,
            "timestamp": self.timestamp.isoformat(),
            "state": self.state,
            "agents": self.agents,
            "phase_snapshot": self.phase_snapshot
        }


# ============================================================
# STATE MACHINE
# ============================================================

class AetherStateMachine:
    """
    State machine for AETHER orchestration with checkpointing.

    Each transition:
    1. Saves checkpoint before transition
    2. Executes transition logic
    3. Saves checkpoint after transition
    4. Returns new state

    Enables:
    - Recovery from failures
    - Debugging with state history
    - Observability into all transitions
    """

    def __init__(
        self,
        colony: Colony,
        checkpoint_dir: Optional[Path] = None
    ):
        self.colony = colony

        # Current state
        self.current_state: SystemState = SystemState.IDLE
        self.state_context: Dict[str, Any] = {}

        # State history
        self.state_history: List[StateTransition] = []
        self.transition_counter: int = 0

        # Checkpointing
        self.checkpoint_dir = checkpoint_dir or Path(".aether/checkpoints")
        self.checkpoint_dir.mkdir(parents=True, exist_ok=True)
        self.checkpoints: List[Checkpoint] = []

        # Observability
        self.event_log: List[Event] = []

    async def transition(self, event: Event) -> AgentState:
        """
        Main state transition function with checkpointing.

        Args:
            event: Event triggering the transition

        Returns:
            New state after transition
        """
        # Log event
        self.event_log.append(event)

        # Save checkpoint BEFORE transition
        await self._save_checkpoint(self.current_state, event, "before")

        # Execute transition
        new_state = await self._execute_transition(self.current_state, event)

        # Save checkpoint AFTER transition
        await self._save_checkpoint(new_state, event, "after")

        # Record transition in history
        self._record_transition(self.current_state, new_state, event)

        # Update current state
        old_state = self.current_state
        self.current_state = new_state

        return self._build_agent_state(new_state, event)

    async def _execute_transition(
        self,
        current_state: SystemState,
        event: Event
    ) -> SystemState:
        """
        Execute specific state transition based on current state and event.

        Implements the state machine logic.
        """

        # IDLE state transitions
        if current_state == SystemState.IDLE:
            if event.type == EventType.TASK_RECEIVED:
                return await self._transition_to_analyzing(event)

        # ANALYZING state transitions
        elif current_state == SystemState.ANALYZING:
            if event.type == EventType.ANALYSIS_COMPLETE:
                return await self._transition_to_planning(event)
            elif event.type == EventType.ERROR:
                return await self._transition_to_failed(event)

        # PLANNING state transitions
        elif current_state == SystemState.PLANNING:
            if event.type == EventType.PLAN_READY:
                return await self._transition_to_executing(event)
            elif event.type == EventType.ERROR:
                return await self._transition_to_failed(event)
            elif event.type == EventType.QUEEN_ADJUSTMENT:
                # Stay in planning, update context
                await self._handle_queen_adjustment(event)
                return SystemState.PLANNING

        # EXECUTING state transitions
        elif current_state == SystemState.EXECUTING:
            if event.type == EventType.EXECUTION_COMPLETE:
                return await self._transition_to_verifying(event)
            elif event.type == EventType.ERROR:
                return await self._transition_to_failed(event)

        # VERIFYING state transitions
        elif current_state == SystemState.VERIFYING:
            if event.type == EventType.VERIFICATION_COMPLETE:
                return await self._transition_to_completed(event)
            elif event.type == EventType.ERROR:
                # Verification failed - go back to executing
                return await self._transition_to_executing(event)

        # COMPLETED state - terminal, no outgoing transitions
        elif current_state == SystemState.COMPLETED:
            # Stay completed
            return SystemState.COMPLETED

        # FAILED state - terminal, no outgoing transitions
        elif current_state == SystemState.FAILED:
            # Stay failed
            return SystemState.FAILED

        # No valid transition - stay in current state
        return current_state

    async def _transition_to_analyzing(self, event: Event) -> SystemState:
        """Transition from IDLE to ANALYZING"""
        # Mobilize Mapper Ant
        task = event.data.get("task")
        if task:
            await self.colony.worker_ants["mapper"].explore_codebase(task)

        self.state_context["analysis_started_at"] = datetime.now()
        return SystemState.ANALYZING

    async def _transition_to_planning(self, event: Event) -> SystemState:
        """Transition from ANALYZING to PLANNING"""
        # Hand off to Planner
        analysis_result = event.data.get("analysis_result")

        self.state_context["analysis_result"] = analysis_result
        self.state_context["planning_started_at"] = datetime.now()
        return SystemState.PLANNING

    async def _transition_to_executing(self, event: Event) -> SystemState:
        """Transition from PLANNING or VERIFYING to EXECUTING"""
        # Mobilize Executor and other workers

        self.state_context["execution_started_at"] = datetime.now()
        return SystemState.EXECUTING

    async def _transition_to_verifying(self, event: Event) -> SystemState:
        """Transition from EXECUTING to VERIFYING"""
        # Mobilize Verifier
        execution_result = event.data.get("execution_result")

        self.state_context["execution_result"] = execution_result
        self.state_context["verification_started_at"] = datetime.now()
        return SystemState.VERIFYING

    async def _transition_to_completed(self, event: Event) -> SystemState:
        """Transition from VERIFYING to COMPLETED"""
        # Phase complete - await Queen approval

        self.state_context["completed_at"] = datetime.now()
        return SystemState.COMPLETED

    async def _transition_to_failed(self, event: Event) -> SystemState:
        """Transition to FAILED state"""
        # Log error
        error_data = event.data.get("error", {})
        self.state_context["error"] = error_data
        self.state_context["failed_at"] = datetime.now()
        return SystemState.FAILED

    async def _handle_queen_adjustment(self, event: Event):
        """Handle Queen adjustment without changing state"""
        adjustment = event.data.get("adjustment")
        self.state_context["queen_adjustment"] = adjustment

    def _record_transition(
        self,
        from_state: SystemState,
        to_state: SystemState,
        event: Event
    ):
        """Record transition in history"""
        self.transition_counter += 1

        transition = StateTransition(
            from_state=from_state,
            to_state=to_state,
            event=event,
            timestamp=datetime.now(),
            transition_id=f"transition_{self.transition_counter}",
            decision_reason=self._explain_transition(from_state, to_state, event),
            context=self.state_context.copy()
        )

        self.state_history.append(transition)

    def _explain_transition(
        self,
        from_state: SystemState,
        to_state: SystemState,
        event: Event
    ) -> str:
        """Generate human-readable explanation for transition"""
        return f"Transitioned from {from_state.value} to {to_state.value} due to {event.type.value}"

    def _build_agent_state(self, system_state: SystemState, event: Event) -> AgentState:
        """Build AgentState dict from current system state"""
        return {
            "phase": system_state.value.upper(),
            "semantic_context": self.state_context,
            "current_task": event.data.get("task"),
            "agent_assignments": self._get_agent_assignments(),
            "checkpoint_data": {
                "transition_count": self.transition_counter,
                "last_event": event.to_dict()
            },
            "pheromone_signals": [],  # Would integrate with pheromone system
            "timestamp": datetime.now().isoformat(),
            "state_id": f"state_{self.transition_counter}"
        }

    def _get_agent_assignments(self) -> Dict[str, Any]:
        """Get current agent assignments"""
        return {
            ant_name: {
                "current_task": getattr(ant, "current_task", None),
                "status": getattr(ant, "status", "active")
            }
            for ant_name, ant in self.colony.worker_ants.items()
        }

    # ============================================================
    # CHECKPOINTING
    # ============================================================

    async def _save_checkpoint(
        self,
        state: SystemState,
        event: Event,
        position: str  # "before" or "after"
    ):
        """
        Save checkpoint for recovery.

        Checkpoints include:
        - Current state
        - State context
        - Agent states
        - Event that triggered transition
        """
        self.transition_counter += 1

        checkpoint = Checkpoint(
            id=f"checkpoint_{self.transition_counter}_{position}",
            timestamp=datetime.now(),
            state=self._build_agent_state(state, event),
            agents=await self._serialize_agents(),
            phase_snapshot=self.state_context.copy()
        )

        self.checkpoints.append(checkpoint)

        # Persist to disk
        await self._persist_checkpoint(checkpoint)

    async def _serialize_agents(self) -> Dict[str, Any]:
        """Serialize agent states for checkpointing"""
        serialized = {}

        for ant_name, ant in self.colony.worker_ants.items():
            serialized[ant_name] = {
                "name": ant_name,
                "current_task": getattr(ant, "current_task", None),
                "subagents_spawned": getattr(ant, "subagents_spawned", 0),
                "status": "active"
            }

        return serialized

    async def _persist_checkpoint(self, checkpoint: Checkpoint):
        """Persist checkpoint to disk"""
        checkpoint_file = self.checkpoint_dir / f"{checkpoint.id}.json"

        checkpoint_data = {
            "id": checkpoint.id,
            "timestamp": checkpoint.timestamp.isoformat(),
            "state": checkpoint.state,
            "agents": checkpoint.agents,
            "phase_snapshot": checkpoint.phase_snapshot
        }

        with open(checkpoint_file, "w") as f:
            json.dump(checkpoint_data, f, indent=2, default=str)

    async def recover_from_checkpoint(self, checkpoint_id: str) -> AgentState:
        """
        Recover state from checkpoint.

        Args:
            checkpoint_id: ID of checkpoint to recover from

        Returns:
            Restored agent state
        """
        # Find checkpoint
        checkpoint = next(
            (c for c in self.checkpoints if c.id == checkpoint_id),
            None
        )

        if not checkpoint:
            # Try loading from disk
            checkpoint = await self._load_checkpoint_from_disk(checkpoint_id)

        if not checkpoint:
            raise ValueError(f"Checkpoint {checkpoint_id} not found")

        # Restore state
        self.state_context = checkpoint.state.get("semantic_context", {})
        self.current_state = SystemState(checkpoint["state"]["phase"].lower())

        # Rehydrate agents (in a real system, this would restore agent states)
        await self._restore_agents(checkpoint.agents)

        return checkpoint.state

    async def _load_checkpoint_from_disk(self, checkpoint_id: str) -> Optional[Checkpoint]:
        """Load checkpoint from disk"""
        checkpoint_file = self.checkpoint_dir / f"{checkpoint_id}.json"

        if not checkpoint_file.exists():
            return None

        with open(checkpoint_file, "r") as f:
            data = json.load(f)

        return Checkpoint(
            id=data["id"],
            timestamp=datetime.fromisoformat(data["timestamp"]),
            state=data["state"],
            agents=data["agents"],
            phase_snapshot=data.get("phase_snapshot")
        )

    async def _restore_agents(self, agents: Dict[str, Any]):
        """Restore agents from checkpoint (simplified)"""
        # In a real system, this would restore full agent state
        # For now, we just note that agents are restored
        pass

    # ============================================================
    # OBSERVABILITY
    # ============================================================

    def get_state_history(self) -> List[Dict[str, Any]]:
        """Get complete state transition history"""
        return [t.to_dict() for t in self.state_history]

    def get_current_state(self) -> Dict[str, Any]:
        """Get current state info"""
        return {
            "state": self.current_state.value,
            "context": self.state_context,
            "transition_count": self.transition_counter,
            "checkpoint_count": len(self.checkpoints)
        }

    def get_checkpoints(self) -> List[Dict[str, Any]]:
        """Get all checkpoints"""
        return [c.to_dict() for c in self.checkpoints]

    def get_event_log(self) -> List[Dict[str, Any]]:
        """Get event log"""
        return [e.to_dict() for e in self.event_log]

    def get_state_machine_summary(self) -> Dict[str, Any]:
        """Get comprehensive state machine summary"""
        return {
            "current_state": self.current_state.value,
            "state_history_count": len(self.state_history),
            "checkpoints_count": len(self.checkpoints),
            "events_logged": len(self.event_log),
            "recent_transitions": [
                t.to_dict() for t in self.state_history[-5:]
            ],
            "available_checkpoints": [c.id for c in self.checkpoints[-5:]]
        }


# ============================================================
# INTEGRATION WITH PHASE ENGINE
# ============================================================

class StateMachinePhaseEngine:
    """
    Phase Engine integrated with State Machine.

    Wraps the PhaseEngine to add state machine orchestration.
    """

    def __init__(
        self,
        colony: Colony,
        state_machine: AetherStateMachine
    ):
        self.colony = colony
        self.state_machine = state_machine

        # Track current phase
        self.current_phase: Optional[Any] = None  # Phase object, imported dynamically

    async def execute_phase_with_state_machine(
        self,
        phase: Any  # Phase object
    ) -> Any:  # Phase object
        """
        Execute phase using state machine orchestration.

        This replaces the simple execution in PhaseEngine with
        full state machine tracking and checkpointing.
        """

        # Start with TASK_RECEIVED event
        event = Event(
            type=EventType.TASK_RECEIVED,
            data={"task": phase.description, "phase": phase.to_dict()},
            source="phase_engine"
        )

        # Initial transition
        state = await self.state_machine.transition(event)

        # Execute state machine flow
        while state["phase"] not in ["COMPLETED", "FAILED"]:
            # Process based on current state
            if state["phase"] == "ANALYZING":
                await self._handle_analyzing(phase)
                next_event = Event(
                    type=EventType.ANALYSIS_COMPLETE,
                    data={"phase": phase.to_dict()},
                    source="phase_engine"
                )

            elif state["phase"] == "PLANNING":
                await self._handle_planning(phase)
                next_event = Event(
                    type=EventType.PLAN_READY,
                    data={"phase": phase.to_dict()},
                    source="phase_engine"
                )

            elif state["phase"] == "EXECUTING":
                await self._handle_executing(phase)
                next_event = Event(
                    type=EventType.EXECUTION_COMPLETE,
                    data={"phase": phase.to_dict()},
                    source="phase_engine"
                )

            elif state["phase"] == "VERIFYING":
                await self._handle_verifying(phase)
                next_event = Event(
                    type=EventType.VERIFICATION_COMPLETE,
                    data={"phase": phase.to_dict()},
                    source="phase_engine"
                )
                break

            # Transition to next state
            state = await self.state_machine.transition(next_event)

        # Update phase status - import PhaseStatus dynamically
        from .phase_engine import PhaseStatus
        if state["phase"] == "COMPLETED":
            phase.status = PhaseStatus.COMPLETED
        elif state["phase"] == "FAILED":
            phase.status = PhaseStatus.FAILED

        return phase

    async def _handle_analyzing(self, phase: Any):
        """Handle ANALYZING state - Mapper explores"""
        mapper = self.colony.worker_ants["mapper"]
        await mapper.explore_codebase(phase.description)

    async def _handle_planning(self, phase: Any):
        """Handle PLANNING state - create task structure"""
        # Planning handled by Planner
        pass

    async def _handle_executing(self, phase: Any):
        """Handle EXECUTING state - colony works on tasks"""
        # Execute with emergence
        completed_tasks = []

        while phase.get_pending_tasks() or phase.get_in_progress_tasks():
            for task in phase.get_pending_tasks():
                if task.is_ready(completed_tasks):
                    task.status = "in_progress"
                    task.started_at = datetime.now()
                    # Task assignment handled by colony

            # Check for completion
            if phase.get_completion_percentage() >= 100:
                break

            await asyncio.sleep(0.1)

    async def _handle_verifying(self, phase: Any):
        """Handle VERIFYING state - Verifier checks work"""
        verifier = self.colony.worker_ants["verifier"]
        await verifier.verify_phase(phase.to_dict())


# ============================================================
# FACTORY
# ============================================================

def create_state_machine(
    colony: Colony,
    checkpoint_dir: Optional[Path] = None
) -> AetherStateMachine:
    """Create a new state machine"""
    return AetherStateMachine(colony, checkpoint_dir)


# ============================================================
# DEMO
# ============================================================

async def demo_state_machine():
    """Demonstration of state machine orchestration"""
    print("🔄 State Machine Orchestration Demo\n")

    from .worker_ants import create_colony

    # Create system
    colony = create_colony()
    state_machine = create_state_machine(colony)

    print("=" * 60)
    print("State Machine: IDLE")
    print("=" * 60)

    # Simulate receiving a task
    event = Event(
        type=EventType.TASK_RECEIVED,
        data={"task": "Build authentication system"},
        source="queen"
    )

    print(f"\n📨 Event: {event.type.value}")
    state = await state_machine.transition(event)
    print(f"✅ New State: {state['phase']}")

    print(f"\n📊 State: {state_machine.get_current_state()}")

    # Simulate analysis complete
    event = Event(
        type=EventType.ANALYSIS_COMPLETE,
        data={"analysis_result": "Codebase mapped"},
        source="mapper"
    )

    print(f"\n📨 Event: {event.type.value}")
    state = await state_machine.transition(event)
    print(f"✅ New State: {state['phase']}")

    # Simulate plan ready
    event = Event(
        type=EventType.PLAN_READY,
        data={"plan": "Tasks created"},
        source="planner"
    )

    print(f"\n📨 Event: {event.type.value}")
    state = await state_machine.transition(event)
    print(f"✅ New State: {state['phase']}")

    # Show history
    print("\n" + "=" * 60)
    print("State Transition History")
    print("=" * 60)

    for transition in state_machine.get_state_history():
        print(f"\n{transition['transition_id']}:")
        print(f"  {transition['from_state']} → {transition['to_state']}")
        print(f"  Reason: {transition['decision_reason']}")

    # Show checkpoints
    print("\n" + "=" * 60)
    print("Checkpoints Created")
    print("=" * 60)

    for checkpoint in state_machine.get_checkpoints():
        print(f"\n📁 {checkpoint['id']}:")
        print(f"   Timestamp: {checkpoint['timestamp']}")
        print(f"   State: {checkpoint['state']['phase']}")

    # Show state machine summary
    print("\n" + "=" * 60)
    print("State Machine Summary")
    print("=" * 60)

    summary = state_machine.get_state_machine_summary()
    print(f"Current State: {summary['current_state']}")
    print(f"Total Transitions: {summary['state_history_count']}")
    print(f"Total Checkpoints: {summary['checkpoints_count']}")
    print(f"Events Logged: {summary['events_logged']}")

    print("\n" + "=" * 60)
    print("✅ State Machine Demo Complete")
    print("=" * 60)
    print("\nKey Features:")
    print("  ✅ Explicit state transitions")
    print("  ✅ Automatic checkpointing")
    print("  ✅ State history tracking")
    print("  ✅ Event logging")
    print("  ✅ State recovery capability")


if __name__ == "__main__":
    asyncio.run(demo_state_machine())
