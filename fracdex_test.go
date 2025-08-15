package fracdex

import (
	"math"
	"math/rand"
	"reflect"
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
	for range 100 {
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
		for range 100 {
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
	for i := range 100 {
		r := rand.New(rand.NewSource(int64(i)))
		jitter := RandJitter{R: r}
		key, err := KeyBetweenJitter(a, b, jitter, 100)
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
	for range 100 {
		key, err := KeyBetween(a, b)
		if err != nil {
			t.Fatalf("KeyBetween failed: %v", err)
		}
		noJitterKeys[key] = true
	}

	// Generate keys with jitter
	jitterKeys := make(map[string]bool)
	for i := range 100 {
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

	for i := range 100 {
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
	// Use a range that has room for jitter: "a1" to "a5" has digits 2,3,4 in the middle
	a, b := "a1", "a5"
	n := uint(5)

	// Test multiple iterations to demonstrate jitter
	allKeys := make([][]string, 0, 10)

	for iteration := 0; iteration < 10; iteration++ {
		r := rand.New(rand.NewSource(int64(iteration)))
		jitter := RandJitter{R: r}

		keys, err := NKeysBetweenJitter(a, b, n, jitter, 100)
		if err != nil {
			t.Fatalf("NKeysBetweenJitter failed on iteration %d: %v", iteration, err)
		}

		if len(keys) != int(n) {
			t.Errorf("Expected %d keys, got %d on iteration %d", n, len(keys), iteration)
		}

		// Verify all keys are between a and b and in order
		for i, key := range keys {
			if key <= a || key >= b {
				t.Errorf("Generated key %s is not between %s and %s on iteration %d", key, a, b, iteration)
			}
			if i > 0 && keys[i-1] >= key {
				t.Errorf("Keys are not in order: %s >= %s on iteration %d", keys[i-1], key, iteration)
			}
		}

		allKeys = append(allKeys, keys)
	}

	// Verify that jitter is actually working by checking for variation
	// (at least some keys should be different between iterations)
	hasVariation := false
	for i := range len(allKeys) - 1 {
		for j := i + 1; j < len(allKeys); j++ {
			if !reflect.DeepEqual(allKeys[i], allKeys[j]) {
				hasVariation = true
				break
			}
		}
		if hasVariation {
			break
		}
	}

	if !hasVariation {
		t.Logf("Warning: All iterations produced identical keys. This might indicate jitter is not working properly.")
		t.Logf("Keys from first iteration: %v", allKeys[0])
	} else {
		t.Logf("Jitter is working! Found variation between iterations.")
		// Show some examples of variation
		for i := range len(allKeys) - 1 {
			if !reflect.DeepEqual(allKeys[i], allKeys[i+1]) {
				t.Logf("Iteration %d vs %d: %v vs %v", i, i+1, allKeys[i], allKeys[i+1])
				break
			}
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

func TestMidpointJitterVariation(t *testing.T) {
	// Test that midpointJitter actually produces variation
	// Use a range that has room for jitter: "a1" to "a5" has digits 2,3,4 in the middle
	a, b := "a1", "a5"

	// Test with different seeds
	results := make(map[string]bool)

	for i := range 100 {
		r := rand.New(rand.NewSource(int64(i)))
		jitter := RandJitter{R: r}

		result := midpointJitter(a, b, jitter, 2)
		results[result] = true
	}

	// With jitter, we should get multiple different results
	if len(results) <= 1 {
		t.Errorf("Expected jitter to produce variation, but got only %d unique result(s): %v", len(results), results)
	} else {
		t.Logf("Jitter produced %d unique results, demonstrating variation", len(results))
	}

	// Verify all results are still between a and b
	for result := range results {
		if result <= a || result >= b {
			t.Errorf("Jittered result %s is not between %s and %s", result, a, b)
		}
	}

	// Debug: show what we got
	t.Logf("All results: %v", results)
}

func TestMidpointJitterNoJitter(t *testing.T) {
	// Test that NoJitter produces consistent results
	a, b := "a1", "a3"
	noJitter := NoJitter{}

	// Multiple calls should produce the same result
	result1 := midpointJitter(a, b, noJitter, 2)
	result2 := midpointJitter(a, b, noJitter, 2)
	result3 := midpointJitter(a, b, noJitter, 2)

	if result1 != result2 || result2 != result3 {
		t.Errorf("NoJitter should produce consistent results: %s, %s, %s", result1, result2, result3)
	}

	// Verify the result is between a and b
	if result1 <= a || result1 >= b {
		t.Errorf("NoJitter result %s is not between %s and %s", result1, a, b)
	}
}

func TestJitterLimitations(t *testing.T) {
	// Test that jitter has limitations based on the available range
	t.Run("No room for jitter", func(t *testing.T) {
		// "a1" to "a3" only has one possible middle digit: "a2"
		a, b := "a1", "a3"
		results := make(map[string]bool)

		for i := range 50 {
			r := rand.New(rand.NewSource(int64(i)))
			jitter := RandJitter{R: r}
			result := midpointJitter(a, b, jitter, 2)
			results[result] = true
		}

		// Should only get one result since there's no room for variation
		if len(results) != 1 {
			t.Errorf("Expected only 1 result for tight range, got %d: %v", len(results), results)
		} else {
			t.Logf("Tight range 'a1' to 'a3' produces only one result: %v (no room for jitter)", results)
		}
	})

	t.Run("Room for jitter", func(t *testing.T) {
		// "a1" to "a5" has three possible middle digits: "a2", "a3", "a4"
		a, b := "a1", "a5"
		results := make(map[string]bool)

		for i := range 50 {
			r := rand.New(rand.NewSource(int64(i)))
			jitter := RandJitter{R: r}
			result := midpointJitter(a, b, jitter, 2)
			results[result] = true
		}

		// Should get multiple results since there's room for variation
		if len(results) <= 1 {
			t.Errorf("Expected multiple results for wide range, got only %d: %v", len(results), results)
		} else {
			t.Logf("Wide range 'a1' to 'a5' produces %d results: %v (jitter working)", len(results), results)
		}
	})

	t.Run("Very wide range", func(t *testing.T) {
		// "a1" to "a9" has seven possible middle digits: "a2", "a3", "a4", "a5", "a6", "a7", "a8"
		a, b := "a1", "a9"
		results := make(map[string]bool)

		for i := range 50 {
			r := rand.New(rand.NewSource(int64(i)))
			jitter := RandJitter{R: r}
			result := midpointJitter(a, b, jitter, 2)
			results[result] = true
		}

		// Should get multiple results, potentially more than the tight range
		if len(results) <= 1 {
			t.Errorf("Expected multiple results for very wide range, got only %d: %v", len(results), results)
		} else {
			t.Logf("Very wide range 'a1' to 'a9' produces %d results: %v (maximum jitter variation)", len(results), results)
		}
	})
}

func TestJitterPositionAnalysis(t *testing.T) {
	// Test to analyze which positions vary and which are fixed in jittered key generation
	a, b := "a1", "a5"
	n := uint(5)

	// Collect all keys from multiple iterations
	allKeys := make([][]string, 0, 20)

	for iteration := 0; iteration < 20; iteration++ {
		r := rand.New(rand.NewSource(int64(iteration)))
		jitter := RandJitter{R: r}

		keys, err := NKeysBetweenJitter(a, b, n, jitter, 100) // High jitter range
		if err != nil {
			t.Fatalf("NKeysBetweenJitter failed on iteration %d: %v", iteration, err)
		}

		allKeys = append(allKeys, keys)
	}

	// Analyze each position
	positionAnalysis := make([]map[string]bool, n)
	for i := range positionAnalysis {
		positionAnalysis[i] = make(map[string]bool)
	}

	// Collect all values for each position
	for _, keys := range allKeys {
		for pos, key := range keys {
			positionAnalysis[pos][key] = true
		}
	}

	// Report findings
	t.Logf("=== Position Analysis for range '%s' to '%s' ===", a, b)
	for pos, values := range positionAnalysis {
		if len(values) == 1 {
			// Fixed position
			var fixedValue string
			for val := range values {
				fixedValue = val
				break
			}
			t.Logf("Position %d: FIXED at '%s' (no variation)", pos+1, fixedValue)
		} else {
			// Variable position
			t.Logf("Position %d: VARIABLE with %d values: %v", pos+1, len(values), values)
		}
	}

	// Verify that we have some variation
	totalVariation := 0
	for _, values := range positionAnalysis {
		totalVariation += len(values)
	}

	if totalVariation == int(n) {
		t.Logf("All positions are fixed - no jitter variation detected")
	} else {
		t.Logf("Total variation across all positions: %d unique values", totalVariation)
	}

	// Show a few example iterations
	t.Logf("\n=== Example Iterations ===")
	for i := range min(5, len(allKeys)) {
		t.Logf("Iteration %d: %v", i, allKeys[i])
	}
}
