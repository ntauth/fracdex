# Fractional Indexing

This is based on [Implementing Fractional Indexing
](https://observablehq.com/@dgreensp/implementing-fractional-indexing) by [David Greenspan
](https://github.com/dgreensp).

Fractional indexing is a technique to create an ordering that can be used for [Realtime Editing of Ordered Sequences](https://www.figma.com/blog/realtime-editing-of-ordered-sequences/).

This implementation includes variable-length integers, and the prepend/append optimization described in David's article.

This should be byte-for-byte compatible with https://github.com/rocicorp/fractional-indexing.

## Features

- **Standard fractional indexing**: Implements Greenspan's algorithm with all optimizations
- **Jitter support**: Adds collision resistance for concurrent writers
- **Lexicographic ordering**: Maintains strict ordering guarantees
- **Invariant preservation**: No trailing '0' in fractional parts

## Example

```go
package main

import (
	"fmt"

	"roci.dev/fracdex"
)

func main() {
	first, _ := fracdex.KeyBetween("", "") // a0
	fmt.Println(first)

	// Insert after 1st
	second, _ := fracdex.KeyBetween(first, "") // "a1"
	fmt.Println(second)

	// Insert after 2nd
	third, _ := fracdex.KeyBetween(second, "") // "a2"
	fmt.Println(third)

	// Insert before 1st
	zeroth, _ := fracdex.KeyBetween("", first) // "Zz"
	fmt.Println(zeroth)

	// Insert in between 2nd and 3rd
	secondAndHalf, _ := fracdex.KeyBetween(second, third) // "a1V"
	fmt.Println(secondAndHalf)
}
```

## Jitter Support

Jitter adds randomization to key generation to reduce collisions when multiple writers generate keys between the same `(a,b)` at the same time. This is particularly useful in distributed systems where concurrent operations can create identical keys.

### Basic Jitter Usage

```go
package main

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/ntauth/fracdex"
)

func main() {
	// Create a random jitter source
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	jitter := fracdex.RandJitter{R: r}
	
	// Generate a key with jitter
	key, err := fracdex.KeyBetweenJitter("a1", "a3", jitter, 2)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Generated key: %s\n", key)
}
```

### Jitter Interfaces

The package provides two jitter implementations:

```go
// NoJitter - deterministic behavior (no randomization)
noJitter := fracdex.NoJitter{}
key, _ := fracdex.KeyBetweenJitter("a1", "a3", noJitter, 2)

// RandJitter - randomized behavior using math/rand
r := rand.New(rand.NewSource(42))
randJitter := fracdex.RandJitter{R: r}
key, _ := fracdex.KeyBetweenJitter("a1", "a3", randJitter, 2)
```

### Jitter Range

The `jitterRange` parameter controls how much randomization is applied:

- **jitterRange = 0**: No randomization (equivalent to `NoJitter`)
- **jitterRange = 1**: Small randomization (±1 digit step)
- **jitterRange = 2**: Medium randomization (±2 digit steps)
- **jitterRange = 3**: Larger randomization (±3 digit steps)

**Important**: Jitter can only create variation when there's room in the lexicographic space between `a` and `b`. For example:
- `"a1"` to `"a3"` has only one possible middle digit (`"a2"`), so jitter cannot create variation
- `"a1"` to `"a5"` has three possible middle digits (`"a2"`, `"a3"`, `"a4"`), so jitter can create variation
- `"a1"` to `"a9"` has seven possible middle digits, allowing maximum jitter variation

### Multiple Keys with Jitter

```go
// Generate 5 keys between a1 and a3 with jitter
keys, err := fracdex.NKeysBetweenJitter("a1", "a3", 5, randJitter, 2)
if err != nil {
	panic(err)
}

for i, key := range keys {
	fmt.Printf("Key %d: %s\n", i+1, key)
}
```

### When to Use Jitter

**Use jitter when:**
- Multiple clients may generate keys simultaneously
- You need collision resistance
- You're building a distributed system
- You want to reduce the need for retry logic

**Don't use jitter when:**
- You need deterministic key generation
- You're in a single-threaded environment
- You're doing testing or debugging

### Collision Handling

Even with jitter, collisions are still possible. Your server should:

1. **Maintain a unique index** on `(scope, key)`
2. **Retry on collision**: Re-read neighbors and generate a new key
3. **Use exponential backoff** for retries

```go
// Example collision handling
for attempts := 0; attempts < maxRetries; attempts++ {
	key, err := fracdex.KeyBetweenJitter(a, b, jitter, 2)
	if err != nil {
		continue
	}
	
	// Try to insert with this key
	err = insertItem(key, item)
	if err == nil {
		break // Success
	}
	
	if isCollisionError(err) {
		// Re-read neighbors and retry
		a, b = reReadNeighbors()
		continue
	}
	
	// Other error, don't retry
	break
}
```

## API Reference

### Core Functions

- `KeyBetween(a, b string) (string, error)` - Generate key between a and b
- `NKeysBetween(a, b string, n uint) ([]string, error)` - Generate n keys between a and b
- `Float64Approx(key string) (float64, error)` - Convert key to approximate float64

### Jitter Functions

- `KeyBetweenJitter(a, b string, j Jitter, jitterRange int) (string, error)` - Generate key with jitter
- `NKeysBetweenJitter(a, b string, n uint, j Jitter, jitterRange int) ([]string, error)` - Generate n keys with jitter

### Jitter Interface

```go
type Jitter interface {
	IntnRange(min, max int) int
}
```

### Implementations

- `NoJitter{}` - Always returns 0 (deterministic)
- `RandJitter{R: *rand.Rand}` - Uses math/rand for randomization

## Performance

Benchmarks on Apple M3 Pro:

```
BenchmarkKeyBetweenJitter-11            10219615               111.9 ns/op
BenchmarkNKeysBetweenJitter-11            579646              1938 ns/op
```

Jitter adds minimal overhead while providing significant collision resistance benefits.

## Testing

Run the full test suite:

```bash
go test -v
```

Run benchmarks:

```bash
go test -bench=.
```
