# Coding Conventions

**Analysis Date:** 2025-02-01

## Naming Patterns

**Files:**
- `snake_case` for all Python files (e.g., `worker_ants.py`, `pheromone_system.py`, `error_prevention.py`)
- `__init__.py` for package initialization in `.aether/memory/` directory
- Main entry point: `__main__.py` in `.aether/` directory

**Functions:**
- `snake_case` for all function and method names
- Async functions use `async def` prefix with same `snake_case` naming
- Private methods prefix with underscore: `_load_state()`, `_find_or_create_pattern()`, `_persist()`
- Factory functions use `create_` prefix: `create_colony()`, `create_pheromone_layer()`, `create_phase_engine()`

**Variables:**
- `snake_case` for all variables
- Class attributes use `snake_case` (e.g., `current_task`, `last_activity`, `error_ledger`)
- Constants use `UPPER_SNAKE_CASE`: `SENSITIVITY_PROFILES`, `ErrorCategory`, `PheromoneType`

**Types:**
- `PascalCase` for class names (e.g., `WorkerAnt`, `PheromoneSignal`, `ErrorRecord`, `MetaLearner`)
- `PascalCase` for Enum classes (e.g., `ErrorCategory`, `ErrorSeverity`, `TaskOutcome`, `PheromoneType`)
- Type aliases use `PascalCase`: `MemoryLayer`, `Colony`, `SpecialistType`

## Code Style

**Formatting:**
- No explicit formatting tool detected (no `.prettierrc`, `black.toml`, or `ruff.toml`)
- 4-space indentation used throughout
- Line length appears to be around 100-120 characters based on observed code
- Double quotes for strings: `"string"`
- Consistent spacing around operators and after commas

**Linting:**
- No explicit linting configuration detected (no `.eslintrc`, `pylintrc`, or `flake8` config)
- Code follows PEP 8 style conventions for Python
- Type hints used extensively: `List[str]`, `Dict[str, Any]`, `Optional[bool]`

## Import Organization

**Order:**
1. Standard library imports (`from typing import...`, `from datetime import...`, `import json`)
2. Third-party imports (none detected in this codebase)
3. Local/relative imports with try/except fallback pattern

**Import Pattern (Observed throughout):**
```python
try:
    from .worker_ants import Colony, create_colony
    from .pheromone_system import PheromoneLayer, PheromoneType
except ImportError:
    from worker_ants import Colony, create_colony
    from pheromone_system import PheromoneLayer, PheromoneType
```

**Path Aliases:**
- No explicit path aliases configured (no `pyproject.toml` or `setup.py` with aliases)
- Uses relative imports with `.` prefix for package imports
- Falls back to absolute imports for standalone execution

**TYPE_CHECKING Pattern:**
```python
from typing import TYPE_CHECKING
if TYPE_CHECKING:
    from .memory.triple_layer_memory import TripleLayerMemory
```

## Error Handling

**Patterns:**
- Extensive use of try/except for import fallbacks
- Specific exception catching in async operations
- ErrorLedger system for structured error tracking in `.aether/error_prevention.py`
- Error categorization using `ErrorCategory` enum (SYNTAX, IMPORT, RUNTIME, SPAWNING, CAPABILITY, etc.)
- Severity levels: CRITICAL, HIGH, MEDIUM, LOW, INFO

**Error Logging Pattern:**
```python
try:
    # operation
except Exception as e:
    if self.error_ledger:
        log_exception(
            self.error_ledger,
            e,
            symptom="Operation failed",
            agent_id=self.agent_id,
            task_context=task.description,
            category=ErrorCategory.SPAWNING
        )
    return None
```

**Return Values on Error:**
- Functions return `None`, `[]`, or `{}` on failure rather than raising
- Graceful degradation throughout the codebase
- Empty returns detected in: `short_term_memory.py`, `long_term_memory.py`, `semantic_layer.py`, `worker_ants.py`

## Logging

**Framework:** No external logging framework (no `structlog`, `loguru` configuration detected)
- Uses `print()` statements for demo output and debugging
- ErrorLedger system for structured error tracking
- Custom `log_exception()` function in `error_prevention.py`

**Patterns:**
- Demo functions use extensive `print()` with emoji indicators (e.g., `"✅ "`, `"🐜 "`, `"📊 "`)
- Status reports use formatted strings with headers and separators: `"=" * 60`
- Errors logged to ErrorLedger with full context (file, line number, function, stack trace)
- Status methods return dictionaries with structured data

**When to Log:**
- Import failures (silent fallback, no log)
- ErrorLedger records (full context with traceback)
- Demo/showcase functions (verbose print output)
- State transitions and phase changes

## Comments

**When to Comment:**
- Module docstrings at top of every file with research citations
- Class docstrings explaining purpose and usage
- Method docstrings for all public APIs
- Inline comments for complex logic (Bayesian calculations, decay formulas)
- Section separators with `# ===` markers for logical grouping

**JSDoc/TSDoc (Python Docstrings):**
- Google-style docstrings used throughout
- Args sections for all parameters
- Returns sections for return values
- Usage examples in docstrings for main APIs
- Research citations in module docstrings

**Docstring Pattern:**
```python
def record_spawn(
    self,
    parent_agent: str,
    task_description: str,
    task_category: str,
    specialist_type: str,
    capability_gap: Set[str],
    inherited_context: Dict[str, Any]
) -> str:
    """
    Record a specialist spawning event.

    Returns the event ID for later outcome tracking.

    Args:
        parent_agent: Which Worker Ant caste spawned this
        task_description: Description of the task
        task_category: e.g., "database", "frontend", "security"
        specialist_type: Type of specialist spawned
        capability_gap: Capabilities that triggered spawning
        inherited_context: Context passed to specialist

    Returns:
        Event ID for outcome tracking
    """
```

## Function Design

**Size:** No explicit size limit observed
- Large functions exist (e.g., demo functions ~200 lines)
- Methods typically 20-50 lines
- Complex operations split into multiple private methods

**Parameters:**
- Extensive use of type hints
- Optional parameters with defaults: `strength: float = 0.5`, `metadata: Optional[Dict[str, Any]] = None`
- Dataclass parameters for complex state (e.g., `PheromoneSignal`, `ErrorRecord`, `Task`)
- `**kwargs` pattern rarely used; prefers explicit parameters

**Return Values:**
- Functions return typed values: `-> str`, `-> Dict[str, Any]`, `-> List[PheromoneSignal]`
- Return dictionaries for structured data: `{"message": "...", "goal": "...", "phase": {...}}`
- Return `None` for failure cases (not exceptions)
- Return empty containers (`[]`, `{}`) for "no results" cases

## Module Design

**Exports:** No explicit `__all__` exports detected
- Public APIs exposed through factory functions: `create_colony()`, `create_pheromone_layer()`
- Main entry point via `__main__.py` with demo functions
- Singleton pattern with `get_` functions: `get_meta_learner()`

**Barrel Files:**
- `.aether/memory/__init__.py` imports from submodules
- Each module imports its dependencies with try/except fallback

## Data Structure Patterns

**Dataclasses (Extensive Use):**
- `@dataclass` decorator for structured data throughout
- `field(default_factory=list)` for mutable defaults
- `frozen=True` for immutable data (e.g., `Capability` dataclass)
- Methods to convert to/from dict: `to_dict()`, `from_dict()`

**Enum Usage:**
- Enums for fixed categories: `PheromoneType`, `ErrorCategory`, `ErrorSeverity`, `TaskOutcome`
- Enum values are lowercase strings: `INIT = "init"`, `FOCUS = "focus"`

**Async/Await Pattern:**
- All I/O and colony operations are async
- Event loop management with `asyncio.run()` in `__main__`
- Async methods in main classes: `async def init()`, `async def focus()`, `async def status()`

## State Management

**Persistence Pattern:**
- JSON-based persistence to `.aether/` directory
- State loaded in `__init__` via `_load()` methods
- State saved on updates via `_persist()` methods
- File paths configurable with defaults: `.aether/memory/`, `.aether/errors/`

**In-Memory State:**
- Class attributes for runtime state: `self.current_task`, `self.last_activity`
- Lists for collections: `self.subagents: List[Subagent] = []`
- Dicts for lookups: `self.worker_ants: Dict[str, WorkerAnt] = {}`
- Optional attributes for lazy initialization: `self.memory_layer: Optional[TripleLayerMemory] = None`

## Factory Pattern

**Factory Functions:**
- `create_<entity>()` functions for object creation
- Used instead of direct class instantiation for dependency injection
- Example: `create_colony(memory_layer=None)` vs `Colony(memory_layer=None)`

**Factories Detected:**
- `create_colony()` in `worker_ants.py`
- `create_pheromone_layer()` in `pheromone_system.py`
- `create_phase_engine()` in `phase_engine.py`
- `create_queen_ant_system()` in `queen_ant_system.py`
- `create_semantic_layer()` in `semantic_layer.py`

## Demo/Testing Pattern

**Demo Functions:**
- Each major module has `demo_<module>()` function
- Demo functions use `if __name__ == "__main__":` guard
- Demos showcase features with verbose print output
- Async demos use `asyncio.run(demo_<module>())`

**Demo Output Format:**
```python
print("=" * 60)
print("STEP 1: Description")
print("=" * 60)
print(f"  Output with indentation")
```

---

*Convention analysis: 2025-02-01*
