"""
Queen Ant Colony - Pheromone Signal System

User signals that guide colony behavior without commands.

Pheromones use TTL (time-to-live) expiration. Signals expire based on
their expires_at field -- default is "phase_end" (lasts until phase completes).

Based on research from:
- Phase 1: Semantic protocols, communication bandwidth reduction
- Phase 4: Anticipatory context prediction, feedback loops
- Phase 6: Multi-agent coordination patterns

Updated to TTL-based model (Feb 2026) for simplicity and predictability.
"""

from typing import List, Dict, Any, Optional, Callable, Union
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
    Default TTL: end of phase."""

    REDIRECT = "redirect"
    """Strong repel signal. Warns colony away from approach/pattern.
    Default TTL: end of phase (can be extended with --ttl)."""

    FEEDBACK = "feedback"
    """Variable signal. Adjusts colony behavior based on Queen's feedback.
    Default TTL: end of phase."""


class Priority(Enum):
    """Signal priority levels"""
    HIGH = "high"      # REDIRECT signals
    NORMAL = "normal"  # FOCUS signals
    LOW = "low"        # FEEDBACK signals


# Default TTLs by signal type (None = phase_end)
DEFAULT_TTLS = {
    PheromoneType.INIT: None,        # Phase end
    PheromoneType.FOCUS: None,       # Phase end
    PheromoneType.REDIRECT: timedelta(hours=24),  # 24 hours default
    PheromoneType.FEEDBACK: None,    # Phase end
}

# Priority by signal type
SIGNAL_PRIORITIES = {
    PheromoneType.INIT: Priority.HIGH,
    PheromoneType.FOCUS: Priority.NORMAL,
    PheromoneType.REDIRECT: Priority.HIGH,
    PheromoneType.FEEDBACK: Priority.LOW,
}


@dataclass
class PheromoneSignal:
    """A pheromone signal from the Queen"""

    signal_type: PheromoneType
    content: str
    created_at: datetime
    expires_at: Optional[datetime] = None  # None = phase_end, datetime = specific expiry
    priority: Priority = Priority.NORMAL
    metadata: Dict[str, Any] = field(default_factory=dict)

    def __post_init__(self):
        if self.expires_at is None:
            # Set default TTL based on signal type
            default_ttl = DEFAULT_TTLS.get(self.signal_type)
            if default_ttl is not None:
                self.expires_at = self.created_at + default_ttl
            # else: expires_at stays None = phase_end

        if self.priority == Priority.NORMAL:
            # Set default priority based on signal type
            self.priority = SIGNAL_PRIORITIES.get(self.signal_type, Priority.NORMAL)

    def is_active(self) -> bool:
        """
        Check if signal is still active.

        Active means: expires_at is None (phase_end) OR now < expires_at
        """
        if self.expires_at is None:
            # Phase-end signals are always active until explicitly cleared
            return True
        return datetime.now() < self.expires_at

    def is_expired(self) -> bool:
        """Check if signal has expired"""
        return not self.is_active()

    def is_phase_end(self) -> bool:
        """Check if signal expires at phase end"""
        return self.expires_at is None

    def age_seconds(self) -> float:
        """Get age of signal in seconds"""
        return (datetime.now() - self.created_at).total_seconds()

    def time_until_expiry(self) -> Optional[timedelta]:
        """
        Get time until expiry, or None if phase_end
        """
        if self.expires_at is None:
            return None
        remaining = self.expires_at - datetime.now()
        return remaining if remaining.total_seconds() > 0 else timedelta(0)

    def to_dict(self) -> Dict[str, Any]:
        """Convert to dictionary for JSON serialization"""
        return {
            "type": self.signal_type.value,
            "content": self.content,
            "priority": self.priority.value,
            "created_at": self.created_at.isoformat(),
            "expires_at": self.expires_at.isoformat() if self.expires_at else "phase_end",
            "active": self.is_active(),
            "metadata": self.metadata
        }

    def __str__(self) -> str:
        """String representation for debugging"""
        age_mins = self.age_seconds() / 60
        expiry_str = "phase_end" if self.is_phase_end() else f"expires in {self.time_until_expiry()}"
        return (
            f"[{self.signal_type.value.upper()}] "
            f"\"{self.content[:50]}...\" "
            f"(priority: {self.priority.value}, age: {age_mins:.1f}m, {expiry_str})"
        )


@dataclass
class SensitivityProfile:
    """
    A Worker Ant's sensitivity to different pheromone types.

    Higher sensitivity = more likely to respond to pheromones.
    Works with priority levels for filtering.
    """

    caste: str
    init: float = 0.5      # Response to init signals
    focus: float = 0.5     # Response to focus signals
    redirect: float = 0.5  # Response to redirect signals
    feedback: float = 0.5  # Response to feedback signals

    # Minimum priority to respond to
    min_priority: Priority = Priority.LOW

    def get_sensitivity(self, signal_type: PheromoneType) -> float:
        """Get sensitivity for a specific signal type"""
        sensitivities = {
            PheromoneType.INIT: self.init,
            PheromoneType.FOCUS: self.focus,
            PheromoneType.REDIRECT: self.redirect,
            PheromoneType.FEEDBACK: self.feedback,
        }
        return sensitivities.get(signal_type, 0.0)

    def should_respond(self, signal: PheromoneSignal) -> bool:
        """
        Determine if this ant should respond to a signal.

        Responds if:
        1. Signal priority meets minimum threshold
        2. Sensitivity to this signal type > 0
        """
        # Check priority threshold
        priority_order = {Priority.LOW: 0, Priority.NORMAL: 1, Priority.HIGH: 2}
        if priority_order.get(signal.priority, 0) < priority_order.get(self.min_priority, 0):
            return False

        # Check sensitivity
        return self.get_sensitivity(signal.signal_type) > 0.3


class PheromoneLayer:
    """
    Manages all pheromone signals in the colony.

    The Pheromone Layer:
    - Receives signals from Queen
    - Tracks signal expiration (TTL-based)
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
        ttl: Optional[Union[timedelta, str]] = None,
        priority: Optional[Priority] = None,
        metadata: Optional[Dict[str, Any]] = None
    ) -> PheromoneSignal:
        """
        Emit a new pheromone signal.

        Args:
            signal_type: Type of signal (init, focus, redirect, feedback)
            content: Signal content/description
            ttl: Time-to-live (timedelta, or None for phase_end)
            priority: Signal priority (defaults based on type)
            metadata: Optional metadata

        Returns:
            The created signal
        """
        now = datetime.now()

        # Calculate expires_at from ttl
        expires_at = None
        if ttl is not None:
            if isinstance(ttl, str):
                # Parse string TTL like "2h", "1d"
                ttl = self._parse_ttl(ttl)
            if ttl is not None:
                expires_at = now + ttl

        signal = PheromoneSignal(
            signal_type=signal_type,
            content=content,
            created_at=now,
            expires_at=expires_at,
            priority=priority or SIGNAL_PRIORITIES.get(signal_type, Priority.NORMAL),
            metadata=metadata or {}
        )

        self.signals.append(signal)

        # Add to semantic index if enabled
        if self.semantic_layer is not None:
            signal_id = f"{signal_type.value}_{now.strftime('%Y%m%d_%H%M%S_%f')[:17]}"
            self.semantic_layer.add_signal(
                signal_id=signal_id,
                content=content,
                metadata={
                    **(metadata or {}),
                    "signal_type": signal_type.value,
                    "priority": signal.priority.value
                }
            )
            signal.metadata["semantic_id"] = signal_id

        return signal

    def _parse_ttl(self, ttl_str: str) -> Optional[timedelta]:
        """Parse TTL string like '2h', '1d', '30m'"""
        import re
        match = re.match(r'^(\d+)([hmd])$', ttl_str.lower())
        if not match:
            return None

        value = int(match.group(1))
        unit = match.group(2)

        if unit == 'm':
            return timedelta(minutes=value)
        elif unit == 'h':
            return timedelta(hours=value)
        elif unit == 'd':
            return timedelta(days=value)
        return None

    def get_active_signals(self, signal_type: Optional[PheromoneType] = None) -> List[PheromoneSignal]:
        """
        Get all active signals, optionally filtered by type.

        Active means: not expired (expires_at is None or in the future)
        """
        active = [s for s in self.signals if s.is_active()]

        if signal_type is not None:
            active = [s for s in active if s.signal_type == signal_type]

        # Sort by priority (high first), then by creation time (newest first)
        priority_order = {Priority.HIGH: 0, Priority.NORMAL: 1, Priority.LOW: 2}
        active.sort(key=lambda s: (priority_order.get(s.priority, 1), -s.created_at.timestamp()))

        return active

    def get_signals_for_ant(
        self,
        sensitivity: SensitivityProfile
    ) -> List[PheromoneSignal]:
        """
        Get all signals that an ant should respond to.

        Returns signals where ant's sensitivity profile indicates response.
        """
        relevant = []

        for signal in self.get_active_signals():
            if sensitivity.should_respond(signal):
                relevant.append(signal)

        return relevant

    def cleanup_expired(self):
        """Remove expired signals from active list"""
        active = [s for s in self.signals if s.is_active()]
        expired = [s for s in self.signals if s.is_expired()]

        # Move expired to history
        self.signal_history.extend(expired)
        self.signals = active

        # Trim history if needed
        if len(self.signal_history) > self.max_history:
            self.signal_history = self.signal_history[-self.max_history:]

    def clear_phase_end_signals(self):
        """Clear all signals that expire at phase end"""
        phase_end_signals = [s for s in self.signals if s.is_phase_end()]
        self.signal_history.extend(phase_end_signals)
        self.signals = [s for s in self.signals if not s.is_phase_end()]

    def get_signal_summary(self) -> Dict[str, Any]:
        """Get summary of all signals"""
        active_by_type = {
            signal_type.value: len(self.get_active_signals(signal_type))
            for signal_type in PheromoneType
        }

        active = self.get_active_signals()
        return {
            "total_active": len(active),
            "by_type": active_by_type,
            "total_history": len(self.signal_history),
            "newest_signal": active[0] if active else None
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

        Uses simple keyword matching (Jaccard similarity).
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

    def force_expire_all(self):
        """Force expire all signals (for testing)"""
        self.signal_history.extend(self.signals)
        self.signals = []


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

    async def detect_pheromones(self) -> List[PheromoneSignal]:
        """
        Detect pheromones that this responder should react to.

        Returns signals matching sensitivity profile.
        """
        signals = self.pheromone_layer.get_signals_for_ant(self.sensitivity)

        # Filter out recently responded signals (avoid thrashing)
        if self.last_response_time:
            recent_threshold = datetime.now() - timedelta(minutes=1)
            signals = [
                s for s in signals
                if s.created_at > self.last_response_time or
                   s.priority == Priority.HIGH  # Always respond to high priority
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
            priority=Priority.HIGH,
            metadata={"command": "init"}
        )

    async def focus(self, area: str, ttl: Optional[str] = None) -> PheromoneSignal:
        """
        /ant:focus <area> [--ttl <duration>]

        Emit focus pheromone to guide colony attention.
        """
        ttl_delta = self.pheromone_layer._parse_ttl(ttl) if ttl else None
        return await self.pheromone_layer.emit(
            PheromoneType.FOCUS,
            area,
            ttl=ttl_delta,
            priority=Priority.NORMAL,
            metadata={"command": "focus"}
        )

    async def redirect(self, pattern: str, ttl: Optional[str] = None) -> PheromoneSignal:
        """
        /ant:redirect <pattern> [--ttl <duration>]

        Emit redirect pheromone to warn colony away from approach.
        """
        ttl_delta = self.pheromone_layer._parse_ttl(ttl) if ttl else timedelta(hours=24)
        return await self.pheromone_layer.emit(
            PheromoneType.REDIRECT,
            pattern,
            ttl=ttl_delta,
            priority=Priority.HIGH,
            metadata={"command": "redirect"}
        )

    async def feedback(self, message: str, ttl: Optional[str] = None) -> PheromoneSignal:
        """
        /ant:feedback <message> [--ttl <duration>]

        Emit feedback pheromone to guide colony behavior.
        """
        ttl_delta = self.pheromone_layer._parse_ttl(ttl) if ttl else None
        return await self.pheromone_layer.emit(
            PheromoneType.FEEDBACK,
            message,
            ttl=ttl_delta,
            priority=Priority.LOW,
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
                    "priority": s.priority.value,
                    "expires": "phase_end" if s.is_phase_end() else s.expires_at.isoformat(),
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
    feedback: float = 0.5,
    min_priority: Priority = Priority.LOW
) -> SensitivityProfile:
    """Create a sensitivity profile for a Worker Ant caste"""
    return SensitivityProfile(
        caste=caste,
        init=init,
        focus=focus,
        redirect=redirect,
        feedback=feedback,
        min_priority=min_priority
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
