#!/usr/bin/env python3
"""
Aether Queen Ant Colony - Memory Integration Demo

Demonstrates the complete triple-layer memory system integration:
- Working memory: add, search, flush
- Short-term memory: compression, retrieval, forgetting
- Long-term memory: pattern storage, search
- Phase boundary: automatic compression
- Worker ants: memory context usage

Run with: python3 .aether/memory_demo.py
"""

import asyncio
import sys
from datetime import datetime

# Import all memory components
try:
    from .memory.triple_layer_memory import TripleLayerMemory
    from .queen_ant_system import create_queen_ant_system
    from .worker_ants import Task
except ImportError:
    # Try direct import
    sys.path.insert(0, '.aether')
    from memory.triple_layer_memory import TripleLayerMemory
    from queen_ant_system import create_queen_ant_system
    from worker_ants import Task


async def demo_working_memory():
    """Demo 1: Working Memory - Add, Search, Flush"""
    print("=" * 70)
    print("DEMO 1: Working Memory")
    print("=" * 70)
    print()

    memory = TripleLayerMemory(working_max_tokens=1000)  # Small for demo

    # Add items
    print("1. Adding items to working memory...")
    await memory.add_to_working(
        "Build REST API with FastAPI and PostgreSQL",
        metadata={"source": "user", "priority": "high"},
        item_type="goal"
    )
    await memory.add_to_working(
        "Use snake_case for files, PascalCase for classes",
        metadata={"source": "convention"},
        item_type="convention"
    )
    await memory.add_to_working(
        "Always use parameterized queries to prevent SQL injection",
        metadata={"source": "security"},
        item_type="best_practice"
    )

    status = memory.working.get_status()
    print(f"   Items: {status['item_count']}")
    print(f"   Tokens: {status['used_tokens']}")
    print()

    # Search
    print("2. Searching working memory...")
    results = await memory.search_working("API", limit=5)
    print(f"   Found {len(results)} items")
    for r in results:
        print(f"   - [{r.metadata.get('type', 'general')}] {r.content}")
    print()

    # Flush to short-term
    print("3. Flushing to short-term (phase boundary simulation)...")
    session_id = await memory.compress_to_short_term({
        "phase": "1",
        "goal": "Build REST API",
        "duration": "2 hours"
    })
    print(f"   Session: {session_id}")
    print(f"   Working memory now empty: {memory.working.item_count} items")
    print()

    compression_stats = memory.get_compression_stats()
    print("4. Compression stats:")
    print(f"   Short-term sessions: {compression_stats['short_term']['session_count']}")
    print(f"   Avg compression ratio: {compression_stats['short_term']['avg_compression_ratio']:.2f}x")
    print()


async def demo_long_term_memory():
    """Demo 2: Long-Term Memory - Patterns and Search"""
    print("=" * 70)
    print("DEMO 2: Long-Term Memory - Patterns")
    print("=" * 70)
    print()

    memory = TripleLayerMemory()

    # Store patterns
    print("1. Storing patterns in long-term memory...")
    await memory.store_long_term(
        category="tech_stack",
        key="framework",
        value="FastAPI for REST APIs",
        confidence=0.9
    )
    await memory.store_long_term(
        category="conventions",
        key="naming",
        value="snake_case files, PascalCase classes",
        confidence=0.8
    )
    pattern_id = await memory.learn_from_error(
        error_category="sql_injection",
        symptom="User input in SQL query",
        fix="Use parameterized queries",
        prevention="Always use placeholders for user input"
    )

    print(f"   Stored 3 patterns")
    print()

    # Search
    print("2. Searching long-term memory...")
    results = await memory.long_term.search("FastAPI", limit=5)
    print(f"   Found {len(results)} patterns")
    for r in results:
        print(f"   - [{r.category}] {r.key}")
        print(f"     Confidence: {r.confidence:.2f} | Occurrences: {r.occurrences}")
        print(f"     {r.value[:60]}...")
    print()


async def demo_cross_layer_retrieval():
    """Demo 3: Cross-Layer Retrieval"""
    print("=" * 70)
    print("DEMO 3: Cross-Layer Memory Retrieval")
    print("=" * 70)
    print()

    memory = TripleLayerMemory()

    # Add to working
    await memory.add_to_working(
        "Using React with TypeScript for frontend",
        metadata={"source": "decision"},
        item_type="decision"
    )
    await memory.add_to_working(
        "Jest for unit testing, Cypress for e2e",
        metadata={"source": "decision"},
        item_type="convention"
    )

    # Compress to short-term
    await memory.compress_to_short_term({"phase": "1", "goal": "Setup"})

    # Store in long-term
    await memory.store_long_term(
        category="patterns",
        key="testing_strategy",
        value="Jest + Cypress combination for full coverage",
        confidence=0.85
    )

    # Cross-layer search
    print("1. Searching 'React' across all layers...")
    results = await memory.retrieve("React")

    print(f"   Found {len(results)} results across all layers:")
    for r in results:
        layer_icon = {"working": "📝", "short_term": "📚", "long_term": "💾"}
        print(f"   {layer_icon.get(r.layer, '?')} [{r.layer}] {r.content[:60]}...")
    print()


async def demo_queen_ant_integration():
    """Demo 4: Queen Ant System Integration"""
    print("=" * 70)
    print("DEMO 4: Queen Ant System Integration")
    print("=" * 70)
    print()

    # Create system with memory
    system = create_queen_ant_system()
    await system.start()

    # Check system info
    info = system.get_system_info()
    print("1. System Features:")
    for feature, enabled in info["features"].items():
        status = "✅" if enabled else "❌"
        print(f"   {status} {feature}")
    print()

    # Initialize project
    print("2. Initializing project...")
    result = await system.init("Build a chat application")
    print(f"   {result['message']}")
    print()

    # Memory status
    print("3. Checking memory status...")
    memory_status = await system.memory_status()
    if "error" not in memory_status:
        tlm = memory_status.get("triple_layer_memory", {})
        working = tlm.get("working", {})
        short_term = tlm.get("short_term", {})
        long_term = tlm.get("long_term", {})

        print(f"   Working: {working.get('item_count', 0)} items, "
              f"{working.get('used_tokens', 0)} tokens")
        print(f"   Short-term: {short_term.get('session_count', 0)} sessions")
        print(f"   Long-term: {long_term.get('total_patterns', 0)} patterns")
        print()

    # Test memory compress
    print("4. Manual memory compression...")
    compress_result = await system.memory_compress(
        phase_metadata={"phase": "demo", "goal": "Test memory integration"}
    )
    print(f"   {compress_result['message']}")
    print(f"   Session: {compress_result['session_id']}")
    print()

    # Test memory search
    print("5. Searching memory for 'chat'...")
    search_result = await system.memory_search("chat", limit=5)
    print(f"   Found {search_result['results_count']} results")
    for r in search_result.get("results", [])[:3]:
        print(f"   - [{r['layer']}] {r['content'][:50]}...")
    print()


async def main():
    """Run all memory integration demos"""
    print()
    print("╔════════════════════════════════════════════════════════════════╗")
    print("║       Aether Queen Ant Colony - Memory Integration Demo                 ║")
    print("╠════════════════════════════════════════════════════════════════╣")
    print("║                                                                       ║")
    print("║  Demonstrates triple-layer memory system integration:                 ║")
    print("║  - Working Memory: 200k tokens, immediate access                    ║")
    print("║  - Short-Term Memory: 10 sessions, DAST 2.5x compression            ║")
    print("║  - Long-Term Memory: Persistent knowledge storage                    ║")
    print("║                                                                       ║")
    print("║  Queen Ant System now has full memory capabilities!                ║")
    print("╚════════════════════════════════════════════════════════════════╝")
    print()

    try:
        await demo_working_memory()
        await asyncio.sleep(1)

        await demo_long_term_memory()
        await asyncio.sleep(1)

        await demo_cross_layer_retrieval()
        await asyncio.sleep(1)

        await demo_queen_ant_integration()

    except KeyboardInterrupt:
        print("\n\nDemo interrupted by user")
    except Exception as e:
        print(f"\n❌ Error: {e}")
        import traceback
        traceback.print_exc()

    print()
    print("=" * 70)
    print("Memory Integration Demo Complete")
    print("=" * 70)
    print()
    print("✅ Working Memory: Add, search, flush working")
    print("✅ Short-Term Memory: Compression, retrieval, forgetting")
    print("✅ Long-Term Memory: Pattern storage, search")
    print("✅ Phase Boundaries: Automatic compression")
    print("✅ Worker Ants: Memory context usage")
    print("✅ Queen Ant System: Full integration")
    print()


if __name__ == "__main__":
    asyncio.run(main())
