#!/usr/bin/env python3
"""
Aether Queen Ant Colony - Triple-Layer Memory System

Orchestrator for the three-layer memory architecture:
- Working Memory: 200k tokens, current session
- Short-Term Memory: 10 sessions, DAST compressed
- Long-Term Memory: Persistent, maximum compression

Based on MEMORY_ARCHITECTURE_RESEARCH.md:
"Three-tier hierarchical memory mirrors human cognition:
Working (immediate) → Short-term (recent) → Long-term (persistent)
"""

import asyncio
from typing import Dict, List, Any, Optional, Literal
from dataclasses import dataclass
from datetime import datetime

try:
    from .working_memory import WorkingMemory, ContextItem, estimate_tokens
    from .short_term_memory import ShortTermMemory, SessionSummary
    from .long_term_memory import LongTermMemory, KnowledgePattern
except ImportError:
    from working_memory import WorkingMemory, ContextItem, estimate_tokens
    from short_term_memory import ShortTermMemory, SessionSummary
    from long_term_memory import LongTermMemory, KnowledgePattern


MemoryLayer = Literal["working", "short_term", "long_term"]


@dataclass
class MemoryQuery:
    """Query for memory retrieval"""
    query: str
    layers: List[MemoryLayer] = None
    limit: int = 10
    item_type: Optional[str] = None
    category: Optional[str] = None
    min_confidence: float = 0.0


@dataclass
class MemoryResult:
    """Result from memory query"""
    content: str
    layer: MemoryLayer
    relevance_score: float
    metadata: Dict[str, Any]
    source_id: str


class TripleLayerMemory:
    """
    Three-layer memory system

    Memory Flow:
    1. Add to Working Memory (immediate access)
    2. At phase boundary: Compress to Short-Term (10 sessions)
    3. Apply DAST compression (2.5x ratio)
    4. Extract patterns for Long-Term (persistent)
    5. Apply forgetting (LRU for short-term)

    Retrieval:
    - Search all layers in parallel
    - Rank by relevance and recency
    - Expand through associative links
    """

    def __init__(
        self,
        working_max_tokens: int = 200_000,
        short_term_max_sessions: int = 10,
        long_term_storage_path: str = ".aether/memory/long_term.json"
    ):
        """
        Initialize triple-layer memory

        Args:
            working_max_tokens: Working memory budget
            short_term_max_sessions: Short-term session limit
            long_term_storage_path: Long-term storage path
        """
        # Initialize layers
        self.working = WorkingMemory(max_tokens=working_max_tokens)
        self.short_term = ShortTermMemory(max_sessions=short_term_max_sessions)
        self.long_term = LongTermMemory(storage_path=long_term_storage_path)

        # Statistics
        self.stats = {
            "total_additions": 0,
            "total_queries": 0,
            "total_compressions": 0,
            "phase_boundaries": 0
        }

    # ============================================================
    # Working Memory Operations
    # ============================================================

    async def add_to_working(
        self,
        content: str,
        metadata: Optional[Dict[str, Any]] = None,
        item_type: str = "general"
    ) -> Optional[str]:
        """
        Add content to working memory

        Args:
            content: Content to add
            metadata: Associated metadata
            item_type: Type/category

        Returns:
            Item ID if added, None if budget exceeded
        """
        item_id = await self.working.add(content, metadata, item_type)
        if item_id:
            self.stats["total_additions"] += 1
        return item_id

    async def get_from_working(self, item_id: str) -> Optional[ContextItem]:
        """Get item from working memory"""
        return await self.working.get(item_id)

    async def search_working(
        self,
        query: str,
        limit: int = 10,
        item_type: Optional[str] = None
    ) -> List[ContextItem]:
        """Search working memory"""
        return await self.working.search(query, limit, item_type)

    # ============================================================
    # Phase Boundary Operations
    # ============================================================

    async def compress_to_short_term(
        self,
        session_metadata: Dict[str, Any]
    ) -> str:
        """
        Compress working memory to short-term at phase boundary

        Args:
            session_metadata: Metadata about the session/phase

        Returns:
            Session ID of compressed session
        """
        # Flush working memory
        items = await self.working.flush()

        # Combine into session content
        session_content = self._items_to_session_content(items)

        # Compress to short-term
        session_id = await self.short_term.add_session(
            content=session_content,
            metadata=session_metadata
        )

        self.stats["total_compressions"] += 1
        self.stats["phase_boundaries"] += 1

        # Extract patterns for long-term
        await self._extract_and_store_patterns(items, session_metadata)

        return session_id

    def _items_to_session_content(self, items: List[ContextItem]) -> str:
        """Convert working memory items to session content"""
        parts = []

        for item in items:
            # Add type prefix
            item_type = item.metadata.get("type", "general")
            prefix = f"[{item_type.upper()}]" if item_type != "general" else ""

            # Add content
            if prefix:
                parts.append(f"{prefix} {item.content}")
            else:
                parts.append(item.content)

        return "\n".join(parts)

    async def _extract_and_store_patterns(
        self,
        items: List[ContextItem],
        session_metadata: Dict[str, Any]
    ) -> None:
        """Extract patterns from items for long-term storage"""
        # Group by type
        by_type: Dict[str, List[ContextItem]] = {}
        for item in items:
            item_type = item.metadata.get("type", "general")
            if item_type not in by_type:
                by_type[item_type] = []
            by_type[item_type].append(item)

        # Extract patterns from each type
        for item_type, type_items in by_type.items():
            # Look for patterns (items that appear multiple times)
            content_counts: Dict[str, int] = {}
            for item in type_items:
                content_counts[item.content] = content_counts.get(item.content, 0) + 1

            # Store repeated patterns
            for content, count in content_counts.items():
                if count >= 2:  # Pattern threshold
                    await self.long_term.store(
                        category=item_type if item_type in self.long_term.CATEGORIES else "patterns",
                        key=content[:50],  # First 50 chars as key
                        value=content,
                        confidence=min(0.5 + (count * 0.1), 1.0),
                        metadata={
                            "occurrences": count,
                            "source": "pattern_extraction",
                            "session": session_metadata.get("phase", "unknown")
                        }
                    )

    # ============================================================
    # Cross-Layer Retrieval
    # ============================================================

    async def retrieve(
        self,
        query: str,
        layers: Optional[List[MemoryLayer]] = None,
        limit: int = 10
    ) -> List[MemoryResult]:
        """
        Retrieve from multiple memory layers

        Args:
            query: Search query
            layers: Layers to search (default: all)
            limit: Max results per layer

        Returns:
            Combined and ranked results
        """
        if layers is None:
            layers = ["working", "short_term", "long_term"]

        self.stats["total_queries"] += 1

        results: List[MemoryResult] = []

        # Search working memory
        if "working" in layers:
            working_items = await self.working.search(query, limit)
            for item in working_items:
                results.append(MemoryResult(
                    content=item.content,
                    layer="working",
                    relevance_score=1.0,  # Working memory is most relevant
                    metadata=item.metadata,
                    source_id=item.item_id
                ))

        # Search short-term memory
        if "short_term" in layers:
            short_term_sessions = await self.short_term.search(query, limit)
            for session in short_term_sessions:
                results.append(MemoryResult(
                    content=session.content,
                    layer="short_term",
                    relevance_score=0.7,  # Short-term is less recent
                    metadata=session.metadata,
                    source_id=session.session_id
                ))

        # Search long-term memory
        if "long_term" in layers:
            long_term_patterns = await self.long_term.search(query, limit=limit)
            for pattern in long_term_patterns:
                results.append(MemoryResult(
                    content=pattern.value,
                    layer="long_term",
                    relevance_score=pattern.confidence,
                    metadata=pattern.metadata,
                    source_id=pattern.pattern_id
                ))

        # Sort by relevance (working first, then by score)
        results.sort(
            key=lambda r: (r.layer != "working", -r.relevance_score),
            reverse=False
        )

        return results[:limit]

    # ============================================================
    # Long-Term Operations
    # ============================================================

    async def store_long_term(
        self,
        category: str,
        key: str,
        value: str,
        confidence: float = 0.5
    ) -> str:
        """Store pattern in long-term memory"""
        return await self.long_term.store(category, key, value, confidence)

    async def learn_from_error(
        self,
        error_category: str,
        symptom: str,
        fix: str,
        prevention: str
    ) -> str:
        """Learn from error for future prevention"""
        return await self.long_term.learn_from_error(
            error_category, symptom, fix, prevention
        )

    # ============================================================
    # Status and Diagnostics
    # ============================================================

    def get_status(self) -> Dict[str, Any]:
        """Get status of all memory layers"""
        return {
            "triple_layer_memory": {
                "working": self.working.get_status(),
                "short_term": self.short_term.get_status(),
                "long_term": self.long_term.get_status(),
                "stats": self.stats
            }
        }

    def get_compression_stats(self) -> Dict[str, Any]:
        """Get compression statistics"""
        return {
            "working": {
                "used_tokens": self.working.used_tokens,
                "max_tokens": self.working.max_tokens,
                "usage_percent": round(self.working.usage_percent, 2)
            },
            "short_term": {
                "session_count": self.short_term.session_count,
                "max_sessions": self.short_term.max_sessions,
                "total_saved_tokens": self.short_term.total_saved_tokens,
                "avg_compression_ratio": round(
                    self.short_term.stats.get("avg_compression_ratio", 0), 2
                )
            },
            "long_term": {
                "total_patterns": self.long_term.total_patterns,
                "categories": self.long_term.stats["category_counts"]
            }
        }


# Demo
async def demo_triple_layer_memory():
    """Demonstrate triple-layer memory"""
    print("=" * 60)
    print("Triple-Layer Memory Demo")
    print("=" * 60)
    print()

    tlm = TripleLayerMemory(
        working_max_tokens=5000,  # Small for demo
        short_term_max_sessions=5
    )

    print("1. Adding to working memory...")
    await tlm.add_to_working(
        "Build REST API with FastAPI and PostgreSQL",
        metadata={"source": "user", "priority": "high"},
        item_type="goal"
    )
    await tlm.add_to_working(
        "Use snake_case for files, PascalCase for classes",
        metadata={"source": "convention"},
        item_type="convention"
    )
    await tlm.add_to_working(
        "Always use parameterized queries to prevent SQL injection",
        metadata={"source": "security"},
        item_type="best_practice"
    )

    working_status = tlm.working.get_status()
    print(f"   Items: {working_status['item_count']}")
    print(f"   Tokens: {working_status['used_tokens']}/{working_status['max_tokens']}")
    print()

    print("2. Searching working memory...")
    results = await tlm.search_working("API")
    for r in results:
        print(f"   Found: {r.content}")
    print()

    print("3. Compressing to short-term (phase boundary)...")
    session_id = await tlm.compress_to_short_term({
        "goal": "Build REST API",
        "phase": "1",
        "duration": "2 hours"
    })
    print(f"   Session: {session_id}")
    print(f"   Working memory now empty: {tlm.working.item_count} items")
    print()

    print("4. Compression stats:")
    stats = tlm.get_compression_stats()
    print(f"   Short-term sessions: {stats['short_term']['session_count']}")
    print(f"   Avg compression ratio: {stats['short_term']['avg_compression_ratio']}x")
    print()

    print("5. Cross-layer retrieval:")
    results = await tlm.retrieve("FastAPI")
    for r in results:
        print(f"   [{r.layer}] {r.content[:50]}...")
    print()


if __name__ == "__main__":
    asyncio.run(demo_triple_layer_memory())
