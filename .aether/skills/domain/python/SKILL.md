---
name: python
description: Use when the project uses Python for backend, scripting, or data work
type: domain
domains: [backend, scripting, data]
agent_roles: [builder]
detect_files: ["*.py", "requirements.txt", "pyproject.toml"]
priority: normal
version: "1.0"
---

# Python Best Practices

## Project Setup

Use virtual environments for every project -- never install packages globally. Prefer `pyproject.toml` over `setup.py` for project configuration. Pin dependency versions in `requirements.txt` or use a lock file via `pip-tools`, `poetry`, or `uv`.

Structure packages with `__init__.py` files. Use `src/` layout for libraries to prevent accidental imports from the project root.

## Code Style

Follow PEP 8. Use a formatter (black, ruff format) and linter (ruff, flake8) -- do not manually enforce style. Type hints are strongly encouraged for function signatures and return values. Use `mypy` or `pyright` for type checking.

Use f-strings for string formatting, not `%` or `.format()`. They are faster and more readable.

## Error Handling

Catch specific exceptions, never bare `except:`. This swallows `KeyboardInterrupt` and `SystemExit`, making the program impossible to stop. Always `except SomeError as e:` and log or re-raise.

Use context managers (`with` statements) for resource management: files, database connections, locks. They guarantee cleanup even when exceptions occur.

## Functions and Classes

Keep functions focused and short. Use keyword arguments for functions with more than 3 parameters to avoid positional ambiguity. Provide default values where sensible.

Prefer composition over deep inheritance hierarchies. Dataclasses (`@dataclass`) or named tuples for simple data containers -- avoid dictionaries when the shape is known.

## Common Gotchas

Mutable default arguments are shared across calls: `def f(items=[])` is a bug. Use `def f(items=None): items = items or []` instead.

String concatenation in loops is O(n^2). Use `"".join(parts)` or list comprehensions. List comprehensions are preferred over `map`/`filter` for readability.

## Testing

Use `pytest` over `unittest`. Write test files with `test_` prefix. Use fixtures for setup/teardown. Parametrize tests with `@pytest.mark.parametrize` to cover multiple cases without duplicating test code.

## Performance

Profile before optimizing -- use `cProfile` or `py-spy`. For CPU-bound work, consider `multiprocessing` (the GIL prevents true threading for CPU tasks). For I/O-bound work, use `asyncio` or `concurrent.futures.ThreadPoolExecutor`.
