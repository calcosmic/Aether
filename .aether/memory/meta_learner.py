"""
Aether Queen Ant Colony - Meta-Learning Loop

Tracks spawning outcomes and evolves capability confidence scoring based on
Ralph's research on LLM-based feedback loops and autonomous agent systems.

Research Foundation:
- Bayesian updating of capability scores
- Cross-task generalization tracking
- Dynamic taxonomy evolution
- Meta-learning for autonomous spawning

Based on:
- "Emergent Behaviors in LLM-Driven Autonomous Agent Networks"
- Ralph's research on test generation feedback loops
"""

from typing import Dict, List, Any, Optional, Set, Tuple
from dataclasses import dataclass, field
from datetime import datetime
from enum import Enum
import json
import math
from collections import defaultdict

try:
    from .worker_ants import SpecialistType
except ImportError:
    # Fallback for standalone execution
    class SpecialistType:
        DATABASE_SPECIALIST = "database_specialist"
        FRONTEND_SPECIALIST = "frontend_specialist"
        API_SPECIALIST = "api_specialist"
        SECURITY_SPECIALIST = "security_specialist"
        TEST_SPECIALIST = "test_specialist"
        PERFORMANCE_SPECIALIST = "performance_specialist"


class TaskOutcome(Enum):
    """Result of a specialist task execution"""
    SUCCESS = "success"
    PARTIAL_SUCCESS = "partial_success"
    FAILURE = "failure"
    TIMEOUT = "timeout"
    ERROR = "error"


@dataclass
class SpawnEvent:
    """Record of a specialist spawning event"""
    event_id: str
    timestamp: datetime
    parent_agent: str  # Which Worker Ant caste spawned this
    task_description: str
    task_category: str  # e.g., "database", "frontend", "security"
    specialist_type: str
    capability_gap: Set[str]  # Capabilities that triggered spawning
    inherited_context: Dict[str, Any]

    # Outcome tracking
    outcome: Optional[TaskOutcome] = None
    completion_time: Optional[float] = None  # seconds
    quality_score: Optional[float] = None  # 0.0 to 1.0
    innovation_score: Optional[float] = None  # 0.0 to 1.0, deviation from baseline

    # Learning signals
    user_feedback: Optional[str] = None
    peer_feedback: List[str] = field(default_factory=list)

    def to_dict(self) -> Dict[str, Any]:
        """Convert to dictionary for storage"""
        return {
            "event_id": self.event_id,
            "timestamp": self.timestamp.isoformat(),
            "parent_agent": self.parent_agent,
            "task_description": self.task_description,
            "task_category": self.task_category,
            "specialist_type": self.specialist_type,
            "capability_gap": list(self.capability_gap),
            "inherited_context": self.inherited_context,
            "outcome": self.outcome.value if self.outcome else None,
            "completion_time": self.completion_time,
            "quality_score": self.quality_score,
            "innovation_score": self.innovation_score,
            "user_feedback": self.user_feedback,
            "peer_feedback": self.peer_feedback
        }


@dataclass
class CapabilityConfidence:
    """Confidence score for a specialist type's capability"""
    specialist_type: str
    task_category: str

    # Bayesian parameters
    alpha: float = 1.0  # Success count prior
    beta: float = 1.0   # Failure count prior

    # Confidence metrics
    total_spawns: int = 0
    successful_spawns: int = 0
    failed_spawns: int = 0

    # Performance tracking
    avg_completion_time: float = 0.0
    avg_quality_score: float = 0.0
    avg_innovation_score: float = 0.0

    # Cross-task generalization
    generalizes_to: List[str] = field(default_factory=list)
    generalization_confidence: Dict[str, float] = field(default_factory=dict)

    @property
    def confidence_score(self) -> float:
        """
        Bayesian posterior mean for capability confidence.

        Uses Beta distribution posterior mean: E[X] = alpha / (alpha + beta)
        """
        return self.alpha / (self.alpha + self.beta)

    @property
    def confidence_interval(self) -> Tuple[float, float]:
        """
        95% confidence interval using Beta distribution.

        Returns (lower_bound, upper_bound)
        """
        # Approximation using normal for large alpha, beta
        mean = self.confidence_score
        variance = (self.alpha * self.beta) / ((self.alpha + self.beta)**2 * (self.alpha + self.beta + 1))
        std = math.sqrt(variance)

        lower = max(0.0, mean - 1.96 * std)
        upper = min(1.0, mean + 1.96 * std)

        return (lower, upper)

    @property
    def sample_size(self) -> int:
        """Effective sample size for this confidence estimate"""
        return self.total_spawns + int(self.alpha + self.beta - 2)  # Remove prior counts

    def update(self, outcome: TaskOutcome, quality: float, innovation: float, duration: float):
        """
        Bayesian update of capability confidence.

        Args:
            outcome: Task outcome
            quality: Quality score (0-1)
            innovation: Innovation score (0-1)
            duration: Completion time in seconds
        """
        self.total_spawns += 1

        # Update Bayesian parameters
        if outcome == TaskOutcome.SUCCESS:
            self.alpha += 2.0  # Strong success signal
            self.successful_spawns += 1
        elif outcome == TaskOutcome.PARTIAL_SUCCESS:
            self.alpha += 1.0  # Weak success signal
            self.successful_spawns += 1
        elif outcome == TaskOutcome.FAILURE:
            self.beta += 1.0  # Failure signal
            self.failed_spawns += 1
        elif outcome == TaskOutcome.ERROR:
            self.beta += 2.0  # Strong failure signal
            self.failed_spawns += 1

        # Update running averages
        n = self.total_spawns
        self.avg_quality_score = (
            (self.avg_quality_score * (n - 1) + quality) / n
        )
        self.avg_innovation_score = (
            (self.avg_innovation_score * (n - 1) + innovation) / n
        )
        self.avg_completion_time = (
            (self.avg_completion_time * (n - 1) + duration) / n
        )

    def to_dict(self) -> Dict[str, Any]:
        """Convert to dictionary for storage"""
        return {
            "specialist_type": self.specialist_type,
            "task_category": self.task_category,
            "alpha": self.alpha,
            "beta": self.beta,
            "confidence_score": self.confidence_score,
            "confidence_interval": self.confidence_interval,
            "total_spawns": self.total_spawns,
            "successful_spawns": self.successful_spawns,
            "failed_spawns": self.failed_spawns,
            "avg_completion_time": self.avg_completion_time,
            "avg_quality_score": self.avg_quality_score,
            "avg_innovation_score": self.avg_innovation_score,
            "generalizes_to": self.generalizes_to,
            "generalization_confidence": self.generalization_confidence,
            "sample_size": self.sample_size
        }


class MetaLearner:
    """
    Meta-learning system for evolving autonomous spawning.

    Tracks spawning outcomes and updates capability confidences using
    Bayesian updating. Enables the colony to learn which specialists
    work best for which tasks.
    """

    def __init__(self, storage_path: str = ".aether/memory/meta_learning.json"):
        self.storage_path = storage_path

        # Capability confidence matrix
        # Key: (specialist_type, task_category)
        # Value: CapabilityConfidence
        self.capability_confidence: Dict[Tuple[str, str], CapabilityConfidence] = {}

        # Spawn history
        self.spawn_history: List[SpawnEvent] = []

        # Taxonomy evolution
        self.active_specialist_types: Set[str] = set([
            "database_specialist",
            "frontend_specialist",
            "api_specialist",
            "security_specialist",
            "test_specialist",
            "performance_specialist",
            "realtime_specialist"
        ])

        # Deprecated specialist types (evolutionary淘汰)
        self.deprecated_specialist_types: Set[str] = set()

        # Emergent capabilities (newly discovered)
        self.emergent_capabilities: Dict[str, float] = {}

        # Cross-task generalization tracking
        self.generalization_matrix: Dict[str, Dict[str, float]] = defaultdict(lambda: defaultdict(float))

        # Load existing state
        self._load_state()

    def _load_state(self):
        """Load meta-learning state from storage"""
        try:
            with open(self.storage_path, 'r') as f:
                data = json.load(f)

            # Restore capability confidences
            for key, value in data.get("capability_confidence", {}).items():
                specialist_type, task_category = eval(key)
                conf = CapabilityConfidence(
                    specialist_type=value["specialist_type"],
                    task_category=value["task_category"],
                    alpha=value["alpha"],
                    beta=value["beta"],
                    total_spawns=value["total_spawns"],
                    successful_spawns=value["successful_spawns"],
                    failed_spawns=value["failed_spawns"],
                    avg_completion_time=value["avg_completion_time"],
                    avg_quality_score=value["avg_quality_score"],
                    avg_innovation_score=value["avg_innovation_score"]
                )
                self.capability_confidence[(specialist_type, task_category)] = conf

            # Restore spawn history
            for event_data in data.get("spawn_history", []):
                event = SpawnEvent(
                    event_id=event_data["event_id"],
                    timestamp=datetime.fromisoformat(event_data["timestamp"]),
                    parent_agent=event_data["parent_agent"],
                    task_description=event_data["task_description"],
                    task_category=event_data["task_category"],
                    specialist_type=event_data["specialist_type"],
                    capability_gap=set(event_data["capability_gap"]),
                    inherited_context=event_data["inherited_context"],
                    outcome=TaskOutcome(event_data["outcome"]) if event_data.get("outcome") else None,
                    completion_time=event_data.get("completion_time"),
                    quality_score=event_data.get("quality_score"),
                    innovation_score=event_data.get("innovation_score"),
                    user_feedback=event_data.get("user_feedback"),
                    peer_feedback=event_data.get("peer_feedback", [])
                )
                self.spawn_history.append(event)

            # Restore other state
            self.active_specialist_types = set(data.get("active_specialist_types", []))
            self.deprecated_specialist_types = set(data.get("deprecated_specialist_types", []))
            self.emergent_capabilities = data.get("emergent_capabilities", {})
            self.generalization_matrix = defaultdict(
                lambda: defaultdict(float),
                data.get("generalization_matrix", {})
            )

        except FileNotFoundError:
            # First run, initialize with default confidences
            self._initialize_default_confidences()

    def _initialize_default_confidences(self):
        """Initialize default capability confidences for known specialist types"""
        default_pairs = [
            ("database_specialist", "database"),
            ("database_specialist", "sql"),
            ("frontend_specialist", "frontend"),
            ("frontend_specialist", "react"),
            ("frontend_specialist", "vue"),
            ("api_specialist", "api"),
            ("api_specialist", "websocket"),
            ("security_specialist", "authentication"),
            ("security_specialist", "jwt"),
            ("security_specialist", "owasp"),
            ("test_specialist", "testing"),
            ("test_specialist", "unit"),
            ("performance_specialist", "optimization"),
            ("realtime_specialist", "websocket"),
        ]

        for specialist_type, task_category in default_pairs:
            key = (specialist_type, task_category)
            self.capability_confidence[key] = CapabilityConfidence(
                specialist_type=specialist_type,
                task_category=task_category,
                alpha=1.0,  # Prior: 1 success
                beta=1.0    # Prior: 1 failure
            )

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
        """
        event_id = f"spawn_{datetime.now().strftime('%Y%m%d%H%M%S')}_{len(self.spawn_history)}"

        event = SpawnEvent(
            event_id=event_id,
            timestamp=datetime.now(),
            parent_agent=parent_agent,
            task_description=task_description,
            task_category=task_category,
            specialist_type=specialist_type,
            capability_gap=capability_gap,
            inherited_context=inherited_context
        )

        self.spawn_history.append(event)

        return event_id

    def record_outcome(
        self,
        event_id: str,
        outcome: TaskOutcome,
        quality_score: float,
        innovation_score: float,
        duration: float,
        user_feedback: Optional[str] = None,
        peer_feedback: Optional[List[str]] = None
    ):
        """
        Record the outcome of a spawn event and update confidences.

        Args:
            event_id: Spawn event ID
            outcome: Task outcome
            quality_score: Quality (0-1)
            innovation_score: Innovation (0-1)
            duration: Completion time (seconds)
            user_feedback: Optional user feedback
            peer_feedback: Optional peer feedback
        """
        # Find the spawn event
        event = next((e for e in self.spawn_history if e.event_id == event_id), None)
        if not event:
            raise ValueError(f"Spawn event {event_id} not found")

        # Update event
        event.outcome = outcome
        event.quality_score = quality_score
        event.innovation_score = innovation_score
        event.completion_time = duration
        event.user_feedback = user_feedback
        if peer_feedback:
            event.peer_feedback = peer_feedback

        # Update capability confidence
        key = (event.specialist_type, event.task_category)

        # Create new confidence if doesn't exist
        if key not in self.capability_confidence:
            self.capability_confidence[key] = CapabilityConfidence(
                specialist_type=event.specialist_type,
                task_category=event.task_category
            )

        # Bayesian update
        self.capability_confidence[key].update(
            outcome=outcome,
            quality=quality_score,
            innovation=innovation_score,
            duration=duration
        )

        # Update generalization matrix
        self._update_generalization(event, outcome, quality_score)

        # Check for taxonomy evolution
        self._check_taxonomy_evolution(event)

        # Save state
        self._save_state()

    def _update_generalization(self, event: SpawnEvent, outcome: TaskOutcome, quality: float):
        """
        Update cross-task generalization tracking.

        If a specialist performs well on a task category, it may generalize
        to related categories.
        """
        if outcome == TaskOutcome.SUCCESS and quality > 0.7:
            # Track that this specialist may generalize to related categories
            specialist = event.specialist_type
            category = event.task_category

            # Find related categories (e.g., "jwt" → "authentication")
            related = self._find_related_categories(category)

            for related_cat in related:
                # Boost generalization confidence
                self.generalization_matrix[specialist][related_cat] += 0.1

                # If strong enough, add to generalizes_to
                if self.generalization_matrix[specialist][related_cat] > 0.7:
                    key = (specialist, related_cat)
                    if key in self.capability_confidence:
                        self.capability_confidence[key].generalizes_to.append(related_cat)
                        self.capability_confidence[key].generalization_confidence[related_cat] = (
                            self.generalization_matrix[specialist][related_cat]
                        )

    def _find_related_categories(self, category: str) -> List[str]:
        """Find categories related to the given one"""
        # Hardcoded relationships for common categories
        relationships = {
            "jwt": ["authentication", "security"],
            "authentication": ["jwt", "session", "oauth"],
            "sql": ["database", "query_optimization"],
            "database": ["sql", "orm", "migrations"],
            "websocket": ["realtime", "api"],
            "react": ["frontend", "ui"],
            "vue": ["frontend", "ui"],
            "unit": ["testing"],
            "testing": ["unit", "integration", "e2e"],
        }

        return relationships.get(category, [])

    def _check_taxonomy_evolution(self, event: SpawnEvent):
        """
        Check if taxonomy should evolve based on outcomes.

        Evolution scenarios:
        1. Specialist consistently fails → Deprecate
        2. New capability pattern emerges → Create new specialist type
        3. Two specialists converge → Merge
        """
        key = (event.specialist_type, event.task_category)
        conf = self.capability_confidence.get(key)

        if not conf or conf.sample_size < 5:
            return  # Not enough data

        # Check for deprecation (consistent failure)
        if conf.confidence_score < 0.3 and conf.sample_size >= 10:
            self.deprecated_specialist_types.add(event.specialist_type)
            self.active_specialist_types.discard(event.specialist_type)

        # Check for new capability patterns (high innovation)
        if event.innovation_score and event.innovation_score > 0.8:
            # This specialist is doing something novel
            capability_key = f"{event.task_category}_novel"
            self.emergent_capabilities[capability_key] = (
                self.emergent_capabilities.get(capability_key, 0.0) + 0.1
            )

    def recommend_specialist(
        self,
        task_description: str,
        task_category: str,
        capability_gap: Set[str]
    ) -> Tuple[Optional[str], float]:
        """
        Recommend a specialist type for the given task.

        Returns (specialist_type, confidence) or (None, 0.0) if no recommendation.

        Uses:
        1. Direct confidence matching
        2. Cross-task generalization
        3. Fallback to generalists
        """
        # Check direct confidence
        best_confidence = 0.0
        best_specialist = None

        for (specialist, category), conf in self.capability_confidence.items():
            if category == task_category or category in capability_gap:
                confidence = conf.confidence_score

                # Weight by sample size (more data = more weight)
                sample_weight = min(1.0, conf.sample_size / 10.0)
                weighted_confidence = confidence * (0.5 + 0.5 * sample_weight)

                if weighted_confidence > best_confidence:
                    best_confidence = weighted_confidence
                    best_specialist = specialist

        # Check generalization if no direct match
        if best_specialist is None:
            for specialist, related_cats in self.generalization_matrix.items():
                for related_cat, gen_conf in related_cats.items():
                    if related_cat == task_category and gen_conf > 0.5:
                        if gen_conf > best_confidence:
                            best_confidence = gen_conf
                            best_specialist = specialist

        # Filter out deprecated specialists
        if best_specialist in self.deprecated_specialist_types:
            best_specialist = None
            best_confidence = 0.0

        return best_specialist, best_confidence

    def get_spawning_stats(self) -> Dict[str, Any]:
        """Get statistics about spawning performance"""
        total_spawns = len(self.spawn_history)
        successful = sum(1 for e in self.spawn_history if e.outcome == TaskOutcome.SUCCESS)
        failed = sum(1 for e in self.spawn_history if e.outcome in [TaskOutcome.FAILURE, TaskOutcome.ERROR])

        # Top performing specialists
        specialist_performance = defaultdict(lambda: {"success": 0, "total": 0})
        for event in self.spawn_history:
            if event.outcome:
                specialist_performance[event.specialist_type]["total"] += 1
                if event.outcome == TaskOutcome.SUCCESS:
                    specialist_performance[event.specialist_type]["success"] += 1

        top_specialists = sorted(
            [(s, stats["success"] / max(stats["total"], 1))
             for s, stats in specialist_performance.items()],
            key=lambda x: x[1],
            reverse=True
        )[:5]

        return {
            "total_spawns": total_spawns,
            "successful_spawns": successful,
            "failed_spawns": failed,
            "success_rate": successful / max(total_spawns, 1),
            "active_specialist_types": len(self.active_specialist_types),
            "deprecated_specialist_types": len(self.deprecated_specialist_types),
            "top_specialists": top_specialists,
            "emergent_capabilities": len(self.emergent_capabilities)
        }

    def _save_state(self):
        """Save meta-learning state to storage"""
        data = {
            "capability_confidence": {
                str(key): conf.to_dict()
                for key, conf in self.capability_confidence.items()
            },
            "spawn_history": [e.to_dict() for e in self.spawn_history],
            "active_specialist_types": list(self.active_specialist_types),
            "deprecated_specialist_types": list(self.deprecated_specialist_types),
            "emergent_capabilities": self.emergent_capabilities,
            "generalization_matrix": dict(self.generalization_matrix)
        }

        # Ensure directory exists
        import os
        os.makedirs(os.path.dirname(self.storage_path), exist_ok=True)

        with open(self.storage_path, 'w') as f:
            json.dump(data, f, indent=2)


# Singleton instance
_meta_learner: Optional[MetaLearner] = None

def get_meta_learner() -> MetaLearner:
    """Get the singleton MetaLearner instance"""
    global _meta_learner
    if _meta_learner is None:
        _meta_learner = MetaLearner()
    return _meta_learner


# Demo
def demo_meta_learning():
    """Demonstrate meta-learning capabilities"""
    print("=" * 60)
    print("Aether Meta-Learning Loop Demo")
    print("=" * 60)
    print()

    ml = MetaLearner(storage_path=".aether/memory/meta_learning_demo.json")

    print("1. Recording spawn events...")

    # Simulate some spawn events
    event1 = ml.record_spawn(
        parent_agent="Executor",
        task_description="Implement JWT authentication",
        task_category="jwt",
        specialist_type="security_specialist",
        capability_gap={"jwt", "authentication"},
        inherited_context={"goal": "Build REST API"}
    )

    event2 = ml.record_spawn(
        parent_agent="Executor",
        task_description="Setup PostgreSQL database",
        task_category="database",
        specialist_type="database_specialist",
        capability_gap={"sql", "database"},
        inherited_context={"goal": "Build REST API"}
    )

    event3 = ml.record_spawn(
        parent_agent="Executor",
        task_description="Create React frontend",
        task_category="react",
        specialist_type="frontend_specialist",
        capability_gap={"frontend", "react"},
        inherited_context={"goal": "Build REST API"}
    )

    print(f"   Recorded 3 spawn events")
    print()

    print("2. Recording outcomes (Bayesian updating)...")

    # Record outcomes
    ml.record_outcome(
        event_id=event1,
        outcome=TaskOutcome.SUCCESS,
        quality_score=0.9,
        innovation_score=0.3,
        duration=1800  # 30 minutes
    )

    ml.record_outcome(
        event_id=event2,
        outcome=TaskOutcome.SUCCESS,
        quality_score=0.85,
        innovation_score=0.2,
        duration=2400  # 40 minutes
    )

    ml.record_outcome(
        event_id=event3,
        outcome=TaskOutcome.PARTIAL_SUCCESS,
        quality_score=0.6,
        innovation_score=0.5,
        duration=3600  # 60 minutes
    )

    print(f"   Updated capability confidences")
    print()

    print("3. Current capability confidences:")

    for key, conf in ml.capability_confidence.items():
        if conf.total_spawns > 0:
            print(f"   {key[0]} for {key[1]}:")
            print(f"     Confidence: {conf.confidence_score:.2%}")
            print(f"     Interval: ({conf.confidence_interval[0]:.2%}, {conf.confidence_interval[1]:.2%})")
            print(f"     Sample size: {conf.sample_size}")
            print()

    print("4. Specialist recommendation:")
    task = "Implement OAuth2 authentication"
    recommended, confidence = ml.recommend_specialist(
        task_description=task,
        task_category="authentication",
        capability_gap={"oauth", "authentication"}
    )

    if recommended:
        print(f"   Task: {task}")
        print(f"   Recommended: {recommended}")
        print(f"   Confidence: {confidence:.2%}")
    else:
        print(f"   No recommendation (insufficient data)")

    print()

    print("5. Statistics:")
    stats = ml.get_spawning_stats()
    for key, value in stats.items():
        print(f"   {key}: {value}")


if __name__ == "__main__":
    demo_meta_learning()
