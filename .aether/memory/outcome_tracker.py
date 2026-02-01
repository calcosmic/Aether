"""
Outcome Tracking for Emergent Testing

Tracks the relationship between testing approaches and outcomes.
Worker Ants learn from this data what works best.

Based on research from:
- Ralph et al. on LLM-based test generation
- Feedback loops and learning in software testing
- Emergent behavior patterns in ant colonies
"""

from dataclasses import dataclass, field
from datetime import datetime
from typing import Literal, Optional, Dict, Any
import json
from pathlib import Path


TestingApproach = Literal[
    "test_first",           # Test written before implementation
    "test_after",           # Test written after implementation
    "test_parallel",        # Test written alongside implementation
    "no_test",              # No test written
    "test_only",            # Only test, no implementation yet
]

Outcome = Literal[
    "success",              # Worked correctly, no issues
    "had_bugs",             # Bugs found later
    "failed_tests",         # Tests failed after completion
    "needed_refactor",      # Required significant refactoring
    "caused_breakage",      # Broke existing functionality
]

TestStyle = Literal[
    "unit",                 # Unit test
    "integration",          # Integration test
    "e2e",                  # End-to-end test
    "property_based",       # Property-based test
]


@dataclass
class TestingOutcome:
    """Record of a testing approach and its outcome"""
    task_id: str
    task_description: str
    approach: TestingApproach
    outcome: Outcome
    time_to_complete: int  # minutes
    defects_found: int     # bugs found later
    rework_needed: bool
    timestamp: str = None
    metadata: Dict[str, Any] = field(default_factory=dict)

    def __post_init__(self):
        if self.timestamp is None:
            self.timestamp = datetime.now().isoformat()

    def to_dict(self) -> Dict[str, Any]:
        """Convert to dictionary for serialization"""
        return {
            "task_id": self.task_id,
            "task_description": self.task_description,
            "approach": self.approach,
            "outcome": self.outcome,
            "time_to_complete": self.time_to_complete,
            "defects_found": self.defects_found,
            "rework_needed": self.rework_needed,
            "timestamp": self.timestamp,
            "metadata": self.metadata
        }

    @classmethod
    def from_dict(cls, data: Dict[str, Any]) -> 'TestingOutcome':
        """Create from dictionary"""
        return cls(**data)


@dataclass
class TestGenerationResult:
    """Result from test generation"""
    test_content: str
    test_path: str
    style: TestStyle
    estimated_coverage: float
    dependencies: list[str] = field(default_factory=list)
    mock_objects: list[str] = field(default_factory=list)

    def to_dict(self) -> Dict[str, Any]:
        """Convert to dictionary"""
        return {
            "test_content": self.test_content,
            "test_path": self.test_path,
            "style": self.style,
            "estimated_coverage": self.estimated_coverage,
            "dependencies": self.dependencies,
            "mock_objects": self.mock_objects
        }


class OutcomeTracker:
    """Track testing outcomes for learning"""

    def __init__(self, memory_layer=None, storage_path: str = None):
        self.memory_layer = memory_layer
        self.outcomes: list[TestingOutcome] = []
        self.storage_path = storage_path or ".aether/data/testing_outcomes.json"

        # Load existing outcomes
        self._load_outcomes()

    def _load_outcomes(self):
        """Load outcomes from persistent storage"""
        try:
            path = Path(self.storage_path)
            if path.exists():
                with open(path, 'r') as f:
                    data = json.load(f)
                    self.outcomes = [TestingOutcome.from_dict(o) for o in data]
        except Exception as e:
            # Start fresh if load fails
            self.outcomes = []

    def _save_outcomes(self):
        """Save outcomes to persistent storage"""
        try:
            path = Path(self.storage_path)
            path.parent.mkdir(parents=True, exist_ok=True)

            with open(path, 'w') as f:
                data = [o.to_dict() for o in self.outcomes]
                json.dump(data, f, indent=2)
        except Exception as e:
            # Non-blocking: continue even if save fails
            pass

    async def record_outcome(self, outcome: TestingOutcome):
        """Record a testing outcome"""
        self.outcomes.append(outcome)

        # Persist to disk
        self._save_outcomes()

        # Store in working memory if available
        if self.memory_layer:
            await self.memory_layer.add_to_working(
                content=f"Testing approach: {outcome.approach} → {outcome.outcome}",
                metadata={
                    "task": outcome.task_id,
                    "approach": outcome.approach,
                    "outcome": outcome.outcome,
                    "defects": outcome.defects_found,
                    "rework": outcome.rework_needed,
                    "time": outcome.time_to_complete
                },
                item_type="testing_outcome"
            )

    async def analyze_approaches(self) -> Dict[str, Dict[str, Any]]:
        """
        Analyze which approaches work best

        Returns:
            {
                "test_first": {
                    "success_rate": 0.85,
                    "avg_defects": 0.3,
                    "avg_time": 45,
                    "confidence": 0.9,
                    "sample_size": 50
                },
                ...
            }
        """
        results = {}

        # Group outcomes by approach
        for approach in [v for v in TestingApproach]:
            approach_outcomes = [o for o in self.outcomes if o.approach == approach]

            if len(approach_outcomes) == 0:
                continue

            # Calculate metrics
            success_count = len([o for o in approach_outcomes if o.outcome == "success"])
            success_rate = success_count / len(approach_outcomes)

            avg_defects = sum(o.defects_found for o in approach_outcomes) / len(approach_outcomes)
            avg_time = sum(o.time_to_complete for o in approach_outcomes) / len(approach_outcomes)

            rework_count = len([o for o in approach_outcomes if o.rework_needed])
            rework_rate = rework_count / len(approach_outcomes)

            # Confidence increases with sample size (max 0.95)
            # Start at 0.3, add 0.1 per sample up to max
            confidence = min(0.95, 0.3 + (len(approach_outcomes) * 0.05))

            results[approach] = {
                "success_rate": success_rate,
                "avg_defects": avg_defects,
                "avg_time": avg_time,
                "rework_rate": rework_rate,
                "confidence": confidence,
                "sample_size": len(approach_outcomes)
            }

        return results

    async def recommend_approach(
        self,
        task_context: Dict[str, Any] = None
    ) -> tuple[TestingApproach, float]:
        """
        Recommend testing approach based on learned outcomes

        Considers:
        - Historical success rates
        - Task complexity
        - Current pheromone signals
        - Colony preferences

        Returns:
            (approach, confidence)
        """
        analysis = await self.analyze_approaches()

        if not analysis:
            # No data yet, recommend balanced approach
            return ("test_parallel", 0.3)

        # Weight by success rate and confidence
        # Prioritize low defects and high success rate
        best_approach = None
        best_score = -1

        for approach, metrics in analysis.items():
            # Composite score: success rate (0.6) - defects penalty (0.3) + confidence (0.1)
            defects_penalty = min(metrics["avg_defects"] * 0.1, 0.3)
            score = (
                metrics["success_rate"] * 0.6 -
                defects_penalty +
                metrics["confidence"] * 0.1
            )

            if score > best_score:
                best_score = score
                best_approach = approach

        confidence = analysis[best_approach]["confidence"]

        return (best_approach, confidence)

    async def get_trends(self, window_size: int = 20) -> Dict[str, Any]:
        """
        Get trends in testing outcomes over time

        Args:
            window_size: Number of recent outcomes to analyze

        Returns:
            Trend data showing evolution of approaches
        """
        if len(self.outcomes) < window_size * 2:
            return {"error": "Not enough data for trend analysis"}

        # Compare recent vs older outcomes
        recent = self.outcomes[-window_size:]
        older = self.outcomes[-(window_size * 2):-window_size]

        def analyze_batch(batch):
            by_approach = {}
            for o in batch:
                if o.approach not in by_approach:
                    by_approach[o.approach] = {"success": 0, "total": 0, "defects": 0}
                by_approach[o.approach]["total"] += 1
                if o.outcome == "success":
                    by_approach[o.approach]["success"] += 1
                by_approach[o.approach]["defects"] += o.defects_found

            return {
                approach: {
                    "success_rate": data["success"] / data["total"],
                    "avg_defects": data["defects"] / data["total"]
                }
                for approach, data in by_approach.items()
            }

        return {
            "recent": analyze_batch(recent),
            "older": analyze_batch(older),
            "trend": "improving"  # Could be calculated more precisely
        }

    async def store_learned_patterns(self):
        """
        Store learned testing patterns in long-term memory

        Based on research on feedback loops and learning.
        Stores patterns when confidence threshold is reached.
        """
        if not self.memory_layer:
            return

        analysis = await self.analyze_approaches()

        for approach, metrics in analysis.items():
            # Only store confident learnings
            if metrics["confidence"] >= 0.7:
                pattern_value = (
                    f"Testing approach '{approach}': "
                    f"{metrics['success_rate']*100:.0f}% success rate, "
                    f"{metrics['avg_defects']:.1f} avg defects, "
                    f"{metrics['avg_time']:.0f}min avg time, "
                    f"{metrics['rework_rate']*100:.0f}% rework rate"
                )

                # Store in long-term memory
                try:
                    await self.memory_layer.store_long_term(
                        category="testing_patterns",
                        key=approach,
                        value=pattern_value,
                        confidence=metrics["confidence"],
                        metadata={
                            "success_rate": metrics["success_rate"],
                            "avg_defects": metrics["avg_defects"],
                            "avg_time": metrics["avg_time"],
                            "rework_rate": metrics["rework_rate"],
                            "sample_size": metrics["sample_size"],
                            "learned_from": "outcome_tracking"
                        }
                    )
                except Exception as e:
                    # Non-blocking: continue even if storage fails
                    pass

    async def get_summary(self) -> Dict[str, Any]:
        """Get summary of all testing outcomes"""
        if not self.outcomes:
            return {
                "total_outcomes": 0,
                "message": "No testing outcomes recorded yet"
            }

        analysis = await self.analyze_approaches()

        return {
            "total_outcomes": len(self.outcomes),
            "approaches_analyzed": len(analysis),
            "best_approach": max(analysis.items(), key=lambda x: x[1]["success_rate"])[0] if analysis else None,
            "overall_success_rate": sum(o.outcome == "success" for o in self.outcomes) / len(self.outcomes),
            "approach_breakdown": analysis
        }
