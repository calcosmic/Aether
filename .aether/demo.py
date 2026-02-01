#!/usr/bin/env python3
"""
Aether Queen Ant Colony - Full System Demo

Demonstrates the complete Queen Ant Colony system:
- Initialize colony with goal
- Create phase structure
- Emit pheromone signals
- Execute phase with Worker Ant spawning
- Memory compression at phase boundary

Run with: python3 .aether/demo.py
"""

import asyncio
import sys
from datetime import datetime

# Import system components
try:
    from .queen_ant_system import create_queen_ant_system
except ImportError:
    sys.path.insert(0, '.aether')
    from queen_ant_system import create_queen_ant_system


async def demo_full_system():
    """Demonstrate full Queen Ant Colony workflow"""
    print()
    print("╔════════════════════════════════════════════════════════════════╗")
    print("║         Aether Queen Ant Colony - Full System Demo            ║")
    print("╠════════════════════════════════════════════════════════════════╣")
    print("║                                                                ║")
    print("║  The first AI system where Worker Ants autonomously spawn     ║")
    print("║  other Worker Ants to complete complex tasks.                  ║")
    print("║                                                                ║")
    print("║  Features:                                                     ║")
    print("║  - Triple-Layer Memory (200k tokens, DAST compression)        ║")
    print("║  - 6 Worker Ant Castes (Mapper, Planner, Executor, etc.)      ║")
    print("║  - Semantic Pheromone Layer for coordination                 ║")
    print("║  - Phase Engine for state orchestration                       ║")
    print("║  - Error Ledger for pattern flagging                          ║")
    print("╚════════════════════════════════════════════════════════════════╝")
    print()

    # Step 1: Create and initialize system
    print("🐜 Step 1: Creating Queen Ant Colony...")
    print("-" * 70)
    system = create_queen_ant_system()
    await system.start()

    info = system.get_system_info()
    print(f"✅ Colony initialized")
    print(f"   Features: {', '.join([f for f, e in info['features'].items() if e])}")
    print()

    # Step 2: Initialize project with goal
    print("🐜 Step 2: Initializing project with goal...")
    print("-" * 70)
    goal = "Build a REST API with FastAPI, PostgreSQL, and JWT authentication"
    print(f"   Goal: {goal}")
    print()

    result = await system.init(goal)
    print(f"✅ {result['message']}")
    print(f"   Phases created: {result.get('phases_created', 0)}")
    print()

    # Step 3: Show phase structure
    print("🐜 Step 3: Phase structure created by Queen...")
    print("-" * 70)
    plan_result = await system.plan()
    print("✅ Phase structure ready")

    # Display first few phases
    if 'phases' in plan_result and plan_result['phases']:
        print()
        print("   Initial phases:")
        for phase in list(plan_result['phases'])[:4]:
            status_emoji = "⏳" if phase.get('status') == 'pending' else "✅"
            print(f"   {status_emoji} Phase {phase.get('id', '?')}: {phase.get('name', 'Untitled')}")
            print(f"      Tasks: {phase.get('tasks_count', 0)}")

        if plan_result.get('current'):
            print()
            print(f"   Current: Phase {plan_result['current'].get('id', '?')}")
    print()

    # Step 4: Emit focus pheromone
    print("🐜 Step 4: Emitting focus pheromone to guide colony...")
    print("-" * 70)
    focus_result = await system.focus("security")
    print(f"✅ {focus_result['message']}")
    signal = focus_result.get('signal', {})
    print(f"   Focus: {signal.get('content', 'N/A')}")
    print(f"   Type: {signal.get('type', 'N/A')}")
    print(f"   Strength: {signal.get('strength', 0):.2f}")
    print()

    # Step 5: Show colony status
    print("🐜 Step 5: Colony status before execution...")
    print("-" * 70)
    status_result = await system.status()
    print(f"✅ Colony Status:")
    if 'state' in status_result:
        print(f"   Current Phase: {status_result['state'].get('current_phase', 'N/A')}")
        print(f"   State: {status_result['state'].get('state', 'N/A')}")
    print()

    # Step 6: Show memory status
    print("🐜 Step 6: Memory system status...")
    print("-" * 70)
    memory_status = await system.memory_status()
    if "error" not in memory_status:
        tlm = memory_status.get("triple_layer_memory", {})
        working = tlm.get("working", {})
        short_term = tlm.get("short_term", {})
        long_term = tlm.get("long_term", {})

        print("✅ Triple-Layer Memory:")
        print(f"   📝 Working: {working.get('item_count', 0)} items, "
              f"{working.get('used_tokens', 0)} / {working.get('max_tokens', 0)} tokens")
        print(f"   📚 Short-Term: {short_term.get('session_count', 0)} / "
              f"{short_term.get('max_sessions', 0)} sessions")
        print(f"   💾 Long-Term: {long_term.get('total_patterns', 0)} patterns")
    print()

    # Step 7: Execute first phase (simulated)
    print("🐜 Step 7: Executing Phase 1 (Mapper Ant spawning...)...")
    print("-" * 70)
    print("   Note: This is a simulated execution. Real execution would spawn")
    print("   Worker Ants that autonomously spawn other Worker Ants.")
    print()
    print("   Simulating...")
    print()

    # Simulate some working memory activity
    await system.memory_layer.add_to_working(
        "FastAPI selected for REST API framework",
        metadata={"source": "planner", "confidence": 0.9},
        item_type="decision"
    )
    await system.memory_layer.add_to_working(
        "Use async/await patterns for database operations",
        metadata={"source": "best_practice"},
        item_type="convention"
    )
    await system.memory_layer.add_to_working(
        "JWT token validation middleware needed for protected routes",
        metadata={"source": "security"},
        item_type="requirement"
    )

    print("✅ Phase 1 simulated completion:")
    print("   ✅ Mapper Ant explored codebase structure")
    print("   ✅ Planner Ant created detailed task breakdown")
    print("   ✅ Executor Ant implemented API endpoints")
    print("   ✅ Verifier Ant validated implementation")
    print()

    # Step 8: Phase boundary - compress to short-term memory
    print("🐜 Step 8: Phase boundary - Compressing memory to short-term...")
    print("-" * 70)
    compress_result = await system.memory_compress(
        phase_metadata={
            "phase": "1",
            "goal": goal,
            "duration": "45 minutes",
            "tasks_completed": 8
        }
    )
    print(f"✅ {compress_result['message']}")
    print(f"   Session ID: {compress_result['session_id']}")
    print(f"   Items archived: {compress_result.get('items_archived', 0)}")
    print(f"   Compression ratio: {compress_result.get('compression_ratio', 0):.2f}x")
    print()

    # Step 9: Cross-layer search
    print("🐜 Step 9: Cross-layer memory search...")
    print("-" * 70)
    search_query = "API"
    search_result = await system.memory_search(search_query, limit=5)
    print(f"✅ Searching for '{search_query}' across all memory layers...")
    print(f"   Found {search_result['results_count']} results:")

    for r in search_result.get("results", [])[:3]:
        layer_icon = {"working": "📝", "short_term": "📚", "long_term": "💾"}
        icon = layer_icon.get(r['layer'], '?')
        print(f"   {icon} [{r['layer'].upper()}] {r['content'][:60]}...")
    print()

    # Step 10: Final summary
    print("🐜 Step 10: Demo Complete - System Summary")
    print("-" * 70)

    # Get final status
    final_status = await system.status()
    final_memory = await system.memory_status()

    print("✅ Queen Ant Colony System:")
    print(f"   State: {final_status.get('state', {}).get('state', 'active')}")
    print(f"   Current Phase: {final_status.get('state', {}).get('current_phase', '1')}")
    print()
    print("✅ Memory System:")
    if "triple_layer_memory" in final_memory:
        working = final_memory["triple_layer_memory"].get("working", {})
        short_term = final_memory["triple_layer_memory"].get("short_term", {})
        print(f"   Working items: {working.get('item_count', 0)}")
        print(f"   Short-term sessions: {short_term.get('session_count', 0)}")
    print()
    print("✅ Worker Ant Castes Available:")
    castes = [
        "🐜 Mapper Ant - Explores and maps codebase",
        "🐜 Planner Ant - Creates execution plans",
        "🐜 Executor Ant - Implements tasks",
        "🐜 Verifier Ant - Validates implementation",
        "🐜 Researcher Ant - Gathers information",
        "🐜 Synthesizer Ant - Combines research results"
    ]
    for caste in castes:
        print(f"   {caste}")
    print()

    # Cleanup
    await system.stop()

    print()
    print("╔════════════════════════════════════════════════════════════════╗")
    print("║                    Demo Complete!                              ║")
    print("╠════════════════════════════════════════════════════════════════╣")
    print("║                                                                ║")
    print("║  To continue exploring:                                        ║")
    print("║  - Run REPL:      python3 .aether/repl.py                      ║")
    print("║  - Memory demo:   python3 .aether/memory_demo.py               ║")
    print("║  - CLI help:      python3 .aether/cli.py --help                ║")
    print("║                                                                ║")
    print("╚════════════════════════════════════════════════════════════════╝")
    print()


async def main():
    """Main demo entry point"""
    try:
        await demo_full_system()
    except KeyboardInterrupt:
        print("\n\nDemo interrupted by user")
    except Exception as e:
        print(f"\n❌ Error: {e}")
        import traceback
        traceback.print_exc()


if __name__ == "__main__":
    asyncio.run(main())
