# Coding Standards Discipline

## Purpose

Universal code quality rules that apply to all worker output. Code is read more than written - optimize for readability and maintainability.

## Core Principles

### 1. Readability First
- Clear variable and function names
- Self-documenting code preferred over comments
- Consistent formatting

### 2. KISS (Keep It Simple, Stupid)
- Simplest solution that works
- No premature optimization
- Easy to understand > clever code

### 3. DRY (Don't Repeat Yourself)
- Extract common logic into functions
- Create reusable components
- No copy-paste programming

### 4. YAGNI (You Aren't Gonna Need It)
- Don't build features before needed
- Add complexity only when required
- Start simple, refactor when needed

## Naming Conventions

### Variables
```
✅ GOOD: Descriptive names
const marketSearchQuery = 'election'
const isUserAuthenticated = true

❌ BAD: Unclear names
const q = 'election'
const flag = true
```

### Functions
```
✅ GOOD: Verb-noun pattern
function fetchMarketData(marketId: string) { }
function calculateSimilarity(a: number[], b: number[]) { }
function isValidEmail(email: string): boolean { }

❌ BAD: Unclear or noun-only
function market(id: string) { }
function similarity(a, b) { }
```

## Critical Patterns

### Immutability (ALWAYS)
```
✅ ALWAYS use spread operator
const updatedUser = { ...user, name: 'New Name' }
const updatedArray = [...items, newItem]

❌ NEVER mutate directly
user.name = 'New Name'  // BAD
items.push(newItem)     // BAD
```

### Error Handling
```
✅ GOOD: Comprehensive
async function fetchData(url: string) {
  try {
    const response = await fetch(url)
    if (!response.ok) {
      throw new Error(`HTTP ${response.status}`)
    }
    return await response.json()
  } catch (error) {
    console.error('Fetch failed:', error)
    throw new Error('Failed to fetch data')
  }
}

❌ BAD: No error handling
async function fetchData(url) {
  const response = await fetch(url)
  return response.json()
}
```

### Async/Await
```
✅ GOOD: Parallel when possible
const [users, markets] = await Promise.all([
  fetchUsers(),
  fetchMarkets()
])

❌ BAD: Sequential when unnecessary
const users = await fetchUsers()
const markets = await fetchMarkets()
```

### Type Safety
```
✅ GOOD: Proper types
function getMarket(id: string): Promise<Market> { }

❌ BAD: Using 'any'
function getMarket(id: any): Promise<any> { }
```

## Code Smells to Avoid

### Long Functions
```
❌ BAD: Function > 50 lines
✅ GOOD: Split into smaller functions
```

### Deep Nesting
```
❌ BAD: 5+ levels of nesting
✅ GOOD: Early returns
if (!user) return
if (!user.isAdmin) return
// Do something
```

### Magic Numbers
```
❌ BAD: Unexplained numbers
if (retryCount > 3) { }

✅ GOOD: Named constants
const MAX_RETRIES = 3
if (retryCount > MAX_RETRIES) { }
```

## Comments

### When to Comment
```
✅ GOOD: Explain WHY, not WHAT
// Use exponential backoff to avoid overwhelming the API
const delay = Math.min(1000 * Math.pow(2, retryCount), 30000)

❌ BAD: Stating the obvious
// Increment counter by 1
count++
```

## Testing Standards

### AAA Pattern
```
test('calculates similarity correctly', () => {
  // Arrange
  const vector1 = [1, 0, 0]
  const vector2 = [0, 1, 0]

  // Act
  const similarity = calculateCosineSimilarity(vector1, vector2)

  // Assert
  expect(similarity).toBe(0)
})
```

### Test Naming
```
✅ GOOD: Descriptive
test('returns empty array when no markets match query', () => { })
test('throws error when API key is missing', () => { })

❌ BAD: Vague
test('works', () => { })
test('test search', () => { })
```

## Quick Checklist

Before completing any code:
- [ ] Names are clear and descriptive
- [ ] No deep nesting (use early returns)
- [ ] No magic numbers (use constants)
- [ ] Error handling is comprehensive
- [ ] Async operations parallelize where possible
- [ ] No `any` types
- [ ] Functions are < 50 lines
- [ ] Comments explain WHY, not WHAT

## Why This Matters

- Clear code enables rapid development
- Maintainable code enables confident refactoring
- Quality code reduces debugging time
- Standards enable team collaboration
