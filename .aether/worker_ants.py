"""
Queen Ant Colony - Worker Ant Castes

Six pre-defined specialist castes that respond to pheromones
and spawn subagents as needed.

Based on research from:
- Phase 6: Multi-Agent System Integration Patterns
- Phase 1: Context Engine Foundation
- Phase 3: Semantic Codebase Understanding
"""

from typing import List, Dict, Any, Optional, Callable
from dataclasses import dataclass, field
from enum import Enum
import asyncio
from datetime import datetime, timedelta

try:
    from .pheromone_system import PheromoneType, PheromoneSignal, PheromoneLayer, SensitivityProfile, SENSITIVITY_PROFILES
except ImportError:
    from pheromone_system import PheromoneType, PheromoneSignal, PheromoneLayer, SensitivityProfile, SENSITIVITY_PROFILES


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
    """A spawned subagent"""
    name: str
    purpose: str
    parent: 'WorkerAnt'
    spawned_at: datetime
    status: str = "active"  # active, completed, terminated

    def terminate(self):
        """Terminate this subagent"""
        self.status = "terminated"


class WorkerAnt:
    """Base class for all Worker Ant castes"""

    caste: str
    capabilities: List[str]
    sensitivity: Dict[PheromoneType, float]
    spawns: List[str]  # Types of subagents this caste can spawn

    def __init__(self, colony: 'Colony'):
        self.colony = colony
        self.subagents: List[Subagent] = []
        self.current_task: Optional[str] = None
        self.last_activity: datetime = datetime.now()

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

    def __init__(self, colony: 'Colony'):
        super().__init__(colony)
        self.semantic_index: Dict[str, Any] = {}
        self.dependency_graph: Dict[str, List[str]] = {}

    async def respond_to_signal(self, signal: PheromoneSignal):
        if signal.signal_type == PheromoneType.INIT:
            await self.explore_codebase(signal.content)
        elif signal.signal_type == PheromoneType.FOCUS:
            await self.map_specific_area(signal.content)

    async def explore_codebase(self, goal: str):
        """Explore entire codebase for new goal"""
        self.current_task = f"Exploring codebase for: {goal}"

        # Spawn graph builder
        graph_builder = self.spawn_subagent(
            "graph_builder",
            "Build dependency graph for codebase"
        )

        # Spawn search agents
        search_agent = self.spawn_subagent(
            "search_agent",
            "Search for relevant code patterns"
        )

        # Build semantic index
        await self.build_semantic_index()

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

    def __init__(self, colony: 'Colony'):
        super().__init__(colony)
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

    def __init__(self, colony: 'Colony'):
        super().__init__(colony)
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

    def __init__(self, colony: 'Colony'):
        super().__init__(colony)
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

    def __init__(self, colony: 'Colony'):
        super().__init__(colony)
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

    def __init__(self, colony: 'Colony'):
        super().__init__(colony)
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
    The Colony - Manages Worker Ants and pheromones

    Colony self-organizes based on pheromones.
    Worker Ants coordinate peer-to-peer.
    """

    def __init__(self):
        self.worker_ants: Dict[str, WorkerAnt] = {}
        self.pheromones: List[PheromoneSignal] = []
        self.subagents: List[Subagent] = []
        self.current_phase: Optional[Dict] = None
        self.phases: List[Dict] = []

        # Initialize Worker Ants
        self._init_worker_ants()

    def _init_worker_ants(self):
        """Initialize all Worker Ant castes"""
        self.worker_ants["mapper"] = MapperAnt(self)
        self.worker_ants["planner"] = PlannerAnt(self)
        self.worker_ants["executor"] = ExecutorAnt(self)
        self.worker_ants["verifier"] = VerifierAnt(self)
        self.worker_ants["researcher"] = ResearcherAnt(self)
        self.worker_ants["synthesizer"] = SynthesizerAnt(self)

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
        """Register a spawned subagent"""
        self.subagents.append(subagent)

    def get_status(self) -> Dict:
        """Get colony status"""
        return {
            "worker_ants": {
                name: {
                    "current_task": ant.current_task,
                    "subagents_count": len(ant.subagents),
                    "last_activity": ant.last_activity
                }
                for name, ant in self.worker_ants.items()
            },
            "active_pheromones": len([p for p in self.pheromones if p.is_active()]),
            "total_subagents": len(self.subagents),
            "current_phase": self.current_phase
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
def create_colony() -> Colony:
    """Create a new Queen Ant colony"""
    return Colony()
