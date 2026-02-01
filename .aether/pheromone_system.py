"""
Queen Ant Colony - Pheromone Signal System

User signals that guide colony behavior without commands.

Pheromones decay over time. Recent signals are stronger.
Worker Ants respond based on their sensitivity profiles.

Based on research from:
- Phase 1: Semantic protocols, communication bandwidth reduction
- Phase 4: Anticipatory context prediction, feedback loops
- Phase 6: Multi-agent coordination patterns
"""

from typing import List, Dict, Any, Optional, Callable
from dataclasses import dataclass, field
from enum import Enum
from datetime import datetime, timedelta
import asyncio

# Optional semantic layer import
try:
    from .semantic_layer import SemanticPheromoneLayer, create_semantic_layer
except ImportError:
    try:
        from semantic_layer import SemanticPheromoneLayer, create_semantic_layer
    except ImportError:
        SemanticPheromoneLayer = None
        create_semantic_layer = None


class PheromoneType(Enum):
    """Types of pheromone signals from Queen"""

    INIT = "init"
    """Strong attract signal. Triggers colony mobilization and planning.
    Persists until phase complete."""

    FOCUS = "focus"
    """Medium attract signal. Guides colony attention to specific area.
    Decays over 1 hour."""

    REDIRECT = "redirect"
    """Strong repel signal. Warns colony away from approach/pattern.
    Decays over 24 hours."""

    FEEDBACK = "feedback"
    """Variable signal. Adjusts colony behavior based on Queen's feedback.
    Decays over 6 hours."""


@dataclass
class PheromoneSignal:
    """A pheromone signal from the Queen"""

    signal_type: PheromoneType
    content: str
    strength: float  # 0.0 to 1.0
    created_at: datetime
    half_life: timedelta = field(default=None)
    metadata: Dict[str, Any] = field(default_factory=dict)

    def __post_init__(self):
        if self.half_life is None:
            # Set default half-life based on signal type
            half_lives = {
                PheromoneType.INIT: timedelta(hours=24),     # Persists
                PheromoneType.FOCUS: timedelta(hours=1),     # Short
                PheromoneType.REDIRECT: timedelta(hours=24), # Long
                PheromoneType.FEEDBACK: timedelta(hours=6),  # Medium
            }
            self.half_life = half_lives[self.signal_type]

    def current_strength(self) -> float:
        """
        Calculate current strength based on decay.

        Strength(t) = InitialStrength × e^(-t/HalfLife)

        Examples:
        - Init (strength 1.0): Stays 1.0 (persists)
        - Focus (strength 0.5): 0.25 after 1 hour
        - Redirect (strength 0.7): 0.35 after 24 hours
        """
        if self.signal_type == PheromoneType.INIT:
            # Init pheromones persist until phase complete
            return self.strength

        age = datetime.now() - self.created_at
        half_lives_passed = age.total_seconds() / self.half_life.total_seconds()
        decay_factor = 0.5 ** half_lives_passed
        return self.strength * decay_factor

    def is_active(self, threshold: float = 0.01) -> bool:
        """Check if signal is still active above threshold"""
        return self.current_strength() > threshold

    def age_seconds(self) -> float:
        """Get age of signal in seconds"""
        return (datetime.now() - self.created_at).total_seconds()

    def time_until_expiry(self, threshold: float = 0.01) -> timedelta:
        """Calculate time until signal expires below threshold"""
        current = self.current_strength()
        if current <= threshold:
            return timedelta(0)

        # Calculate half-lives until threshold
        ratio = threshold / current
        half_lives_needed = -1 * (ratio.bit_length() - 1)  # Approximate
        exact_half_lives = (threshold / self.strength) ** (1/2)  # Inverse of decay

        return timedelta(seconds=exact_half_lives * self.half_life.total_seconds())

    def __str__(self) -> str:
        """String representation for debugging"""
        age_mins = self.age_seconds() / 60
        strength_pct = self.current_strength() * 100
        return (
            f"[{self.signal_type.value.upper()}] "
            f"\"{self.content[:50]}...\" "
            f"(strength: {strength_pct:.1f}%, age: {age_mins:.1f}m)"
        )


@dataclass
class SensitivityProfile:
    """
    A Worker Ant's sensitivity to different pheromone types.

    Higher sensitivity = stronger response to pheromones.
    """

    caste: str
    init: float = 0.5      # Response to init signals
    focus: float = 0.5     # Response to focus signals
    redirect: float = 0.5  # Response to redirect signals
    feedback: float = 0.5  # Response to feedback signals

    def get_sensitivity(self, signal_type: PheromoneType) -> float:
        """Get sensitivity for a specific signal type"""
        sensitivities = {
            PheromoneType.INIT: self.init,
            PheromoneType.FOCUS: self.focus,
            PheromoneType.REDIRECT: self.redirect,
            PheromoneType.FEEDBACK: self.feedback,
        }
        return sensitivities.get(signal_type, 0.0)

    def effective_strength(self, signal: PheromoneSignal) -> float:
        """
        Calculate effective strength of a signal for this ant.

        EffectiveStrength = SignalStrength × AntSensitivity
        """
        return signal.current_strength() * self.get_sensitivity(signal.signal_type)

    def should_respond(self, signal: PheromoneSignal, threshold: float = 0.1) -> bool:
        """
        Determine if this ant should respond to a signal.

        Responds if effective strength exceeds threshold.
        """
        return self.effective_strength(signal) > threshold


class PheromoneLayer:
    """
    Manages all pheromone signals in the colony.

    The Pheromone Layer:
    - Receives signals from Queen
    - Tracks signal strength and decay
    - Propagates signals to Worker Ants
    - Maintains signal history
    - Semantic understanding (optional)
    """

    def __init__(self, enable_semantic: bool = True):
        self.signals: List[PheromoneSignal] = []
        self.signal_history: List[PheromoneSignal] = []
        self.max_history: int = 1000

        # Optional semantic layer
        self.semantic_layer: Optional[SemanticPheromoneLayer] = None
        if enable_semantic and create_semantic_layer is not None:
            try:
                self.semantic_layer = create_semantic_layer()
                print("✅ Semantic layer enabled for pheromones")
            except Exception as e:
                print(f"⚠️  Failed to enable semantic layer: {e}")

    async def emit(
        self,
        signal_type: PheromoneType,
        content: str,
        strength: float = 0.5,
        metadata: Optional[Dict[str, Any]] = None
    ) -> PheromoneSignal:
        """
        Emit a new pheromone signal.

        Args:
            signal_type: Type of signal (init, focus, redirect, feedback)
            content: Signal content/description
            strength: Signal strength (0.0 to 1.0)
            metadata: Optional metadata

        Returns:
            The created signal
        """
        signal = PheromoneSignal(
            signal_type=signal_type,
            content=content,
            strength=strength,
            created_at=datetime.now(),
            metadata=metadata or {}
        )

        self.signals.append(signal)

        # Add to semantic index if enabled
        if self.semantic_layer is not None:
            signal_id = f"{signal_type.value}_{datetime.now().strftime('%Y%m%d_%H%M%S_%f')[:17]}"
            self.semantic_layer.add_signal(
                signal_id=signal_id,
                content=content,
                metadata={**(metadata or {}), "signal_type": signal_type.value, "strength": strength}
            )
            # Store signal_id in metadata for later retrieval
            signal.metadata["semantic_id"] = signal_id

        return signal

    def get_active_signals(self, signal_type: Optional[PheromoneType] = None) -> List[PheromoneSignal]:
        """
        Get all active signals, optionally filtered by type.

        Active means: current_strength > threshold (default 0.01)
        """
        active = [s for s in self.signals if s.is_active()]

        if signal_type is not None:
            active = [s for s in active if s.signal_type == signal_type]

        # Sort by strength (strongest first)
        active.sort(key=lambda s: s.current_strength(), reverse=True)

        return active

    def get_signals_for_ant(
        self,
        sensitivity: SensitivityProfile,
        threshold: float = 0.1
    ) -> List[PheromoneSignal]:
        """
        Get all signals that an ant should respond to.

        Returns signals where effective strength > threshold.
        """
        relevant = []

        for signal in self.get_active_signals():
            if sensitivity.should_respond(signal, threshold):
                relevant.append(signal)

        return relevant

    def cleanup_expired(self):
        """Remove expired signals from active list"""
        active = [s for s in self.signals if s.is_active()]
        expired = [s for s in self.signals if not s.is_active()]

        # Move expired to history
        self.signal_history.extend(expired)
        self.signals = active

        # Trim history if needed
        if len(self.signal_history) > self.max_history:
            self.signal_history = self.signal_history[-self.max_history:]

    def get_signal_summary(self) -> Dict[str, Any]:
        """Get summary of all signals"""
        active_by_type = {
            signal_type.value: len(self.get_active_signals(signal_type))
            for signal_type in PheromoneType
        }

        return {
            "total_active": len(self.get_active_signals()),
            "by_type": active_by_type,
            "total_history": len(self.signal_history),
            "strongest_signal": self.get_active_signals()[0] if self.get_active_signals() else None
        }

    # ========================================================================
    # SEMANTIC SEARCH METHODS
    # ========================================================================

    def find_similar_signals_semantic(
        self,
        query: str,
        top_k: int = 5,
        threshold: float = 0.6,
        signal_type: Optional[PheromoneType] = None
    ) -> List[Dict[str, Any]]:
        """
        Find semantically similar signals using vector embeddings

        Args:
            query: Query text
            top_k: Maximum results
            threshold: Minimum similarity (0-1)
            signal_type: Optional signal type filter

        Returns:
            List of similar signals with similarity scores
        """
        if self.semantic_layer is None:
            return []

        signal_type_str = signal_type.value if signal_type else None
        return self.semantic_layer.find_similar_signals(
            query=query,
            top_k=top_k,
            threshold=threshold,
            signal_type=signal_type_str
        )

    def get_semantic_distance(self, text1: str, text2: str) -> float:
        """
        Calculate semantic distance between two texts

        Args:
            text1: First text
            text2: Second text

        Returns:
            Cosine similarity (0-1, higher = more similar)
        """
        if self.semantic_layer is None:
            return 0.0
        return self.semantic_layer.semantic_distance(text1, text2)

    def compress_redundant_signals(self, signals: List[PheromoneSignal]) -> List[PheromoneSignal]:
        """
        Compress redundant signals using semantic similarity

        Args:
            signals: List of signals to compress

        Returns:
            Compressed list
        """
        if self.semantic_layer is None:
            return signals

        # Convert to dict format
        signal_dicts = [
            {
                "id": s.metadata.get("semantic_id", f"signal_{i}"),
                "content": s.content,
                "signal_type": s.signal_type.value
            }
            for i, s in enumerate(signals)
        ]

        # Compress
        compressed_dicts = self.semantic_layer.compress_redundant_signals(signal_dicts)

        # Map back to signals (preserve signal objects)
        compressed_contents = {d["content"] for d in compressed_dicts}
        return [s for s in signals if s.content in compressed_contents]

    def get_semantic_stats(self) -> Dict[str, Any]:
        """Get semantic layer statistics"""
        if self.semantic_layer is None:
            return {"enabled": False}

        return {
            "enabled": True,
            **self.semantic_layer.get_compression_stats()
        }

    def find_similar_signals(
        self,
        content: str,
        similarity_threshold: float = 0.7,
        signal_type: Optional[PheromoneType] = None
    ) -> List[PheromoneSignal]:
        """
        Find signals similar to given content.

        Uses simple keyword matching. Could be enhanced with semantic similarity.
        """
        content_lower = content.lower()
        content_words = set(content_lower.split())

        similar = []

        for signal in self.get_active_signals(signal_type):
            signal_lower = signal.content.lower()
            signal_words = set(signal_lower.split())

            # Jaccard similarity
            intersection = content_words & signal_words
            union = content_words | signal_words
            similarity = len(intersection) / len(union) if union else 0

            if similarity >= similarity_threshold:
                similar.append((signal, similarity))

        # Sort by similarity
        similar.sort(key=lambda x: x[1], reverse=True)

        return [s[0] for s in similar]

    def clear_type(self, signal_type: PheromoneType):
        """Clear all signals of a specific type"""
        self.signals = [s for s in self.signals if s.signal_type != signal_type]

    def decay_all(self):
        """Force decay of all signals (for testing)"""
        # Shift all creation times back by 1 half-life
        for signal in self.signals:
            signal.created_at = signal.created_at - signal.half_life


class PheromoneResponder:
    """
    Base class for objects that respond to pheromones.

    Worker Ants inherit from this to gain pheromone response capability.
    """

    def __init__(self, sensitivity: SensitivityProfile, pheromone_layer: PheromoneLayer):
        self.sensitivity = sensitivity
        self.pheromone_layer = pheromone_layer
        self.last_response_time: Optional[datetime] = None
        self.response_count: int = 0

    async def detect_pheromones(self, threshold: float = 0.1) -> List[PheromoneSignal]:
        """
        Detect pheromones that this responder should react to.

        Returns signals where effective strength > threshold.
        """
        signals = self.pheromone_layer.get_signals_for_ant(
            self.sensitivity,
            threshold
        )

        # Filter out recently responded signals (avoid thrashing)
        if self.last_response_time:
            recent_threshold = datetime.now() - timedelta(minutes=1)
            signals = [
                s for s in signals
                if s.created_at > self.last_response_time or
                   s.current_strength() > 0.5  # Always respond to strong signals
            ]

        return signals

    async def respond_to_pheromones(self, signals: List[PheromoneSignal]):
        """
        Respond to detected pheromones.

        Override in subclasses to implement specific behavior.
        """
        for signal in signals:
            await self.respond_to_signal(signal)

        if signals:
            self.last_response_time = datetime.now()
            self.response_count += len(signals)

    async def respond_to_signal(self, signal: PheromoneSignal):
        """
        Respond to a single pheromone signal.

        Override in subclasses to implement specific behavior.
        """
        pass


class PheromoneHistory:
    """
    Tracks pheromone history for learning and pattern detection.

    The colony learns from pheromone patterns:
    - What does the Queen focus on?
    - What does the Queen redirect away from?
    - What feedback does the Queen give?
    """

    def __init__(self, pheromone_layer: PheromoneLayer):
        self.pheromone_layer = pheromone_layer
        self.patterns: Dict[str, Any] = {}

    def analyze_focus_patterns(self) -> Dict[str, int]:
        """
        Analyze what the Queen focuses on most.

        Returns: {topic: count}
        """
        focus_signals = [
            s for s in self.pheromone_layer.signal_history
            if s.signal_type == PheromoneType.FOCUS
        ]

        topic_counts: Dict[str, int] = {}

        for signal in focus_signals:
            topic = signal.content.lower()
            topic_counts[topic] = topic_counts.get(topic, 0) + 1

        # Sort by count
        return dict(sorted(topic_counts.items(), key=lambda x: x[1], reverse=True))

    def analyze_redirect_patterns(self) -> Dict[str, int]:
        """
        Analyze what the Queen redirects away from.

        Returns: {pattern: count}
        """
        redirect_signals = [
            s for s in self.pheromone_layer.signal_history
            if s.signal_type == PheromoneType.REDIRECT
        ]

        pattern_counts: Dict[str, int] = {}

        for signal in redirect_signals:
            pattern = signal.content.lower()
            pattern_counts[pattern] = pattern_counts.get(pattern, 0) + 1

        return dict(sorted(pattern_counts.items(), key=lambda x: x[1], reverse=True))

    def analyze_feedback_patterns(self) -> Dict[str, List[str]]:
        """
        Analyze feedback patterns.

        Returns: {category: [feedback_items]}
        """
        feedback_signals = [
            s for s in self.pheromone_layer.signal_history
            if s.signal_type == PheromoneType.FEEDBACK
        ]

        categories = {
            "quality": [],
            "speed": [],
            "direction": [],
            "other": []
        }

        for signal in feedback_signals:
            content = signal.content.lower()

            if "bug" in content or "quality" in content or "test" in content:
                categories["quality"].append(signal.content)
            elif "slow" in content or "fast" in content or "speed" in content:
                categories["speed"].append(signal.content)
            elif "wrong" in content or "approach" in content or "direction" in content:
                categories["direction"].append(signal.content)
            else:
                categories["other"].append(signal.content)

        return categories

    def get_learned_preferences(self) -> Dict[str, Any]:
        """
        Get learned preferences from pheromone history.

        This helps the colony adapt to Queen's preferences over time.
        """
        return {
            "focus_topics": self.analyze_focus_patterns(),
            "avoid_patterns": self.analyze_redirect_patterns(),
            "feedback_categories": self.analyze_feedback_patterns()
        }


class PheromoneCommands:
    """
    Command interface for emitting pheromones.

    This is what the /ant: commands use.
    """

    def __init__(self, pheromone_layer: PheromoneLayer):
        self.pheromone_layer = pheromone_layer

    async def init(self, goal: str) -> PheromoneSignal:
        """
        /ant:init <goal>

        Emit init pheromone to start new project/phase.
        """
        return await self.pheromone_layer.emit(
            PheromoneType.INIT,
            goal,
            strength=1.0,
            metadata={"command": "init"}
        )

    async def focus(self, area: str, strength: float = 0.5) -> PheromoneSignal:
        """
        /ant:focus <area>

        Emit focus pheromone to guide colony attention.
        """
        return await self.pheromone_layer.emit(
            PheromoneType.FOCUS,
            area,
            strength=strength,
            metadata={"command": "focus"}
        )

    async def redirect(self, pattern: str, strength: float = 0.7) -> PheromoneSignal:
        """
        /ant:redirect <pattern>

        Emit redirect pheromone to warn colony away from approach.
        """
        return await self.pheromone_layer.emit(
            PheromoneType.REDIRECT,
            pattern,
            strength=strength,
            metadata={"command": "redirect"}
        )

    async def feedback(self, message: str, strength: float = 0.5) -> PheromoneSignal:
        """
        /ant:feedback <message>

        Emit feedback pheromone to guide colony behavior.
        """
        return await self.pheromone_layer.emit(
            PheromoneType.FEEDBACK,
            message,
            strength=strength,
            metadata={"command": "feedback"}
        )

    def get_status(self) -> Dict[str, Any]:
        """
        Get pheromone system status.

        Used by /ant:status command.
        """
        summary = self.pheromone_layer.get_signal_summary()

        return {
            "pheromone_summary": summary,
            "active_signals": [
                {
                    "type": s.signal_type.value,
                    "content": s.content,
                    "strength": f"{s.current_strength():.2f}",
                    "age": f"{s.age_seconds()/60:.1f}m"
                }
                for s in self.pheromone_layer.get_active_signals()
            ]
        }


# Factory functions
def create_pheromone_layer(enable_semantic: bool = True) -> PheromoneLayer:
    """
    Create a new pheromone layer

    Args:
        enable_semantic: Whether to enable semantic understanding (default: True)
    """
    return PheromoneLayer(enable_semantic=enable_semantic)


def create_sensitivity_profile(
    caste: str,
    init: float = 0.5,
    focus: float = 0.5,
    redirect: float = 0.5,
    feedback: float = 0.5
) -> SensitivityProfile:
    """Create a sensitivity profile for a Worker Ant caste"""
    return SensitivityProfile(
        caste=caste,
        init=init,
        focus=focus,
        redirect=redirect,
        feedback=feedback
    )


# Pre-defined sensitivity profiles for each caste
SENSITIVITY_PROFILES = {
    "mapper": SensitivityProfile(
        caste="mapper",
        init=1.0,      # Always responds to init
        focus=0.7,     # Responds to focus on areas
        redirect=0.3,  # Less affected by redirect
        feedback=0.5
    ),
    "planner": SensitivityProfile(
        caste="planner",
        init=1.0,      # Triggers planning
        focus=0.5,     # Adjusts priorities
        redirect=0.8,  # Avoids redirected approaches
        feedback=0.7
    ),
    "executor": SensitivityProfile(
        caste="executor",
        init=0.5,      # Awaits planning
        focus=0.9,     # Highly responsive to focus
        redirect=0.9,  # Strongly avoids redirected patterns
        feedback=0.7
    ),
    "verifier": SensitivityProfile(
        caste="verifier",
        init=0.3,      # Waits for code to test
        focus=0.8,     # Increases scrutiny on focus area
        redirect=0.5,
        feedback=0.9   # Highly responsive to quality feedback
    ),
    "researcher": SensitivityProfile(
        caste="researcher",
        init=0.7,      # Learns new domain
        focus=0.9,     # Researches focused topic
        redirect=0.4,
        feedback=0.5
    ),
    "synthesizer": SensitivityProfile(
        caste="synthesizer",
        init=0.2,      # Not very responsive
        focus=0.4,
        redirect=0.3,
        feedback=0.6
    ),
}
