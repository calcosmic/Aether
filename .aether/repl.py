#!/usr/bin/env python3
"""
Aether Queen Ant Colony - Interactive REPL

Read-Eval-Print Loop for real-time colony control.
Provides command history, tab completion, and live status.
"""

import asyncio
import sys
from typing import Optional, List
from datetime import datetime

try:
    from prompt_toolkit import PromptSession
    from prompt_toolkit.completion import WordCompleter
    from prompt_toolkit.history import FileHistory
    from prompt_toolkit.auto_suggest import AutoSuggestFromHistory
    from prompt_toolkit.formatted_text import HTML
    PROMPT_TOOLKIT_AVAILABLE = True
except ImportError:
    PROMPT_TOOLKIT_AVAILABLE = False

try:
    import readline
    READLINE_AVAILABLE = True
except ImportError:
    READLINE_AVAILABLE = False

from datetime import datetime


class AetherREPL:
    """
    Interactive REPL for Aether Queen Ant Colony

    Features:
    - Command history (up/down arrows)
    - Tab completion for commands and arguments
    - Live status updates
    - Visual feedback for colony activity
    - Clean prompt interface
    """

    def __init__(self, commands, history_file: str = ".aether/repl_history.txt"):
        """
        Initialize REPL

        Args:
            commands: InteractiveCommands instance
            history_file: Path to command history file
        """
        self.commands = commands
        self.history_file = history_file
        self.running = False
        self.history_enabled = False

        # Available commands
        self.command_names = [
            "init", "plan", "phase", "execute", "review",
            "focus", "redirect", "feedback", "status", "memory",
            "colonize", "pause-colony", "resume-colony",
            "help", "quit", "exit", "clear"
        ]

        # Command descriptions for help
        self.command_help = {
            "init": "Initialize new project: init <goal>",
            "plan": "Show all phases with tasks",
            "phase": "Show phase details: phase [id]",
            "execute": "Execute a phase: execute <id>",
            "review": "Review completed phase: review <id>",
            "focus": "Guide colony attention: focus <area>",
            "redirect": "Warn colony away: redirect <pattern>",
            "feedback": "Provide guidance: feedback <message>",
            "status": "Show colony status",
            "memory": "Memory system: memory [status|working|short-term|long-term|compress]",
            "colonize": "Analyze codebase before starting",
            "pause-colony": "Save session mid-phase",
            "resume-colony": "Restore saved session",
            "help": "Show this help message",
            "quit": "Exit REPL",
            "exit": "Exit REPL",
            "clear": "Clear screen"
        }

    async def run(self, history_enabled: bool = True):
        """
        Run the REPL loop

        Args:
            history_enabled: Enable command history
        """
        self.history_enabled = history_enabled
        self.running = True

        # Show welcome message
        self._show_welcome()

        # Use prompt_toolkit if available, otherwise fallback
        if PROMPT_TOOLKIT_AVAILABLE:
            await self._run_with_prompt_toolkit()
        elif READLINE_AVAILABLE:
            await self._run_with_readline()
        else:
            await self._run_basic()

    def _show_welcome(self):
        """Display welcome message"""
        print()
        print("╔══════════════════════════════════════════════════════════════╗")
        print("║          🐜 Aether Queen Ant Colony - Interactive REPL        ║")
        print("╠══════════════════════════════════════════════════════════════╣")
        print("║  The first AI system where Worker Ants autonomously spawn     ║")
        print("║  other Worker Ants without human orchestration.              ║")
        print("╠══════════════════════════════════════════════════════════════╣")
        print("║  Type 'help' for commands | 'quit' to exit                   ║")
        print("╚══════════════════════════════════════════════════════════════╝")
        print()

    async def _run_with_prompt_toolkit(self):
        """Run REPL with prompt_toolkit (best experience)"""
        # Create completer
        completer = WordCompleter(
            self.command_names,
            ignore_case=True,
            sentence=True
        )

        # Create session
        session = PromptSession(
            history=FileHistory(self.history_file) if self.history_enabled else None,
            auto_suggest=AutoSuggestFromHistory(),
            enable_history_search=True,
            completer=completer,
        )

        # Main loop
        while self.running:
            try:
                # Get input with styled prompt
                user_input = await session.prompt_async(
                    HTML("<style fg='ansigreen' bold>🐜 aether</style><style fg='ansiblue'> &gt; </style>"),
                    async_=True
                )

                if not user_input.strip():
                    continue

                # Execute command
                await self._execute_command(user_input)

            except KeyboardInterrupt:
                print("\n\nUse 'quit' or 'exit' to close REPL")
            except EOFError:
                print("\n\nGoodbye! 👋")
                break
            except Exception as e:
                print(f"\n❌ Error: {e}\n")

    async def _run_with_readline(self):
        """Run REPL with readline (basic experience)"""
        # Setup readline
        if self.history_enabled:
            try:
                readline.read_history_file(self.history_file)
            except FileNotFoundError:
                pass

        readline.parse_and_bind("tab: complete")
        readline.set_completer(self._readline_completer)

        # Main loop
        while self.running:
            try:
                user_input = input("🐜 aether> ").strip()

                if not user_input:
                    continue

                # Save to history
                if self.history_enabled:
                    readline.write_history_file(self.history_file)

                # Execute command
                await self._execute_command(user_input)

            except KeyboardInterrupt:
                print("\n\nUse 'quit' or 'exit' to close REPL")
            except EOFError:
                print("\n\nGoodbye! 👋")
                break
            except Exception as e:
                print(f"\n❌ Error: {e}\n")

    async def _run_basic(self):
        """Run REPL with basic input only"""
        # Main loop
        while self.running:
            try:
                user_input = input("🐜 aether> ").strip()

                if not user_input:
                    continue

                # Execute command
                await self._execute_command(user_input)

            except KeyboardInterrupt:
                print("\n\nUse 'quit' or 'exit' to close REPL")
            except EOFError:
                print("\n\nGoodbye! 👋")
                break
            except Exception as e:
                print(f"\n❌ Error: {e}\n")

    def _readline_completer(self, text: str, state: int) -> Optional[str]:
        """Readline tab completion"""
        options = [cmd for cmd in self.command_names if cmd.startswith(text)]
        if state < len(options):
            return options[state]
        return None

    async def _execute_command(self, user_input: str):
        """
        Parse and execute command

        Args:
            user_input: Raw user input string
        """
        parts = user_input.strip().split()
        if not parts:
            return

        cmd = parts[0].lower()
        args = parts[1:]

        # Handle commands
        if cmd in ["quit", "exit"]:
            self.running = False
            print("Goodbye! 👋\n")

        elif cmd == "clear":
            print("\n" * 100)

        elif cmd == "help":
            self._show_help()

        elif cmd == "init":
            if len(args) < 1:
                print("❌ Usage: init <goal>")
                return
            goal = " ".join(args)
            output = await self.commands.init(goal)
            print(output)

        elif cmd == "plan":
            output = await self.commands.plan()
            print(output)

        elif cmd == "phase":
            phase_id = int(args[0]) if args else None
            output = await self.commands.phase(phase_id)
            print(output)

        elif cmd == "execute":
            if len(args) < 1:
                print("❌ Usage: execute <phase_id>")
                return
            output = await self.commands.execute(int(args[0]))
            print(output)

        elif cmd == "review":
            if len(args) < 1:
                print("❌ Usage: review <phase_id>")
                return
            output = await self.commands.review(int(args[0]))
            print(output)

        elif cmd == "focus":
            if len(args) < 1:
                print("❌ Usage: focus <area>")
                return
            area = " ".join(args)
            output = await self.commands.focus(area)
            print(output)

        elif cmd == "redirect":
            if len(args) < 1:
                print("❌ Usage: redirect <pattern>")
                return
            pattern = " ".join(args)
            output = await self.commands.redirect(pattern)
            print(output)

        elif cmd == "feedback":
            if len(args) < 1:
                print("❌ Usage: feedback <message>")
                return
            message = " ".join(args)
            output = await self.commands.feedback(message)
            print(output)

        elif cmd == "status":
            output = await self.commands.status()
            print(output)

        elif cmd == "memory":
            # Memory subcommands
            if not args:
                # Default: show memory status
                output = await self.commands.memory()
                print(output)
            else:
                # Import memory system
                try:
                    from .memory.triple_layer_memory import TripleLayerMemory
                except ImportError:
                    from aether.memory.triple_layer_memory import TripleLayerMemory

                # Get or create memory
                if not hasattr(self.commands, 'memory_layer') or self.commands.memory_layer is None:
                    memory = TripleLayerMemory()
                else:
                    memory = self.commands.memory_layer

                subcommand = args[0].lower()

                if subcommand == "status":
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

                elif subcommand == "working":
                    query = " ".join(args[1:]) if len(args) > 1 else None
                    if query:
                        results = await memory.search_working(query, limit=10)
                        print(f"📝 Working Memory Search: '{query}'")
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

                elif subcommand == "short-term":
                    query = " ".join(args[1:]) if len(args) > 1 else None
                    if query:
                        results = await memory.short_term.search(query, limit=5)
                        print(f"📚 Short-Term Memory Search: '{query}'")
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

                elif subcommand == "long-term":
                    if len(args) < 2:
                        print("❌ Usage: memory long-term <query> [--category <cat>]")
                        return
                    query = " ".join(args[1:])
                    # Simple parsing (doesn't handle --category flag in basic form)
                    results = await memory.long_term.search(query, limit=10)
                    print(f"💾 Long-Term Memory Search: '{query}'")
                    print()
                    for pattern in results:
                        print(f"  [{pattern.category}] {pattern.key}")
                        print(f"  Confidence: {pattern.confidence:.2f} | Occurrences: {pattern.occurrences}")
                        print(f"  {pattern.value}")
                        print()
                    if not results:
                        print("  No matches found")
                    print()

                elif subcommand == "compress":
                    print("🔄 Compressing working memory to short-term...")
                    session_id = await memory.compress_to_short_term({
                        "phase": "manual",
                        "trigger": "repl",
                        "timestamp": datetime.now().isoformat()
                    })
                    print(f"   Session: {session_id}")
                    print(f"   Working memory: {memory.working.item_count} items, {memory.working.used_tokens} tokens")
                    print()

                else:
                    print(f"❌ Unknown memory subcommand: {subcommand}")
                    print("   Available: status, working, short-term, long-term, compress")
                    print()

        elif cmd == "colonize":
            output = await self.commands.colonize()
            print(output)

        elif cmd == "pause-colony":
            output = await self.commands.pause_colony()
            print(output)

        elif cmd == "resume-colony":
            output = await self.commands.resume_colony()
            print(output)

        else:
            print(f"❌ Unknown command: {cmd}")
            print("Type 'help' for available commands\n")

    def _show_help(self):
        """Display help message"""
        print()
        print("╔══════════════════════════════════════════════════════════════╗")
        print("║                      Available Commands                     ║")
        print("╠══════════════════════════════════════════════════════════════╣")
        print()

        # Group commands
        core = ["init", "plan", "phase", "execute", "review"]
        guidance = ["focus", "redirect", "feedback"]
        info = ["status", "memory", "colonize"]
        session = ["pause-colony", "resume-colony"]
        system = ["help", "clear", "quit", "exit"]

        for cmd_list in [core, guidance, info, session, system]:
            for cmd in cmd_list:
                description = self.command_help.get(cmd, "")
                print(f"  {cmd:15} {description}")
            print()

        print("╚══════════════════════════════════════════════════════════════╝")
        print()

        # Show usage tips
        print("💡 Tips:")
        print("  • Use TAB to complete commands")
        print("  • Use UP/DOWN arrows for command history")
        print("  • Use 'clear' to clean the screen")
        print()
        print("🧠 Memory Subcommands:")
        print("  memory status     - Show all memory layers")
        print("  memory working    - Show working memory")
        print("  memory short-term - Show compressed sessions")
        print("  memory long-term <query> - Search persistent knowledge")
        print("  memory compress   - Compress working to short-term")
        print()


async def main():
    """Main entry point for standalone REPL"""
    from .interactive_commands import InteractiveCommands

    commands = InteractiveCommands()
    repl = AetherREPL(commands)
    await repl.run(history_enabled=True)


if __name__ == "__main__":
    asyncio.run(main())
