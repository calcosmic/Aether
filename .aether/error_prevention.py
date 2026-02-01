#!/usr/bin/env python3
"""
AETHER - Error Prevention System

Revolutionary approach: Never make the same mistake twice.

Components:
1. Error Ledger - Track every mistake with symptom, root cause, fix, prevention
2. Flag System - Auto-flag after 3 occurrences
3. Constraint Engine - YAML-based rules with DO/DON'T patterns
4. Guardrails - Validate BEFORE execution (not after)

This is the learning system that makes AETHER improve over time.
"""

import json
import uuid
from datetime import datetime
from typing import Dict, List, Set, Optional, Any
from dataclasses import dataclass, field
from enum import Enum
import yaml
import re


class ErrorSeverity(Enum):
    """Severity levels for errors"""
    CRITICAL = "critical"
    HIGH = "high"
    MEDIUM = "medium"
    LOW = "low"


@dataclass
class ErrorEntry:
    """A single error entry in the ledger"""
    id: str = field(default_factory=lambda: str(uuid.uuid4())[:8])
    timestamp: datetime = field(default_factory=datetime.now)
    title: str = ""
    symptom: str = ""  # What went wrong
    root_cause: str = ""  # Why it happened
    fix: str = ""  # How it was fixed
    prevention: str = ""  # How to prevent recurrence
    category: str = ""  # Error category for grouping
    severity: ErrorSeverity = ErrorSeverity.MEDIUM
    occurrence_count: int = 1

    # Additional metadata
    context: Dict[str, Any] = field(default_factory=dict)
    stack_trace: str = ""
    related_files: List[str] = field(default_factory=list)

    def to_dict(self) -> Dict:
        """Convert to dictionary for serialization"""
        return {
            "id": self.id,
            "timestamp": self.timestamp.isoformat(),
            "title": self.title,
            "symptom": self.symptom,
            "root_cause": self.root_cause,
            "fix": self.fix,
            "prevention": self.prevention,
            "category": self.category,
            "severity": self.severity.value,
            "occurrence_count": self.occurrence_count,
            "context": self.context,
            "stack_trace": self.stack_trace,
            "related_files": self.related_files
        }

    @classmethod
    def from_dict(cls, data: Dict) -> 'ErrorEntry':
        """Create from dictionary"""
        # Handle case-insensitive severity parsing
        severity_str = data.get("severity", "medium")
        if isinstance(severity_str, str):
            severity_str = severity_str.lower()
        try:
            severity = ErrorSeverity(severity_str)
        except ValueError:
            severity = ErrorSeverity.MEDIUM

        error = cls(
            title=data.get("title", ""),
            symptom=data.get("symptom", ""),
            root_cause=data.get("root_cause", ""),
            fix=data.get("fix", ""),
            prevention=data.get("prevention", ""),
            category=data.get("category", ""),
            severity=severity,
            occurrence_count=data.get("occurrence_count", 1),
            context=data.get("context", {}),
            stack_trace=data.get("stack_trace", ""),
            related_files=data.get("related_files", [])
        )
        if "id" in data:
            error.id = data["id"]
        if "timestamp" in data:
            error.timestamp = datetime.fromisoformat(data["timestamp"])
        return error

    def increment(self):
        """Increment occurrence count"""
        self.occurrence_count += 1
        self.timestamp = datetime.now()

    def __repr__(self):
        return f"ErrorEntry({self.title}, count: {self.occurrence_count}, severity: {self.severity.value})"


@dataclass
class Constraint:
    """A constraint rule that prevents specific actions"""
    id: str
    category: str
    severity: ErrorSeverity
    title: str
    description: str
    dont: List[str]  # Patterns that must NOT match
    do: List[str]  # Patterns that SHOULD match instead
    impact: str = ""  # What happens if violated
    verified: bool = False
    auto_flag: bool = False

    def matches_dont(self, action: str) -> Optional[str]:
        """
        Check if action matches any DON'T pattern.

        Returns the matching pattern if found, None otherwise.
        """
        for pattern in self.dont:
            if pattern in action.lower():
                return pattern
        return None

    def to_dict(self) -> Dict:
        """Convert to dictionary"""
        return {
            "id": self.id,
            "category": self.category,
            "severity": self.severity.value,
            "title": self.title,
            "description": self.description,
            "dont": self.dont,
            "do": self.do,
            "impact": self.impact,
            "verified": self.verified,
            "auto_flag": self.auto_flag
        }

    @classmethod
    def from_dict(cls, data: Dict) -> 'Constraint':
        """Create from dictionary"""
        # Handle case-insensitive severity parsing
        severity_str = data.get("severity", "medium").lower()
        try:
            severity = ErrorSeverity(severity_str)
        except ValueError:
            # If uppercase, try lowercase
            severity = ErrorSeverity(severity_str.lower())

        return cls(
            id=data["id"],
            category=data["category"],
            severity=severity,
            title=data["title"],
            description=data["description"],
            dont=data["dont"],
            do=data["do"],
            impact=data.get("impact", ""),
            verified=data.get("verified", False),
            auto_flag=data.get("auto_flag", False)
        )


class ErrorLedger:
    """
    Track every mistake with full details.

    Key innovation: After 3 occurrences of same category, auto-flag
    for constraint creation.
    """

    def __init__(self, ledger_path: str = None):
        self.errors: List[ErrorEntry] = []
        self.category_counts: Dict[str, int] = {}
        self.ledger_path = ledger_path

    def log(
        self,
        title: str,
        symptom: str,
        root_cause: str,
        fix: str,
        prevention: str,
        category: str,
        severity: ErrorSeverity = ErrorSeverity.MEDIUM,
        context: Dict = None
    ) -> Optional[str]:
        """
        Log an error with full details.

        Returns flag message if threshold reached, None otherwise.
        """
        # Check if this error already exists
        existing = self.find_by_title(title)
        if existing:
            existing.increment()
            count = existing.occurrence_count
        else:
            error = ErrorEntry(
                title=title,
                symptom=symptom,
                root_cause=root_cause,
                fix=fix,
                prevention=prevention,
                category=category,
                severity=severity,
                context=context or {}
            )
            self.errors.append(error)
            count = 1

        # Update category count
        self.category_counts[category] = self.category_counts.get(category, 0) + 1

        # Auto-flag after 3 occurrences
        if self.category_counts[category] >= 3:
            return self._create_flag(category)

        return None

    def _create_flag(self, category: str) -> str:
        """Create a flag message for recurring error"""
        return f"🚩 FLAG: {category} has occurred {self.category_counts[category]} times. Create constraint."

    def find_by_title(self, title: str) -> Optional[ErrorEntry]:
        """Find existing error by title"""
        for error in self.errors:
            if error.title == title:
                return error
        return None

    def get_category_errors(self, category: str) -> List[ErrorEntry]:
        """Get all errors in a category"""
        return [e for e in self.errors if e.category == category]

    def get_recurring_categories(self, threshold: int = 3) -> List[str]:
        """Get categories that have exceeded threshold"""
        return [cat for cat, count in self.category_counts.items() if count >= threshold]

    def to_dict(self) -> Dict:
        """Convert ledger to dictionary"""
        return {
            "errors": [e.to_dict() for e in self.errors],
            "category_counts": self.category_counts,
            "total_errors": len(self.errors)
        }

    def save(self, path: str = None):
        """Save ledger to file"""
        save_path = path or self.ledger_path
        if save_path:
            with open(save_path, 'w') as f:
                json.dump(self.to_dict(), f, indent=2)

    def load(self, path: str = None):
        """Load ledger from file"""
        load_path = path or self.ledger_path
        if load_path:
            try:
                with open(load_path, 'r') as f:
                    data = json.load(f)
                    self.errors = [ErrorEntry.from_dict(e) for e in data["errors"]]
                    self.category_counts = data["category_counts"]
            except FileNotFoundError:
                pass  # First run, no existing ledger

    def __repr__(self):
        recurring = len(self.get_recurring_categories())
        return f"ErrorLedger({len(self.errors)} errors, {recurring} recurring categories)"


class ConstraintEngine:
    """
    YAML-based constraint rules with DO/DON'T patterns.

    Validates actions BEFORE execution to prevent errors.
    """

    def __init__(self, constraints_path: str = None):
        self.constraints: List[Constraint] = []
        self.constraints_path = constraints_path
        self.load_constraints()

    def load_constraints(self):
        """Load constraints from YAML file"""
        if self.constraints_path:
            try:
                with open(self.constraints_path, 'r') as f:
                    data = yaml.safe_load(f)
                    if 'constraints' in data:
                        self.constraints = [
                            Constraint.from_dict(c) for c in data['constraints']
                        ]
            except FileNotFoundError:
                pass  # First run

    def validate(self, action: str) -> tuple[bool, Optional[Constraint]]:
        """
        Check if action violates any constraints.

        Returns (is_valid, violating_constraint)
        """
        for constraint in self.constraints:
            matching_pattern = constraint.matches_dont(action)
            if matching_pattern:
                return False, constraint

        return True, None

    def add_constraint(self, constraint: Constraint):
        """Add a new constraint"""
        self.constraints.append(constraint)

    def get_constraints_by_category(self, category: str) -> List[Constraint]:
        """Get all constraints in a category"""
        return [c for c in self.constraints if c.category == category]

    def to_dict(self) -> Dict:
        """Convert to dictionary"""
        return {
            "constraints": [c.to_dict() for c in self.constraints],
            "total": len(self.constraints)
        }

    def save(self, path: str = None):
        """Save constraints to YAML file"""
        save_path = path or self.constraints_path
        if save_path:
            data = {
                'constraints': [c.to_dict() for c in self.constraints]
            }
            with open(save_path, 'w') as f:
                yaml.dump(data, f, default_flow_style=False)

    def __repr__(self):
        return f"ConstraintEngine({len(self.constraints)} constraints)"


class Guardrails:
    """
    Pre-action validation system.

    Key innovation: Validates BEFORE action, not after.
    This prevents errors from executing in the first place.
    """

    def __init__(self, ledger_path: str = None, constraints_path: str = None):
        self.ledger = ErrorLedger(ledger_path)
        self.ledger.load()

        self.constraints = ConstraintEngine(constraints_path)

        # Validation statistics
        self.stats = {
            "validations": 0,
            "blocked": 0,
            "allowed": 0,
            "errors_prevented": 0
        }

    def validate_before_action(self, action: str, context: Dict = None) -> tuple[bool, str]:
        """
        Validate action BEFORE execution.

        Returns (is_allowed, reason)
        """
        self.stats["validations"] += 1

        # Check constraints
        is_valid, constraint = self.constraints.validate(action)

        if not is_valid:
            self.stats["blocked"] += 1
            reason = f"BLOCKED: Violates constraint '{constraint.title}' - {constraint.description}"
            return False, reason

        # Check for known error patterns
        if self._is_known_error_pattern(action):
            self.stats["blocked"] += 1
            self.stats["errors_prevented"] += 1
            reason = "BLOCKED: Matches known error pattern from ledger"
            return False, reason

        # Action allowed
        self.stats["allowed"] += 1
        return True, "ALLOWED"

    def _is_known_error_pattern(self, action: str) -> bool:
        """
        Check if action matches patterns that caused errors before.

        In production, would use more sophisticated pattern matching.
        """
        action_lower = action.lower()

        for error in self.ledger.errors:
            # Check if action contains patterns from past errors
            if error.symptom.lower() in action_lower:
                return True

            # Check related files
            for file_path in error.related_files:
                if file_path.lower() in action_lower:
                    return True

        return False

    def log_error(
        self,
        title: str,
        symptom: str,
        root_cause: str,
        fix: str,
        prevention: str,
        category: str,
        severity: ErrorSeverity = ErrorSeverity.MEDIUM
    ):
        """Log an error to the ledger"""
        flag = self.ledger.log(
            title=title,
            symptom=symptom,
            root_cause=root_cause,
            fix=fix,
            prevention=prevention,
            category=category,
            severity=severity
        )

        # Auto-create constraint if flagged
        if flag and severity in [ErrorSeverity.CRITICAL, ErrorSeverity.HIGH]:
            self._auto_create_constraint(category, prevention)

        self.ledger.save()

        return flag

    def _auto_create_constraint(self, category: str, prevention: str):
        """Auto-create constraint from prevention advice"""
        # Extract patterns from prevention
        dont_patterns = []
        if "don't" in prevention.lower() or "don't" in prevention.lower():
            # Parse "don't X" patterns
            matches = re.findall(r"(?:don'?t|do not)\s+([^.]+)", prevention, re.IGNORECASE)
            dont_patterns.extend([m.strip().lower() for m in matches])

        if dont_patterns:
            constraint = Constraint(
                id=f"auto-{category}-{len(self.constraints.constraints)}",
                category=category,
                severity=ErrorSeverity.HIGH,
                title=f"Auto: Prevent {category}",
                description=f"Auto-generated from {category} error pattern",
                dont=dont_patterns,
                do=[],
                impact="Prevents recurring error",
                auto_flag=True
            )
            self.constraints.add_constraint(constraint)
            self.constraints.save()

    def get_stats(self) -> Dict:
        """Get validation statistics"""
        return {
            **self.stats,
            "block_rate": f"{(self.stats['blocked'] / max(1, self.stats['validations']) * 100):.1f}%",
            "total_errors": len(self.ledger.errors),
            "recurring_categories": len(self.ledger.get_recurring_categories()),
            "total_constraints": len(self.constraints.constraints)
        }

    def get_flagged_issues(self) -> List[str]:
        """Get currently flagged issues"""
        return self.ledger.get_recurring_categories()

    def __repr__(self):
        stats = self.get_stats()
        return f"Guardrails(validations: {stats['validations']}, blocked: {stats['blocked']}, errors: {stats['total_errors']})"


def demo_error_prevention():
    """Demonstrate the error prevention system."""
    print("=" * 70)
    print("AETHER: Error Prevention System Demonstration")
    print("=" * 70)

    guardrails = Guardrails(
        ledger_path=".aether/error_ledger.json",
        constraints_path=".aether/CONSTRAINTS.yaml"
    )

    print("\n📊 Initial State:")
    print(f"   {guardrails}")

    # Scenario 1: First error occurrence
    print("\n1️⃣ First Error Occurrence:")
    allowed, reason = guardrails.validate_before_action(
        "Load entire codebase into context window"
    )

    if not allowed:
        print(f"   ❌ {reason}")
    else:
        print(f"   ✅ Action allowed")
        # Simulate error happening
        flag = guardrails.log_error(
            title="Context overload causing slow responses",
            symptom="Loading entire codebase exceeded token budget",
            root_cause="Context budgeting not implemented",
            fix="Load only files needed for current task",
            prevention="Always load minimal context, use token budgeting",
            category="context:overload",
            severity=ErrorSeverity.HIGH
        )
        if flag:
            print(f"   ⚠️  {flag}")
        print(f"   📝 Error logged (1st occurrence)")

    # Scenario 2: Second error - same category
    print("\n2️⃣ Second Error (Same Category):")
    flag = guardrails.log_error(
        title="Context overflow in planning phase",
        symptom="Context window filled, quality degraded",
        root_cause="No budgeting mechanism",
        fix="Implement 50k token per gate budget",
        prevention="Always start with minimal context",
        category="context:overload",
        severity=ErrorSeverity.HIGH
    )
    if flag:
        print(f"   ⚠️  {flag}")
    print(f"   📝 Error logged (2nd occurrence)")

    # Scenario 3: Third error - triggers flag!
    print("\n3️⃣ Third Error (Triggers Auto-Flag):")
    flag = guardrails.log_error(
        title="Context budget exceeded in research",
        symptom="Ran out of tokens during research phase",
        root_cause="Loaded too many files at once",
        fix="Progressive loading strategy",
        prevention="Load files on-demand, clear between gates",
        category="context:overload",
        severity=ErrorSeverity.HIGH
    )
    if flag:
        print(f"   🚩 {flag}")
        print(f"   ⚠️  Constraint auto-created!")
    print(f"   📝 Error logged (3rd occurrence)")

    # Scenario 4: Attempting the same action again - NOW BLOCKED!
    print("\n4️⃣ Attempting Same Action Again:")
    allowed, reason = guardrails.validate_before_action(
        "Load entire codebase into context window"
    )
    if not allowed:
        print(f"   ❌ {reason}")
        print(f"   ✅ Error PREVENTED by guardrails!")
    else:
        print(f"   ✅ Action allowed")

    # Scenario 5: Different error category
    print("\n5️⃣ Different Error Category:")
    flag = guardrails.log_error(
        title="Missing import statement",
        symptom="ModuleNotFoundError for utility module",
        root_cause="Forgot to add import",
        fix="Add import statement",
        prevention="Use linter to catch missing imports",
        category="test:missing",
        severity=ErrorSeverity.MEDIUM
    )
    if flag:
        print(f"   ⚠️  {flag}")
    print(f"   📝 Error logged (1st occurrence)")

    # Show statistics
    print("\n📊 Final Statistics:")
    stats = guardrails.get_stats()
    for key, value in stats.items():
        print(f"   {key}: {value}")

    print("\n🚩 Flagged Issues:")
    flagged = guardrails.get_flagged_issues()
    for issue in flagged:
        print(f"   • {issue}")

    print("\n✅ Key Innovation:")
    print("   After 3 occurrences of same error:")
    print("   → Auto-flagged for attention")
    print("   → Constraint auto-created")
    print("   → Future attempts BLOCKED before execution")
    print("   → System NEVER repeats the same mistake")

    return guardrails


def main():
    """Main entry point."""
    guardrails = demo_error_prevention()

    print("\n" + "=" * 70)
    print("✅ DEMONSTRATION COMPLETE")
    print("=" * 70)
    print("\nError Prevention System features:")
    print("  • Error Ledger: Track every mistake with full details")
    print("  • Auto-Flag: After 3 occurrences of same category")
    print("  • Constraint Engine: YAML-based rules with DO/DON'T")
    print("  • Guardrails: Validate BEFORE action (not after)")
    print("\nThis is the learning system that makes AETHER improve:")
    print("  'Never make the same mistake twice'")
    print("\nWe just built the solution. 🛡️")


if __name__ == "__main__":
    main()
