#!/usr/bin/env python3
"""
Aether Queen Ant Colony - CLI Interface

Command-line interface for the Aether multi-agent system.
Provides access to all colony commands via terminal.

Usage:
    python -m aether.cli init "Build a REST API"
    python -m aether.cli plan
    python -m aether.cli repl
"""

import argparse
import asyncio
import sys
from typing import Optional
from datetime import datetime


def main():
    """Main CLI entry point"""
    parser = argparse.ArgumentParser(
        prog="aether",
        description="Aether Queen Ant Colony - Autonomous multi-agent development system",
        epilog="The first AI system where Worker Ants autonomously spawn other Worker Ants.",
        formatter_class=argparse.RawDescriptionHelpFormatter,
    )

    parser.add_argument(
        "--version",
        action="version",
        version="Aether 1.2.0 - Queen Ant Colony"
    )

    subparsers = parser.add_subparsers(
        dest="command",
        title="Available Commands",
        description="Use 'aether <command> --help' for command-specific help",
        metavar="COMMAND",
        help="Command to run"
    )

    # ============================================================
    # /ant:init <goal>
    # ============================================================
    init_parser = subparsers.add_parser(
        "init",
        help="Initialize a new project with a goal",
        description="Initialize a new project. The Queen sets intention and the colony creates phase structure."
    )
    init_parser.add_argument(
        "goal",
        help="Project goal/description (e.g., 'Build a REST API with user auth')"
    )
    init_parser.add_argument(
        "--colony",
        action="store_true",
        help="Colonize codebase first (analyze existing code)"
    )

    # ============================================================
    # /ant:plan
    # ============================================================
    plan_parser = subparsers.add_parser(
        "plan",
        help="Show all phases with tasks and details",
        description="Display the complete phase plan with tasks, milestones, and status."
    )

    # ============================================================
    # /ant:phase [N]
    # ============================================================
    phase_parser = subparsers.add_parser(
        "phase",
        help="Show phase details (current or specific)",
        description="Display detailed information about a phase or the current phase."
    )
    phase_parser.add_argument(
        "id",
        type=int,
        nargs="?",
        help="Phase number (leave empty for current phase)"
    )

    # ============================================================
    # /ant:execute <N>
    # ============================================================
    execute_parser = subparsers.add_parser(
        "execute",
        help="Execute a phase with pure emergence",
        description="Execute a phase. Worker Ants spawn autonomously to complete tasks."
    )
    execute_parser.add_argument(
        "phase_id",
        type=int,
        help="Phase number to execute"
    )
    execute_parser.add_argument(
        "--verbose", "-v",
        action="store_true",
        help="Show detailed progress"
    )

    # ============================================================
    # /ant:review <N>
    # ============================================================
    review_parser = subparsers.add_parser(
        "review",
        help="Review completed phase",
        description="Review what was built, key learnings, and issues resolved."
    )
    review_parser.add_argument(
        "phase_id",
        type=int,
        help="Phase number to review"
    )

    # ============================================================
    # /ant:focus <area>
    # ============================================================
    focus_parser = subparsers.add_parser(
        "focus",
        help="Guide colony attention with focus pheromone",
        description="Emit a focus pheromone to guide Worker Ant attention to specific areas."
    )
    focus_parser.add_argument(
        "area",
        help="Area to focus on (e.g., 'security', 'test coverage')"
    )
    focus_parser.add_argument(
        "--strength", "-s",
        type=float,
        default=0.5,
        help="Signal strength 0.0-1.0 (default: 0.5)"
    )

    # ============================================================
    # /ant:redirect <pattern>
    # ============================================================
    redirect_parser = subparsers.add_parser(
        "redirect",
        help="Warn colony away from approach",
        description="Emit a redirect pheromone to warn Worker Ants away from patterns."
    )
    redirect_parser.add_argument(
        "pattern",
        help="Pattern to avoid (e.g., 'circular imports')"
    )
    redirect_parser.add_argument(
        "--strength", "-s",
        type=float,
        default=0.7,
        help="Signal strength 0.0-1.0 (default: 0.7)"
    )

    # ============================================================
    # /ant:feedback <message>
    # ============================================================
    feedback_parser = subparsers.add_parser(
        "feedback",
        help="Provide guidance feedback to colony",
        description="Emit feedback pheromone to teach colony preferences."
    )
    feedback_parser.add_argument(
        "message",
        help="Feedback message"
    )
    feedback_parser.add_argument(
        "--strength", "-s",
        type=float,
        default=0.5,
        help="Signal strength 0.0-1.0 (default: 0.5)"
    )

    # ============================================================
    # /ant:status
    # ============================================================
    status_parser = subparsers.add_parser(
        "status",
        help="Show colony status",
        description="Display current colony state including Worker Ants, pheromones, and progress."
    )

    # ============================================================
    # /ant:colonize
    # ============================================================
    colonize_parser = subparsers.add_parser(
        "colonize",
        help="Analyze codebase before starting",
        description="Colonize codebase - analyze existing code to understand patterns and architecture."
    )

    # ============================================================
    # /ant:pause-colony
    # ============================================================
    pause_parser = subparsers.add_parser(
        "pause-colony",
        help="Save session mid-phase",
        description="Pause colony work and create handoff document for resuming later."
    )

    # ============================================================
    # /ant:resume-colony
    # ============================================================
    resume_parser = subparsers.add_parser(
        "resume-colony",
        help="Restore saved session",
        description="Resume colony from saved handoff document."
    )

    # ============================================================
    # /ant:repl
    # ============================================================
    repl_parser = subparsers.add_parser(
        "repl",
        help="Start interactive REPL",
        description="Launch interactive Read-Eval-Print Loop for real-time colony control."
    )
    repl_parser.add_argument(
        "--history",
        action="store_true",
        help="Enable command history"
    )

    # ============================================================
    # /ant:memory
    # ============================================================
    memory_parser = subparsers.add_parser(
        "memory",
        help="Memory system operations",
        description="Access triple-layer memory system (working, short-term, long-term)."
    )
    memory_subparsers = memory_parser.add_subparsers(
        dest="memory_action",
        title="Memory Actions",
        description="Use 'memory <action> --help' for action-specific help",
        metavar="ACTION"
    )

    # memory status
    memory_status_parser = memory_subparsers.add_parser(
        "status",
        help="Show memory system status",
        description="Display status of all three memory layers."
    )

    # memory working
    memory_working_parser = memory_subparsers.add_parser(
        "working",
        help="Show working memory contents",
        description="Display current working memory contents."
    )
    memory_working_parser.add_argument(
        "query",
        nargs="?",
        help="Search query (leave empty to show all)"
    )

    # memory short-term
    memory_short_parser = memory_subparsers.add_parser(
        "short-term",
        help="Show short-term memory sessions",
        description="Display compressed sessions in short-term memory."
    )
    memory_short_parser.add_argument(
        "query",
        nargs="?",
        help="Search query (leave empty to show all)"
    )

    # memory long-term
    memory_long_parser = memory_subparsers.add_parser(
        "long-term",
        help="Search long-term memory",
        description="Search persistent knowledge in long-term memory."
    )
    memory_long_parser.add_argument(
        "query",
        help="Search query"
    )
    memory_long_parser.add_argument(
        "--category", "-c",
        help="Filter by category"
    )

    # memory compress
    memory_compress_parser = memory_subparsers.add_parser(
        "compress",
        help="Manually compress working to short-term",
        description="Trigger compression of working memory to short-term."
    )
    memory_compress_parser.add_argument(
        "--phase", "-p",
        default="manual",
        help="Phase identifier (default: 'manual')"
    )

    # Parse arguments
    args = parser.parse_args()

    # If no command, show help
    if not args.command:
        parser.print_help()
        return 0

    # Execute command
    try:
        return asyncio.run(run_command(args))
    except KeyboardInterrupt:
        print("\n\nInterrupted by user")
        return 130
    except Exception as e:
        print(f"\n❌ Error: {e}", file=sys.stderr)
        return 1


async def run_command(args: argparse.Namespace) -> int:
    """Execute the specified command"""

    # Import here to avoid circular imports
    try:
        from .interactive_commands import InteractiveCommands
    except ImportError:
        try:
            from aether.interactive_commands import InteractiveCommands
        except ImportError:
            from interactive_commands import InteractiveCommands

    commands = InteractiveCommands()

    # ============================================================
    # init
    # ============================================================
    if args.command == "init":
        if args.colony:
            print("Colonizing codebase first...\n")
            colonize_output = await commands.colonize()
            print(colonize_output)
            print("\n" + "="*70 + "\n")

        output = await commands.init(args.goal)
        print(output)
        return 0

    # ============================================================
    # plan
    # ============================================================
    elif args.command == "plan":
        output = await commands.plan()
        print(output)
        return 0

    # ============================================================
    # phase
    # ============================================================
    elif args.command == "phase":
        output = await commands.phase(args.id)
        print(output)
        return 0

    # ============================================================
    # execute
    # ============================================================
    elif args.command == "execute":
        output = await commands.execute(args.phase_id)
        print(output)
        return 0

    # ============================================================
    # review
    # ============================================================
    elif args.command == "review":
        output = await commands.review(args.phase_id)
        print(output)
        return 0

    # ============================================================
    # focus
    # ============================================================
    elif args.command == "focus":
        output = await commands.focus(args.area, args.strength)
        print(output)
        return 0

    # ============================================================
    # redirect
    # ============================================================
    elif args.command == "redirect":
        output = await commands.redirect(args.pattern, args.strength)
        print(output)
        return 0

    # ============================================================
    # feedback
    # ============================================================
    elif args.command == "feedback":
        output = await commands.feedback(args.message, args.strength)
        print(output)
        return 0

    # ============================================================
    # status
    # ============================================================
    elif args.command == "status":
        output = await commands.status()
        print(output)
        return 0

    # ============================================================
    # colonize
    # ============================================================
    elif args.command == "colonize":
        output = await commands.colonize()
        print(output)
        return 0

    # ============================================================
    # pause-colony
    # ============================================================
    elif args.command == "pause-colony":
        output = await commands.pause_colony()
        print(output)
        return 0

    # ============================================================
    # resume-colony
    # ============================================================
    elif args.command == "resume-colony":
        output = await commands.resume_colony()
        print(output)
        return 0

    # ============================================================
    # repl
    # ============================================================
    elif args.command == "repl":
        try:
            from .repl import AetherREPL
        except ImportError:
            from aether.repl import AetherREPL

        repl = AetherREPL(commands)
        await repl.run(history_enabled=args.history)
        return 0

    # ============================================================
    # memory
    # ============================================================
    elif args.command == "memory":
        # Import memory system
        try:
            from .memory.triple_layer_memory import TripleLayerMemory
        except ImportError:
            try:
                from aether.memory.triple_layer_memory import TripleLayerMemory
            except ImportError:
                from memory.triple_layer_memory import TripleLayerMemory

        # Get memory from commands if available
        if not hasattr(commands, 'memory_layer') or commands.memory_layer is None:
            memory = TripleLayerMemory()
        else:
            memory = commands.memory_layer

        if args.memory_action == "status":
            status = memory.get_status()
            print("╔══════════════════════════════════════════════════════════════╗")
            print("║                  🧠 Memory System Status                    ║")
            print("╠══════════════════════════════════════════════════════════════╣")
            print()

            # Working memory
            working = status["triple_layer_memory"]["working"]
            print(f"📝 WORKING MEMORY")
            print(f"   Items: {working['item_count']}")
            print(f"   Tokens: {working['used_tokens']}/{working['max_tokens']} ({working['usage_percent']:.1f}%)")
            print()

            # Short-term memory
            short_term = status["triple_layer_memory"]["short_term"]
            print(f"📚 SHORT-TERM MEMORY")
            print(f"   Sessions: {short_term['session_count']}/{short_term['max_sessions']}")
            print(f"   Saved tokens: {short_term['total_saved_tokens']}")
            print(f"   Avg compression: {short_term['avg_compression_ratio']}x")
            print()

            # Long-term memory
            long_term = status["triple_layer_memory"]["long_term"]
            print(f"💾 LONG-TERM MEMORY")
            print(f"   Total patterns: {long_term['total_patterns']}")
            if long_term['categories']:
                print(f"   Categories:")
                for cat, count in long_term['categories'].items():
                    if count > 0:
                        print(f"      {cat}: {count}")
            print()

            print("╚══════════════════════════════════════════════════════════════╝")
            return 0

        elif args.memory_action == "working":
            if hasattr(args, 'query') and args.query:
                results = await memory.search_working(args.query, limit=10)
                print(f"📝 Working Memory Search: '{args.query}'")
                print()
                for item in results:
                    print(f"  [{item.metadata.get('type', 'general')}] {item.content}")
                if not results:
                    print("  No matches found")
            else:
                status = memory.working.get_status()
                print(f"📝 Working Memory ({status['item_count']} items, {status['used_tokens']} tokens)")
                print()
                for item in list(memory.working.items.values())[:10]:
                    print(f"  [{item.metadata.get('type', 'general')}] {item.content[:60]}...")
                if status['item_count'] > 10:
                    print(f"  ... and {status['item_count'] - 10} more items")
            print()
            return 0

        elif args.memory_action == "short-term":
            if hasattr(args, 'query') and args.query:
                results = await memory.short_term.search(args.query, limit=5)
                print(f"📚 Short-Term Memory Search: '{args.query}'")
                print()
                for session in results:
                    print(f"  {session.session_id}")
                    print(f"  {session.content}")
                if not results:
                    print("  No matches found")
            else:
                status = memory.short_term.get_status()
                print(f"📚 Short-Term Memory ({status['session_count']} sessions)")
                print()
                for session in memory.short_term.sessions.values():
                    print(f"  {session.session_id}")
                    print(f"  Compression: {session.compression_ratio:.2f}x")
                    print(f"  {session.content[:80]}...")
                    print()
            return 0

        elif args.memory_action == "long-term":
            category = getattr(args, 'category', None)
            results = await memory.long_term.search(args.query, categories=[category] if category else None, limit=10)
            print(f"💾 Long-Term Memory Search: '{args.query}'")
            if category:
                print(f"   Category: {category}")
            print()
            for pattern in results:
                print(f"  [{pattern.category}] {pattern.key}")
                print(f"  Confidence: {pattern.confidence:.2f} | Occurrences: {pattern.occurrences}")
                print(f"  {pattern.value}")
                print()
            if not results:
                print("  No matches found")
            print()
            return 0

        elif args.memory_action == "compress":
            print("🔄 Compressing working memory to short-term...")
            session_id = await memory.compress_to_short_term({
                "phase": getattr(args, 'phase', 'manual'),
                "trigger": "manual",
                "timestamp": datetime.now().isoformat()
            })
            print(f"   Session: {session_id}")
            print(f"   Working memory: {memory.working.item_count} items, {memory.working.used_tokens} tokens")
            print()
            return 0

        else:
            memory_parser.print_help()
            return 0

    return 0


if __name__ == "__main__":
    sys.exit(main())
