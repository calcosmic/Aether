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
    # /ant:memory
    # ============================================================
    memory_parser = subparsers.add_parser(
        "memory",
        help="Show learned patterns and memory",
        description="Display colony memory including learned preferences and constraints."
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
        from aether.interactive_commands import InteractiveCommands

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
    # memory
    # ============================================================
    elif args.command == "memory":
        output = await commands.memory()
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

    return 0


if __name__ == "__main__":
    sys.exit(main())
