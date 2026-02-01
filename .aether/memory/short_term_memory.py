#!/usr/bin/env python3
"""
Aether Queen Ant Colony - Short-Term Memory Layer

Short-Term Memory: Compressed session storage with DAST compression.
- 10 sessions limit
- DAST (Dynamic Allocation of Soft Tokens): 2.5x compression
- Associative links to related sessions
- Forgetting mechanism: LRU after 10 sessions

Based on MEMORY_ARCHITECTURE_RESEARCH.md:
"Short-term memory stores DAST-compressed sessions. Maintains 10 most recent
sessions with associative linking for context retrieval."
"""

import asyncio
import json
import re
from typing import Dict, List, Any, Optional, Set
from dataclasses import dataclass, field, asdict
from datetime import datetime, timedelta
from collections import OrderedDict
import hashlib

try:
    from .working_memory import ContextItem, estimate_tokens
except ImportError:
    from working_memory import ContextItem, estimate_tokens


@dataclass
class SessionSummary:
    """
    Compressed summary of a session/phase

    Attributes:
        session_id: Unique identifier
        content: DAST-compressed content
        original_tokens: Original token count before compression
        compressed_tokens: Token count after compression
        compression_ratio: Ratio of original/compressed
        metadata: Session metadata (goal, phase, duration, etc.)
        created_at: When session was stored
        accessed_at: Last access time (for LRU)
        associations: Links to related sessions
    """
    session_id: str
    content: str
    original_tokens: int
    compressed_tokens: int
    compression_ratio: float
    metadata: Dict[str, Any] = field(default_factory=dict)
    created_at: str = field(default_factory=lambda: datetime.now().isoformat())
    accessed_at: str = field(default_factory=lambda: datetime.now().isoformat())
    associations: Set[str] = field(default_factory=set)

    def __post_init__(self):
        """Calculate compression ratio if not set"""
        if self.compression_ratio == 0 and self.compressed_tokens > 0:
            self.compression_ratio = self.original_tokens / self.compressed_tokens

    @property
    def age_hours(self) -> float:
        """Age of session in hours"""
        created = datetime.fromisoformat(self.created_at)
        now = datetime.now()
        return (now - created).total_seconds() / 3600

    def touch(self):
        """Update accessed time for LRU"""
        self.accessed_at = datetime.now().isoformat()


@dataclass
class AssociativeLink:
    """
    Link between related sessions

    Attributes:
        from_session: Source session ID
        to_session: Target session ID
        link_type: Type of association (continuation, similar, related, etc.)
        strength: Link strength 0.0-1.0
        created_at: When link was created
    """
    from_session: str
    to_session: str
    link_type: str
    strength: float
    created_at: str = field(default_factory=lambda: datetime.now().isoformat())


class DASTCompressor:
    """
    DAST (Dynamic Allocation of Soft Tokens) Compressor

    Semantic compression that preserves meaning while reducing tokens.
    Achieves ~2.5x compression ratio per research.

    Techniques:
    1. Extract key entities and relationships
    2. Summarize while preserving semantic structure
    3. Remove redundancy while keeping essential info
    4. Use concise language
    """

    def __init__(self, compression_target: float = 0.4):
        """
        Initialize DAST compressor

        Args:
            compression_target: Target compression ratio (0.4 = 40% = 2.5x)
        """
        self.compression_target = compression_target

    async def compress(self, content: str, metadata: Dict[str, Any]) -> str:
        """
        Compress content using DAST

        Args:
            content: Content to compress
            metadata: Context for compression

        Returns:
            Compressed content
        """
        original_tokens = estimate_tokens(content)
        target_tokens = int(original_tokens * self.compression_target)

        # Extract key information
        key_info = await self._extract_key_info(content, metadata)

        # Build compressed summary
        compressed = await self._build_summary(key_info, target_tokens)

        return compressed

    async def _extract_key_info(
        self,
        content: str,
        metadata: Dict[str, Any]
    ) -> Dict[str, Any]:
        """Extract key information from content"""
        info = {
            "goal": metadata.get("goal", ""),
            "phase": metadata.get("phase", ""),
            "tasks_completed": [],
            "decisions": [],
            "issues": [],
            "key_entities": set(),
            "patterns": []
        }

        # Extract sentences
        sentences = re.split(r'[.!?]+', content)

        # Analyze each sentence
        for sentence in sentences:
            sentence = sentence.strip()
            if not sentence:
                continue

            # Look for task completion indicators
            if any(word in sentence.lower() for word in ["completed", "finished", "done", "built"]):
                info["tasks_completed"].append(sentence)

            # Look for decisions
            elif any(word in sentence.lower() for word in ["decided", "chose", "selected", "using"]):
                info["decisions"].append(sentence)

            # Look for issues
            elif any(word in sentence.lower() for word in ["error", "issue", "problem", "bug"]):
                info["issues"].append(sentence)

            # Extract potential entities (capitalized words)
            entities = re.findall(r'\b[A-Z][a-zA-Z]+\b', sentence)
            info["key_entities"].update(entities)

        # Convert sets to lists for JSON serialization
        info["key_entities"] = list(info["key_entities"])

        return info

    async def _build_summary(
        self,
        key_info: Dict[str, Any],
        target_tokens: int
    ) -> str:
        """Build compressed summary from key info"""
        parts = []

        # Goal/Context
        if key_info.get("goal"):
            parts.append(f"Goal: {key_info['goal']}")

        # Phase
        if key_info.get("phase"):
            parts.append(f"Phase: {key_info['phase']}")

        # Tasks completed (summary)
        tasks = key_info.get("tasks_completed", [])
        if tasks:
            if len(tasks) <= 3:
                parts.append(f"Completed: {'; '.join(tasks)}")
            else:
                parts.append(f"Completed: {len(tasks)} tasks")

        # Decisions (summary)
        decisions = key_info.get("decisions", [])
        if decisions:
            if len(decisions) <= 2:
                parts.append(f"Decisions: {'; '.join(decisions)}")
            else:
                parts.append(f"Decisions: {len(decisions)} made")

        # Issues (summary)
        issues = key_info.get("issues", [])
        if issues:
            parts.append(f"Issues: {len(issues)} encountered and resolved")

        # Key entities (if any)
        entities = key_info.get("key_entities", [])
        if entities:
            parts.append(f"Key: {', '.join(entities[:5])}")

        summary = ". ".join(parts)

        # Check if we need more aggressive compression
        current_tokens = estimate_tokens(summary)
        if current_tokens > target_tokens:
            # Apply more aggressive compression
            summary = await self._compress_aggressively(summary, target_tokens)

        return summary

    async def _compress_aggressively(
        self,
        content: str,
        target_tokens: int
    ) -> str:
        """Apply aggressive compression"""
        # Split into parts
        parts = content.split(". ")

        # Keep most important parts (first ones)
        compressed_parts = []
        current_tokens = 0

        for part in parts:
            part_tokens = estimate_tokens(part)
            if current_tokens + part_tokens <= target_tokens:
                compressed_parts.append(part)
                current_tokens += part_tokens
            else:
                break

        return ". ".join(compressed_parts)


class ShortTermMemory:
    """
    Short-Term Memory Layer

    Features:
    - 10 sessions limit (LRU eviction)
    - DAST compression (~2.5x ratio)
    - Associative linking
    - Semantic search across sessions
    """

    def __init__(self, max_sessions: int = 10):
        """
        Initialize short-term memory

        Args:
            max_sessions: Maximum sessions to store
        """
        self.max_sessions = max_sessions

        # Use OrderedDict for LRU ordering
        self.sessions: OrderedDict[str, SessionSummary] = OrderedDict()

        # Associative links
        self.links: List[AssociativeLink] = []

        # DAST compressor
        self.compressor = DASTCompressor()

        # Statistics
        self.stats = {
            "total_sessions": 0,
            "total_compressed": 0,
            "total_evicted": 0,
            "total_links": 0,
            "avg_compression_ratio": 0.0
        }

    async def add_session(
        self,
        content: str,
        metadata: Dict[str, Any],
        associations: Optional[List[str]] = None
    ) -> str:
        """
        Add compressed session to short-term memory

        Args:
            content: Session content (will be compressed)
            metadata: Session metadata (goal, phase, duration, etc.)
            associations: IDs of related sessions

        Returns:
            Session ID
        """
        # Estimate original tokens
        original_tokens = estimate_tokens(content)

        # Compress using DAST
        compressed_content = await self.compressor.compress(content, metadata)
        compressed_tokens = estimate_tokens(compressed_content)

        # Calculate compression ratio
        compression_ratio = original_tokens / max(compressed_tokens, 1)

        # Create session summary
        session_id = f"session_{datetime.now().strftime('%Y%m%d_%H%M%S')}"

        session = SessionSummary(
            session_id=session_id,
            content=compressed_content,
            original_tokens=original_tokens,
            compressed_tokens=compressed_tokens,
            compression_ratio=compression_ratio,
            metadata=metadata,
            associations=set(associations or [])
        )

        # Check if we need to evict
        while len(self.sessions) >= self.max_sessions:
            await self._evict_oldest()

        # Add to storage (newest at end)
        self.sessions[session_id] = session

        # Create associative links
        for assoc_id in (associations or []):
            if assoc_id in self.sessions:
                link = AssociativeLink(
                    from_session=session_id,
                    to_session=assoc_id,
                    link_type="related",
                    strength=0.5
                )
                self.links.append(link)
                self.stats["total_links"] += 1

        # Update stats
        self.stats["total_sessions"] += 1
        self.stats["total_compressed"] += 1

        # Update average compression ratio
        if self.stats["total_sessions"] > 0:
            total_ratio = (
                self.stats["avg_compression_ratio"] * (self.stats["total_sessions"] - 1)
                + compression_ratio
            )
            self.stats["avg_compression_ratio"] = total_ratio / self.stats["total_sessions"]

        return session_id

    async def get(self, session_id: str) -> Optional[SessionSummary]:
        """
        Get session by ID (updates LRU)

        Args:
            session_id: Session ID

        Returns:
            SessionSummary if found
        """
        if session_id not in self.sessions:
            return None

        # Move to end (most recently used)
        session = self.sessions.pop(session_id)
        session.touch()
        self.sessions[session_id] = session

        return session

    async def search(self, query: str, limit: int = 5) -> List[SessionSummary]:
        """
        Search sessions by content

        Args:
            query: Search query
            limit: Max results

        Returns:
            Matching sessions
        """
        results = []
        query_lower = query.lower()

        for session in self.sessions.values():
            # Search content
            if query_lower in session.content.lower():
                results.append(session)
                continue

            # Search metadata
            for key, value in session.metadata.items():
                if isinstance(value, str) and query_lower in value.lower():
                    results.append(session)
                    break

        # Sort by recency
        results.sort(key=lambda x: x.accessed_at, reverse=True)

        return results[:limit]

    async def get_associated(self, session_id: str) -> List[SessionSummary]:
        """
        Get sessions associated with given session

        Args:
            session_id: Session ID

        Returns:
            Associated sessions
        """
        session = await self.get(session_id)
        if not session:
            return []

        results = []
        for assoc_id in session.associations:
            if assoc_id in self.sessions:
                results.append(self.sessions[assoc_id])

        return results

    async def _evict_oldest(self) -> None:
        """Evict oldest session (LRU)"""
        if not self.sessions:
            return

        # Get oldest (first in OrderedDict)
        oldest_id = next(iter(self.sessions))

        # Remove associations
        self.links = [
            link for link in self.links
            if link.from_session != oldest_id and link.to_session != oldest_id
        ]

        # Remove session
        del self.sessions[oldest_id]

        self.stats["total_evicted"] += 1

    async def apply_forgetting(self) -> int:
        """
        Apply forgetting mechanism (remove beyond max_sessions)

        Returns:
            Number of sessions evicted
        """
        evicted = 0

        while len(self.sessions) > self.max_sessions:
            await self._evict_oldest()
            evicted += 1

        return evicted

    @property
    def session_count(self) -> int:
        """Number of sessions stored"""
        return len(self.sessions)

    @property
    def total_saved_tokens(self) -> int:
        """Total tokens saved by compression"""
        saved = 0
        for session in self.sessions.values():
            saved += session.original_tokens - session.compressed_tokens
        return saved

    def to_dict(self) -> Dict[str, Any]:
        """Convert to dictionary"""
        return {
            "max_sessions": self.max_sessions,
            "session_count": self.session_count,
            "total_saved_tokens": self.total_saved_tokens,
            "avg_compression_ratio": round(self.stats["avg_compression_ratio"], 2),
            "stats": self.stats,
            "sessions": [
                {
                    "session_id": s.session_id,
                    "content": s.content,
                    "compression_ratio": round(s.compression_ratio, 2),
                    "original_tokens": s.original_tokens,
                    "compressed_tokens": s.compressed_tokens,
                    "created_at": s.created_at,
                    "age_hours": round(s.age_hours, 1)
                }
                for s in self.sessions.values()
            ]
        }

    def get_status(self) -> Dict[str, Any]:
        """Get current status"""
        return {
            "layer": "short_term_memory",
            "max_sessions": self.max_sessions,
            "session_count": self.session_count,
            "total_saved_tokens": self.total_saved_tokens,
            "avg_compression_ratio": round(self.stats["avg_compression_ratio"], 2),
            "stats": self.stats
        }


# Demo
async def demo_short_term_memory():
    """Demonstrate short-term memory"""
    print("=" * 60)
    print("Short-Term Memory Demo")
    print("=" * 60)
    print()

    stm = ShortTermMemory(max_sessions=5)

    # Add some sessions
    print("Adding sessions...")

    for i in range(3):
        content = f"""
        Session {i+1}: Building the authentication system.
        We completed {5-i} tasks including user login, password reset,
        and JWT token implementation. Decided to use FastAPI with
        PostgreSQL. Encountered 2 issues with token expiration but
        resolved them by adjusting the TTL.
        """

        session_id = await stm.add_session(
            content=content,
            metadata={
                "goal": "Build REST API with auth",
                "phase": f"{i+1}",
                "duration": "2 hours"
            }
        )

        print(f"  Added {session_id}")
        print(f"    Compression: {stm.sessions[session_id].compression_ratio:.2f}x")

    print()
    print("Status:")
    status = stm.get_status()
    for key, value in status.items():
        print(f"  {key}: {value}")
    print()


if __name__ == "__main__":
    asyncio.run(demo_short_term_memory())
