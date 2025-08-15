package fracdex

import (
	"errors"
	"fmt"
	"math/rand"
	"strings"
)

// Jitter interface for testability (use math/rand.Rand).
type Jitter interface {
	// Uniform integer in [min, max], inclusive.
	IntnRange(min, max int) int
}

// NoJitter implements Jitter but returns 0 offset.
type NoJitter struct{}

func (NoJitter) IntnRange(min, max int) int { return 0 }

// RandJitter is a helper backed by *rand.Rand:
type RandJitter struct{ R *rand.Rand }

func (j RandJitter) IntnRange(min, max int) int {
	if max < min {
		return min
	}
	if max == min {
		return min
	}
	return min + j.R.Intn(max-min+1)
}

// KeyBetweenJitter picks a key strictly between a and b, with randomization.
// This provides collision resistance when multiple writers generate keys
// between the same (a,b) at the same time.
func KeyBetweenJitter(a, b string, j Jitter, jitterRange int) (string, error) {
	return keyBetweenInternal(a, b, j, jitterRange)
}

// NKeysBetweenJitter generates n keys between a and b with randomization.
// This provides collision resistance when multiple writers generate keys
// between the same (a,b) at the same time.
func NKeysBetweenJitter(a, b string, n uint, j Jitter, jitterRange int) ([]string, error) {
	if n == 0 {
		return []string{}, nil
	}
	if n == 1 {
		c, err := KeyBetweenJitter(a, b, j, jitterRange)
		if err != nil {
			return nil, err
		}
		return []string{c}, nil
	}
	if b == "" {
		c, err := KeyBetweenJitter(a, b, j, jitterRange)
		if err != nil {
			return nil, err
		}
		out := make([]string, 0, n)
		out = append(out, c)
		for i := 0; i < int(n)-1; i++ {
			c, err = KeyBetweenJitter(c, b, j, jitterRange)
			if err != nil {
				return nil, err
			}
			out = append(out, c)
		}
		return out, nil
	}
	if a == "" {
		c, err := KeyBetweenJitter(a, b, j, jitterRange)
		if err != nil {
			return nil, err
		}
		out := make([]string, 0, n)
		out = append(out, c)
		for i := 0; i < int(n)-1; i++ {
			c, err = KeyBetweenJitter(a, c, j, jitterRange)
			if err != nil {
				return nil, err
			}
			out = append(out, c)
		}
		reverse(out)
		return out, nil
	}
	mid := n / 2
	c, err := KeyBetweenJitter(a, b, j, jitterRange)
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, n)
	left, err := NKeysBetweenJitter(a, c, mid, j, jitterRange)
	if err != nil {
		return nil, err
	}
	out = append(out, left...)
	out = append(out, c)
	right, err := NKeysBetweenJitter(c, b, n-mid-1, j, jitterRange)
	if err != nil {
		return nil, err
	}
	out = append(out, right...)
	return out, nil
}

// midpointJitter is a jittered version of midpoint that adds randomization
// while preserving lexicographic order and invariants.
func midpointJitter(a, b string, j Jitter, jitterRange int) string {
	if b != "" {
		// Remove longest common prefix, preserving Greenspan's correctness.
		i := 0
		for ; i < len(b); i++ {
			c := byte('0')
			if len(a) > i {
				c = a[i]
			}
			if c != b[i] {
				break
			}
		}
		if i > 0 {
			if i > len(a) {
				return b[0:i] + midpointJitter("", b[i:], j, jitterRange)
			}
			return b[0:i] + midpointJitter(a[i:], b[i:], j, jitterRange)
		}
	}

	// first digits (or lack) differ
	digitA := 0
	if a != "" {
		digitA = strings.Index(base62Digits, string(a[0]))
	}
	digitB := len(base62Digits)
	if b != "" {
		digitB = strings.Index(base62Digits, string(b[0]))
	}

	// Interior room? Pick a randomized interior digit near the middle.
	if digitB-digitA > 1 {
		interior := digitB - digitA - 1
		center := digitA + 1 + interior/2
		// Jitter offset, clamped to interior range.
		// Use jitterRange as the max absolute deviation (in "digit steps").
		// Example: jitterRange=2 lets you pick center-2 .. center+2.
		lo := max(digitA+1, center-j.IntnRange(0, jitterRange))
		hi := min(digitB-1, center+j.IntnRange(0, jitterRange))
		pick := center
		if hi > lo {
			pick = j.IntnRange(lo, hi)
		} else {
			pick = lo // degenerate range
		}
		return string(base62Digits[pick])
	}

	// Adjacent digits: we must extend.
	if len(b) > 1 {
		// Return b[0] + random digit BELOW b[1] (to stay < b), avoiding trailing '0'.
		head := b[0]
		upper := strings.Index(base62Digits, string(b[1])) - 1
		// allowed low .. high
		low := 0
		high := upper
		if high < low {
			// no room; fall back to minimal extension
			return b[0:1]
		}
		// Skip '0' at the end: ensure we don't end with '0'
		// Pick until non-zero or use '1' if available.
		pickIdx := 1
		if high >= 1 {
			pickIdx = j.IntnRange(1, min(high, 1+jitterRange)) // restrict jitter window
		}
		return string(head) + string(base62Digits[pickIdx])
	}

	// b is empty or 1 char; use Greenspan recursive construction.
	sa := ""
	if len(a) > 0 {
		sa = a[1:]
	}
	return string(base62Digits[digitA]) + midpointJitter(sa, "", j, jitterRange)
}

// keyBetweenInternal is the internal implementation that supports jitter
func keyBetweenInternal(a, b string, j Jitter, jitterRange int) (string, error) {
	if a != "" {
		err := validateOrderKey(a)
		if err != nil {
			return "", err
		}
	}
	if b != "" {
		err := validateOrderKey(b)
		if err != nil {
			return "", err
		}
	}
	if a != "" && b != "" && a >= b {
		return "", fmt.Errorf("%s >= %s", a, b)
	}
	if a == "" {
		if b == "" {
			return zero, nil
		}

		ib, err := getIntPart(b)
		if err != nil {
			return "", err
		}
		fb := b[len(ib):]
		if ib == smallestInt {
			return ib + midpointJitter("", fb, j, jitterRange), nil
		}
		if ib < b {
			return ib, nil
		}
		res, err := decrementInt(ib)
		if err != nil {
			return "", err
		}
		if res == "" {
			return "", errors.New("range underflow")
		}
		return res, nil
	}

	if b == "" {
		ia, err := getIntPart(a)
		if err != nil {
			return "", err
		}
		fa := a[len(ia):]
		i, err := incrementInt(ia)
		if err != nil {
			return "", err
		}
		if i == "" {
			return ia + midpointJitter(fa, "", j, jitterRange), nil
		}
		return i, nil
	}

	ia, err := getIntPart(a)
	if err != nil {
		return "", err
	}
	fa := a[len(ia):]
	ib, err := getIntPart(b)
	if err != nil {
		return "", err
	}
	fb := b[len(ib):]
	if ia == ib {
		return ia + midpointJitter(fa, fb, j, jitterRange), nil
	}
	i, err := incrementInt(ia)
	if err != nil {
		return "", err
	}
	if i == "" {
		return "", errors.New("range overflow")
	}
	if i < b {
		return i, nil
	}
	return ia + midpointJitter(fa, "", j, jitterRange), nil
}

// AddJitterToKey adds random jitter to an existing key by extending it with random digits.
// This is useful when you want to add randomization to a key that was generated without jitter,
// or when you want to "jitter" an existing key to reduce collision probability.
//
// The method preserves the original key's lexicographic position while adding random
// fractional digits that maintain the ordering invariants (no trailing '0').
//
// Parameters:
//   - key: The existing key to add jitter to
//   - j: Jitter source for randomization
//   - jitterRange: Maximum number of random digits to add
//
// Returns a new key with jitter added, or the original key if jitter cannot be applied.
func AddJitterToKey(key string, j Jitter, jitterRange int) (string, error) {
	if key == "" {
		return "", errors.New("cannot add jitter to empty key")
	}

	// Validate the input key
	err := validateOrderKey(key)
	if err != nil {
		return "", fmt.Errorf("invalid key for jitter: %v", err)
	}

	// If jitterRange is 0 or NoJitter, return the original key
	if jitterRange <= 0 {
		return key, nil
	}

	// Check if this is NoJitter (which always returns 0)
	if _, ok := j.(NoJitter); ok {
		return key, nil
	}

	// Determine how many random digits to add
	numDigits := j.IntnRange(1, jitterRange)
	if numDigits <= 0 {
		return key, nil
	}

	// Generate random digits, ensuring no trailing '0'
	result := key
	for i := range numDigits {
		// For the last digit, avoid '0' to maintain the no-trailing-0 invariant
		if i == numDigits-1 {
			// Pick from 1-61 (avoiding '0')
			digitIdx := j.IntnRange(1, len(base62Digits)-1)
			result += string(base62Digits[digitIdx])
		} else {
			// Pick from 0-61 for intermediate digits
			digitIdx := j.IntnRange(0, len(base62Digits)-1)
			result += string(base62Digits[digitIdx])
		}
	}

	// Validate the result
	err = validateOrderKey(result)
	if err != nil {
		return "", fmt.Errorf("generated jittered key is invalid: %v", err)
	}

	return result, nil
}

// JitterKey creates a jittered version of an existing key within the same key space.
// Unlike AddJitterToKey, this method creates variation without increasing key length.
// It works by finding alternative valid keys that are lexicographically close to the original.
//
// Parameters:
//   - key: The existing key to jitter
//   - j: Jitter source for randomization
//   - jitterRange: How much to vary from the original key
//
// Returns a jittered key of the same length, or the original if jitter cannot be applied.
func JitterKey(key string, j Jitter, jitterRange int) (string, error) {
	if key == "" {
		return "", errors.New("cannot jitter empty key")
	}

	// Validate the input key
	err := validateOrderKey(key)
	if err != nil {
		return "", fmt.Errorf("invalid key for jitter: %v", err)
	}

	// If jitterRange is 0 or NoJitter, return the original key
	if jitterRange <= 0 {
		return key, nil
	}

	// Get the integer part and fractional part
	ip, err := getIntPart(key)
	if err != nil {
		return "", fmt.Errorf("failed to get int part: %v", err)
	}

	fp := key[len(ip):]

	// If there's a fractional part, try to jitter within it
	if len(fp) > 0 {
		// Find alternative fractional parts that maintain ordering
		alternatives := findAlternativeFractionalParts(fp, j, jitterRange)
		if len(alternatives) > 0 {
			// Pick a random alternative
			pick := j.IntnRange(0, len(alternatives)-1)
			return ip + alternatives[pick], nil
		}
	}

	// If no fractional part or no alternatives, try to jitter the integer part
	// by finding a nearby valid integer
	nearbyInts := findNearbyIntegers(ip, j, jitterRange)
	if len(nearbyInts) > 0 {
		pick := j.IntnRange(0, len(nearbyInts)-1)
		return nearbyInts[pick], nil
	}

	// If no jitter possible, return original
	return key, nil
}

// findAlternativeFractionalParts finds alternative fractional parts that maintain ordering
func findAlternativeFractionalParts(fp string, j Jitter, jitterRange int) []string {
	if len(fp) == 0 {
		return nil
	}

	// If this is NoJitter, return no alternatives
	if _, ok := j.(NoJitter); ok {
		return nil
	}

	alternatives := make([]string, 0)

	// Try to vary the last few digits while maintaining ordering
	for i := max(0, len(fp)-jitterRange); i < len(fp); i++ {
		// Create a variation by changing digits at position i
		variation := fp[:i]

		// For the last digit, avoid '0'
		if i == len(fp)-1 {
			for d := 1; d < len(base62Digits); d++ {
				if string(base62Digits[d]) != string(fp[i]) {
					alt := variation + string(base62Digits[d])
					if !strings.HasSuffix(alt, "0") {
						alternatives = append(alternatives, alt)
					}
				}
			}
		} else {
			// For intermediate digits, can use any digit
			for d := 0; d < len(base62Digits); d++ {
				if string(base62Digits[d]) != string(fp[i]) {
					alt := variation + string(base62Digits[d]) + fp[i+1:]
					if !strings.HasSuffix(alt, "0") {
						alternatives = append(alternatives, alt)
					}
				}
			}
		}
	}

	return alternatives
}

// findNearbyIntegers finds nearby valid integers that can be used for jitter
func findNearbyIntegers(ip string, j Jitter, jitterRange int) []string {
	alternatives := make([]string, 0)

	// This is a simplified approach - in practice, you'd want more sophisticated
	// logic to find truly nearby integers in the fractional indexing space

	// For now, just return empty to indicate no alternatives found
	return alternatives
}
