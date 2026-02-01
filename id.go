package fail

import (
	"encoding/json"
	"fmt"
	"log"
	"sort"
	"sync"
)

// reservedDomain is the domain reserved for internal error IDs.
// Users cannot register IDs on this domain through the public ID() function.
const reservedDomain = "FAIL"

// ErrorID represents a trusted, deterministically-generated error identifier
// IDs are generated from names using explicit numbering for stability across versions
// Static (S) and Dynamic (D) have separate counters within each domain
// Format: LEVEL_DOMAIN_NUM_TYPE (e.g., "0_AUTH_0042_S")
// Level indicates severity but does not affect uniqueness
type ErrorID struct {
	name     string
	domain   string
	level    int // Severity level
	isStatic bool
	number   int  // Explicitly assigned, stable across versions
	trusted  bool // Internal flag - only IDs created by ID() are trusted
}

// String returns the formatted error ID (e.g., "0_AUTH_0042_S")
func (id ErrorID) String() string {
	typeChar := "D"
	if id.isStatic {
		typeChar = "S"
	}
	return fmt.Sprintf("%d_%s_%04d_%s", id.level, id.domain, id.number, typeChar)
}

// Name returns the full error name (e.g., "AuthInvalidCredentials")
func (id ErrorID) Name() string {
	return id.name
}

// Domain returns the error domain (e.g., "AUTH", "USER")
func (id ErrorID) Domain() string {
	return id.domain
}

// Level returns the severity level
func (id ErrorID) Level() int {
	return id.level
}

// Number returns the error number (explicitly assigned for stability)
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

// OverrideAllowIDRuntimePanics sets global id registry override
func OverrideAllowIDRuntimePanics(allow bool) {
	globalIDRegistry.OverrideAllowRuntimePanics(allow)
}

// OverrideAllowIDRuntimeRegistrationForTestingOnly sets global id registry override for tests
func OverrideAllowIDRuntimeRegistrationForTestingOnly(allow bool) {
	globalIDRegistry.mu.Lock()
	defer globalIDRegistry.mu.Unlock()
	globalIDRegistry.allowRuntimeRegistration = allow
}

// OverrideAllowRuntimePanics sets per-registry override
func (r *IDRegistry) OverrideAllowRuntimePanics(allow bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.allowRuntimePanics = &allow
}

// IDRegistry manages error ID generation and validation
// Numbers are explicitly assigned per domain and type (static/dynamic)
type IDRegistry struct {
	mu            sync.Mutex
	registeredIDs map[string]ErrorID // name -> ErrorID
	numberIndex   map[string]ErrorID // "domain:static:number" -> ErrorID (collision detection)

	allowRuntimePanics       *bool
	allowRuntimeRegistration bool
}

// Global ID registry
var globalIDRegistry = &IDRegistry{
	registeredIDs: make(map[string]ErrorID),
	numberIndex:   make(map[string]ErrorID),
}

// ID creates a new trusted ErrorID with explicit numbering
// This is the ONLY way to create a trusted ErrorID
//
// WARNING: Must be called at package level (var declaration).
// Calling inside func init() or runtime functions causes unstable numbering.
// Use go:generate or static analysis to verify.
//
// Parameters:
//   - name: Full error name (e.g., "AuthInvalidCredentials", "UserNotFound")
//   - domain: Error domain (e.g., "AUTH", "USER") - must be a prefix of the name
//   - static: true for static message, false for dynamic
//   - level: severity level (0-9 recommended)
//   - number: explicit number for this ID (must be unique within domain+type)
//
// Panics if:
//   - Name doesn't start with domain
//   - Name already exists in registry
//   - Name is too similar to existing name (Levenshtein distance < 3)
//   - Domain is "FAIL" (reserved for internal errors)
//   - Number already used in this domain+type combination
//
// Example:
//
//	var AuthInvalidCredentials = fail.ID(0, "AUTH", 0, true, "AuthInvalidCredentials")   // 0_AUTH_0000_S
//	var AuthInvalidPassword    = fail.ID(0, "AUTH", 1, true, "AuthInvalidPassword")      // 0_AUTH_0001_S
//	var AuthCustomError        = fail.ID(0, "AUTH", 0, false, "AuthCustomError")         // 1_AUTH_0000_D
//	var AuthAnotherError       = fail.ID(0, "AUTH", 1, false, "AuthAnotherError")        // 0_AUTH_0001_D
//	// v0.0.2 - add new ID, gaps are fine
//	var AuthNewFeature         = fail.ID("AuthNewFeature", "AUTH", true, 0, 100)         // 0_AUTH_0100_S
func ID(level int, domain string, number int, static bool, name string) ErrorID {
	return globalIDRegistry.ID(name, domain, static, level, number)
}

var RuntimeIDInvalid = internalID(9, 16, true, "FailRuntimeIDInvalid")

// ID creates a new trusted ErrorID for this registry
func (r *IDRegistry) ID(name, domain string, static bool, level, number int) ErrorID {
	// Critical safety check: ID() must only be called during init/var time
	r.mu.Lock()
	allowRuntime := r.allowRuntimeRegistration
	r.mu.Unlock()

	if !calledBeforeMain() && !allowRuntime {
		r.mu.Lock()
		// Force log regardless of allowInternalLogs - this is critical misuse
		callerFile, callerLine := getCallerInfo(2)
		log.Printf("[FAIL CRITICAL] ID() called at runtime by %s:%d - name='%s' domain='%s'. "+
			"All error IDs must be defined at package initialization time (var level or init()). "+
			"Returning invalid ID.", callerFile, callerLine, name, domain)

		// Check if we should panic
		shouldPanic := false
		if r.allowRuntimePanics != nil && *r.allowRuntimePanics {
			shouldPanic = true
		} else if allowRuntimePanics {
			shouldPanic = true
		}
		r.mu.Unlock()

		if shouldPanic {
			panic(fmt.Sprintf(
				"[FAIL CRITICAL] ID() called at runtime by %s:%d - name='%s' domain='%s'. "+
					"All error IDs must be defined at package initialization time",
				callerFile, callerLine, name, domain,
			))
		}

		return RuntimeIDInvalid
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	// Validation 0: Domain cannot be reserved
	if domain == reservedDomain {
		panic(fmt.Sprintf(
			"domain '%s' is reserved for internal errors and cannot be used",
			reservedDomain,
		))
	}

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

	// Validation 4: Number must be unique within domain+type
	numberKey := fmt.Sprintf("%s:%v:%d", domain, static, number)
	if existing, exists := r.numberIndex[numberKey]; exists {
		panic(fmt.Sprintf(
			"number %d already used in domain '%s' (static=%v) by '%s' (%s)",
			number, domain, static, existing.name, existing.String(),
		))
	}

	// Create the ID with explicit number
	id := ErrorID{
		name:     name,
		domain:   domain,
		level:    level,
		isStatic: static,
		number:   number,
		trusted:  true,
	}
	r.registeredIDs[name] = id
	r.numberIndex[numberKey] = id

	return id
}

// internalID creates a trusted ErrorID in the reserved "FAIL" domain.
// This is for internal library use only and bypasses the reserved domain restriction.
// It enforces that the name must start with "FAIL".
//
// Example:
//
//	var FailRegistryCorrupted = internalID(9, 0, true, "FailRegistryCorrupted")  // 9_FAIL_0000_S
func internalID(level, number int, static bool, name string) ErrorID {
	return globalIDRegistry.internalID(level, number, static, name)
}

// internalID creates a new trusted ErrorID for the reserved FAIL domain.
func (r *IDRegistry) internalID(level, number int, static bool, name string) ErrorID {
	domain := reservedDomain

	r.mu.Lock()
	defer r.mu.Unlock()

	// Validation 1: Name must start with domain (FAIL)
	if !hasPrefix(name, domain) {
		panic(fmt.Sprintf(
			"internal error name '%s' must start with domain '%s' (e.g., %sRegistryCorrupted)",
			name, domain, domain,
		))
	}

	// Validation 2: Name must not already exist
	if existing, exists := r.registeredIDs[name]; exists {
		panic(fmt.Sprintf(
			"internal error name '%s' already registered as %s",
			name, existing.String(),
		))
	}

	// Validation 3: Name must not be too similar to existing names
	for existingName := range r.registeredIDs {
		distance := levenshteinDistance(name, existingName)
		if distance <= 3 {
			panic(fmt.Sprintf(
				"internal error name '%s' is too similar to existing name '%s' (distance: %d, must be > 3)",
				name, existingName, distance,
			))
		}
	}

	// Validation 4: Number must be unique within FAIL domain+type
	numberKey := fmt.Sprintf("%s:%v:%d", domain, static, number)
	if existing, exists := r.numberIndex[numberKey]; exists {
		panic(fmt.Sprintf(
			"number %d from %s already used in internal domain '%s' (static=%v) by '%s'",
			number, name, domain, static, existing.name,
		))
	}

	// Create the ID
	id := ErrorID{
		name:     name,
		domain:   domain,
		level:    level,
		isStatic: static,
		number:   number,
		trusted:  true,
	}
	r.registeredIDs[name] = id
	r.numberIndex[numberKey] = id

	return id
}

// Reset clears all registered IDs (useful for testing)
func (r *IDRegistry) Reset() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.registeredIDs = make(map[string]ErrorID)
	r.numberIndex = make(map[string]ErrorID)
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
		numberIndex:   make(map[string]ErrorID),
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

// ValidateIDs checks for gaps and duplicates. Call in main() after all init().
func ValidateIDs() {
	globalIDRegistry.validateNoGaps()
}

// ValidateIDs checks for gaps and duplicates. Call in main() after all init().
func (r *IDRegistry) ValidateIDs() {
	r.validateNoGaps()
}

// validateNoGaps checks for numbering gaps within each domain+type combination
// Gaps indicate possible mistakes in manual numbering (skipped numbers)
func (r *IDRegistry) validateNoGaps() {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Group by domain+type
	type groupKey struct {
		domain string
		static bool
	}
	groups := make(map[groupKey][]int) // key -> list of numbers

	for _, id := range r.registeredIDs {
		key := groupKey{domain: id.domain, static: id.isStatic}
		groups[key] = append(groups[key], id.number)
	}

	// Check each group for gaps
	var gaps []string
	for key, numbers := range groups {
		if len(numbers) <= 1 {
			continue // Single ID or empty, no gaps possible
		}

		sort.Ints(numbers)

		for i := 1; i < len(numbers); i++ {
			if numbers[i] != numbers[i-1]+1 {
				// Gap detected
				expected := numbers[i-1] + 1
				actual := numbers[i]
				gaps = append(gaps, fmt.Sprintf(
					"%s (static=%v): missing %d (jumped from %d to %d)",
					key.domain, key.static, expected, numbers[i-1], actual,
				))
			}
		}
	}

	if len(gaps) > 0 {
		panic(fmt.Sprintf(
			"fail: ID numbering gaps detected (missing numbers):\n%s\n"+
				"Hint: IDs must be numbered sequentially starting from 0 within each domain+type combination. "+
				"Gaps indicate skipped numbers or future-proofing, which breaks the stability contract.",
			formatGaps(gaps),
		))
	}
}

func formatGaps(gaps []string) string {
	result := ""
	for _, g := range gaps {
		result += "  - " + g + "\n"
	}
	return result
}

// ExportIDList returns all registered error IDs as JSON bytes
func ExportIDList() ([]byte, error) {
	return globalIDRegistry.ExportIDList()
}

// ExportIDList returns all registered error IDs as JSON for this registry
func (r *IDRegistry) ExportIDList() ([]byte, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	type exportEntry struct {
		Name   string `json:"name"`
		Domain string `json:"domain"`
		Static bool   `json:"static"`
		Level  int    `json:"level"`
		Number int    `json:"number"`
		ID     string `json:"id"`
	}

	// Collect IDs
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

	entries := make([]exportEntry, len(ids))
	for i, id := range ids {
		entries[i] = exportEntry{
			Name:   id.name,
			Domain: id.domain,
			Static: id.isStatic,
			Level:  id.level,
			Number: id.number,
			ID:     id.String(),
		}
	}

	return json.MarshalIndent(entries, "", "  ")
}
