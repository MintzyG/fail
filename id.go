package fail

import (
	"fmt"
	"sort"
	"sync"
)

// FIXME add a level to the ID so we can know easily the severity example LVL_ID_NUM_TYPE (0_USER_0001_S)
// Level shouldn't impact number meaning (0_USER_0001_S can't coexist with 1_USER_0001_S) Only domain and type affect it

// ErrorID represents a trusted, deterministically-generated error identifier
// IDs are generated from names using sorting to ensure compilation-order independence
// Static (S) and Dynamic (D) have separate counters within each domain
type ErrorID struct {
	name     string
	domain   string
	isStatic bool
	number   int  // Derived from sorted position within domain and type
	trusted  bool // Internal flag - only IDs created by ID() are trusted
}

// String returns the formatted error ID (e.g., "AUTH_0042_S")
func (id ErrorID) String() string {
	typeChar := "D"
	if id.isStatic {
		typeChar = "S"
	}
	return fmt.Sprintf("%s_%04d_%s", id.domain, id.number, typeChar)
}

// Name returns the full error name (e.g., "AuthInvalidCredentials")
func (id ErrorID) Name() string {
	return id.name
}

// Domain returns the error domain (e.g., "AUTH", "USER")
func (id ErrorID) Domain() string {
	return id.domain
}

// Number returns the error number (deterministic based on sorted order within domain and type)
func (id ErrorID) Number() int {
	return id.number
}

// IsStatic returns true if this is a static error
func (id ErrorID) IsStatic() bool {
	return id.isStatic
}

// IsTrusted returns true if this ID was created through the proper ID() function
func (id ErrorID) IsTrusted() bool {
	return id.trusted
}

// IDRegistry manages error ID generation and validation
// Numbers are assigned per-domain and per-type (static/dynamic) based on alphabetical ordering
type IDRegistry struct {
	mu            sync.Mutex
	registeredIDs map[string]ErrorID // name -> ErrorID
}

// Global ID registry
var globalIDRegistry = &IDRegistry{
	registeredIDs: make(map[string]ErrorID),
}

// ID creates a new trusted ErrorID with deterministic sequential numbering
// This is the ONLY way to create a trusted ErrorID
//
// Parameters:
//   - name: Full error name (e.g., "AuthInvalidCredentials", "UserNotFound")
//   - domain: Error domain (e.g., "AUTH", "USER") - must be a prefix of the name
//   - static: true for static message, false for dynamic
//
// Panics if:
//   - Name doesn't start with domain (e.g., name="UserNotFound" but domain="AUTH")
//   - Name already exists in registry
//   - Name is too similar to existing name (Levenshtein distance < 3)
//
// Example:
//
//	var AuthInvalidCredentials = fail.ID("AuthInvalidCredentials", "AUTH", true)  // AUTH_0000_S
//	var AuthInvalidPassword    = fail.ID("AuthInvalidPassword", "AUTH", true)     // AUTH_0001_S
//	var AuthCustomError        = fail.ID("AuthCustomError", "AUTH", false)        // AUTH_0000_D
func ID(name, domain string, static bool) ErrorID {
	return globalIDRegistry.ID(name, domain, static)
}

// ID creates a new trusted ErrorID for this registry
func (r *IDRegistry) ID(name, domain string, static bool) ErrorID {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Validation 1: Name must start with domain
	if !hasPrefix(name, domain) {
		panic(fmt.Sprintf(
			"error name '%s' must start with domain '%s' (e.g., %sInvalidCredentials)",
			name, domain, domain,
		))
	}

	// Validation 2: Name must not already exist
	if existing, exists := r.registeredIDs[name]; exists {
		panic(fmt.Sprintf(
			"error name '%s' already registered as %s",
			name, existing.String(),
		))
	}

	// Validation 3: Name must not be too similar to existing names
	for existingName := range r.registeredIDs {
		distance := levenshteinDistance(name, existingName)
		if distance <= 3 {
			panic(fmt.Sprintf(
				"error name '%s' is too similar to existing name '%s' (distance: %d, must be > 3)",
				name, existingName, distance,
			))
		}
	}

	// Create the ID with placeholder number
	id := ErrorID{
		name:     name,
		domain:   domain,
		isStatic: static,
		number:   -1, // temporary, will be reassigned
		trusted:  true,
	}
	r.registeredIDs[name] = id

	// Reassign all numbers in this domain to ensure contiguous ordering
	// Separate assignments for static and dynamic
	r.renumberDomain(domain)

	return r.registeredIDs[name]
}

// renumberDomain reassigns numbers to all IDs in a domain to ensure:
// 1. Static and dynamic have separate sequences
// 2. Numbers are contiguous (0, 1, 2, 3...)
// 3. Numbers are assigned based on alphabetical order of names
func (r *IDRegistry) renumberDomain(domain string) {
	// Separate static and dynamic names
	var staticNames, dynamicNames []string

	for name, id := range r.registeredIDs {
		if id.domain == domain {
			if id.isStatic {
				staticNames = append(staticNames, name)
			} else {
				dynamicNames = append(dynamicNames, name)
			}
		}
	}

	// Sort alphabetically
	sort.Strings(staticNames)
	sort.Strings(dynamicNames)

	// Assign numbers to static IDs
	for i, name := range staticNames {
		id := r.registeredIDs[name]
		id.number = i
		r.registeredIDs[name] = id
	}

	// Assign numbers to dynamic IDs
	for i, name := range dynamicNames {
		id := r.registeredIDs[name]
		id.number = i
		r.registeredIDs[name] = id
	}
}

// Reset clears all registered IDs (useful for testing)
func (r *IDRegistry) Reset() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.registeredIDs = make(map[string]ErrorID)
}

// GetAllIDs returns all registered error IDs sorted by domain, type, then number
func (r *IDRegistry) GetAllIDs() []ErrorID {
	r.mu.Lock()
	defer r.mu.Unlock()

	ids := make([]ErrorID, 0, len(r.registeredIDs))
	for _, id := range r.registeredIDs {
		ids = append(ids, id)
	}

	// Sort by domain, then by type (static first), then by number
	sort.Slice(ids, func(i, j int) bool {
		if ids[i].domain != ids[j].domain {
			return ids[i].domain < ids[j].domain
		}
		if ids[i].isStatic != ids[j].isStatic {
			return ids[i].isStatic // true comes before false
		}
		return ids[i].number < ids[j].number
	})

	return ids
}

// NewIDRegistry creates a new isolated ID registry (useful for testing or multi-app)
func NewIDRegistry() *IDRegistry {
	return &IDRegistry{
		registeredIDs: make(map[string]ErrorID),
	}
}

// hasPrefix checks if name starts with domain (case-insensitive check of first letters)
func hasPrefix(name, domain string) bool {
	if len(name) < len(domain) {
		return false
	}

	// Simple prefix check - compare first len(domain) characters
	namePrefix := name[:len(domain)]

	// Case-insensitive comparison
	return toLower(namePrefix) == toLower(domain)
}

// toLower converts string to lowercase (simple ASCII version)
func toLower(s string) string {
	result := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c = c + ('a' - 'A')
		}
		result[i] = c
	}
	return string(result)
}

// levenshteinDistance calculates the Levenshtein distance between two strings
// This measures how similar two strings are (lower = more similar)
func levenshteinDistance(s1, s2 string) int {
	len1, len2 := len(s1), len(s2)

	// Create matrix
	matrix := make([][]int, len1+1)
	for i := range matrix {
		matrix[i] = make([]int, len2+1)
	}

	// Initialize first row and column
	for i := 0; i <= len1; i++ {
		matrix[i][0] = i
	}
	for j := 0; j <= len2; j++ {
		matrix[0][j] = j
	}

	// Fill matrix
	for i := 1; i <= len1; i++ {
		for j := 1; j <= len2; j++ {
			cost := 0
			if s1[i-1] != s2[j-1] {
				cost = 1
			}

			matrix[i][j] = minInt(
				matrix[i-1][j]+1,      // deletion
				matrix[i][j-1]+1,      // insertion
				matrix[i-1][j-1]+cost, // substitution
			)
		}
	}

	return matrix[len1][len2]
}

func minInt(a, b, c int) int {
	if a < b {
		if a < c {
			return a
		}
		return c
	}
	if b < c {
		return b
	}
	return c
}

// ExportIDList prints all registered error IDs as JSON to stdout
// Format: [{"name": "AuthInvalidCredentials", "domain": "AUTH", "static": true, "id": "AUTH_0000_S"}, ...]
func ExportIDList() {
	globalIDRegistry.ExportIDList()
}

// ExportIDList prints all registered error IDs as JSON for this registry
func (r *IDRegistry) ExportIDList() {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Collect IDs
	type entry struct {
		id ErrorID
	}
	entries := make([]entry, 0, len(r.registeredIDs))
	for _, id := range r.registeredIDs {
		entries = append(entries, entry{id: id})
	}

	// Sort by domain, then by type (static first), then by number
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].id.domain != entries[j].id.domain {
			return entries[i].id.domain < entries[j].id.domain
		}
		if entries[i].id.isStatic != entries[j].id.isStatic {
			return entries[i].id.isStatic // true comes before false
		}
		return entries[i].id.number < entries[j].id.number
	})

	fmt.Println("[")

	first := true
	for _, e := range entries {
		if !first {
			fmt.Println(",")
		}
		first = false

		staticStr := "true"
		if !e.id.isStatic {
			staticStr = "false"
		}

		fmt.Printf("  {\"name\": \"%s\", \"domain\": \"%s\", \"static\": %s, \"id\": \"%s\"}",
			e.id.name, e.id.domain, staticStr, e.id.String())
	}

	fmt.Println("\n]")
}
