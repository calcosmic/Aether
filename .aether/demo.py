"""
Queen Ant Colony System - Demo
"""

import sys
import os
import asyncio

# Add parent directory to path to import from .aether
sys.path.insert(0, os.path.dirname(__file__))

from queen_ant_system import demo_queen_ant_system

if __name__ == "__main__":
    asyncio.run(demo_queen_ant_system())
