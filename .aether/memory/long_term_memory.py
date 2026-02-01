#!/usr/bin/env python3
"""
Aether Queen Ant Colony - Long-Term Memory Layer

Long-Term Memory: Persistent knowledge storage with maximum compression.
- Categories: tech_stack, conventions, patterns, errors, best_practices
- Maximum compression for efficiency
- Persistent across all sessions
- Semantic search and retrieval

Based on MEMORY_ARCHITECTURE_RESEARCH.md:
"Long-term memory stores abstracted patterns and persistent knowledge.
Maximum compression with categorical organization for efficient retrieval."
"""

import asyncio
import json
import os
from typing import Dict, List, Any, Optional, Set
from dataclasses import dataclass, field
from datetime import datetime
import hashlib

try:
    from .working_memory import estimate_tokens
except ImportError:
    from working_memory import estimate_tokens


@dataclass
class KnowledgePattern:
    """
    Abstracted pattern stored in long-term memory

    Attributes:
        pattern_id: Unique identifier
        category: Category (tech_stack, conventions, patterns, errors, etc.)
        key: Key for retrieval
        value: Pattern value (maximally compressed)
        occurrences: How many times this pattern was seen
        confidence: Confidence score 0.0-1.0
        created_at: First seen
        updated_at: Last updated
        metadata: Additional metadata
    """
    pattern_id: str
    category: str
    key: str
    value: str
    occurrences: int = 1
    confidence: float = 0.5
    created_at: str = field(default_factory=lambda: datetime.now().isoformat())
    updated_at: str = field(default_factory=lambda: datetime.now().isoformat())
    metadata: Dict[str, Any] = field(default_factory=dict)

    def touch(self):
        """Update timestamp and increment occurrences"""
        self.updated_at = datetime.now().isoformat()
        self.occurrences += 1

    def update_confidence(self, was_correct: bool, alpha: float = 0.1):
        """
        Update confidence using exponential moving average

        Args:
            was_correct: Whether pattern was correct/applied
            alpha: Learning rate (default 0.1)
        """
        target = 1.0 if was_correct else 0.0
        self.confidence = alpha * target + (1 - alpha) * self.confidence


class PersistentStorage:
    """
    File-based persistent storage

    Stores patterns as JSON for durability across sessions.
    """

    def __init__(self, storage_path: str = ".aether/memory/long_term.json"):
        """
        Initialize persistent storage

        Args:
            storage_path: Path to storage file
        """
        self.storage_path = storage_path
        self._ensure_directory()

    def _ensure_directory(self):
        """Ensure storage directory exists"""
        directory = os.path.dirname(self.storage_path)
        if directory:
            os.makedirs(directory, exist_ok=True)

    async def save(self, data: Dict[str, Any]) -> None:
        """Save data to disk"""
        self._ensure_directory()
        with open(self.storage_path, 'w') as f:
            json.dump(data, f, indent=2)

    async def load(self) -> Dict[str, Any]:
        """Load data from disk"""
        if not os.path.exists(self.storage_path):
            return {}

        with open(self.storage_path, 'r') as f:
            return json.load(f)

    async def append(self, key: str, value: Any) -> None:
        """Append to storage"""
        data = await self.load()
        data[key] = value
        await self.save(data)

    async def clear(self) -> None:
        """Clear all storage"""
        if os.path.exists(self.storage_path):
            os.remove(self.storage_path)


class LongTermMemory:
    """
    Long-Term Memory Layer

    Features:
    - Persistent storage across sessions
    - Categorical organization
    - Maximum compression
    - Confidence-based learning
    - Semantic search
    """

    # Standard categories
    CATEGORIES = [
        "tech_stack",
        "conventions",
        "patterns",
        "errors",
        "best_practices",
        "anti_patterns",
        "preferences",
        "learnings"
    ]

    def __init__(self, storage_path: str = ".aether/memory/long_term.json"):
        """
        Initialize long-term memory

        Args:
            storage_path: Path for persistent storage
        """
        self.storage = PersistentStorage(storage_path)

        # In-memory cache organized by category
        self.patterns: Dict[str, Dict[str, KnowledgePattern]] = {
            cat: {} for cat in self.CATEGORIES
        }

        # Statistics
        self.stats = {
            "total_patterns": 0,
            "total_retrievals": 0,
            "total_stores": 0,
            "category_counts": {cat: 0 for cat in self.CATEGORIES}
        }

        # Load from storage
        asyncio.create_task(self._load_from_storage())

    async def _load_from_storage(self):
        """Load patterns from persistent storage"""
        data = await self.storage.load()

        for category_str, patterns in data.items():
            if category_str not in self.patterns:
                self.patterns[category_str] = {}

            for key, pattern_data in patterns.items():
                pattern = KnowledgePattern(**pattern_data)
                self.patterns[category_str][key] = pattern
                self.stats["total_patterns"] += 1

                if category_str in self.stats["category_counts"]:
                    self.stats["category_counts"][category_str] += 1

    async def store(
        self,
        category: str,
        key: str,
        value: str,
        confidence: float = 0.5,
        metadata: Optional[Dict[str, Any]] = None
    ) -> str:
        """
        Store a pattern in long-term memory

        Args:
            category: Category (must be in CATEGORIES)
            key: Unique key within category
            value: Pattern value (maximally compressed)
            confidence: Initial confidence
            metadata: Additional metadata

        Returns:
            Pattern ID
        """
        # Validate category
        if category not in self.patterns:
            # Add new category
            self.patterns[category] = {}
            self.stats["category_counts"][category] = 0

        # Check if pattern exists
        if key in self.patterns[category]:
            # Update existing
            pattern = self.patterns[category][key]
            pattern.value = value
            pattern.touch()
            pattern.metadata.update(metadata or {})
        else:
            # Create new pattern
            pattern_id = f"{category}_{hashlib.md5(key.encode()).hexdigest()[:8]}"

            pattern = KnowledgePattern(
                pattern_id=pattern_id,
                category=category,
                key=key,
                value=value,
                confidence=confidence,
                metadata=metadata or {}
            )

            self.patterns[category][key] = pattern
            self.stats["total_patterns"] += 1
            self.stats["category_counts"][category] += 1

        self.stats["total_stores"] += 1

        # Persist
        await self._persist()

        return pattern.pattern_id

    async def retrieve(self, category: str, key: str) -> Optional[KnowledgePattern]:
        """
        Retrieve a pattern by category and key

        Args:
            category: Category
            key: Pattern key

        Returns:
            KnowledgePattern if found
        """
        if category not in self.patterns:
            return None

        pattern = self.patterns[category].get(key)
        if pattern:
            self.stats["total_retrievals"] += 1

        return pattern

    async def search(
        self,
        query: str,
        categories: Optional[List[str]] = None,
        limit: int = 10
    ) -> List[KnowledgePattern]:
        """
        Search patterns across categories

        Args:
            query: Search query
            categories: Categories to search (default: all)
            limit: Max results

        Returns:
            Matching patterns sorted by confidence and occurrences
        """
        results = []
        query_lower = query.lower()

        # Determine categories to search
        search_categories = categories or list(self.patterns.keys())

        for category in search_categories:
            if category not in self.patterns:
                continue

            for key, pattern in self.patterns[category].items():
                # Search key
                if query_lower in key.lower():
                    results.append(pattern)
                    continue

                # Search value
                if query_lower in pattern.value.lower():
                    results.append(pattern)
                    continue

                # Search metadata
                for meta_key, meta_value in pattern.metadata.items():
                    if isinstance(meta_value, str) and query_lower in meta_value.lower():
                        results.append(pattern)
                        break

        # Sort by confidence and occurrences
        results.sort(
            key=lambda p: (p.confidence, p.occurrences),
            reverse=True
        )

        return results[:limit]

    async def get_patterns_by_category(
        self,
        category: str,
        min_confidence: float = 0.0
    ) -> List[KnowledgePattern]:
        """
        Get all patterns in a category

        Args:
            category: Category
            min_confidence: Minimum confidence threshold

        Returns:
            List of patterns
        """
        if category not in self.patterns:
            return []

        patterns = list(self.patterns[category].values())

        if min_confidence > 0:
            patterns = [p for p in patterns if p.confidence >= min_confidence]

        return patterns

    async def learn_from_error(
        self,
        error_category: str,
        symptom: str,
        fix: str,
        prevention: str
    ) -> str:
        """
        Learn from an error for future prevention

        Args:
            error_category: Type of error
            symptom: What went wrong
            fix: How it was fixed
            prevention: How to prevent

        Returns:
            Pattern ID
        """
        # Compress into a pattern
        value = f"Prevention: {prevention}. Fix: {fix}"

        return await self.store(
            category="errors",
            key=error_category,
            value=value,
            confidence=0.7,  # Start with higher confidence for errors
            metadata={
                "symptom": symptom,
                "fix": fix,
                "prevention": prevention
            }
        )

    async def update_feedback(
        self,
        category: str,
        key: str,
        was_correct: bool
    ) -> bool:
        """
        Update pattern confidence based on feedback

        Args:
            category: Pattern category
            key: Pattern key
            was_correct: Whether pattern was correct

        Returns:
            True if updated
        """
        pattern = await self.retrieve(category, key)
        if not pattern:
            return False

        pattern.update_confidence(was_correct)
        await self._persist()

        return True

    async def _persist(self) -> None:
        """Persist patterns to storage"""
        data = {}

        for category, patterns in self.patterns.items():
            data[category] = {}
            for key, pattern in patterns.items():
                data[category][key] = {
                    "pattern_id": pattern.pattern_id,
                    "category": pattern.category,
                    "key": pattern.key,
                    "value": pattern.value,
                    "occurrences": pattern.occurrences,
                    "confidence": pattern.confidence,
                    "created_at": pattern.created_at,
                    "updated_at": pattern.updated_at,
                    "metadata": pattern.metadata
                }

        await self.storage.save(data)

    @property
    def total_patterns(self) -> int:
        """Total patterns stored"""
        return self.stats["total_patterns"]

    def get_status(self) -> Dict[str, Any]:
        """Get current status"""
        return {
            "layer": "long_term_memory",
            "total_patterns": self.total_patterns,
            "categories": {
                cat: count
                for cat, count in self.stats["category_counts"].items()
                if count > 0
            },
            "stats": self.stats
        }

    async def get_high_confidence_patterns(
        self,
        category: Optional[str] = None,
        min_confidence: float = 0.7
    ) -> List[KnowledgePattern]:
        """
        Get high-confidence patterns

        Args:
            category: Specific category (default: all)
            min_confidence: Minimum confidence

        Returns:
            High-confidence patterns
        """
        if category:
            patterns_dict = {category: self.patterns.get(category, {})}
        else:
            patterns_dict = self.patterns

        results = []
        for cat, patterns in patterns_dict.items():
            for pattern in patterns.values():
                if pattern.confidence >= min_confidence:
                    results.append(pattern)

        # Sort by confidence
        results.sort(key=lambda p: p.confidence, reverse=True)

        return results


# Demo
async def demo_long_term_memory():
    """Demonstrate long-term memory"""
    print("=" * 60)
    print("Long-Term Memory Demo")
    print("=" * 60)
    print()

    ltm = LongTermMemory(storage_path=".aether/memory/demo_ltm.json")

    # Store some patterns
    print("Storing patterns...")

    await ltm.store(
        category="tech_stack",
        key="framework",
        value="FastAPI for REST APIs",
        confidence=0.9,
        metadata={"reason": "fast, async, modern"}
    )

    await ltm.store(
        category="conventions",
        key="naming",
        value="snake_case for files, PascalCase for classes",
        confidence=0.8
    )

    await ltm.learn_from_error(
        error_category="sql_injection",
        symptom="User input concatenated into SQL query",
        fix="Use parameterized queries",
        prevention="Always use placeholders for user input"
    )

    print("  Stored 3 patterns")
    print()

    # Search
    print("Searching for 'API'...")
    results = await ltm.search("API")
    for r in results:
        print(f"  [{r.category}] {r.key}: {r.value}")
    print()

    # Get status
    print("Status:")
    status = ltm.get_status()
    for key, value in status.items():
        if key != "stats":
            print(f"  {key}: {value}")
    print()

    # Cleanup
    await ltm.storage.clear()


if __name__ == "__main__":
    asyncio.run(demo_long_term_memory())
