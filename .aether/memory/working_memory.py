#!/usr/bin/env python3
"""
Aether Queen Ant Colony - Working Memory Layer

Working Memory: Immediate context, uncompressed, current session.
- 200k token budget for active context
- Shared by all agents
- Fast access, no compression
- Cleared/flushed to short-term at phase boundaries

Based on MEMORY_ARCHITECTURE_RESEARCH.md:
"Working Memory holds immediate context with 200k token budget.
All agents share working memory for fast access to current session data."
"""

import asyncio
import json
import re
from typing import Dict, List, Any, Optional, Set
from dataclasses import dataclass, field, asdict
from datetime import datetime
from collections import OrderedDict
import hashlib


@dataclass
class ContextItem:
    """
    A single item in working memory

    Attributes:
        content: The actual content
        metadata: Associated metadata (source, timestamp, type, etc.)
        token_count: Estimated token count
        created_at: When this item was added
    """
    content: str
    metadata: Dict[str, Any] = field(default_factory=dict)
    token_count: int = 0
    created_at: str = field(default_factory=lambda: datetime.now().isoformat())
    item_id: str = field(default="")

    def __post_init__(self):
        """Generate ID and estimate tokens after creation"""
        if not self.item_id:
            content_hash = hashlib.md5(self.content.encode()).hexdigest()[:8]
            timestamp = datetime.now().strftime("%H%M%S")
            self.item_id = f"ctx_{timestamp}_{content_hash}"

        if self.token_count == 0:
            self.token_count = estimate_tokens(self.content)


@dataclass
class TokenBudget:
    """
    Token budget manager for working memory

    Attributes:
        max_tokens: Maximum budget (default: 200k)
        used_tokens: Currently used tokens
    """
    max_tokens: int = 200_000
    used_tokens: int = 0

    @property
    def available_tokens(self) -> int:
        """Remaining tokens available"""
        return max(0, self.max_tokens - self.used_tokens)

    @property
    def usage_percent(self) -> float:
        """Usage as percentage"""
        if self.max_tokens == 0:
            return 0.0
        return (self.used_tokens / self.max_tokens) * 100

    def can_fit(self, token_count: int) -> bool:
        """Check if tokens fit in budget"""
        return self.used_tokens + token_count <= self.max_tokens

    def add_tokens(self, count: int) -> bool:
        """Add tokens to budget, return True if successful"""
        if self.can_fit(count):
            self.used_tokens += count
            return True
        return False

    def remove_tokens(self, count: int) -> None:
        """Remove tokens from budget"""
        self.used_tokens = max(0, self.used_tokens - count)


def estimate_tokens(text: str) -> int:
    """
    Estimate token count for text

    Uses heuristic: ~4 characters per token for English text.
    This is approximate but fast and good enough for budgeting.

    Args:
        text: Text to estimate

    Returns:
        Estimated token count
    """
    # Count characters
    char_count = len(text)

    # Heuristic: ~4 characters per token
    # Accounts for whitespace, punctuation, etc.
    estimated = char_count // 4

    # Minimum of 1 token
    return max(1, estimated)


class WorkingMemory:
    """
    Working Memory Layer

    Features:
    - 200k token budget
    - Fast access (no compression)
    - Shared by all agents
    - LRU eviction when full
    - Flush to short-term at phase boundaries
    """

    def __init__(self, max_tokens: int = 200_000):
        """
        Initialize working memory

        Args:
            max_tokens: Maximum token budget (default: 200k)
        """
        self.max_tokens = max_tokens
        self.budget = TokenBudget(max_tokens=max_tokens)

        # Use OrderedDict for O(1) access and LRU ordering
        self.items: OrderedDict[str, ContextItem] = OrderedDict()

        # Index for fast search
        self._content_index: Dict[str, Set[str]] = {}
        self._type_index: Dict[str, Set[str]] = {}

        # Statistics
        self.stats = {
            "total_added": 0,
            "total_evicted": 0,
            "total_retrieved": 0,
            "flush_count": 0
        }

    async def add(
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
            item_type: Type/category of item

        Returns:
            Item ID if added, None if rejected (budget exceeded)
        """
        # Create context item
        item = ContextItem(
            content=content,
            metadata=metadata or {},
            token_count=estimate_tokens(content)
        )

        # Add type to metadata
        item.metadata["type"] = item_type

        # Check budget
        if not self.budget.can_fit(item.token_count):
            # Try to make room by evicting oldest
            if not await self._evict_for_tokens(item.token_count):
                return None

        # Add to storage
        self.items[item.item_id] = item

        # Update budget
        self.budget.add_tokens(item.token_count)

        # Update indexes
        self._index_item(item)

        # Update stats
        self.stats["total_added"] += 1

        return item.item_id

    async def get(self, item_id: str) -> Optional[ContextItem]:
        """
        Get item by ID (updates LRU order)

        Args:
            item_id: Item ID to retrieve

        Returns:
            ContextItem if found, None otherwise
        """
        if item_id not in self.items:
            return None

        # Move to end (most recently used)
        item = self.items.pop(item_id)
        self.items[item_id] = item

        # Update stats
        self.stats["total_retrieved"] += 1

        return item

    async def search(
        self,
        query: str,
        limit: int = 10,
        item_type: Optional[str] = None
    ) -> List[ContextItem]:
        """
        Search working memory by content

        Args:
            query: Search query
            limit: Max results
            item_type: Filter by type

        Returns:
            List of matching items (ordered by relevance)
        """
        results = []

        query_lower = query.lower()

        for item_id, item in self.items.items():
            # Type filter
            if item_type and item.metadata.get("type") != item_type:
                continue

            # Content search
            if query_lower in item.content.lower():
                results.append(item)

            # Metadata search
            for key, value in item.metadata.items():
                if isinstance(value, str) and query_lower in value.lower():
                    if item not in results:
                        results.append(item)
                    break

        # Sort by recency (most recent first)
        results.sort(key=lambda x: x.created_at, reverse=True)

        return results[:limit]

    async def remove(self, item_id: str) -> bool:
        """
        Remove item from working memory

        Args:
            item_id: Item ID to remove

        Returns:
            True if removed, False if not found
        """
        if item_id not in self.items:
            return False

        item = self.items[item_id]

        # Remove from indexes
        self._unindex_item(item)

        # Remove from storage
        del self.items[item_id]

        # Update budget
        self.budget.remove_tokens(item.token_count)

        return True

    async def _evict_for_tokens(self, needed_tokens: int) -> bool:
        """
        Evict oldest items to free up tokens

        Args:
            needed_tokens: Tokens needed

        Returns:
            True if enough space freed, False otherwise
        """
        freed = 0

        # Evict oldest items until we have enough space
        while (
            self.items
            and self.budget.available_tokens < needed_tokens
        ):
            # Get oldest item (first in OrderedDict)
            oldest_id = next(iter(self.items))
            oldest_item = self.items[oldest_id]

            # Remove it
            await self.remove(oldest_id)

            freed += oldest_item.token_count
            self.stats["total_evicted"] += 1

            # Safety check
            if freed > self.max_tokens:
                break

        return self.budget.available_tokens >= needed_tokens

    def _index_item(self, item: ContextItem) -> None:
        """Add item to search indexes"""
        # Content keywords (for search)
        words = set(re.findall(r'\w+', item.content.lower()))
        for word in words:
            if word not in self._content_index:
                self._content_index[word] = set()
            self._content_index[word].add(item.item_id)

        # Type index
        item_type = item.metadata.get("type", "general")
        if item_type not in self._type_index:
            self._type_index[item_type] = set()
        self._type_index[item_type].add(item.item_id)

    def _unindex_item(self, item: ContextItem) -> None:
        """Remove item from search indexes"""
        # Content keywords
        words = set(re.findall(r'\w+', item.content.lower()))
        for word in words:
            if word in self._content_index:
                self._content_index[word].discard(item.item_id)

        # Type index
        item_type = item.metadata.get("type", "general")
        if item_type in self._type_index:
            self._type_index[item_type].discard(item.item_id)

    async def flush(self) -> List[ContextItem]:
        """
        Flush all contents for compression to short-term

        Clears working memory and returns all items.

        Returns:
            All items that were in working memory
        """
        items = list(self.items.values())

        # Clear everything
        self.items.clear()
        self._content_index.clear()
        self._type_index.clear()

        # Reset budget
        self.budget = TokenBudget(max_tokens=self.max_tokens)

        # Update stats
        self.stats["flush_count"] += 1

        return items

    @property
    def item_count(self) -> int:
        """Number of items in working memory"""
        return len(self.items)

    @property
    def used_tokens(self) -> int:
        """Tokens currently used"""
        return self.budget.used_tokens

    @property
    def available_tokens(self) -> int:
        """Tokens available"""
        return self.budget.available_tokens

    @property
    def usage_percent(self) -> float:
        """Usage as percentage"""
        return self.budget.usage_percent

    def to_dict(self) -> Dict[str, Any]:
        """Convert to dictionary for serialization"""
        return {
            "max_tokens": self.max_tokens,
            "used_tokens": self.used_tokens,
            "available_tokens": self.available_tokens,
            "usage_percent": self.usage_percent,
            "item_count": self.item_count,
            "stats": self.stats,
            "items": [
                {
                    "item_id": item.item_id,
                    "content": item.content[:200] + "..." if len(item.content) > 200 else item.content,
                    "metadata": item.metadata,
                    "token_count": item.token_count,
                    "created_at": item.created_at
                }
                for item in list(self.items.values())[:10]  # First 10 only
            ]
        }

    def get_status(self) -> Dict[str, Any]:
        """Get current status"""
        return {
            "layer": "working_memory",
            "max_tokens": self.max_tokens,
            "used_tokens": self.used_tokens,
            "available_tokens": self.available_tokens,
            "usage_percent": round(self.usage_percent, 2),
            "item_count": self.item_count,
            "stats": self.stats,
            "items_by_type": {
                item_type: len(item_ids)
                for item_type, item_ids in self._type_index.items()
            }
        }


# Demo function
async def demo_working_memory():
    """Demonstrate working memory functionality"""
    print("=" * 60)
    print("Working Memory Demo")
    print("=" * 60)
    print()

    # Create working memory
    wm = WorkingMemory(max_tokens=1000)  # Small for demo

    print("Initial state:")
    print(f"  Max tokens: {wm.max_tokens}")
    print(f"  Available: {wm.available_tokens}")
    print()

    # Add some items
    print("Adding items...")

    item1 = await wm.add(
        "Build a REST API with user authentication",
        metadata={"source": "user", "priority": "high"},
        item_type="goal"
    )
    print(f"  Added goal: {item1}")

    item2 = await wm.add(
        "Use FastAPI framework with PostgreSQL database",
        metadata={"source": "planner", "phase": "1"},
        item_type="decision"
    )
    print(f"  Added decision: {item2}")

    item3 = await wm.add(
        "Implement JWT tokens for authentication",
        metadata={"source": "executor", "phase": "1"},
        item_type="task"
    )
    print(f"  Added task: {item3}")

    print()
    print("After adding:")
    print(f"  Items: {wm.item_count}")
    print(f"  Used: {wm.used_tokens} tokens ({wm.usage_percent:.1f}%)")
    print()

    # Search
    print("Searching for 'API'...")
    results = await wm.search("API")
    for result in results:
        print(f"  - {result.content[:50]}...")
    print()

    # Get status
    print("Status:")
    status = wm.get_status()
    for key, value in status.items():
        if key != "items_by_type":
            print(f"  {key}: {value}")
    print()

    # Flush
    print("Flushing working memory...")
    flushed = await wm.flush()
    print(f"  Flushed {len(flushed)} items")
    print(f"  Now empty: {wm.item_count} items, {wm.used_tokens} tokens")
    print()


if __name__ == "__main__":
    asyncio.run(demo_working_memory())
