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
