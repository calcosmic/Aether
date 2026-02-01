# Technology Stack

**Analysis Date:** 2025-02-01

## Languages

**Primary:**
- Python 3.8+ - Core implementation (entire `.aether/` codebase)

**Secondary:**
- Markdown - Documentation (README.md, ARCHITECTURE.md, etc.)
- JSON - Data persistence and configuration

## Runtime

**Environment:**
- Python 3.8+ (async/await required)
- Operating System: Cross-platform (macOS, Linux, Windows)

**Package Manager:**
- Not explicitly declared (no requirements.txt, pyproject.toml, or setup.py found)
- Uses standard library extensively
- Optional dependencies: `numpy`, `sentence-transformers` for semantic features

## Frameworks

**Core:**
- asyncio (Python stdlib) - Async/await concurrency model throughout
- dataclasses (Python stdlib) - Data structures
- typing (Python stdlib) - Type hints (TypedDict, Literal, TYPE_CHECKING)
- enum (Python stdlib) - Enumerations (PheromoneType, ErrorCategory, etc.)

**Testing:**
- pytest - Referenced in `worker_ants.py` for test patterns
- hypothesis - Property-based testing framework referenced

**Build/Dev:**
- None detected (no build system configured)

## Key Dependencies

**Critical:**
- asyncio - Core async framework for all agent operations
- dataclasses - Data structures throughout (`@dataclass` used extensively)
- typing - Type system for complex types

**Infrastructure:**
- json - Data persistence (`.aether/data/`, `.aether/memory/`, `.aether/checkpoints/`)
- pathlib - Path operations for file system access
- collections (OrderedDict, defaultdict) - Specialized data structures
- datetime/timedelta - Time-based operations
- hashlib - Content hashing for IDs
- re - Regular expressions for pattern matching

**Optional (for semantic features):**
- numpy - Vector operations for embeddings
- sentence-transformers - Text embeddings (model: all-MiniLM-L6-v2)

## Configuration

**Environment:**
- File-based configuration (JSON files)
- No environment variable usage detected
- No .env file pattern

**Build:**
- No build configuration files detected
- Direct Python module execution (`python -m aether`)

## Platform Requirements

**Development:**
- Python 3.8+ (async/await, dataclasses, typing features)
- Optional: numpy, sentence-transformers for semantic layer
- Standard library only for core functionality

**Production:**
- Python runtime
- File system access for state persistence (`.aether/` directory)
- No external services required

---

*Stack analysis: 2025-02-01*
