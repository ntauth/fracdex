package fracdex

import (
	"errors"
	"fmt"
	"math"
	"math/rand"
	"strings"
)

const base62Digits = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
const smallestInt = "A00000000000000000000000000"
const zero = "a0"

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

// KeyBetween returns a key that sorts lexicographically between a and b.
// Either a or b can be empty strings. If a is empty it indicates smallest key,
// If b is empty it indicates largest key.
// b must be empty string or > a.
func KeyBetween(a, b string) (string, error) {
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
			return ib + midpoint("", fb), nil
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
			return ia + midpoint(fa, ""), nil
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
		return ia + midpoint(fa, fb), nil
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
	return ia + midpoint(fa, ""), nil
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

// KeyBetweenJitter picks a key strictly between a and b, with randomization.
// This provides collision resistance when multiple writers generate keys
// between the same (a,b) at the same time.
func KeyBetweenJitter(a, b string, j Jitter, jitterRange int) (string, error) {
	return keyBetweenInternal(a, b, j, jitterRange)
}

// `a < b` lexicographically if `b` is non-empty.
// a == "" means first possible string.
// b == "" means last possible string.
func midpoint(a string, b string) string {
	if b != "" {
		// remove longest common prefix.  pad `a` with 0s as we
		// go.  note that we don't need to pad `b`, because it can't
		// end before `a` while traversing the common prefix.
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
				return b[0:i] + midpoint("", b[i:])
			}
			return b[0:i] + midpoint(a[i:], b[i:])
		}
	}

	// first digits (or lack of digit) are different
	digitA := 0
	if a != "" {
		digitA = strings.Index(base62Digits, string(a[0]))
	}
	digitB := len(base62Digits)
	if b != "" {
		digitB = strings.Index(base62Digits, string(b[0]))
	}
	if digitB-digitA > 1 {
		midDigit := int(math.Round(0.5 * float64(digitA+digitB)))
		return string(base62Digits[midDigit])
	}

	// first digits are consecutive
	if len(b) > 1 {
		return b[0:1]
	}

	// `b` is empty or has length 1 (a single digit).
	// the first digit of `a` is the previous digit to `b`,
	// or 9 if `b` is null.
	// given, for example, midpoint('49', '5'), return
	// '4' + midpoint('9', null), which will become
	// '4' + '9' + midpoint('', null), which is '495'
	sa := ""
	if len(a) > 0 {
		sa = a[1:]
	}
	return string(base62Digits[digitA]) + midpoint(sa, "")
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

// helper functions for min/max
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func validateInt(i string) error {
	exp, err := getIntLen(i[0])
	if err != nil {
		return err
	}
	if len(i) != exp {
		return fmt.Errorf("invalid integer part of order key: %s", i)
	}
	return nil
}

func getIntLen(head byte) (int, error) {
	if head >= 'a' && head <= 'z' {
		return int(head - 'a' + 2), nil
	} else if head >= 'A' && head <= 'Z' {
		return int('Z' - head + 2), nil
	} else {
		return 0, fmt.Errorf("invalid order key head: %s", string(head))
	}
}

func getIntPart(key string) (string, error) {
	intPartLen, err := getIntLen(key[0])
	if err != nil {
		return "", err
	}
	if intPartLen > len(key) {
		return "", fmt.Errorf("invalid order key: %s", key)
	}
	return key[0:intPartLen], nil
}

func validateOrderKey(key string) error {
	if key == smallestInt {
		return fmt.Errorf("invalid order key: %s", key)
	}
	// getIntPart will return error if the first character is bad,
	// or the key is too short.  we'd call it to check these things
	// even if we didn't need the result
	i, err := getIntPart(key)
	if err != nil {
		return err
	}
	f := key[len(i):]
	if strings.HasSuffix(f, "0") {
		return fmt.Errorf("invalid order key: %s", key)
	}
	return nil
}

// returns error if x is invalid, or if range is exceeded
func incrementInt(x string) (string, error) {
	err := validateInt(x)
	if err != nil {
		return "", err
	}
	digs := strings.Split(x, "")
	head := digs[0]
	digs = digs[1:]
	carry := true
	for i := len(digs) - 1; carry && i >= 0; i-- {
		d := strings.Index(base62Digits, digs[i]) + 1
		if d == len(base62Digits) {
			digs[i] = "0"
		} else {
			digs[i] = string(base62Digits[d])
			carry = false
		}
	}
	if carry {
		if head == "Z" {
			return "a0", nil
		}
		if head == "z" {
			return "", nil
		}
		h := string(head[0] + 1)
		if h > "a" {
			digs = append(digs, "0")
		} else {
			digs = digs[1:]
		}
		return string(h) + strings.Join(digs, ""), nil
	}
	return head + strings.Join(digs, ""), nil
}

func decrementInt(x string) (string, error) {
	err := validateInt(x)
	if err != nil {
		return "", err
	}
	digs := strings.Split(x, "")
	head := digs[0]
	digs = digs[1:]
	borrow := true
	for i := len(digs) - 1; borrow && i >= 0; i-- {
		d := strings.Index(base62Digits, digs[i]) - 1
		if d == -1 {
			digs[i] = string(base62Digits[len(base62Digits)-1])
		} else {
			digs[i] = string(base62Digits[d])
			borrow = false
		}
	}

	if borrow {
		if head == "a" {
			return "Z" + string(base62Digits[len(base62Digits)-1]), nil
		}
		if head == "A" {
			return "", nil
		}
		h := head[0] - 1
		if h < 'Z' {
			digs = append(digs, string(base62Digits[len(base62Digits)-1]))
		} else {
			digs = digs[1:]
		}
		return string(h) + strings.Join(digs, ""), nil
	}

	return head + strings.Join(digs, ""), nil
}

// Float64Approx converts a key as generated by KeyBetween() to a float64.
// Because the range of keys is far larger than float64 can represent
// accurately, this is necessarily approximate. But for many use cases it should
// be, as they say, close enough for jazz.
func Float64Approx(key string) (float64, error) {
	if key == "" {
		return 0.0, errors.New("invalid order key")
	}

	err := validateOrderKey(key)
	if err != nil {
		return 0.0, err
	}

	ip, err := getIntPart(key)
	if err != nil {
		return 0.0, err
	}

	digs := strings.Split(ip, "")
	head := digs[0]
	digs = digs[1:]
	rv := float64(0)
	for i := 0; i < len(digs); i++ {
		d := digs[len(digs)-i-1]
		p := strings.Index(base62Digits, d)
		if p == -1 {
			return 0.0, fmt.Errorf("invalid order key: %s", key)
		}
		rv += math.Pow(float64(len(base62Digits)), float64(i)) * float64(p)
	}

	fp := key[len(ip):]
	for i, d := range fp {
		p := strings.Index(base62Digits, string(d))
		if p == -1 {
			return 0.0, fmt.Errorf("invalid key: %s", key)
		}
		rv += (float64(p) / math.Pow(float64(len(base62Digits)), float64(i+1)))
	}

	if head < "a" {
		rv *= -1
	}

	return rv, nil
}

// NKeysBetween returns n keys between a and b that sorts lexicographically.
// Either a or b can be empty strings. If a is empty it indicates smallest key,
// If b is empty it indicates largest key.
// b must be empty string or > a.
func NKeysBetween(a, b string, n uint) ([]string, error) {
	if n == 0 {
		return []string{}, nil
	}
	if n == 1 {
		c, err := KeyBetween(a, b)
		if err != nil {
			return nil, err
		}
		return []string{c}, nil
	}
	if b == "" {
		c, err := KeyBetween(a, b)
		if err != nil {
			return nil, err
		}
		result := make([]string, 0, n)
		result = append(result, c)
		for i := 0; i < int(n)-1; i++ {
			c, err = KeyBetween(c, b)
			if err != nil {
				return nil, err
			}
			result = append(result, c)
		}
		return result, nil
	}
	if a == "" {
		c, err := KeyBetween(a, b)
		if err != nil {
			return nil, err
		}
		result := make([]string, 0, n)
		result = append(result, c)
		for i := 0; i < int(n)-1; i++ {
			c, err = KeyBetween(a, c)
			if err != nil {
				return nil, err
			}
			result = append(result, c)
		}
		reverse(result)
		return result, nil
	}
	mid := n / 2
	c, err := KeyBetween(a, b)
	if err != nil {
		return nil, err
	}
	result := make([]string, 0, n)
	{
		r, err := NKeysBetween(a, c, mid)
		if err != nil {
			return nil, err
		}
		result = append(result, r...)
	}
	result = append(result, c)
	{
		r, err := NKeysBetween(c, b, n-mid-1)
		if err != nil {
			return nil, err
		}
		result = append(result, r...)
	}
	return result, nil
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

func reverse(values []string) {
	for i := 0; i < len(values)/2; i++ {
		j := len(values) - i - 1
		values[i], values[j] = values[j], values[i]
	}
}
