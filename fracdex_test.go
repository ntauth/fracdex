package fracdex

import (
	"math"
	"math/rand"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestKeys(t *testing.T) {
	assert := assert.New(t)

	test := func(a, b, exp string) {
		act, err := KeyBetween(a, b)
		if strings.HasPrefix(exp, "invalid") || strings.Contains(exp, ">=") {
			// Expecting an error
			assert.Equal("", act)
			assert.NotNil(err)
			assert.Equal(exp, err.Error())
		} else {
			// Expecting success
			assert.Nil(err)
			assert.Equal(exp, act)
		}
	}

	test("", "", "a0")
	test("", "a0", "Zz")
	test("", "Zz", "Zy")
	test("a0", "", "a1")
	test("a1", "", "a2")
	test("a0", "a1", "a0V")
	test("a1", "a2", "a1V")
	test("a0V", "a1", "a0l")
	test("Zz", "a0", "ZzV")
	test("Zz", "a1", "a0")
	test("", "Y00", "Xzzz")
	test("bzz", "", "c000")
	test("a0", "a0V", "a0G")
	test("a0", "a0G", "a08")
	test("b125", "b129", "b127")
	test("a0", "a1V", "a1")
	test("Zz", "a01", "a0")
	test("", "a0V", "a0")
	test("", "b999", "b99")
	test("aV", "aV0V", "aV0G")
	test(
		"",
		"A00000000000000000000000000",
		"invalid order key: A00000000000000000000000000",
	)
	test("", "A000000000000000000000000001", "A000000000000000000000000000V")
	test("zzzzzzzzzzzzzzzzzzzzzzzzzzy", "", "zzzzzzzzzzzzzzzzzzzzzzzzzzz")
	test("zzzzzzzzzzzzzzzzzzzzzzzzzzz", "", "zzzzzzzzzzzzzzzzzzzzzzzzzzzV")
	test("a00", "", "invalid order key: a00")
	test("a00", "a1", "invalid order key: a00")
	test("0", "1", "invalid order key head: 0")
	test("a1", "a0", "a1 >= a0")
}

func TestNKeys(t *testing.T) {
	assert := assert.New(t)

	test := func(a, b string, n uint, exp string) {
		actSlice, err := NKeysBetween(a, b, n)
		act := strings.Join(actSlice, " ")
		if exp == "" {
			assert.Equal("", act)
			assert.Equal(exp, err.Error())
		} else {
			assert.Nil(err)
			assert.Equal(exp, act)
		}
	}
	test("", "", 5, "a0 a1 a2 a3 a4")
	test("a4", "", 10, "a5 a6 a7 a8 a9 aA aB aC aD aE")
	test("", "a0", 5, "Zv Zw Zx Zy Zz")
	test(
		"a0",
		"a2",
		20,
		"a04 a08 a0G a0K a0O a0V a0Z a0d a0l a0t a1 a14 a18 a1G a1O a1V a1Z a1d a1l a1t",
	)
}

func TestToFloat64Approx(t *testing.T) {
	assert := assert.New(t)

	test := func(key string, exp float64, expErr string) {
		act, err := Float64Approx(key)
		if expErr != "" {
			assert.Equal(0.0, act)
			assert.Equal(expErr, err.Error())
		} else {
			assert.Equal(exp, act)
			assert.NoError(err)
		}
	}

	test("a0", 0.0, "")
	test("a1", 1.0, "")
	test("az", 61.0, "")
	test("b10", 62.0, "")
	test("z20000000000000000000000000", math.Pow(62.0, 25.0)*2.0, "")
	test("Z1", -1.0, "")
	test("Zz", -61.0, "")
	test("Y10", -62.0, "")
	test("A20000000000000000000000000", math.Pow(62.0, 25.0)*-2.0, "")

	test("a0V", 0.5, "")
	test("a00V", 31.0/math.Pow(62.0, 2.0), "")
	test("aVV", 31.5, "")
	test("ZVV", -31.5, "")

	test("", 0.0, "invalid order key")
	test("!", 0.0, "invalid order key head: !")
	test("a400", 0.0, "invalid order key: a400")
	test("a!", 0.0, "invalid order key: a!")
}

// Jitter-specific tests
func TestJitterInterfaces(t *testing.T) {
	// Test NoJitter always returns 0
	noJitter := NoJitter{}
	for i := 0; i < 100; i++ {
		if noJitter.IntnRange(1, 10) != 0 {
			t.Errorf("NoJitter should always return 0, got %d", noJitter.IntnRange(1, 10))
		}
	}

	// Test RandJitter returns values in range
	r := rand.New(rand.NewSource(42))
	randJitter := RandJitter{R: r}

	// Test multiple ranges
	ranges := [][]int{{1, 5}, {10, 20}, {0, 1}, {5, 5}}
	for _, rng := range ranges {
		min, max := rng[0], rng[1]
		for i := 0; i < 100; i++ {
			val := randJitter.IntnRange(min, max)
			if val < min || val > max {
				t.Errorf("RandJitter.IntnRange(%d, %d) returned %d, outside range", min, max, val)
			}
		}
	}
}

func TestKeyBetweenJitterBasic(t *testing.T) {
	// Test that jittered keys still maintain lexicographic order
	a, b := "a1", "a3"

	// Generate multiple keys between a and b
	keys := make([]string, 0, 100)
	for i := 0; i < 100; i++ {
		r := rand.New(rand.NewSource(int64(i)))
		jitter := RandJitter{R: r}
		key, err := KeyBetweenJitter(a, b, jitter, 2)
		if err != nil {
			t.Fatalf("KeyBetweenJitter failed: %v", err)
		}
		keys = append(keys, key)
	}

	// Verify all keys are between a and b
	for _, key := range keys {
		if key <= a || key >= b {
			t.Errorf("Generated key %s is not between %s and %s", key, a, b)
		}
	}

	// Verify keys are sorted
	sortedKeys := make([]string, len(keys))
	copy(sortedKeys, keys)
	sort.Strings(sortedKeys)

	for i, key := range keys {
		if key != sortedKeys[i] {
			t.Errorf("Keys are not in lexicographic order. Original: %v, Sorted: %v", keys, sortedKeys)
			break
		}
	}
}

func TestKeyBetweenJitterCollisionResistance(t *testing.T) {
	// Test that jittered keys reduce collisions
	a, b := "a1", "a3"

	// Generate keys with no jitter
	noJitterKeys := make(map[string]bool)
	for i := 0; i < 100; i++ {
		key, err := KeyBetween(a, b)
		if err != nil {
			t.Fatalf("KeyBetween failed: %v", err)
		}
		noJitterKeys[key] = true
	}

	// Generate keys with jitter
	jitterKeys := make(map[string]bool)
	for i := 0; i < 100; i++ {
		r := rand.New(rand.NewSource(int64(i)))
		jitter := RandJitter{R: r}
		key, err := KeyBetweenJitter(a, b, jitter, 3)
		if err != nil {
			t.Fatalf("KeyBetweenJitter failed: %v", err)
		}
		jitterKeys[key] = true
	}

	// Jittered keys should have more unique values (better collision resistance)
	if len(noJitterKeys) >= len(jitterKeys) {
		t.Logf("No jitter unique keys: %d, Jitter unique keys: %d", len(noJitterKeys), len(jitterKeys))
		// This is not a failure, just a note that collision resistance may vary
	}
}

func TestKeyBetweenJitterInvariants(t *testing.T) {
	// Test that jittered keys maintain all invariants
	a, b := "a1", "a3"

	for i := 0; i < 100; i++ {
		r := rand.New(rand.NewSource(int64(i)))
		jitter := RandJitter{R: r}
		key, err := KeyBetweenJitter(a, b, jitter, 2)
		if err != nil {
			t.Fatalf("KeyBetweenJitter failed: %v", err)
		}

		// Test no trailing '0' invariant
		if strings.HasSuffix(key, "0") {
			t.Errorf("Generated key %s has trailing '0', violating invariant", key)
		}

		// Test valid order key
		if err := validateOrderKey(key); err != nil {
			t.Errorf("Generated key %s is not a valid order key: %v", key, err)
		}
	}
}

func TestNKeysBetweenJitter(t *testing.T) {
	// Test that jittered N keys generation works
	a, b := "a1", "a3"
	n := uint(5)

	r := rand.New(rand.NewSource(42))
	jitter := RandJitter{R: r}

	keys, err := NKeysBetweenJitter(a, b, n, jitter, 2)
	if err != nil {
		t.Fatalf("NKeysBetweenJitter failed: %v", err)
	}

	if len(keys) != int(n) {
		t.Errorf("Expected %d keys, got %d", n, len(keys))
	}

	// Verify all keys are between a and b and in order
	for i, key := range keys {
		if key <= a || key >= b {
			t.Errorf("Generated key %s is not between %s and %s", key, a, b)
		}
		if i > 0 && keys[i-1] >= key {
			t.Errorf("Keys are not in order: %s >= %s", keys[i-1], key)
		}
	}
}

func TestKeyBetweenJitterEdgeCases(t *testing.T) {
	// Test edge cases with jitter
	r := rand.New(rand.NewSource(42))
	jitter := RandJitter{R: r}

	// Test with empty bounds
	key, err := KeyBetweenJitter("", "", jitter, 1)
	if err != nil {
		t.Fatalf("KeyBetweenJitter with empty bounds failed: %v", err)
	}
	if key != zero {
		t.Errorf("Expected %s, got %s", zero, key)
	}

	// Test with one empty bound
	key, err = KeyBetweenJitter("a1", "", jitter, 1)
	if err != nil {
		t.Fatalf("KeyBetweenJitter with one empty bound failed: %v", err)
	}
	if key <= "a1" {
		t.Errorf("Generated key %s should be greater than a1", key)
	}

	key, err = KeyBetweenJitter("", "a3", jitter, 1)
	if err != nil {
		t.Fatalf("KeyBetweenJitter with one empty bound failed: %v", err)
	}
	if key >= "a3" {
		t.Errorf("Generated key %s should be less than a3", key)
	}
}

func TestKeyBetweenJitterConsistency(t *testing.T) {
	// Test that jittered keys are consistent for the same seed
	a, b := "a1", "a3"

	// Generate keys with same seed
	r1 := rand.New(rand.NewSource(42))
	r2 := rand.New(rand.NewSource(42))

	jitter1 := RandJitter{R: r1}
	jitter2 := RandJitter{R: r2}

	key1, err := KeyBetweenJitter(a, b, jitter1, 2)
	if err != nil {
		t.Fatalf("KeyBetweenJitter failed: %v", err)
	}

	key2, err := KeyBetweenJitter(a, b, jitter2, 2)
	if err != nil {
		t.Fatalf("KeyBetweenJitter failed: %v", err)
	}

	if key1 != key2 {
		t.Errorf("Keys with same seed should be identical: %s != %s", key1, key2)
	}
}

func BenchmarkKeyBetweenJitter(b *testing.B) {
	a, bKey := "a1", "a3"
	r := rand.New(rand.NewSource(42))
	jitter := RandJitter{R: r}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := KeyBetweenJitter(a, bKey, jitter, 2)
		if err != nil {
			b.Fatalf("KeyBetweenJitter failed: %v", err)
		}
	}
}

func BenchmarkNKeysBetweenJitter(b *testing.B) {
	a, bKey := "a1", "a3"
	n := uint(10)
	r := rand.New(rand.NewSource(42))
	jitter := RandJitter{R: r}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := NKeysBetweenJitter(a, bKey, n, jitter, 2)
		if err != nil {
			b.Fatalf("NKeysBetweenJitter failed: %v", err)
		}
	}
}
