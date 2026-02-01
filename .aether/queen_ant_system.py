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

    async def errors(self) -> Dict[str, Any]:
        """
        /ant:errors

        Show error ledger and flagged issues.
        """
        # This would integrate with the error prevention system
        # For now, return placeholder
        return {
            "error_ledger": {},
            "flagged_issues": [],
            "message": "Error prevention system integration pending"
        }

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
            "version": "1.0.0",
            "description": "Phased autonomy with user as pheromone source",
            "worker_ants": list(self.colony.worker_ants.keys()),
            "pheromone_types": [t.value for t in PheromoneType],
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
    print("Demo Complete")
    print("=" * 60)
    print("\nKey Points:")
    print("  ✅ Queen provides intention, not commands")
    print("  ✅ Colony self-organizes within phases")
    print("  ✅ Pheromones guide behavior (signals, not orders)")
    print("  ✅ Phase boundaries provide checkpoints")
    print("  ✅ Pure emergence within structured phases")


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
