"""
Queen Ant Colony - Unified System

The complete Queen Ant colony system integrating:
- Worker Ant castes (6 specialists)
- Pheromone signal system
- Phase execution engine
- Memory and learning
- Error prevention

This is the main entry point for the Queen Ant system.
"""

import asyncio
from typing import List, Dict, Any, Optional
from datetime import datetime
from dataclasses import dataclass

try:
    from .worker_ants import Colony, create_colony
    from .pheromone_system import (
        PheromoneLayer,
        PheromoneType,
        create_pheromone_layer,
        PheromoneCommands,
        PheromoneHistory
    )
    from .phase_engine import (
        PhaseEngine,
        Phase,
        create_phase_engine,
        PhaseCommands
    )
    from .error_prevention import ErrorLedger
except ImportError:
    from worker_ants import Colony, create_colony
    from pheromone_system import (
        PheromoneLayer,
        PheromoneType,
        create_pheromone_layer,
        PheromoneCommands,
        PheromoneHistory
    )
    from phase_engine import (
        PhaseEngine,
        Phase,
        create_phase_engine,
        PhaseCommands
    )
    from error_prevention import ErrorLedger


class QueenAntSystem:
    """
    The Queen Ant Colony System

    A phased autonomy system where:
    - Queen (User) provides intention via pheromones
    - Colony self-organizes within phases
    - Pure emergence, guided by feedback
    - Phase boundaries = checkpoints

    Usage:
        system = QueenAntSystem()
        await system.start()

        # Queen signals
        await system.init("Build a real-time chat app")
        await system.focus("WebSocket security")
        await system.feedback("Great progress")

        # Observe colony
        status = await system.get_status()
        phase_info = await system.phase()
    """

    def __init__(self):
        # Core components
        self.colony = create_colony()
        self.pheromone_layer = create_pheromone_layer()
        self.phase_engine = create_phase_engine(self.colony, self.pheromone_layer)

        # Command interfaces
        self.pheromone_commands = PheromoneCommands(self.pheromone_layer)
        self.phase_commands = PhaseCommands(self.phase_engine)

        # Learning and memory
        self.pheromone_history = PheromoneHistory(self.pheromone_layer)

        # System state
        self.started_at: Optional[datetime] = None
        self.current_goal: Optional[str] = None

    async def start(self):
        """Start the Queen Ant system"""
        self.started_at = datetime.now()
        # System is ready to receive Queen's signals

    async def stop(self):
        """Stop the Queen Ant system"""
        # Graceful shutdown
        pass

    # ============================================================
    # QUEEN COMMANDS (User Interface)
    # ============================================================

    async def init(self, goal: str) -> Dict[str, Any]:
        """
        /ant:init <goal>

        Lay egg. Create intention pheromone.
        Colony will create phase structure.
        """
        self.current_goal = goal

        # Emit init pheromone
        await self.pheromone_commands.init(goal)

        # Create phase structure
        result = await self.phase_commands.init(goal)

        return {
            "message": f"Initiated project: {goal}",
            "goal": goal,
            "phase": result["phase"],
            "colony_status": self.colony.get_status()
        }

    async def phase(self, phase_id: Optional[int] = None) -> Dict[str, Any]:
        """
        /ant:phase

        Show current phase status or specific phase details.
        """
        return await self.phase_commands.phase(phase_id)

    async def plan(self) -> Dict[str, Any]:
        """
        /ant:plan

        Show upcoming phases.
        """
        return await self.phase_commands.plan()

    async def focus(self, area: str, strength: float = 0.5) -> Dict[str, Any]:
        """
        /ant:focus <area>

        Emit focus pheromone. Guide colony attention.
        """
        signal = await self.pheromone_commands.focus(area, strength)

        return {
            "message": f"Focusing on: {area}",
            "signal": {
                "type": signal.signal_type.value,
                "content": signal.content,
                "strength": signal.strength
            }
        }

    async def redirect(self, pattern: str, strength: float = 0.7) -> Dict[str, Any]:
        """
        /ant:redirect <pattern>

        Emit redirect pheromone. Warn colony away from approach.
        """
        signal = await self.pheromone_commands.redirect(pattern, strength)

        return {
            "message": f"Redirecting away from: {pattern}",
            "signal": {
                "type": signal.signal_type.value,
                "content": signal.content,
                "strength": signal.strength
            }
        }

    async def feedback(self, message: str, strength: float = 0.5) -> Dict[str, Any]:
        """
        /ant:feedback <message>

        Emit feedback pheromone. Guide colony behavior.
        """
        signal = await self.pheromone_commands.feedback(message, strength)

        return {
            "message": f"Feedback recorded: {message}",
            "signal": {
                "type": signal.signal_type.value,
                "content": signal.content,
                "strength": signal.strength
            }
        }

    async def status(self) -> Dict[str, Any]:
        """
        /ant:status

        Show comprehensive colony status.
        """
        colony_status = self.colony.get_status()
        pheromone_status = self.pheromone_commands.get_status()
        phase_summary = self.phase_engine.get_phase_summary()

        return {
            "system": {
                "started_at": self.started_at.isoformat() if self.started_at else None,
                "current_goal": self.current_goal,
                "uptime_seconds": (datetime.now() - self.started_at).total_seconds() if self.started_at else 0
            },
            "colony": colony_status,
            "pheromones": pheromone_status,
            "phases": phase_summary
        }

    async def state_machine(self) -> Dict[str, Any]:
        """
        /ant:state-machine

        Show state machine status and history.
        """
        return self.phase_engine.get_state_machine_status()

    async def recover(self, checkpoint_id: str) -> Dict[str, Any]:
        """
        /ant:recover <checkpoint_id>

        Recover state from checkpoint.
        """
        return await self.phase_engine.recover_state(checkpoint_id)

    async def memory(self) -> Dict[str, Any]:
        """
        /ant:memory

        Show memory state and learned patterns.
        """
        # Get learned preferences from pheromone history
        learned = self.pheromone_history.get_learned_preferences()

        return {
            "learned_preferences": learned,
            "pheromone_patterns": {
                "focus_topics": list(learned.get("focus_topics", {}).keys())[:10],
                "avoid_patterns": list(learned.get("avoid_patterns", {}).keys())[:10],
                "feedback_categories": learned.get("feedback_categories", {})
            }
        }

    async def errors(self, show_all: bool = False) -> Dict[str, Any]:
        """
        /ant:errors

        Show error ledger and flagged issues.

        Args:
            show_all: If True, show all errors. If False, show only flagged and unresolved.
        """
        ledger = self.colony.error_ledger

        if show_all:
            # Show full error report
            return {
                "summary": ledger.get_summary(),
                "flagged_patterns": [p.get_summary() for p in ledger.get_flagged_patterns()],
                "unresolved_errors": [e.get_summary() for e in ledger.get_unresolved_errors()[:20]],
                "recent_errors": [e.get_summary() for e in ledger.get_recent_errors(24)],
                "status_report": ledger.get_status_report()
            }
        else:
            # Show summary and flagged issues only
            summary = ledger.get_summary()
            flagged = ledger.get_flagged_patterns()

            return {
                "summary": summary,
                "flagged_patterns": [p.get_summary() for p in flagged],
                "unresolved_count": summary['unresolved_errors'],
                "flagged_count": summary['flagged_patterns'],
                "message": ledger.get_status_report() if flagged else "No flagged issues. System healthy."
            }

    async def error_flagged(self) -> Dict[str, Any]:
        """
        /ant:error-flagged

        Show only flagged patterns (recurring issues).
        """
        ledger = self.colony.error_ledger
        flagged = ledger.get_flagged_patterns()

        if not flagged:
            return {
                "flagged_patterns": [],
                "message": "No flagged patterns. System healthy."
            }

        return {
            "flagged_patterns": [
                {
                    "pattern_id": p.pattern_id,
                    "error_type": p.error_type,
                    "category": p.category.value,
                    "occurrences": p.occurrence_count,
                    "flag_reason": p.flag_reason,
                    "first_occurrence": p.first_occurrence.isoformat(),
                    "last_occurrence": p.last_occurrence.isoformat(),
                    "systematic_fix_deployed": p.fix_deployed
                }
                for p in flagged
            ],
            "total_flagged": len(flagged)
        }

    async def semantic_search(
        self,
        query: str,
        top_k: int = 5,
        threshold: float = 0.6,
        signal_type: Optional[str] = None
    ) -> Dict[str, Any]:
        """
        /ant:semantic-search <query>

        Search for semantically similar pheromone signals.

        Args:
            query: Search query text
            top_k: Maximum number of results
            threshold: Minimum similarity (0-1)
            signal_type: Optional filter by signal type
        """
        # Convert signal_type string to enum if provided
        signal_type_enum = None
        if signal_type:
            try:
                signal_type_enum = PheromoneType(signal_type)
            except ValueError:
                return {"error": f"Invalid signal_type: {signal_type}"}

        results = self.pheromone_layer.find_similar_signals_semantic(
            query=query,
            top_k=top_k,
            threshold=threshold,
            signal_type=signal_type_enum
        )

        return {
            "query": query,
            "results_count": len(results),
            "results": results,
            "semantic_enabled": self.pheromone_layer.semantic_layer is not None
        }

    async def semantic_stats(self) -> Dict[str, Any]:
        """
        /ant:semantic-stats

        Show semantic layer statistics.
        """
        return self.pheromone_layer.get_semantic_stats()

    # ============================================================
    # INTERNAL METHODS
    # ============================================================

    async def _run_pheromone_decay(self):
        """Background task to clean up expired pheromones"""
        while True:
            await asyncio.sleep(60)  # Check every minute
            self.pheromone_layer.cleanup_expired()

    async def _monitor_phase_execution(self):
        """Background task to monitor and respond to pheromones during execution"""
        while True:
            await asyncio.sleep(5)  # Check every 5 seconds
            await self.phase_engine.respond_to_pheromones()

    def get_system_info(self) -> Dict[str, Any]:
        """Get system information"""
        return {
            "name": "Queen Ant Colony System",
            "version": "1.2.0",
            "description": "Phased autonomy with user as pheromone source",
            "worker_ants": list(self.colony.worker_ants.keys()),
            "pheromone_types": [t.value for t in PheromoneType],
            "state_machine_enabled": self.phase_engine.use_state_machine,
            "features": {
                "autonomous_spawning": True,
                "state_machine_orchestration": True,
                "checkpointing": True,
                "state_recovery": True,
                "error_prevention": True,
                "semantic_communication": self.pheromone_layer.semantic_layer is not None
            },
            "semantic_stats": self.pheromone_layer.get_semantic_stats(),
            "research_based": True,
            "research_documents": 25,
            "research_words": 383515
        }


# ============================================================
# DEMO / TESTING
# ============================================================

async def demo_queen_ant_system():
    """
    Demonstration of the Queen Ant system.

    This shows:
    1. Queen initiates project
    2. Colony creates phase structure
    3. Queen provides guidance
    4. Colony executes with emergence
    5. Phase boundary check-in
    6. Queen reviews and approves
    """
    print("🐜 Queen Ant Colony System Demo\n")

    # Create system
    system = QueenAntSystem()
    await system.start()

    print("=" * 60)
    print("STEP 1: Queen initiates project")
    print("=" * 60)

    result = await system.init("Build a real-time chat application")
    print(f"\n✅ Project initiated: {result['goal']}")
    print(f"   Phase 1: {result['phase']['name']}")

    print("\n" + "=" * 60)
    print("STEP 2: Colony created phase structure")
    print("=" * 60)

    plan = await system.plan()
    print(f"\n📋 Total phases: {len(plan['phases'])}")
    for p in plan['phases']:
        print(f"   Phase {p['id']}: {p['name']} ({p['status']})")

    print("\n" + "=" * 60)
    print("STEP 3: Queen provides focus guidance")
    print("=" * 60)

    result = await system.focus("WebSocket security", strength=0.7)
    print(f"\n🎯 Focus: {result['message']}")

    result = await system.focus("message reliability", strength=0.6)
    print(f"🎯 Focus: {result['message']}")

    print("\n" + "=" * 60)
    print("STEP 4: Queen observes colony status")
    print("=" * 60)

    status = await system.status()
    print(f"\n📊 Colony Status:")
    print(f"   Active pheromones: {status['pheromones']['pheromone_summary']['total_active']}")
    print(f"   Total subagents: {status['colony']['total_subagents']}")

    for ant_name, ant_info in status['colony']['worker_ants'].items():
        print(f"   {ant_name.capitalize()}: {ant_info['current_task']}")

    print("\n" + "=" * 60)
    print("STEP 5: Queen provides feedback")
    print("=" * 60)

    result = await system.feedback("Prioritize security features")
    print(f"\n💬 {result['message']}")

    print("\n" + "=" * 60)
    print("STEP 6: Queen reviews learned patterns")
    print("=" * 60)

    memory = await system.memory()
    print(f"\n🧠 Learned Preferences:")
    print(f"   Focus topics: {memory['pheromone_patterns']['focus_topics']}")
    print(f"   Avoid patterns: {memory['pheromone_patterns']['avoid_patterns']}")

    print("\n" + "=" * 60)
    print("STEP 7: State Machine Status")
    print("=" * 60)

    sm_status = await system.state_machine()
    if sm_status.get("enabled"):
        print(f"\n🔄 State Machine Enabled:")
        print(f"   Current State: {sm_status['current_state']['state']}")
        print(f"   Transitions: {sm_status['current_state']['transition_count']}")
        print(f"   Checkpoints: {sm_status['current_state']['checkpoint_count']}")

        if sm_status.get("transition_history"):
            print(f"\n📊 Recent Transitions:")
            for t in sm_status["transition_history"][-3:]:
                print(f"   {t['from_state']} → {t['to_state']} ({t['event']['type']})")

        if sm_status.get("checkpoints"):
            print(f"\n💾 Recent Checkpoints:")
            for c in sm_status["checkpoints"][-3:]:
                print(f"   {c['id']}: {c['state']['phase']}")

    print("\n" + "=" * 60)
    print("Demo Complete")
    print("=" * 60)
    print("\nKey Points:")
    print("  ✅ Queen provides intention, not commands")
    print("  ✅ Colony self-organizes within phases")
    print("  ✅ Pheromones guide behavior (signals, not orders)")
    print("  ✅ Phase boundaries provide checkpoints")
    print("  ✅ Pure emergence within structured phases")
    print("  ✅ State machine enables production-grade reliability")


# ============================================================
# FACTORY
# ============================================================

def create_queen_ant_system() -> QueenAntSystem:
    """Create a new Queen Ant colony system"""
    return QueenAntSystem()


# ============================================================
# MAIN
# ============================================================

if __name__ == "__main__":
    # Run demo
    asyncio.run(demo_queen_ant_system())
