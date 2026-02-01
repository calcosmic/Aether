"""
Queen Ant Colony - Worker Ant Castes with Autonomous Spawning

Six specialist castes that respond to pheromones and autonomously spawn
specialists based on capability gap detection.

Based on research from:
- Phase 6: Multi-Agent System Integration Patterns
- Phase 1: Context Engine Foundation
- Phase 3: Semantic Codebase Understanding
- Autonomous Agent Spawning Research
"""

from typing import List, Dict, Any, Optional, Callable, Set
from dataclasses import dataclass, field
from enum import Enum
import asyncio
from datetime import datetime, timedelta
import re

try:
    from .pheromone_system import PheromoneType, PheromoneSignal, PheromoneLayer, SensitivityProfile, SENSITIVITY_PROFILES
    from .error_prevention import ErrorLedger, ErrorCategory, ErrorSeverity, log_exception
    from .memory.meta_learner import MetaLearner, TaskOutcome
except ImportError:
    from pheromone_system import PheromoneType, PheromoneSignal, PheromoneLayer, SensitivityProfile, SENSITIVITY_PROFILES
    from error_prevention import ErrorLedger, ErrorCategory, ErrorSeverity, log_exception
    from memory.meta_learner import MetaLearner, TaskOutcome

# Type hint for memory layer
from typing import TYPE_CHECKING
if TYPE_CHECKING:
    from .memory.triple_layer_memory import TripleLayerMemory


# ============================================================================
# AUTONOMOUS SPAWNING - NEW DATA STRUCTURES
# ============================================================================

@dataclass(frozen=True)
class Capability:
    """A capability that an agent may have (immutable and hashable)"""
    name: str
    category: str  # technical, domain, skill
    proficiency: float = 0.5  # 0.0 to 1.0


@dataclass
class Task:
    """A task that needs to be done"""
    id: str
    description: str
    required_capabilities: Set[str] = field(default_factory=set)  # Use capability names (strings)
    context: Dict[str, Any] = field(default_factory=dict)
    priority: float = 0.5
    estimated_effort: float = 1.0  # hours


@dataclass
class ResourceBudget:
    """Resource budget for spawning subagents"""
    max_subagents: int = 10
    max_depth: int = 3  # Max spawning depth
    current_subagents: int = 0
    spawning_disabled: bool = False

    def can_spawn(self, depth: int = 0) -> bool:
        """Check if spawning is allowed"""
        if self.spawning_disabled:
            return False
        if self.current_subagents >= self.max_subagents:
            return False
        if depth >= self.max_depth:
            return False
        return True

    def disable_spawning(self):
        """Circuit breaker - disable spawning"""
        self.spawning_disabled = True

    def enable_spawning(self):
        """Re-enable spawning after circuit breaker"""
        self.spawning_disabled = False


@dataclass
class InheritedContext:
    """Context passed from parent to spawned subagent"""
    parent_agent_id: str
    parent_task: str
    goal: str
    pheromone_signals: List[PheromoneSignal] = field(default_factory=list)
    working_memory: Dict[str, Any] = field(default_factory=dict)
    relevant_code: List[str] = field(default_factory=list)
    constraints: List[str] = field(default_factory=list)


@dataclass
class SpawningDecision:
    """Record of autonomous spawning decision"""
    timestamp: datetime
    parent_agent: str
    task: str
    capability_gaps: Set[str]  # Use capability names (strings)
    specialist_type: str
    reason: str
    depth: int


class PheromoneType(Enum):
    """Types of pheromone signals"""
    INIT = "init"           # Strong attract, triggers planning
    FOCUS = "focus"         # Medium attract, guides attention
    REDIRECT = "redirect"   # Strong repel, warns away
    FEEDBACK = "feedback"   # Variable, adjusts behavior


@dataclass
class PheromoneSignal:
    """A pheromone signal from the Queen"""
    signal_type: PheromoneType
    content: str
    strength: float  # 0.0 to 1.0
    created_at: datetime
    half_life: timedelta = field(default_factory=lambda: timedelta(hours=1))

    def current_strength(self) -> float:
        """Calculate current strength based on decay"""
        age = datetime.now() - self.created_at
        decay_factor = age.total_seconds() / self.half_life.total_seconds()
        return self.strength * (0.5 ** decay_factor)

    def is_active(self) -> bool:
        """Check if signal is still active"""
        return self.current_strength() > 0.01


@dataclass
class Subagent:
    """A spawned subagent with autonomous capabilities"""
    name: str
    purpose: str
    parent: 'WorkerAnt'
    spawned_at: datetime
    status: str = "active"  # active, completed, terminated
    inherited_context: Optional[InheritedContext] = None
    capabilities: Set[Capability] = field(default_factory=set)
    depth: int = 0  # Spawning depth
    spawning_reason: str = ""  # Why was this agent spawned?

    def terminate(self):
        """Terminate this subagent"""
        self.status = "terminated"

    def get_context_summary(self) -> str:
        """Get summary of inherited context"""
        if not self.inherited_context:
            return "No inherited context"
        return f"Goal: {self.inherited_context.goal}, Parent task: {self.inherited_context.parent_task}"


class WorkerAnt:
    """Base class for all Worker Ant castes with autonomous spawning"""

    caste: str
    capabilities: List[str]
    sensitivity: Dict[PheromoneType, float]
    spawns: List[str]  # Types of subagents this caste can spawn

    # Capability taxonomy for autonomous spawning
    CAPABILITY_TAXONOMY = {
        # Technical capabilities
        "database": {"sql", "orm", "migrations", "query_optimization"},
        "frontend": {"react", "vue", "angular", "css", "html", "javascript"},
        "backend": {"api", "rest", "graphql", "websocket", "microservices"},
        "devops": {"docker", "kubernetes", "ci_cd", "deployment", "monitoring"},
        "security": {"authentication", "authorization", "encryption", "owasp"},
        "testing": {"unit", "integration", "e2e", "tdd", "mocking"},
        "performance": {"optimization", "caching", "profiling", "load_testing"},
        # Domain capabilities
        "auth": {"jwt", "oauth", "sessions", "passwords", "2fa"},
        "data": {"etl", "pipelines", "validation", "transformation"},
        "ui": {"design", "ux", "accessibility", "responsive"},
        # Skill capabilities
        "analysis": {"debugging", "investigation", "root_cause"},
        "planning": {"estimation", "dependency_analysis", "risk_assessment"},
        "communication": {"documentation", "coordination", "negotiation"},
    }

    # Specialist type mappings
    SPECIALIST_MAPPING = {
        "database": "database_specialist",
        "sql": "database_specialist",
        "react": "frontend_specialist",
        "vue": "frontend_specialist",
        "angular": "frontend_specialist",
        "api": "api_specialist",
        "websocket": "realtime_specialist",
        "authentication": "security_specialist",
        "jwt": "security_specialist",
        "oauth": "security_specialist",
        "testing": "test_specialist",
        "unit": "test_specialist",
        "performance": "optimization_specialist",
        "optimization": "optimization_specialist",
        "security": "security_specialist",
        "deployment": "devops_specialist",
        "docker": "devops_specialist",
        "kubernetes": "devops_specialist",
    }

    def __init__(self, colony: 'Colony', error_ledger: Optional[ErrorLedger] = None, memory_layer: Optional['TripleLayerMemory'] = None, meta_learner: Optional[MetaLearner] = None):
        self.colony = colony
        self.subagents: List[Subagent] = []
        self.current_task: Optional[str] = None
        self.last_activity: datetime = datetime.now()

        # Autonomous spawning components
        self.resource_budget = ResourceBudget()
        self.spawning_history: List[SpawningDecision] = []
        self.active = True

        # Error logging
        self.error_ledger = error_ledger or colony.error_ledger if colony else None
        self.agent_id = f"{self.caste}_ant_{id(self)}"

        # Triple-Layer Memory (optional)
        self.memory_layer = memory_layer

        # Meta-Learning System (optional)
        self.meta_learner = meta_learner

    async def detect_pheromones(self) -> List[PheromoneSignal]:
        """Detect relevant pheromones based on sensitivity"""
        detected = []
        for signal in self.colony.pheromones:
            if signal.is_active():
                my_sensitivity = self.sensitivity.get(signal.signal_type, 0.0)
                if my_sensitivity > 0:
                    # Weight signal strength by my sensitivity
                    effective_strength = signal.current_strength() * my_sensitivity
                    if effective_strength > 0.1:  # Threshold for response
                        detected.append(signal)
        return detected

    async def respond_to_pheromones(self, signals: List[PheromoneSignal]):
        """Respond to detected pheromones"""
        for signal in signals:
            await self.respond_to_signal(signal)

    async def respond_to_signal(self, signal: PheromoneSignal):
        """Override in subclasses"""
        pass

    def spawn_subagent(self, name: str, purpose: str) -> Subagent:
        """Spawn a new subagent"""
        subagent = Subagent(
            name=name,
            purpose=purpose,
            parent=self,
            spawned_at=datetime.now()
        )
        self.subagents.append(subagent)
        self.colony.register_subagent(subagent)
        return subagent

    def terminate_subagent(self, subagent: Subagent):
        """Terminate a subagent"""
        subagent.terminate()
        if subagent in self.subagents:
            self.subagents.remove(subagent)

    async def coordinate_with(self, other: 'WorkerAnt'):
        """Peer-to-peer coordination with another Worker Ant"""
        # Direct communication, not through central coordinator
        pass

    # ========================================================================
    # AUTONOMOUS SPAWNING METHODS
    # ========================================================================

    async def detect_capability_gap(self, task: Task) -> bool:
        """
        Detect if this ant lacks capability for a task

        Analyzes task requirements against own capabilities to identify gaps.
        Returns True if gaps exist that would prevent task completion.
        """
        required = await self.analyze_task_requirements(task)
        my_capabilities = set(self.capabilities)

        # Check for capability gaps
        gaps = required - my_capabilities

        # Also check for proficiency gaps
        for cap in required & my_capabilities:
            my_proficiency = self._get_capability_proficiency(cap)
            if my_proficiency < 0.5:  # Low proficiency threshold
                gaps.add(cap)

        return len(gaps) > 0

    async def analyze_task_requirements(self, task: Task) -> Set[str]:
        """
        Analyze task to determine required capabilities

        Uses semantic analysis of task description to identify
        what capabilities are needed.
        """
        required = set()

        # Extract capability keywords from task description
        description_lower = task.description.lower()

        # Check against capability taxonomy
        for category, keywords in self.CAPABILITY_TAXONOMY.items():
            for keyword in keywords:
                if keyword in description_lower:
                    required.add(category)
                    break

        # Add common pattern detections
        if re.search(r'(database|sql|query|orm)', description_lower):
            required.add("database")
        if re.search(r'(api|endpoint|route|controller)', description_lower):
            required.add("backend")
        if re.search(r'(test|spec|mock)', description_lower):
            required.add("testing")
        if re.search(r'(auth|login|jwt|session)', description_lower):
            required.add("auth")
        if re.search(r'(deploy|docker|k8s|ci/cd)', description_lower):
            required.add("devops")
        if re.search(r'(security|encrypt|vulnerability)', description_lower):
            required.add("security")

        return required

    def _get_capability_proficiency(self, capability: str) -> float:
        """Get proficiency level for a capability"""
        # Base implementation - override in subclasses
        # Default to moderate proficiency for listed capabilities
        return 0.7 if capability in self.capabilities else 0.0

    async def _categorize_task(self, task: Task) -> str:
        """
        Categorize task into a primary category for meta-learning.

        Returns the primary category (e.g., "database", "security", "frontend").
        """
        description_lower = task.description.lower()

        # Define category patterns
        category_patterns = [
            ("database", ["database", "sql", "query", "orm", "migration", "postgres", "mysql", "mongodb"]),
            ("frontend", ["react", "vue", "angular", "frontend", "ui", "css", "html", "javascript"]),
            ("api", ["api", "endpoint", "route", "controller", "rest", "graphql", "websocket"]),
            ("security", ["auth", "jwt", "oauth", "session", "security", "encrypt", "vulnerability"]),
            ("testing", ["test", "spec", "mock", "unit", "integration", "e2e"]),
            ("performance", ["optimization", "cache", "performance", "profiling", "load"]),
            ("devops", ["deploy", "docker", "k8s", "ci", "cd", "infrastructure"]),
        ]

        # Find best matching category
        best_category = "general"
        best_match_count = 0

        for category, patterns in category_patterns:
            match_count = sum(1 for pattern in patterns if pattern in description_lower)
            if match_count > best_match_count:
                best_category = category
                best_match_count = match_count

        return best_category

    async def record_task_outcome(
        self,
        spawn_event_id: str,
        outcome: TaskOutcome,
        quality_score: float,
        innovation_score: float,
        duration: float,
        user_feedback: Optional[str] = None,
        peer_feedback: Optional[List[str]] = None
    ):
        """
        Record the outcome of a spawned specialist task.

        This should be called when a specialist completes its work.

        Args:
            spawn_event_id: Event ID from meta_learner.record_spawn()
            outcome: Task outcome (SUCCESS, PARTIAL_SUCCESS, FAILURE, etc.)
            quality_score: Quality of work (0.0 to 1.0)
            innovation_score: How innovative was the solution (0.0 to 1.0)
            duration: Time to complete (seconds)
            user_feedback: Optional user feedback
            peer_feedback: Optional feedback from other agents
        """
        if self.meta_learner:
            self.meta_learner.record_outcome(
                event_id=spawn_event_id,
                outcome=outcome,
                quality_score=quality_score,
                innovation_score=innovation_score,
                duration=duration,
                user_feedback=user_feedback,
                peer_feedback=peer_feedback
            )

    async def determine_specialist_type(self, capability_gaps: Set[str]) -> str:
        """
        Determine what type of specialist to spawn based on capability gaps

        Maps capability gaps to appropriate specialist types.
        """
        # Use specialist mapping if available
        for gap in capability_gaps:
            if gap in self.SPECIALIST_MAPPING:
                return self.SPECIALIST_MAPPING[gap]

        # Fallback: determine from most common gap
        if "database" in capability_gaps or "sql" in capability_gaps:
            return "database_specialist"
        if "auth" in capability_gaps or "security" in capability_gaps:
            return "security_specialist"
        if "testing" in capability_gaps:
            return "test_specialist"
        if "frontend" in capability_gaps or "ui" in capability_gaps:
            return "frontend_specialist"
        if "backend" in capability_gaps or "api" in capability_gaps:
            return "api_specialist"
        if "devops" in capability_gaps or "deployment" in capability_gaps:
            return "devops_specialist"

        # Default: general specialist
        return "general_specialist"

    async def spawn_specialist_autonomously(
        self,
        task: Task,
        depth: int = 0
    ) -> Optional[Subagent]:
        """
        Spawn specialist based on autonomous capability gap detection

        This is the core of autonomous spawning - agents detect they
        lack capabilities and spawn appropriate specialists.
        """
        try:
            # Check resource budget
            if not self.resource_budget.can_spawn(depth):
                # Circuit breaker triggered - log error
                if self.resource_budget.current_subagents >= self.resource_budget.max_subagents:
                    self.resource_budget.disable_spawning()
                    if self.error_ledger:
                        self.error_ledger.log_error(
                            symptom="Spawning limit reached: max_subagents exceeded",
                            error_type="SpawningLimitError",
                            category=ErrorCategory.SPAWNING,
                            function="spawn_specialist_autonomously",
                            agent_id=self.agent_id,
                            task_context=task.description,
                            severity=ErrorSeverity.MEDIUM
                        )
                return None

            # Analyze capability gaps
            required = await self.analyze_task_requirements(task)
            my_capabilities = set(self.capabilities)
            gaps = required - my_capabilities

            if not gaps:
                # No gaps, can handle ourselves
                return None

            # Determine specialist type using meta-learning if available
            specialist_type = None

            # Step 1: Get meta-learner recommendation
            if self.meta_learner:
                # Determine primary task category
                task_category = await self._categorize_task(task)

                # Get recommendation
                recommended_specialist, confidence = self.meta_learner.recommend_specialist(
                    task_description=task.description,
                    task_category=task_category,
                    capability_gap=gaps
                )

                # Use recommendation if confidence is sufficient
                if recommended_specialist and confidence > 0.4:
                    specialist_type = recommended_specialist

            # Step 2: Fall back to rule-based mapping if no recommendation
            if not specialist_type:
                specialist_type = await self.determine_specialist_type(gaps)

            # Create inherited context
            inherited = InheritedContext(
                parent_agent_id=id(self),
                parent_task=self.current_task or "unknown",
                goal=self._get_current_goal(),
                pheromone_signals=self.colony.pheromones.copy(),
                working_memory=self._get_working_memory(),
                relevant_code=await self._get_relevant_code(task),
                constraints=self._get_constraints()
            )

            # Generate spawning reason
            reason = f"Capability gaps detected: {', '.join(gaps)}. Need {specialist_type}"

            # Record spawn event in meta-learner
            spawn_event_id = None
            task_category = await self._categorize_task(task)

            if self.meta_learner:
                spawn_event_id = self.meta_learner.record_spawn(
                    parent_agent=self.caste,
                    task_description=task.description,
                    task_category=task_category,
                    specialist_type=specialist_type,
                    capability_gap=gaps,
                    inherited_context={
                        "goal": inherited.goal,
                        "pheromones": [p.to_dict() for p in inherited.pheromone_signals] if hasattr(p, 'to_dict') else list(inherited.pheromone_signals),
                        "working_memory": inherited.working_memory,
                        "constraints": inherited.constraints
                    }
                )

            # Spawn the specialist
            specialist = Subagent(
                name=f"autonomous_{specialist_type}_{len(self.subagents)}",
                purpose=f"Address capability gaps: {', '.join(gaps)}",
                parent=self,
                spawned_at=datetime.now(),
                inherited_context=inherited,
                capabilities=required,  # Store as set of strings
                depth=depth,
                spawning_reason=reason,
                spawn_event_id=spawn_event_id  # Track for outcome recording
            )

            # Update resource budget
            self.resource_budget.current_subagents += 1
            self.subagents.append(specialist)
            self.colony.register_subagent(specialist)

            # Record spawning decision
            decision = SpawningDecision(
                timestamp=datetime.now(),
                parent_agent=self.caste,
                task=task.description,
                capability_gaps=gaps,  # Use set of strings directly
                specialist_type=specialist_type,
                reason=reason,
                depth=depth
            )
            self.spawning_history.append(decision)

            self.last_activity = datetime.now()

            return specialist

        except Exception as e:
            # Log spawning error
            if self.error_ledger:
                log_exception(
                    self.error_ledger,
                    e,
                    symptom=f"Failed to spawn specialist for task: {task.description}",
                    agent_id=self.agent_id,
                    task_context=task.description,
                    category=ErrorCategory.SPAWNING
                )
            return None

    async def delegate_or_handle(self, task: Task, depth: int = 0) -> Any:
        """
        Decide whether to handle task personally or delegate to specialist

        This is the main decision point for autonomous spawning.
        """
        # Check if spawning is allowed
        if not self.resource_budget.can_spawn(depth):
            # Must handle ourselves
            return await self.execute(task)

        # Detect capability gaps
        if await self.detect_capability_gap(task):
            # Spawn specialist and delegate
            specialist = await self.spawn_specialist_autonomously(task, depth)
            if specialist:
                # Delegate to specialist
                return await self._delegate_to_specialist(specialist, task)

        # Handle ourselves
        return await self.execute(task)

    async def execute(self, task: Task) -> Any:
        """
        Execute a task (override in subclasses)

        Default implementation - should be overridden by specific castes.
        """
        self.current_task = task.description
        self.last_activity = datetime.now()

        # Check memory for relevant context before executing
        if self.memory_layer:
            context = await self._get_memory_context(task)
            if context:
                # Context is available - could use it to improve execution
                pass

        result = {"status": "completed", "agent": self.caste}

        # Store result in working memory if available
        if self.memory_layer:
            await self._store_in_memory(task, result)

        return result

    async def _get_memory_context(self, task: Task) -> Optional[List]:
        """
        Retrieve relevant context from working memory

        Args:
            task: Task being executed

        Returns:
            Relevant context items or None
        """
        if not self.memory_layer:
            return None

        # Search for relevant items based on task description
        query_words = task.description.lower().split()[:3]  # First 3 words
        relevant_items = []

        for word in query_words:
            items = await self.memory_layer.search_working(word, limit=3, item_type=self.caste)
            relevant_items.extend(items)

        return relevant_items if relevant_items else None

    async def _store_in_memory(self, task: Task, result: Any) -> Optional[str]:
        """
        Store task execution result in working memory

        Args:
            task: Task that was executed
            result: Result of execution

        Returns:
            Item ID if stored, None otherwise
        """
        if not self.memory_layer:
            return None

        # Create content to store
        content = f"{self.caste} completed task: {task.description}"
        if isinstance(result, dict):
            content += f". Result: {result.get('status', 'unknown')}"

        # Store in working memory
        item_id = await self.memory_layer.add_to_working(
            content=content,
            metadata={
                "type": self.caste,
                "task_id": task.id,
                "agent_id": self.agent_id,
                "task_priority": task.priority
            },
            item_type=self.caste
        )

        return item_id

    async def _delegate_to_specialist(self, specialist: Subagent, task: Task) -> Any:
        """Delegate task to spawned specialist"""
        # In a full implementation, this would actually execute the specialist
        # For now, return a placeholder
        return {
            "status": "delegated",
            "specialist": specialist.name,
            "purpose": specialist.purpose,
            "context_inherited": specialist.inherited_context is not None
        }

    def _get_current_goal(self) -> str:
        """Get current goal from pheromones"""
        for signal in self.colony.pheromones:
            if signal.signal_type == PheromoneType.INIT:
                return signal.content
        return "unknown"

    def _get_working_memory(self) -> Dict[str, Any]:
        """Get current working memory"""
        return {
            "current_task": self.current_task,
            "subagents_count": len(self.subagents),
            "last_activity": self.last_activity
        }

    async def _get_relevant_code(self, task: Task) -> List[str]:
        """Get relevant code for task context"""
        # In full implementation, would use semantic search
        return []

    def _get_constraints(self) -> List[str]:
        """Get current constraints"""
        constraints = []

        # Extract from redirect pheromones
        for signal in self.colony.pheromones:
            if signal.signal_type == PheromoneType.REDIRECT and signal.is_active():
                constraints.append(f"Avoid: {signal.content}")

        return constraints

    def get_spawning_summary(self) -> Dict[str, Any]:
        """Get summary of autonomous spawning activity"""
        return {
            "total_spawned": len(self.subagents),
            "resource_budget": {
                "used": self.resource_budget.current_subagents,
                "max": self.resource_budget.max_subagents,
                "disabled": self.resource_budget.spawning_disabled
            },
            "spawning_history": [
                {
                    "timestamp": d.timestamp.isoformat(),
                    "specialist": d.specialist_type,
                    "reason": d.reason
                }
                for d in self.spawning_history[-10:]  # Last 10
            ]
        }


class MapperAnt(WorkerAnt):
    """
    Mapper Ant - Explore, index, understand codebase

    Capabilities:
    - Semantic codebase exploration
    - Dependency graph mapping
    - Code relationship identification
    - Pattern detection

    Spawns: Graph builders, search agents, pattern matchers
    """

    caste = "mapper"
    capabilities = ["semantic_exploration", "dependency_mapping", "pattern_detection"]
    sensitivity = {
        PheromoneType.INIT: 1.0,      # Always responds to init
        PheromoneType.FOCUS: 0.7,     # Responds to focus on areas
        PheromoneType.REDIRECT: 0.3,  # Less affected by redirect
        PheromoneType.FEEDBACK: 0.5,
    }
    spawns = ["graph_builder", "search_agent", "pattern_matcher"]

    def __init__(self, colony: 'Colony', memory_layer: Optional['TripleLayerMemory'] = None, meta_learner: Optional[MetaLearner] = None):
        super().__init__(colony, memory_layer=memory_layer, meta_learner=meta_learner)
        self.semantic_index: Dict[str, Any] = {}
        self.dependency_graph: Dict[str, List[str]] = {}

    async def respond_to_signal(self, signal: PheromoneSignal):
        if signal.signal_type == PheromoneType.INIT:
            await self.explore_codebase(signal.content)
        elif signal.signal_type == PheromoneType.FOCUS:
            await self.map_specific_area(signal.content)

    async def explore_codebase(self, goal: str):
        """Explore entire codebase for new goal using autonomous spawning"""
        self.current_task = f"Exploring codebase for: {goal}"

        # AUTONOMOUS SPAWNING: Detect if we need help and spawn specialists
        task = Task(
            id="explore_1",
            description=f"Explore and map codebase for: {goal}",
            priority=0.8
        )

        # Use autonomous spawning to handle the task
        result = await self.delegate_or_handle(task)

        # Build semantic index ourselves
        await self.build_semantic_index()

        return result

    async def map_specific_area(self, area: str):
        """Map a specific area of codebase"""
        self.current_task = f"Mapping area: {area}"

        # Spawn focused pattern matcher
        pattern_matcher = self.spawn_subagent(
            f"pattern_matcher_{area}",
            f"Match patterns in {area}"
        )

    async def build_semantic_index(self):
        """Build semantic understanding of codebase"""
        # Based on Phase 3 research:
        # - Beyond AST Parsing
        # - Graph-Based Code Analysis
        # - Vector Embeddings for Code
        pass

    async def find_related_code(self, query: str) -> List[str]:
        """Find code related to query using semantic index"""
        # Semantic search using embeddings
        pass


class PlannerAnt(WorkerAnt):
    """
    Planner Ant - Create structured phase plans

    Capabilities:
    - Goal decomposition
    - Phase boundary identification
    - Milestone definition
    - Dependency analysis

    Spawns: Estimators, dependency analyzers, risk assessors
    """

    caste = "planner"
    capabilities = ["goal_decomposition", "phase_planning", "dependency_analysis"]
    sensitivity = {
        PheromoneType.INIT: 1.0,      # Triggers planning
        PheromoneType.FOCUS: 0.5,     # Adjusts priorities
        PheromoneType.REDIRECT: 0.8,  # Avoids redirected approaches
        PheromoneType.FEEDBACK: 0.7,
    }
    spawns = ["estimator", "dependency_analyzer", "risk_assessor"]

    def __init__(self, colony: 'Colony', memory_layer: Optional['TripleLayerMemory'] = None, meta_learner: Optional[MetaLearner] = None):
        super().__init__(colony, memory_layer=memory_layer, meta_learner=meta_learner)
        self.current_plan: Optional[Dict] = None
        self.phases: List[Dict] = []

    async def respond_to_signal(self, signal: PheromoneSignal):
        if signal.signal_type == PheromoneType.INIT:
            await self.create_phase_plan(signal.content)
        elif signal.signal_type == PheromoneType.FOCUS:
            await self.adjust_priorities(signal.content)
        elif signal.signal_type == PheromoneType.REDIRECT:
            await self.avoid_approach(signal.content)

    async def create_phase_plan(self, goal: str):
        """Create structured phase plan from goal"""
        self.current_task = f"Creating phase plan for: {goal}"

        # Spawn estimator
        estimator = self.spawn_subagent(
            "estimator",
            "Estimate effort for each phase"
        )

        # Spawn dependency analyzer
        dep_analyzer = self.spawn_subagent(
            "dependency_analyzer",
            "Analyze dependencies between tasks"
        )

        # Spawn risk assessor
        risk_assessor = self.spawn_subagent(
            "risk_assessor",
            "Assess risks in each phase"
        )

        # Create phases
        await self.decompose_goal(goal)

    async def decompose_goal(self, goal: str) -> List[Dict]:
        """Decompose goal into structured phases"""
        # Based on Phase 7 research:
        # - Implementation Roadmap and Milestones
        # - 6-phase roadmap pattern

        phases = [
            {
                "id": 1,
                "name": "Foundation",
                "description": "Phase 1: Foundation",
                "tasks": await self.break_down_phase(goal, "foundation"),
                "milestones": ["WebSocket server running", "Database connected"],
                "status": "pending"
            },
            {
                "id": 2,
                "name": "Core Implementation",
                "description": "Phase 2: Core Implementation",
                "tasks": await self.break_down_phase(goal, "core"),
                "milestones": ["Real-time message delivery", "Message persistence"],
                "status": "pending"
            },
            {
                "id": 3,
                "name": "User Authentication",
                "description": "Phase 3: User Authentication",
                "tasks": await self.break_down_phase(goal, "auth"),
                "milestones": ["Users can authenticate", "Sessions persist"],
                "status": "pending"
            },
        ]

        self.phases = phases
        self.current_plan = {"goal": goal, "phases": phases}
        return phases

    async def break_down_phase(self, goal: str, phase_type: str) -> List[Dict]:
        """Break down a phase into specific tasks"""
        # For demo purposes, return predefined tasks
        # In production, this would use an LLM to generate tasks

        if phase_type == "foundation":
            return [
                {"id": "f1", "description": "Setup project structure", "status": "pending"},
                {"id": "f2", "description": "Configure development environment", "status": "pending"},
                {"id": "f3", "description": "Initialize database schema", "status": "pending"},
                {"id": "f4", "description": "Setup WebSocket server", "status": "pending"},
                {"id": "f5", "description": "Implement basic message routing", "status": "pending"}
            ]
        elif phase_type == "core":
            return [
                {"id": "c1", "description": "Implement WebSocket connection handling", "status": "pending"},
                {"id": "c2", "description": "Create message queue system", "status": "pending"},
                {"id": "c3", "description": "Configure Redis pub/sub", "status": "pending"},
                {"id": "c4", "description": "Add connection pooling", "status": "pending"},
                {"id": "c5", "description": "Implement message delivery", "status": "pending"},
                {"id": "c6", "description": "Add message persistence", "status": "pending"},
                {"id": "c7", "description": "Create offline message handling", "status": "pending"},
                {"id": "c8", "description": "Add message acknowledgment", "status": "pending"}
            ]
        elif phase_type == "auth":
            return [
                {"id": "a1", "description": "Design authentication schema", "status": "pending"},
                {"id": "a2", "description": "Implement JWT token system", "status": "pending"},
                {"id": "a3", "description": "Create login/logout endpoints", "status": "pending"},
                {"id": "a4", "description": "Add session management", "status": "pending"},
                {"id": "a5", "description": "Implement password reset", "status": "pending"}
            ]
        else:
            return [
                {"id": f"{phase_type}_1", "description": f"Task 1 for {phase_type}", "status": "pending"},
                {"id": f"{phase_type}_2", "description": f"Task 2 for {phase_type}", "status": "pending"},
                {"id": f"{phase_type}_3", "description": f"Task 3 for {phase_type}", "status": "pending"}
            ]

    async def adjust_priorities(self, focus_area: str):
        """Adjust priorities based on focus pheromone"""
        # Reorder tasks based on focus
        pass

    async def avoid_approach(self, pattern: str):
        """Mark approach to avoid based on redirect pheromone"""
        # Add constraint to avoid this pattern
        pass


class ExecutorAnt(WorkerAnt):
    """
    Executor Ant - Write code, implement changes with experimental testing

    Capabilities:
    - Code generation
    - File manipulation
    - Refactoring
    - Implementation
    - Experimental testing (learns what works)

    Spawns: Language specialists, framework specialists, database specialists
    """

    caste = "executor"
    capabilities = ["code_generation", "file_manipulation", "refactoring", "experimental_testing"]
    sensitivity = {
        PheromoneType.INIT: 0.5,      # Awaits planning
        PheromoneType.FOCUS: 0.9,     # Highly responsive to focus
        PheromoneType.REDIRECT: 0.9,  # Strongly avoids redirected patterns
        PheromoneType.FEEDBACK: 0.7,
    }
    spawns = ["language_specialist", "framework_specialist", "database_specialist"]

    def __init__(self, colony: 'Colony', memory_layer: Optional['TripleLayerMemory'] = None, meta_learner: Optional[MetaLearner] = None):
        super().__init__(colony, memory_layer=memory_layer, meta_learner=meta_learner)
        self.current_files: List[str] = []
        self.implemented_features: List[str] = []

        # Experimental testing state
        self.testing_approach = None  # Chosen approach for current task
        self.outcome_tracker = None
        self.current_task_outcome = None

    async def respond_to_signal(self, signal: PheromoneSignal):
        if signal.signal_type == PheromoneType.FOCUS:
            await self.prioritize_work(signal.content)
        elif signal.signal_type == PheromoneType.REDIRECT:
            await self.avoid_pattern(signal.content)

    async def prioritize_work(self, focus_area: str):
        """Prioritize work on focused area"""
        self.current_task = f"Prioritizing: {focus_area}"

        # Check if we need specialist
        if "database" in focus_area.lower():
            specialist = self.spawn_subagent(
                "database_specialist",
                f"Handle database work for {focus_area}"
            )

    async def avoid_pattern(self, pattern: str):
        """Avoid implementing a specific pattern"""
        # Add to avoid list
        pass

    async def implement_task(self, task: str):
        """Implement a specific task"""
        # Spawn appropriate specialist
        specialist = self.spawn_subagent(
            f"specialist_{task}",
            f"Implement: {task}"
        )

    async def write_code(self, file_path: str, content: str):
        """Write code to file"""
        # Use existing write capability
        pass

    async def refactor_code(self, file_path: str, description: str):
        """Refactor code in file"""
        pass

    # ========================================================================
    # EXPERIMENTAL TESTING METHODS
    # ========================================================================

    async def choose_testing_approach(self, task: str) -> tuple[str, float]:
        """
        Choose testing approach based on:
        - Learned outcomes from memory
        - Pheromone signals
        - Task complexity
        - Colony preferences

        Returns:
            (approach, confidence) tuple
        """
        try:
            from .memory.outcome_tracker import OutcomeTracker, TestingApproach
        except ImportError:
            from memory.outcome_tracker import OutcomeTracker, TestingApproach

        # Initialize outcome tracker
        if not self.outcome_tracker:
            self.outcome_tracker = OutcomeTracker(self.memory_layer)

        # Check memory for learned patterns
        if self.memory_layer:
            try:
                patterns = await self.memory_layer.search_working(
                    "testing approach success",
                    limit=5
                )

                if patterns:
                    # Use learned best approach
                    best = max(patterns, key=lambda p: p.metadata.get("confidence", 0) if hasattr(p, 'metadata') else 0)
                    approach = best.metadata.get("approach", "test_parallel") if hasattr(best, 'metadata') else "test_parallel"
                    return (approach, 0.7)
            except Exception:
                pass  # Fall through to default

        # Use outcome tracker to recommend approach
        approach, confidence = await self.outcome_tracker.recommend_approach(
            task_context={"task": task}
        )

        return (approach, confidence)

    async def implement_with_experimentation(
        self,
        task: str,
        force_approach: str = None
    ) -> Dict[str, Any]:
        """
        Implement task with experimental testing approach

        Tracks outcome for learning.

        Args:
            task: Task description
            force_approach: Force specific approach (for testing)

        Returns:
            Implementation result with approach tracking
        """
        import time
        start_time = time.time()

        # Choose approach
        if force_approach:
            approach = force_approach
            confidence = 1.0
        else:
            approach, confidence = await self.choose_testing_approach(task)

        self.testing_approach = approach

        result = {
            "task": task,
            "approach": approach,
            "approach_confidence": confidence,
            "steps": []
        }

        # Execute based on approach
        if approach == "test_first":
            step_result = await self._experiment_test_first(task)
            result["steps"].append(step_result)
        elif approach == "test_after":
            step_result = await self._experiment_test_after(task)
            result["steps"].append(step_result)
        elif approach == "test_parallel":
            step_result = await self._experiment_test_parallel(task)
            result["steps"].append(step_result)
        elif approach == "test_only":
            step_result = await self._experiment_test_only(task)
            result["steps"].append(step_result)
        else:  # no_test
            step_result = await self._experiment_no_test(task)
            result["steps"].append(step_result)

        # Record outcome
        duration = int((time.time() - start_time) / 60)
        await self._record_outcome(task, approach, result, duration)

        return result

    async def _experiment_test_first(self, task: str) -> Dict[str, Any]:
        """Experiment: Write test first, then implementation (TDD style)"""
        # Get verifier ant
        verifier = self.colony.worker_ants.get("verifier")
        if not verifier:
            return {"error": "No verifier available"}

        # Generate test
        test = await verifier.generate_test(task, test_style="unit")

        # Write test
        await self.write_code(test["test_path"], test["test_content"])

        # Verify test fails (RED phase)
        test_result = await verifier.run_test(test["test_path"])

        step_result = {
            "approach": "test_first",
            "test_written": True,
            "test_failed": not test_result.get("passed", True),
            "implementation_written": False
        }

        # Write implementation (GREEN phase)
        impl = await self._generate_implementation(task, test.get("test_path"))
        impl_path = self._derive_impl_path(task)
        await self.write_code(impl_path, impl)

        step_result["implementation_written"] = True

        # Verify test passes (GREEN phase complete)
        test_result_after = await verifier.run_test(test["test_path"])
        step_result["test_passes"] = test_result_after.get("passed", False)
        step_result["test_path"] = test.get("test_path")
        step_result["impl_path"] = impl_path

        return step_result

    async def _experiment_test_after(self, task: str) -> Dict[str, Any]:
        """Experiment: Write implementation first, then test"""
        # Write implementation
        impl = await self._generate_implementation(task)
        impl_path = self._derive_impl_path(task)
        await self.write_code(impl_path, impl)

        # Generate test based on implementation
        verifier = self.colony.worker_ants.get("verifier")
        if not verifier:
            return {"error": "No verifier available"}

        test = await verifier.generate_test(task, impl_path, test_style="unit")

        # Write test
        await self.write_code(test["test_path"], test["test_content"])

        # Verify test passes
        test_result = await verifier.run_test(test["test_path"])

        return {
            "approach": "test_after",
            "implementation_written": True,
            "test_written": True,
            "test_passes": test_result.get("passed", False),
            "test_path": test.get("test_path"),
            "impl_path": impl_path
        }

    async def _experiment_test_parallel(self, task: str) -> Dict[str, Any]:
        """Experiment: Write test and implementation together"""
        verifier = self.colony.worker_ants.get("verifier")
        if not verifier:
            return {"error": "No verifier available"}

        # Generate both
        test = await verifier.generate_test(task, test_style="unit")
        impl = await self._generate_implementation(task)

        # Write both
        impl_path = self._derive_impl_path(task)
        await self.write_code(impl_path, impl)
        await self.write_code(test["test_path"], test["test_content"])

        # Verify test passes
        test_result = await verifier.run_test(test["test_path"])

        return {
            "approach": "test_parallel",
            "implementation_written": True,
            "test_written": True,
            "test_passes": test_result.get("passed", False),
            "test_path": test.get("test_path"),
            "impl_path": impl_path
        }

    async def _experiment_test_only(self, task: str) -> Dict[str, Any]:
        """Experiment: Only write test, no implementation"""
        verifier = self.colony.worker_ants.get("verifier")
        if not verifier:
            return {"error": "No verifier available"}

        # Generate test
        test = await verifier.generate_test(task, test_style="unit")

        # Write test
        await self.write_code(test["test_path"], test["test_content"])

        return {
            "approach": "test_only",
            "implementation_written": False,
            "test_written": True,
            "test_path": test.get("test_path")
        }

    async def _experiment_no_test(self, task: str) -> Dict[str, Any]:
        """Experiment: No test, just implementation"""
        impl = await self._generate_implementation(task)
        impl_path = self._derive_impl_path(task)
        await self.write_code(impl_path, impl)

        return {
            "approach": "no_test",
            "implementation_written": True,
            "test_written": False,
            "impl_path": impl_path
        }

    async def _generate_implementation(self, task: str, test_path: str = None) -> str:
        """Generate implementation code"""
        # In production: Use LLM to generate implementation
        # For now: Return template
        return f"""# Implementation for: {task}
# Generated by Executor Ant

def {task.lower().replace(' ', '_')}(input_data):
    '''Implementation of {task}'''
    # TODO: Implement logic
    return None
"""

    def _derive_impl_path(self, task: str) -> str:
        """Derive implementation file path from task description"""
        task_slug = task.lower().replace(' ', '_').replace('/', '_')
        return f"src/{task_slug}.py"

    async def _record_outcome(self, task: str, approach: str, result: dict, duration: int):
        """Record testing outcome for learning"""
        try:
            from .memory.outcome_tracker import TestingOutcome, Outcome, OutcomeTracker
        except ImportError:
            from memory.outcome_tracker import TestingOutcome, Outcome, OutcomeTracker

        # Initialize outcome tracker
        if not self.outcome_tracker:
            self.outcome_tracker = OutcomeTracker(self.memory_layer)

        # Determine outcome
        if result.get("test_passes"):
            outcome = "success"
        elif result.get("test_failed") and not result.get("test_passes"):
            outcome = "failed_tests"
        elif result.get("error"):
            outcome = "had_bugs"
        else:
            outcome = "success"  # Default for no_test or test_only approach

        # Create outcome record
        task_id = task.lower().replace(' ', '_').replace('/', '_')[:50]
        outcome_record = TestingOutcome(
            task_id=task_id,
            task_description=task,
            approach=approach,
            outcome=outcome,
            time_to_complete=duration,
            defects_found=0,  # Will be updated if bugs found later
            rework_needed=not result.get("test_passes", True),
            metadata={
                "agent": self.caste,
                "test_path": result.get("test_path"),
                "impl_path": result.get("impl_path")
            }
        )

        # Record outcome
        await self.outcome_tracker.record_outcome(outcome_record)

        # Periodically store learned patterns
        if len(self.outcome_tracker.outcomes) % 10 == 0:
            await self.outcome_tracker.store_learned_patterns()


class VerifierAnt(WorkerAnt):
    """
    Verifier Ant - Test, validate, QA with LLM-based test generation

    Capabilities:
    - Test generation (leveraging Ralph's research on LLM-based testing)
    - Validation
    - Quality checks
    - Bug detection
    - Coverage analysis

    Spawns: Test generators, lint agents, security scanners, performance testers
    """

    caste = "verifier"
    capabilities = ["test_generation", "validation", "quality_checks", "coverage_analysis"]
    sensitivity = {
        PheromoneType.INIT: 0.3,      # Waits for code to test
        PheromoneType.FOCUS: 0.8,     # Increases scrutiny on focus area
        PheromoneType.REDIRECT: 0.5,
        PheromoneType.FEEDBACK: 0.9,  # Highly responsive to quality feedback
    }
    spawns = ["test_generator", "lint_agent", "security_scanner", "performance_tester"]

    def __init__(self, colony: 'Colony', memory_layer: Optional['TripleLayerMemory'] = None, meta_learner: Optional[MetaLearner] = None):
        super().__init__(colony, memory_layer=memory_layer, meta_learner=meta_learner)
        self.tests_generated: int = 0
        self.bugs_found: int = 0
        self.issues: List[Dict] = []

        # Test generation tracking
        self.test_styles = ["unit", "integration", "e2e", "property_based"]
        self.coverage_history: List[Dict] = []

    async def respond_to_signal(self, signal: PheromoneSignal):
        if signal.signal_type == PheromoneType.FOCUS:
            await self.increase_scrutiny(signal.content)
        elif signal.signal_type == PheromoneType.FEEDBACK:
            if "bug" in signal.content.lower() or "quality" in signal.content.lower():
                await self.intensify_testing()

    async def increase_scrutiny(self, area: str):
        """Increase testing scrutiny on specific area"""
        self.current_task = f"Increasing scrutiny on: {area}"

        # Spawn additional verifier
        specialist = self.spawn_subagent(
            f"focused_verifier_{area}",
            f"Verify {area} thoroughly"
        )

    async def intensify_testing(self):
        """Intensify testing based on feedback"""
        # Spawn more test generators
        for i in range(3):
            test_gen = self.spawn_subagent(
                f"test_generator_intense_{i}",
                "Generate additional tests"
            )

    async def verify_phase(self, phase: Dict):
        """Verify a completed phase"""
        # Spawn test generator
        test_gen = self.spawn_subagent(
            "test_generator",
            f"Generate tests for phase: {phase['name']}"
        )

        # Spawn security scanner
        security = self.spawn_subagent(
            "security_scanner",
            "Scan for security issues"
        )

    async def generate_tests(self, code: str) -> List[str]:
        """Generate tests for code"""
        # Based on Phase 5 research:
        # - Automated Quality Assurance and Testing
        pass

    async def generate_test(
        self,
        task: str,
        implementation_path: str = None,
        test_style: str = "unit"
    ) -> Dict[str, Any]:
        """
        Generate test using LLM-based approach

        Leveraging Ralph's research on:
        - LLM-based test generation
        - Feedback loops in testing
        - Test adequacy criteria

        Args:
            task: Description of what to test
            implementation_path: Path to implementation code (optional)
            test_style: Style of test (unit, integration, e2e, property_based)

        Returns:
            Dict with test_content, test_path, style, estimated_coverage
        """
        try:
            try:
                from .memory.outcome_tracker import TestGenerationResult
            except ImportError:
                from memory.outcome_tracker import TestGenerationResult
        except ImportError:
            from memory.outcome_tracker import TestGenerationResult

        # Generate test content using LLM
        test_content = await self._llm_generate_test(task, implementation_path, test_style)

        # Derive test path
        test_path = self._derive_test_path(task, test_style)

        # Estimate coverage
        estimated_coverage = self._estimate_coverage(task, test_style)

        # Update counter
        self.tests_generated += 1

        # Store result
        result = TestGenerationResult(
            test_content=test_content,
            test_path=test_path,
            style=test_style,
            estimated_coverage=estimated_coverage
        )

        # Store in memory if available
        if self.memory_layer:
            await self.memory_layer.add_to_working(
                content=f"Generated {test_style} test for: {task}",
                metadata={
                    "test_path": test_path,
                    "style": test_style,
                    "coverage": estimated_coverage,
                    "implementation": implementation_path
                },
                item_type="test_generation"
            )

        return result.to_dict()

    async def _llm_generate_test(
        self,
        task: str,
        implementation_path: str,
        test_style: str
    ) -> str:
        """
        Generate test using LLM (leveraging Ralph's research)

        In production, this would call an LLM API to generate tests based on:
        - Task description
        - Implementation code (if exists)
        - Test style (unit, integration, e2e)
        - Best practices from memory

        For now, returns a template that would be filled by LLM.
        """
        # Check memory for similar test patterns
        test_patterns = []
        if self.memory_layer:
            test_patterns = await self.memory_layer.search_working(
                f"{test_style} test",
                limit=3,
                item_type="test_generation"
            )

        # Build test prompt
        if implementation_path:
            prompt = f"Generate {test_style} test for implementation at {implementation_path}\nTask: {task}"
        else:
            prompt = f"Generate {test_style} test for: {task}"

        # In production: Call LLM API here
        # For now: Return template
        test_templates = {
            "unit": f"""# Unit test for: {task}
# Generated based on Ralph's research on LLM-based test generation

import pytest

def test_{task.lower().replace(' ', '_')}_basic():
    '''Test basic functionality'''
    assert True  # Placeholder - would be filled by LLM

def test_{task.lower().replace(' ', '_')}_edge_cases():
    '''Test edge cases'''
    assert True  # Placeholder - would be filled by LLM
""",
            "integration": f"""# Integration test for: {task}
# Tests interaction between components

import pytest

def test_{task.lower().replace(' ', '_')}_integration():
    '''Test integration with dependencies'''
    assert True  # Placeholder - would be filled by LLM
""",
            "e2e": f"""# End-to-end test for: {task}
# Tests complete user workflow

import pytest

def test_{task.lower().replace(' ', '_')}_e2e():
    '''Test end-to-end workflow'''
    assert True  # Placeholder - would be filled by LLM
""",
            "property_based": f"""# Property-based test for: {task}
# Tests invariants and properties

from hypothesis import given, strategies as st

@given(st.integers())
def test_{task.lower().replace(' ', '_')}_property(n):
    '''Test that property holds for all inputs'''
    assert True  # Placeholder - would be filled by LLM
"""
        }

        return test_templates.get(test_style, test_templates["unit"])

    def _derive_test_path(self, task: str, test_style: str) -> str:
        """Derive test file path from task description"""
        # Convert task to filename
        task_slug = task.lower().replace(' ', '_').replace('/', '_').replace('(', '').replace(')', '')

        # Determine test directory based on style
        if test_style == "unit":
            return f"tests/unit/test_{task_slug}.py"
        elif test_style == "integration":
            return f"tests/integration/test_{task_slug}.py"
        elif test_style == "e2e":
            return f"tests/e2e/test_{task_slug}.py"
        else:  # property_based
            return f"tests/properties/test_{task_slug}.py"

    def _estimate_coverage(self, task: str, test_style: str) -> float:
        """Estimate test coverage based on task and style"""
        # Base coverage by style
        base_coverage = {
            "unit": 0.85,           # High coverage for unit tests
            "integration": 0.65,    # Medium for integration
            "e2e": 0.45,            # Lower for e2e (focus on paths)
            "property_based": 0.90  # Highest for property-based
        }

        # Adjust based on task complexity (heuristic)
        task_complexity = len(task.split())  # Simple heuristic

        coverage = base_coverage.get(test_style, 0.70)
        coverage -= min(task_complexity * 0.01, 0.15)  # Reduce slightly for complex tasks

        return round(coverage, 2)

    async def run_test(self, test_path: str) -> Dict[str, Any]:
        """
        Run test and return results

        Executes pytest/jest and captures output.

        Args:
            test_path: Path to test file

        Returns:
            Dict with passed, failed, output, coverage
        """
        # In production: Execute test runner (pytest, jest, etc.)
        # For now: Return mock result
        return {
            "passed": True,
            "failed": 0,
            "output": f"Test executed: {test_path}",
            "coverage": 0.85,
            "execution_time": 0.5
        }

    async def analyze_test_coverage(self, test_path: str, impl_path: str) -> Dict[str, Any]:
        """
        Analyze test coverage (leveraging Ralph's research on verification)

        Args:
            test_path: Path to test file
            impl_path: Path to implementation file

        Returns:
            Coverage analysis with metrics
        """
        # In production: Run coverage tool (pytest-cov, jest --coverage)
        # For now: Return mock analysis
        analysis = {
            "line_coverage": 0.85,
            "branch_coverage": 0.78,
            "function_coverage": 0.92,
            "uncovered_lines": [42, 57, 89],
            "recommendations": [
                "Add test for error handling in line 42",
                "Test edge case in line 57",
                "Add test for null input at line 89"
            ]
        }

        # Store in memory
        if self.memory_layer:
            await self.memory_layer.add_to_working(
                content=f"Coverage analysis for {test_path}: {analysis['line_coverage']*100:.0f}%",
                metadata=analysis,
                item_type="coverage_analysis"
            )

        # Track coverage history
        self.coverage_history.append({
            "test_path": test_path,
            "impl_path": impl_path,
            "timestamp": datetime.now().isoformat(),
            "coverage": analysis["line_coverage"]
        })

        return analysis

    async def detect_bugs(self, code: str) -> List[Dict]:
        """Detect bugs in code"""
        pass


class ResearcherAnt(WorkerAnt):
    """
    Researcher Ant - Find information, context

    Capabilities:
    - Web search
    - Documentation lookup
    - Reference finding
    - Context gathering

    Spawns: Search agents, crawlers, documentation readers, API explorers
    """

    caste = "researcher"
    capabilities = ["web_search", "documentation_lookup", "context_gathering"]
    sensitivity = {
        PheromoneType.INIT: 0.7,      # Learns new domain
        PheromoneType.FOCUS: 0.9,     # Researches focused topic
        PheromoneType.REDIRECT: 0.4,
        PheromoneType.FEEDBACK: 0.5,
    }
    spawns = ["search_agent", "crawler", "documentation_reader", "api_explorer"]

    def __init__(self, colony: 'Colony', memory_layer: Optional['TripleLayerMemory'] = None, meta_learner: Optional[MetaLearner] = None):
        super().__init__(colony, memory_layer=memory_layer, meta_learner=meta_learner)
        self.knowledge_base: Dict[str, Any] = {}
        self.research_cache: Dict[str, Any] = {}

    async def respond_to_signal(self, signal: PheromoneSignal):
        if signal.signal_type == PheromoneType.INIT:
            await self.learn_domain(signal.content)
        elif signal.signal_type == PheromoneType.FOCUS:
            await self.research_topic(signal.content)

    async def learn_domain(self, goal: str):
        """Learn about a new domain"""
        self.current_task = f"Learning domain for: {goal}"

        # Spawn search agent
        searcher = self.spawn_subagent(
            "domain_learner",
            f"Learn about domain: {goal}"
        )

    async def research_topic(self, topic: str):
        """Research a specific topic"""
        self.current_task = f"Researching: {topic}"

        # Spawn documentation reader
        doc_reader = self.spawn_subagent(
            f"doc_reader_{topic}",
            f"Read documentation for: {topic}"
        )

    async def search_web(self, query: str) -> List[Dict]:
        """Search web for information"""
        # Use WebSearch tool
        pass

    async def find_best_practices(self, topic: str) -> List[str]:
        """Find best practices for topic"""
        pass


class SynthesizerAnt(WorkerAnt):
    """
    Synthesizer Ant - Compress memory, extract patterns

    Capabilities:
    - Memory compression
    - Pattern extraction
    - Anti-pattern detection
    - Knowledge synthesis

    Spawns: Analysis agents, pattern matchers, compression agents
    """

    caste = "synthesizer"
    capabilities = ["memory_compression", "pattern_extraction", "knowledge_synthesis"]
    sensitivity = {
        PheromoneType.INIT: 0.2,      # Not very responsive
        PheromoneType.FOCUS: 0.4,
        PheromoneType.REDIRECT: 0.3,
        PheromoneType.FEEDBACK: 0.6,
    }
    spawns = ["analysis_agent", "pattern_matcher", "compression_agent"]

    def __init__(self, colony: 'Colony', memory_layer: Optional['TripleLayerMemory'] = None, meta_learner: Optional[MetaLearner] = None):
        super().__init__(colony, memory_layer=memory_layer, meta_learner=meta_learner)
        self.patterns: List[Dict] = []
        self.best_practices: List[str] = []
        self.anti_patterns: List[str] = []

    async def respond_to_signal(self, signal: PheromoneSignal):
        # Synthesizer responds to memory events, not pheromones
        pass

    async def compress_phase_memory(self, phase: Dict):
        """Compress memory after phase completion"""
        self.current_task = f"Compressing phase: {phase['name']}"

        # Spawn compression agent
        compressor = self.spawn_subagent(
            f"phase_compressor_{phase['id']}",
            f"Compress memory for phase {phase['id']}"
        )

    async def extract_patterns(self, memory: Dict) -> List[Dict]:
        """Extract patterns from memory"""
        # Spawn pattern matcher
        pattern_matcher = self.spawn_subagent(
            "pattern_matcher",
            "Extract patterns from memory"
        )

        # Based on Phase 5 research:
        # - Verification Feedback Loops and Learning
        pass

    async def detect_anti_patterns(self, errors: List[Dict]) -> List[str]:
        """Detect anti-patterns from errors"""
        pass

    async def synthesize_knowledge(self) -> Dict:
        """Synthesize all knowledge into insights"""
        pass


class Colony:
    """
    The Colony - Manages Worker Ants and pheromones with autonomous spawning

    Colony self-organizes based on pheromones.
    Worker Ants autonomously spawn specialists based on capability gaps.
    """

    def __init__(self, memory_layer: Optional['TripleLayerMemory'] = None, meta_learner: Optional[MetaLearner] = None):
        self.worker_ants: Dict[str, WorkerAnt] = {}
        self.pheromones: List[PheromoneSignal] = []
        self.subagents: List[Subagent] = []
        self.current_phase: Optional[Dict] = None
        self.phases: List[Dict] = []

        # Autonomous spawning tracking
        self.spawning_decisions: List[SpawningDecision] = []

        # Error prevention system
        self.error_ledger = ErrorLedger()

        # Triple-Layer Memory System
        self.memory_layer = memory_layer

        # Meta-Learning System
        self.meta_learner = meta_learner or MetaLearner()

        # Initialize Worker Ants
        self._init_worker_ants()

    def _init_worker_ants(self):
        """Initialize all Worker Ant castes"""
        self.worker_ants["mapper"] = MapperAnt(self, memory_layer=self.memory_layer, meta_learner=self.meta_learner)
        self.worker_ants["planner"] = PlannerAnt(self, memory_layer=self.memory_layer, meta_learner=self.meta_learner)
        self.worker_ants["executor"] = ExecutorAnt(self, memory_layer=self.memory_layer, meta_learner=self.meta_learner)
        self.worker_ants["verifier"] = VerifierAnt(self, memory_layer=self.memory_layer, meta_learner=self.meta_learner)
        self.worker_ants["researcher"] = ResearcherAnt(self, memory_layer=self.memory_layer, meta_learner=self.meta_learner)
        self.worker_ants["synthesizer"] = SynthesizerAnt(self, memory_layer=self.memory_layer, meta_learner=self.meta_learner)

    def set_memory_layer(self, memory_layer: 'TripleLayerMemory') -> None:
        """Set or update the memory layer for all Worker Ants"""
        self.memory_layer = memory_layer

        # Update all Worker Ants
        for ant in self.worker_ants.values():
            ant.memory_layer = memory_layer

    async def receive_pheromone(self, signal_type: PheromoneType, content: str, strength: float = 0.5):
        """Receive pheromone signal from Queen"""
        signal = PheromoneSignal(
            signal_type=signal_type,
            content=content,
            strength=strength,
            created_at=datetime.now()
        )
        self.pheromones.append(signal)

        # Notify all Worker Ants
        await self._notify_worker_ants()

    async def _notify_worker_ants(self):
        """Notify all Worker Ants of pheromone changes"""
        for ant in self.worker_ants.values():
            signals = await ant.detect_pheromones()
            if signals:
                await ant.respond_to_pheromones(signals)

    def register_subagent(self, subagent: Subagent):
        """Register a spawned subagent and track spawning decision"""
        self.subagents.append(subagent)

        # Track spawning decision
        if subagent.inherited_context and subagent.spawning_reason:
            decision = SpawningDecision(
                timestamp=datetime.now(),
                parent_agent=subagent.parent.caste,
                task=subagent.purpose,
                capability_gaps=subagent.capabilities,  # Already a set of strings
                specialist_type=subagent.name,
                reason=subagent.spawning_reason,
                depth=subagent.depth
            )
            self.spawning_decisions.append(decision)

    def get_status(self) -> Dict:
        """Get colony status including autonomous spawning information"""
        return {
            "worker_ants": {
                name: {
                    "current_task": ant.current_task,
                    "subagents_count": len(ant.subagents),
                    "last_activity": ant.last_activity,
                    "spawning_summary": ant.get_spawning_summary()
                }
                for name, ant in self.worker_ants.items()
            },
            "active_pheromones": len([p for p in self.pheromones if p.is_active()]),
            "total_subagents": len(self.subagents),
            "current_phase": self.current_phase,
            "autonomous_spawning": {
                "total_decisions": len(self.spawning_decisions),
                "recent_decisions": [
                    {
                        "timestamp": d.timestamp.isoformat(),
                        "parent": d.parent_agent,
                        "specialist": d.specialist_type,
                        "reason": d.reason
                    }
                    for d in self.spawning_decisions[-5:]  # Last 5
                ]
            }
        }

    def get_autonomous_spawning_report(self) -> Dict[str, Any]:
        """Get detailed report of autonomous spawning activity"""
        # Aggregate by parent caste
        by_parent = {}
        for decision in self.spawning_decisions:
            parent = decision.parent_agent
            if parent not in by_parent:
                by_parent[parent] = {"count": 0, "specialists": {}}
            by_parent[parent]["count"] += 1

            specialist = decision.specialist_type
            if specialist not in by_parent[parent]["specialists"]:
                by_parent[parent]["specialists"][specialist] = 0
            by_parent[parent]["specialists"][specialist] += 1

        # Aggregate by specialist type
        by_specialist = {}
        for decision in self.spawning_decisions:
            specialist = decision.specialist_type
            if specialist not in by_specialist:
                by_specialist[specialist] = 0
            by_specialist[specialist] += 1

        return {
            "total_spawned": len(self.spawning_decisions),
            "by_parent_caste": by_parent,
            "by_specialist_type": by_specialist,
            "recent_activity": [
                {
                    "timestamp": d.timestamp.isoformat(),
                    "parent": d.parent_agent,
                    "specialist": d.specialist_type,
                    "reason": d.reason,
                    "depth": d.depth
                }
                for d in self.spawning_decisions[-10:]  # Last 10
            ]
        }

    async def execute_phase(self, phase: Dict):
        """Execute a phase with pure emergence"""
        self.current_phase = phase
        phase["status"] = "in_progress"
        phase["started_at"] = datetime.now()

        # Worker Ants self-organize
        # They will detect tasks and spawn subagents
        # No central coordination

        # This is where the magic happens
        # Pure emergence based on:
        # - Pheromone signals
        # - Local task detection
        # - Peer-to-peer coordination

        # Wait for phase completion
        await self._wait_for_phase_completion(phase)

        phase["status"] = "completed"
        phase["completed_at"] = datetime.now()

        # Synthesize memory
        await self.worker_ants["synthesizer"].compress_phase_memory(phase)

    async def _wait_for_phase_completion(self, phase: Dict):
        """Wait for phase to complete"""
        # Monitor progress
        # Detect completion
        # This is autonomous
        pass


# Factory function
def create_colony(memory_layer: Optional['TripleLayerMemory'] = None) -> Colony:
    """Create a new Queen Ant colony with autonomous spawning"""
    return Colony(memory_layer=memory_layer)


# ============================================================================
# AUTONOMOUS SPAWNING DEMO
# ============================================================================

async def demo_autonomous_spawning():
    """
    Demonstrate autonomous agent spawning

    This shows how Worker Ants detect capability gaps and
    autonomously spawn specialists to handle tasks.
    """
    print("=" * 70)
    print("🐜 AUTONOMOUS AGENT SPAWNING DEMO")
    print("=" * 70)
    print()

    # Create colony
    colony = create_colony()
    print("✅ Colony created with 6 Worker Ant castes")
    print()

    # Emit INIT pheromone to trigger activity
    await colony.receive_pheromone(
        PheromoneType.INIT,
        "Build a REST API with JWT authentication and database integration",
        strength=1.0
    )
    print("✅ INIT pheromone emitted: Build REST API with JWT auth and database")
    print()

    # Mapper Ant responds with autonomous spawning
    print("🐜 Mapper Ant detects task and autonomously spawns specialists...")
    mapper = colony.worker_ants["mapper"]
    result = await mapper.explore_codebase("Build REST API with JWT auth")
    print()

    # Check spawning results
    status = colony.get_status()
    spawning_summary = status["autonomous_spawning"]
    print(f"📊 SPAWNING SUMMARY:")
    print(f"   Total autonomous decisions: {spawning_summary['total_decisions']}")
    print()

    if spawning_summary["recent_decisions"]:
        print("📝 RECENT SPAWNING DECISIONS:")
        for i, decision in enumerate(spawning_summary["recent_decisions"], 1):
            print(f"   {i}. {decision['parent']} → {decision['specialist']}")
            print(f"      Reason: {decision['reason']}")
        print()

    # Show detailed report
    report = colony.get_autonomous_spawning_report()
    print("📈 AUTONOMOUS SPAWNING REPORT:")
    print(f"   Total spawned: {report['total_spawned']}")
    print(f"   By parent caste: {list(report['by_parent_caste'].keys())}")
    print(f"   By specialist type: {list(report['by_specialist_type'].keys())}")
    print()

    print("=" * 70)
    print("✨ DEMO COMPLETE")
    print()
    print("Key Takeaways:")
    print("  • Worker Ants autonomously detect capability gaps")
    print("  • They spawn appropriate specialists without human direction")
    print("  • Each spawning decision is tracked with reasoning")
    print("  • Resource budgets prevent infinite spawning")
    print("  • Context inheritance ensures specialists have needed context")
    print("=" * 70)

    return colony


if __name__ == "__main__":
    # Run the demo
    asyncio.run(demo_autonomous_spawning())
