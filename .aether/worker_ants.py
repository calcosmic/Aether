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
    from .error_prevention import ErrorLedger, ErrorCategory, log_exception
except ImportError:
    from pheromone_system import PheromoneType, PheromoneSignal, PheromoneLayer, SensitivityProfile, SENSITIVITY_PROFILES
    from error_prevention import ErrorLedger, ErrorCategory, log_exception

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

    def __init__(self, colony: 'Colony', error_ledger: Optional[ErrorLedger] = None, memory_layer: Optional['TripleLayerMemory'] = None):
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

            # Determine specialist type
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

            # Spawn the specialist
            specialist = Subagent(
                name=f"autonomous_{specialist_type}_{len(self.subagents)}",
                purpose=f"Address capability gaps: {', '.join(gaps)}",
                parent=self,
                spawned_at=datetime.now(),
                inherited_context=inherited,
                capabilities=required,  # Store as set of strings
                depth=depth,
                spawning_reason=reason
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

    def __init__(self, colony: 'Colony', memory_layer: Optional['TripleLayerMemory'] = None):
        super().__init__(colony, memory_layer=memory_layer)
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

    def __init__(self, colony: 'Colony', memory_layer: Optional['TripleLayerMemory'] = None):
        super().__init__(colony, memory_layer=memory_layer)
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
    Executor Ant - Write code, implement changes

    Capabilities:
    - Code generation
    - File manipulation
    - Refactoring
    - Implementation

    Spawns: Language specialists, framework specialists, database specialists
    """

    caste = "executor"
    capabilities = ["code_generation", "file_manipulation", "refactoring"]
    sensitivity = {
        PheromoneType.INIT: 0.5,      # Awaits planning
        PheromoneType.FOCUS: 0.9,     # Highly responsive to focus
        PheromoneType.REDIRECT: 0.9,  # Strongly avoids redirected patterns
        PheromoneType.FEEDBACK: 0.7,
    }
    spawns = ["language_specialist", "framework_specialist", "database_specialist"]

    def __init__(self, colony: 'Colony', memory_layer: Optional['TripleLayerMemory'] = None):
        super().__init__(colony, memory_layer=memory_layer)
        self.current_files: List[str] = []
        self.implemented_features: List[str] = []

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


class VerifierAnt(WorkerAnt):
    """
    Verifier Ant - Test, validate, QA

    Capabilities:
    - Test generation
    - Validation
    - Quality checks
    - Bug detection

    Spawns: Test generators, lint agents, security scanners, performance testers
    """

    caste = "verifier"
    capabilities = ["test_generation", "validation", "quality_checks"]
    sensitivity = {
        PheromoneType.INIT: 0.3,      # Waits for code to test
        PheromoneType.FOCUS: 0.8,     # Increases scrutiny on focus area
        PheromoneType.REDIRECT: 0.5,
        PheromoneType.FEEDBACK: 0.9,  # Highly responsive to quality feedback
    }
    spawns = ["test_generator", "lint_agent", "security_scanner", "performance_tester"]

    def __init__(self, colony: 'Colony', memory_layer: Optional['TripleLayerMemory'] = None):
        super().__init__(colony, memory_layer=memory_layer)
        self.tests_generated: int = 0
        self.bugs_found: int = 0
        self.issues: List[Dict] = []

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

    def __init__(self, colony: 'Colony', memory_layer: Optional['TripleLayerMemory'] = None):
        super().__init__(colony, memory_layer=memory_layer)
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

    def __init__(self, colony: 'Colony', memory_layer: Optional['TripleLayerMemory'] = None):
        super().__init__(colony, memory_layer=memory_layer)
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

    def __init__(self, memory_layer: Optional['TripleLayerMemory'] = None):
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

        # Initialize Worker Ants
        self._init_worker_ants()

    def _init_worker_ants(self):
        """Initialize all Worker Ant castes"""
        self.worker_ants["mapper"] = MapperAnt(self, memory_layer=self.memory_layer)
        self.worker_ants["planner"] = PlannerAnt(self, memory_layer=self.memory_layer)
        self.worker_ants["executor"] = ExecutorAnt(self, memory_layer=self.memory_layer)
        self.worker_ants["verifier"] = VerifierAnt(self, memory_layer=self.memory_layer)
        self.worker_ants["researcher"] = ResearcherAnt(self, memory_layer=self.memory_layer)
        self.worker_ants["synthesizer"] = SynthesizerAnt(self, memory_layer=self.memory_layer)

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
