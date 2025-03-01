package consistenthash

import (
	"crypto/md5"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"
)

const (
	// DefaultReplicas is the default number of bucket virtual replicas.
	DefaultReplicas = 16
)

var (
	_ json.Marshaler = (*ConsistentHash)(nil)
	_ fmt.Stringer   = (*ConsistentHash)(nil)
)

// HashFunction is a function that can be used to hash items.
type HashFunction func([]byte) uint64

// StableHash implements the default hash function with
// a stable crc64 table checksum.
func StableHash(data []byte) uint64 {
	hash := md5.Sum(data)
	return binary.BigEndian.Uint64(hash[8:])
}

// Options are the options for the consistent hash type.
type Options struct {
	Replicas     int
	HashFunction HashFunction
}

// Option mutates options.
type Option func(*Options)

// OptReplicas sets the replicas on options.
func OptReplicas(replicas int) Option {
	return func(o *Options) {
		o.Replicas = replicas
	}
}

// OptReplicas sets the replicas on options.
func OptHashFunction(hashFunction HashFunction) Option {
	return func(o *Options) {
		o.HashFunction = hashFunction
	}
}

// New creates a new consistent hash instance.
func New(opts ...Option) *ConsistentHash {
	var options Options
	for _, opt := range opts {
		opt(&options)
	}
	return &ConsistentHash{
		replicas:     options.Replicas,
		hashFunction: options.HashFunction,
	}
}

// ConsistentHash creates hashed assignments for each bucket.
//
// You _must_ use `New` to parameterize ConsistentHash beyond the
// defaults for `replicas` and `hashFunction`.
//
// This is done because these parameters if changed after data has been added
// will lead to inconsistent behavior.
type ConsistentHash struct {
	replicas     int
	hashFunction HashFunction
	mu           sync.RWMutex
	buckets      map[string]struct{}
	hashring     []HashedBucket
}

//
// properties with defaults
//

// Replicas is the default number of bucket virtual replicas.
func (ch *ConsistentHash) Replicas() int {
	if ch.replicas > 0 {
		return ch.replicas
	}
	return DefaultReplicas
}

// HashFunction returns the provided hash function or a default.
func (ch *ConsistentHash) HashFunction() HashFunction {
	if ch.hashFunction != nil {
		return ch.hashFunction
	}
	return StableHash
}

//
// Write methods
//

// AddBuckets adds a list of buckets to the consistent hash, and returns
// a boolean indiciating if _any_ buckets were added.
//
// If any of the new buckets do not exist on the hash ring the
// new bucket will be inserted `ReplicasOrDefault` number
// of times into the internal hashring.
//
// If any of the new buckets already exist on the hash ring
// no action is taken for that bucket (it's effectively skipped).
//
// Calling `AddBuckets` is safe to do concurrently
// and acquires a write lock on the consistent hash reference.
func (ch *ConsistentHash) AddBuckets(newBuckets ...string) (ok bool) {
	ch.mu.Lock()
	defer ch.mu.Unlock()

	if ch.buckets == nil {
		ch.buckets = make(map[string]struct{})
	}
	for _, newBucket := range newBuckets {
		if _, ok := ch.buckets[newBucket]; ok {
			continue
		}
		ok = true
		ch.buckets[newBucket] = struct{}{}
		ch.insertUnsafe(newBucket)
	}
	return
}

// RemoveBucket removes a bucket from the consistent hash, and returns
// a boolean indicating if the provided bucket was found.
//
// If the bucket exists on the hash ring, the bucket and its replicas are removed.
//
// If the bucket does not exist on the ring, no action is taken.
//
// Calling `RemoveBucket` is safe to do concurrently
// and acquires a write lock on the consistent hash reference.
func (ch *ConsistentHash) RemoveBucket(toRemove string) (ok bool) {
	ch.mu.Lock()
	defer ch.mu.Unlock()

	if ch.buckets == nil {
		return
	}
	if _, ok = ch.buckets[toRemove]; !ok {
		return
	}
	// delete the bucket entry
	delete(ch.buckets, toRemove)

	// delete all the replicas from the hash ring for the bucket (there can be many!)
	for x := 0; x < ch.Replicas(); x++ {
		index := ch.search(ch.bucketHashKey(toRemove, x))
		// do slice things to pull it out of the ring.
		ch.hashring = append(ch.hashring[:index], ch.hashring[index+1:]...)
	}
	return
}

//
// Read methods
//

// Buckets returns the buckets.
//
// Calling `Buckets` is safe to do concurrently and acquires
// a read lock on the consistent hash reference.
func (ch *ConsistentHash) Buckets() (buckets []string) {
	ch.mu.RLock()
	defer ch.mu.RUnlock()

	for bucket := range ch.buckets {
		buckets = append(buckets, bucket)
	}
	sort.Strings(buckets)
	return
}

// Assignment returns the bucket assignment for a given item.
//
// Calling `Assignment` is safe to do concurrently and acquires
// a read lock on the consistent hash reference.
func (ch *ConsistentHash) Assignment(item string) (bucket string) {
	ch.mu.RLock()
	defer ch.mu.RUnlock()

	bucket = ch.assignmentUnsafe(item)
	return
}

// IsAssigned returns if a given bucket is assigned a given item.
//
// Calling `IsAssigned` is safe to do concurrently and acquires
// a read lock on the consistent hash reference.
func (ch *ConsistentHash) IsAssigned(bucket, item string) (ok bool) {
	ch.mu.RLock()
	defer ch.mu.RUnlock()

	ok = bucket == ch.assignmentUnsafe(item)
	return
}

// Assignments returns the assignments for a given list of items organized
// by the name of the bucket, and an array of the assigned items.
//
// Calling `Assignments` is safe to do concurrently and acquires
// a read lock on the consistent hash reference.
func (ch *ConsistentHash) Assignments(items ...string) map[string][]string {
	ch.mu.RLock()
	defer ch.mu.RUnlock()

	output := make(map[string][]string)
	for _, item := range items {
		bucket := ch.assignmentUnsafe(item)
		output[bucket] = append(output[bucket], item)
	}
	return output
}

// String returns a string form of the hash for debugging purposes.
//
// Calling `String` is safe to do concurrently and acquires
// a read lock on the consistent hash reference.
func (ch *ConsistentHash) String() string {
	ch.mu.RLock()
	defer ch.mu.RUnlock()

	var output []string
	for _, bucket := range ch.hashring {
		output = append(output, fmt.Sprintf("%d:%s-%02d", bucket.Hashcode, bucket.Bucket, bucket.Replica))
	}
	return strings.Join(output, ", ")
}

// MarshalJSON marshals the consistent hash as json.
//
// The form of the returned json is the underlying []HashedBucket
// and there is no corresponding `UnmarshalJSON` because
// it is uncertain on the other end what the hashfunction is
// because functions can't be json serialized.
//
// You should use MarshalJSON for communicating information
// for debugging purposes only.
//
// Calling `MarshalJSON` is safe to do concurrently and acquires
// a read lock on the consistent hash reference.
func (ch *ConsistentHash) MarshalJSON() ([]byte, error) {
	ch.mu.RLock()
	defer ch.mu.RUnlock()

	return json.Marshal(ch.hashring)
}

//
// internal / unexported helpers
//

// assignmentUnsafe searches for the item's matching bucket based
// on a binary search, and if the index returned is outside the
// ring length, the first index (0) is returned to simulate wrapping around.
func (ch *ConsistentHash) assignmentUnsafe(item string) (bucket string) {
	index := ch.search(item)
	if index >= len(ch.hashring) {
		index = 0
	}
	bucket = ch.hashring[index].Bucket
	return
}

// insert inserts a hashring bucket.
//
// insert uses an insertion sort such that the
// resulting ring will remain sorted after insert.
//
// it will also insert `ReplicasOrDefault` copies of the bucket
// to help distribute items across buckets more evenly.
func (ch *ConsistentHash) insertUnsafe(bucket string) {
	for x := 0; x < ch.Replicas(); x++ {
		ch.insertionSort(HashedBucket{
			Hashcode: ch.hashcode(ch.bucketHashKey(bucket, x)),
			Bucket:   bucket,
			Replica:  x,
		})
	}
}

// insertionSort inserts an bucket into the hashring by binary searching
// for the index which would satisfy the overall "sorted" status of the ring.
func (ch *ConsistentHash) insertionSort(item HashedBucket) {
	destinationIndex := sort.Search(len(ch.hashring), func(index int) bool {
		return ch.hashring[index].Hashcode >= item.Hashcode
	})
	// potentially grow the hashring to accommodate the new entry
	ch.hashring = append(ch.hashring, HashedBucket{})
	// move elements around the new entry index
	copy(ch.hashring[destinationIndex+1:], ch.hashring[destinationIndex:])
	// assign the destination index directly
	ch.hashring[destinationIndex] = item
}

// search does a binary search for the first hashring index whose
// node hashcode is >= the hashcode of a given item.
func (ch *ConsistentHash) search(item string) (index int) {
	index = sort.Search(len(ch.hashring), ch.searchFn(ch.hashcode(item)))
	return
}

// searchFn returns a closure searching for a given hashcode.
func (ch *ConsistentHash) searchFn(hashcode uint64) func(int) bool {
	return func(index int) bool {
		return ch.hashring[index].Hashcode >= hashcode
	}
}

// bucketHashKey formats a hash key for a given bucket virtual replica.
func (ch *ConsistentHash) bucketHashKey(bucket string, index int) string {
	return bucket + "|" + fmt.Sprintf("%02d", index)
}

// hashcode creates a hashcode for a given string
func (ch *ConsistentHash) hashcode(item string) uint64 {
	return ch.HashFunction()([]byte(item))
}

// HashedBucket is a bucket in the hashring
// that holds the hashcode, the bucket name (as Bucket)
// and the virtual replica index.
type HashedBucket struct {
	Hashcode uint64 `json:"hashcode"`
	Bucket   string `json:"bucket"`
	Replica  int    `json:"replica"`
}
