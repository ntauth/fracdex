package fracdex

import "fmt"

// Bucket represents a logical grouping or namespace for lexoranks.
// It's implemented as a uint8, allowing for up to 256 different buckets.
// Buckets are useful for organizing related items or implementing
// multi-tenant systems where different tenants need separate ordering.
type Bucket uint8

// Lexorank represents a lexicographically sortable rank within a bucket.
// It combines a bucket identifier with a fractional index key to create
// a unique, sortable identifier that can be used for ordering items
// in a distributed system.
//
// The key component uses the fractional indexing algorithm to ensure
// that new items can always be inserted between any two existing items
// without requiring reordering of the entire collection.
type Lexorank struct {
	bucket Bucket // The bucket/namespace this rank belongs to
	key    string // The fractional index key for ordering within the bucket
}

// String returns a string representation of the Lexorank in the format "bucket|key".
// This format makes it easy to parse and display lexoranks while maintaining
// the relationship between bucket and key components.
//
// Example: "1|a1" represents bucket 1 with key "a1"
func (rk Lexorank) String() string {
	return fmt.Sprintf("%d|%s", rk.bucket, rk.key)
}

// NewLexorank creates a new Lexorank with the specified bucket and key.
// This is the primary constructor for creating new lexorank instances.
//
// Parameters:
//   - bucket: The bucket/namespace identifier
//   - key: The fractional index key for ordering
//
// Returns a new Lexorank instance.
func NewLexorank(bucket Bucket, key string) Lexorank {
	return Lexorank{bucket: bucket, key: key}
}

// Bucket returns the bucket identifier for this lexorank.
// This allows external code to determine which namespace
// a particular lexorank belongs to.
func (rk Lexorank) Bucket() Bucket {
	return rk.bucket
}

// Key returns the fractional index key for this lexorank.
// The key determines the ordering within the bucket and follows
// the fractional indexing algorithm for maintaining sort order
// while allowing insertions between any two existing items.
func (rk Lexorank) Key() string {
	return rk.key
}
