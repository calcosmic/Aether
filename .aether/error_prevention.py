"""
Queen Ant Colony - Error Prevention System

Foundation for learning from mistakes and preventing recurring errors.
Tracks error patterns, aggregates occurrences, and flags recurring issues.

Based on research:
- Error prevention systems that learn from mistakes
- Pattern recognition for recurring issues
- Proactive error flagging

Features:
- ErrorRecord dataclass for structured error tracking
- ErrorLedger with persistence to .aether/errors/
- Flagging system for recurring errors (3+ occurrences)
- Integration with WorkerAnt error handling
"""

from typing import List, Dict, Any, Optional, Set
from dataclasses import dataclass, field, asdict
from enum import Enum
from datetime import datetime
import json
import os
from pathlib import Path
from collections import defaultdict


# ============================================================================
# ERROR DATA STRUCTURES
# ============================================================================

class ErrorCategory(Enum):
    """Categories of errors for pattern analysis"""
    # Technical errors
    SYNTAX = "syntax"                    # Code syntax errors
    IMPORT = "import"                    # Import/module errors
    RUNTIME = "runtime"                  # Runtime exceptions
    TYPE = "type"                        # Type errors
    # Agent-specific errors
    SPAWNING = "spawning"                # Subagent spawning failures
    CAPABILITY = "capability"            # Capability detection errors
    PHASE = "phase"                      # Phase execution errors
    VERIFICATION = "verification"        # Verification failures
    # External errors
    API = "api"                          # External API failures
    NETWORK = "network"                  # Network/connection errors
    FILE = "file"                        # File I/O errors
    # Quality errors
    LOGIC = "logic"                      # Logic bugs
    PERFORMANCE = "performance"          # Performance issues
    SECURITY = "security"                # Security vulnerabilities


class ErrorSeverity(Enum):
    """Severity levels for errors"""
    CRITICAL = "critical"    # Blocks progress, must fix immediately
    HIGH = "high"            # Significant impact, should fix soon
    MEDIUM = "medium"        # Moderate impact, fix when convenient
    LOW = "low"              # Minor impact, note for future
    INFO = "info"            # Informational, no action needed


@dataclass
class ErrorRecord:
    """
    Structured record of an error occurrence

    Tracks the symptom, root cause analysis, fix applied,
    prevention strategy, and categorization.
    """
    # Identification
    error_id: str = field(default_factory=lambda: f"err_{datetime.now().strftime('%Y%m%d_%H%M%S_%f')[:17]}")
    timestamp: datetime = field(default_factory=datetime.now)

    # What happened
    symptom: str = ""              # What the user/agent observed
    error_type: str = ""           # Exception type or error name
    error_message: str = ""        # Full error message

    # Where it happened
    file_path: str = ""            # File where error occurred
    line_number: int = 0           # Line number
    function: str = ""             # Function/method name

    # Why it happened (root cause analysis)
    root_cause: str = ""           # Underlying cause
    category: ErrorCategory = ErrorCategory.RUNTIME
    severity: ErrorSeverity = ErrorSeverity.MEDIUM

    # How to fix and prevent
    fix: str = ""                  # What fixed it (if resolved)
    prevention: str = ""           # How to prevent in future
    code_snippet: str = ""         # Relevant code context

    # Agent context
    agent_id: str = ""             # Which agent encountered this
    task_context: str = ""         # What task was being worked on
    phase: str = ""                # Which phase (if applicable)

    # Resolution
    resolved: bool = False
    resolved_at: Optional[datetime] = None
    verified_fix: bool = False     # Has the fix been verified?

    # Additional context
    stack_trace: str = ""
    additional_notes: str = ""
    tags: List[str] = field(default_factory=list)

    def to_dict(self) -> Dict[str, Any]:
        """Convert to dictionary for JSON serialization"""
        data = asdict(self)
        data['timestamp'] = self.timestamp.isoformat()
        data['category'] = self.category.value
        data['severity'] = self.severity.value
        if self.resolved_at:
            data['resolved_at'] = self.resolved_at.isoformat()
        return data

    @classmethod
    def from_dict(cls, data: Dict[str, Any]) -> 'ErrorRecord':
        """Create from dictionary"""
        if isinstance(data.get('timestamp'), str):
            data['timestamp'] = datetime.fromisoformat(data['timestamp'])
        if isinstance(data.get('resolved_at'), str):
            data['resolved_at'] = datetime.fromisoformat(data['resolved_at'])
        if isinstance(data.get('category'), str):
            data['category'] = ErrorCategory(data['category'])
        if isinstance(data.get('severity'), str):
            data['severity'] = ErrorSeverity(data['severity'])
        return cls(**data)

    def get_summary(self) -> str:
        """Get brief summary of the error"""
        status = "RESOLVED" if self.resolved else "OPEN"
        return f"[{status}] {self.error_type}: {self.symptom[:100]}"

    def get_age(self) -> timedelta:
        """Get age of the error"""
        return datetime.now() - self.timestamp


from datetime import timedelta


@dataclass
class ErrorPattern:
    """
    Aggregated pattern of similar errors

    Combines multiple ErrorRecords that share common characteristics
    to identify recurring issues that need systematic prevention.
    """
    pattern_id: str
    category: ErrorCategory
    error_type: str
    occurrences: List[ErrorRecord] = field(default_factory=list)

    # Flagging
    flagged: bool = False
    flag_threshold: int = 3  # Flag after N occurrences
    flag_reason: str = ""

    # Pattern analysis
    first_occurrence: datetime = field(default_factory=datetime.now)
    last_occurrence: datetime = field(default_factory=datetime.now)
    frequency_trend: str = "stable"  # increasing, stable, decreasing

    # Systematic fix
    systematic_fix: str = ""
    systematic_prevention: str = ""
    fix_deployed: bool = False

    @property
    def occurrence_count(self) -> int:
        return len(self.occurrences)

    @property
    def should_flag(self) -> bool:
        return self.occurrence_count >= self.flag_threshold

    def add_occurrence(self, error: ErrorRecord):
        """Add a new occurrence to this pattern"""
        self.occurrences.append(error)
        self.last_occurrence = error.timestamp

        # Check if we should flag
        if self.should_flag and not self.flagged:
            self.flagged = True
            self.flag_reason = f"Occurred {self.occurrence_count} times (threshold: {self.flag_threshold})"

    def get_summary(self) -> str:
        """Get summary of this pattern"""
        flag_icon = "🚩" if self.flagged else ""
        return f"{flag_icon} {self.error_type}: {self.occurrence_count} occurrences (last: {self.last_occurrence.strftime('%Y-%m-%d')})"


# ============================================================================
# ERROR LEDGER
# ============================================================================

class ErrorLedger:
    """
    Persistent ledger of all errors encountered by the colony

    Provides:
    - Error logging and persistence
    - Pattern detection and aggregation
    - Flagging of recurring issues
    - Query and filtering capabilities
    """

    def __init__(self, storage_dir: str = ".aether/errors"):
        self.storage_dir = Path(storage_dir)
        self.storage_dir.mkdir(parents=True, exist_ok=True)

        # In-memory storage
        self.errors: List[ErrorRecord] = []
        self.patterns: Dict[str, ErrorPattern] = {}

        # Load persisted errors
        self._load()

    # ============================================================
    # LOGGING
    # ============================================================

    def log_error(
        self,
        symptom: str,
        error_type: str = "",
        error_message: str = "",
        file_path: str = "",
        line_number: int = 0,
        function: str = "",
        root_cause: str = "",
        category: ErrorCategory = ErrorCategory.RUNTIME,
        severity: ErrorSeverity = ErrorSeverity.MEDIUM,
        agent_id: str = "",
        task_context: str = "",
        phase: str = "",
        stack_trace: str = "",
        code_snippet: str = "",
        tags: List[str] = None
    ) -> ErrorRecord:
        """
        Log a new error

        Creates an ErrorRecord and adds it to the ledger.
        Automatically checks for patterns and flags recurring issues.
        """
        record = ErrorRecord(
            symptom=symptom,
            error_type=error_type,
            error_message=error_message,
            file_path=file_path,
            line_number=line_number,
            function=function,
            root_cause=root_cause,
            category=category,
            severity=severity,
            agent_id=agent_id,
            task_context=task_context,
            phase=phase,
            stack_trace=stack_trace,
            code_snippet=code_snippet,
            tags=tags or []
        )

        self.add_error(record)
        return record

    def add_error(self, error: ErrorRecord):
        """Add an existing ErrorRecord to the ledger"""
        self.errors.append(error)

        # Check for pattern match
        pattern = self._find_or_create_pattern(error)
        pattern.add_occurrence(error)

        # Persist
        self._persist()

    def _find_or_create_pattern(self, error: ErrorRecord) -> ErrorPattern:
        """Find existing pattern or create new one"""
        # Pattern key is based on error type and category
        pattern_key = f"{error.category.value}_{error.error_type}"

        if pattern_key not in self.patterns:
            self.patterns[pattern_key] = ErrorPattern(
                pattern_id=pattern_key,
                category=error.category,
                error_type=error.error_type
            )

        return self.patterns[pattern_key]

    # ============================================================
    # QUERIES
    # ============================================================

    def get_all_errors(self) -> List[ErrorRecord]:
        """Get all errors, most recent first"""
        return sorted(self.errors, key=lambda e: e.timestamp, reverse=True)

    def get_unresolved_errors(self) -> List[ErrorRecord]:
        """Get unresolved errors"""
        return [e for e in self.errors if not e.resolved]

    def get_errors_by_category(self, category: ErrorCategory) -> List[ErrorRecord]:
        """Get errors by category"""
        return [e for e in self.errors if e.category == category]

    def get_errors_by_severity(self, severity: ErrorSeverity) -> List[ErrorRecord]:
        """Get errors by severity"""
        return [e for e in self.errors if e.severity == severity]

    def get_errors_by_agent(self, agent_id: str) -> List[ErrorRecord]:
        """Get errors from a specific agent"""
        return [e for e in self.errors if e.agent_id == agent_id]

    def get_flagged_patterns(self) -> List[ErrorPattern]:
        """Get all flagged patterns"""
        return [p for p in self.patterns.values() if p.flagged]

    def get_recent_errors(self, hours: int = 24) -> List[ErrorRecord]:
        """Get errors from last N hours"""
        cutoff = datetime.now() - timedelta(hours=hours)
        return [e for e in self.errors if e.timestamp >= cutoff]

    # ============================================================
    # RESOLUTION
    # ============================================================

    def resolve_error(self, error_id: str, fix: str = "", prevention: str = "") -> bool:
        """Mark an error as resolved"""
        for error in self.errors:
            if error.error_id == error_id:
                error.resolved = True
                error.resolved_at = datetime.now()
                error.fix = fix
                error.prevention = prevention
                self._persist()
                return True
        return False

    def verify_fix(self, error_id: str) -> bool:
        """Mark a fix as verified"""
        for error in self.errors:
            if error.error_id == error_id:
                error.verified_fix = True
                self._persist()
                return True
        return False

    def deploy_systematic_fix(self, pattern_id: str, fix: str, prevention: str) -> bool:
        """Deploy a systematic fix for a pattern"""
        if pattern_id in self.patterns:
            pattern = self.patterns[pattern_id]
            pattern.systematic_fix = fix
            pattern.systematic_prevention = prevention
            pattern.fix_deployed = True
            self._persist()
            return True
        return False

    # ============================================================
    # PERSISTENCE
    # ============================================================

    def _persist(self):
        """Persist errors to disk"""
        # Save individual errors
        for error in self.errors:
            file_path = self.storage_dir / f"{error.error_id}.json"
            with open(file_path, 'w') as f:
                json.dump(error.to_dict(), f, indent=2, default=str)

        # Save pattern index
        index_file = self.storage_dir / "patterns.json"
        patterns_data = {}
        for pattern_id, pattern in self.patterns.items():
            patterns_data[pattern_id] = {
                'pattern_id': pattern.pattern_id,
                'category': pattern.category.value,
                'error_type': pattern.error_type,
                'occurrence_count': pattern.occurrence_count,
                'flagged': pattern.flagged,
                'flag_threshold': pattern.flag_threshold,
                'flag_reason': pattern.flag_reason,
                'first_occurrence': pattern.first_occurrence.isoformat(),
                'last_occurrence': pattern.last_occurrence.isoformat(),
                'frequency_trend': pattern.frequency_trend,
                'systematic_fix': pattern.systematic_fix,
                'systematic_prevention': pattern.systematic_prevention,
                'fix_deployed': pattern.fix_deployed,
                'error_ids': [e.error_id for e in pattern.occurrences]
            }

        with open(index_file, 'w') as f:
            json.dump(patterns_data, f, indent=2, default=str)

    def _load(self):
        """Load errors from disk"""
        # Load error records
        for file_path in self.storage_dir.glob("err_*.json"):
            try:
                with open(file_path, 'r') as f:
                    data = json.load(f)
                    error = ErrorRecord.from_dict(data)
                    self.errors.append(error)

                    # Rebuild patterns
                    pattern = self._find_or_create_pattern(error)
                    if error.error_id not in [e.error_id for e in pattern.occurrences]:
                        pattern.add_occurrence(error)
            except Exception as e:
                print(f"Error loading {file_path}: {e}")

        # Load pattern index for flagging state
        index_file = self.storage_dir / "patterns.json"
        if index_file.exists():
            try:
                with open(index_file, 'r') as f:
                    patterns_data = json.load(f)
                    for pattern_id, data in patterns_data.items():
                        if pattern_id in self.patterns:
                            pattern = self.patterns[pattern_id]
                            pattern.flagged = data.get('flagged', False)
                            pattern.flag_reason = data.get('flag_reason', '')
                            pattern.systematic_fix = data.get('systematic_fix', '')
                            pattern.systematic_prevention = data.get('systematic_prevention', '')
                            pattern.fix_deployed = data.get('fix_deployed', False)
            except Exception as e:
                print(f"Error loading patterns index: {e}")

    # ============================================================
    # STATUS AND SUMMARY
    # ============================================================

    def get_summary(self) -> Dict[str, Any]:
        """Get summary of error ledger"""
        unresolved = self.get_unresolved_errors()
        flagged = self.get_flagged_patterns()

        # Count by category
        category_counts = defaultdict(int)
        for error in self.errors:
            category_counts[error.category.value] += 1

        # Count by severity
        severity_counts = defaultdict(int)
        for error in self.errors:
            severity_counts[error.severity.value] += 0

        return {
            "total_errors": len(self.errors),
            "unresolved_errors": len(unresolved),
            "flagged_patterns": len(flagged),
            "total_patterns": len(self.patterns),
            "by_category": dict(category_counts),
            "by_severity": dict(severity_counts),
            "recent_errors_24h": len(self.get_recent_errors(24)),
            "resolved_errors": len([e for e in self.errors if e.resolved])
        }

    def get_status_report(self) -> str:
        """Get formatted status report"""
        summary = self.get_summary()
        flagged = self.get_flagged_patterns()

        lines = [
            "🐛 AETHER ERROR LEDGER",
            "=" * 50,
            f"Total Errors: {summary['total_errors']}",
            f"Unresolved: {summary['unresolved_errors']}",
            f"Resolved: {summary['resolved_errors']}",
            f"Flagged Patterns: {summary['flagged_patterns']}",
            f"Recent (24h): {summary['recent_errors_24h']}",
            ""
        ]

        if flagged:
            lines.append("🚩 FLAGGED PATTERNS (recurring issues):")
            lines.append("-" * 50)
            for pattern in flagged:
                lines.append(f"  {pattern.get_summary()}")
                if pattern.flag_reason:
                    lines.append(f"    Reason: {pattern.flag_reason}")
            lines.append("")

        if summary['unresolved_errors'] > 0:
            lines.append("⏳ UNRESOLVED ERRORS:")
            lines.append("-" * 50)
            for error in self.get_unresolved_errors()[:10]:
                lines.append(f"  {error.get_summary()}")
            if summary['unresolved_errors'] > 10:
                lines.append(f"  ... and {summary['unresolved_errors'] - 10} more")
            lines.append("")

        return "\n".join(lines)


# ============================================================================
# CONVENIENCE FUNCTIONS FOR WORKER ANTS
# ============================================================================

def log_exception(
    ledger: ErrorLedger,
    e: Exception,
    symptom: str = "",
    agent_id: str = "",
    task_context: str = "",
    category: ErrorCategory = ErrorCategory.RUNTIME
) -> ErrorRecord:
    """
    Convenience function to log an exception

    Usage in WorkerAnt:
        try:
            risky_operation()
        except Exception as e:
            log_exception(self.error_ledger, e, "Operation failed", self.agent_id)
    """
    import traceback
    import inspect

    # Get caller info
    frame = inspect.currentframe().f_back
    file_path = ""
    line_number = 0
    function = ""

    if frame:
        file_path = frame.f_code.co_filename
        line_number = frame.f_lineno
        function = frame.f_code.co_name

    return ledger.log_error(
        symptom=symptom or str(e),
        error_type=type(e).__name__,
        error_message=str(e),
        file_path=file_path,
        line_number=line_number,
        function=function,
        root_cause="",
        category=category,
        agent_id=agent_id,
        task_context=task_context,
        stack_trace=traceback.format_exc()
    )


# ============================================================================
# DEMO / TESTING
# ============================================================================

def demo_error_prevention():
    """
    Demonstration of the error prevention system.

    Shows:
    1. Logging various errors
    2. Pattern detection and flagging
    3. Query and filtering
    4. Resolution tracking
    """
    print("🐛 Error Prevention System Demo\n")

    # Create ledger
    ledger = ErrorLedger()

    print("=" * 60)
    print("STEP 1: Logging errors")
    print("=" * 60)

    # Simulate some errors
    errors_to_log = [
        {
            "symptom": "ImportError: No module named 'nonexistent_module'",
            "error_type": "ImportError",
            "category": ErrorCategory.IMPORT,
            "file_path": "worker_ants.py",
            "line_number": 42,
            "function": "load_module",
            "agent_id": "builder_ant_0",
            "severity": ErrorSeverity.HIGH
        },
        {
            "symptom": "Capability detection failed: unknown capability",
            "error_type": "CapabilityError",
            "category": ErrorCategory.CAPABILITY,
            "file_path": "worker_ants.py",
            "line_number": 156,
            "function": "detect_capability_gaps",
            "agent_id": "planner_ant_1",
            "severity": ErrorSeverity.MEDIUM
        },
        {
            "symptom": "Spawning limit reached: max_subagents",
            "error_type": "SpawningLimitError",
            "category": ErrorCategory.SPAWNING,
            "file_path": "worker_ants.py",
            "line_number": 234,
            "function": "spawn_subagent",
            "agent_id": "builder_ant_0",
            "severity": ErrorSeverity.MEDIUM
        }
    ]

    for err in errors_to_log:
        record = ledger.log_error(**err)
        print(f"  Logged: {err['error_type']} - {err['symptom'][:50]}")

    print("\n" + "=" * 60)
    print("STEP 2: Trigger flagging by repeating errors")
    print("=" * 60)

    # Log more of the same errors to trigger flagging
    for i in range(3):
        ledger.log_error(
            symptom="ImportError: No module named 'nonexistent_module'",
            error_type="ImportError",
            category=ErrorCategory.IMPORT,
            file_path="worker_ants.py",
            line_number=42,
            function="load_module",
            agent_id=f"builder_ant_{i}",
            severity=ErrorSeverity.HIGH
        )

    print("  Logged 3 more ImportError occurrences")
    print("  This should trigger flagging (threshold: 3)")

    print("\n" + "=" * 60)
    print("STEP 3: Check status")
    print("=" * 60)

    print("\n" + ledger.get_status_report())

    print("=" * 60)
    print("STEP 4: Resolve an error")
    print("=" * 60)

    # Resolve one error
    unresolved = ledger.get_unresolved_errors()
    if unresolved:
        error_id = unresolved[0].error_id
        ledger.resolve_error(
            error_id,
            fix="Added module to requirements.txt",
            prevention="Always check requirements.txt before importing"
        )
        print(f"  Resolved: {error_id}")

    print("\n" + "=" * 60)
    print("STEP 5: Deploy systematic fix for flagged pattern")
    print("=" * 60)

    flagged = ledger.get_flagged_patterns()
    if flagged:
        pattern = flagged[0]
        ledger.deploy_systematic_fix(
            pattern.pattern_id,
            fix="Add import validation and try/except blocks",
            prevention="Implement module availability check before imports"
        )
        print(f"  Deployed fix for pattern: {pattern.pattern_id}")

    print("\n" + ledger.get_status_report())

    print("=" * 60)
    print("Demo Complete")
    print("=" * 60)
    print("\nKey Points:")
    print("  ✅ Structured error logging with full context")
    print("  ✅ Automatic pattern detection")
    print("  ✅ Flagging of recurring issues (3+ occurrences)")
    print("  ✅ Resolution tracking and verification")
    print("  ✅ Systematic fixes for patterns")
    print("  ✅ Persistent storage to .aether/errors/")


if __name__ == "__main__":
    demo_error_prevention()
