package fail

import (
	"container/list"
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
	name         string
	domain       string
	level        int // Severity level
	isStatic     bool
	number       int  // Explicitly assigned, stable across versions
	isRegistered bool // Internal flag - only IDs created by ID() have this as true
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

// IsRegistered returns true if this ID was created through the proper ID() function
func (id ErrorID) IsRegistered() bool {
	return id.isRegistered
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

type numberNode struct {
	number int
	name   string
	id     string
}

// IDRegistry manages error ID generation and validation
// Numbers are explicitly assigned per domain and type (static/dynamic)
type IDRegistry struct {
	mu                       sync.Mutex
	registeredIDs            map[string]ErrorID    // name -> ErrorID
	numberIndex              map[string]*list.List // "domain:static" -> sorted list of numberNode
	allowRuntimePanics       *bool
	allowRuntimeRegistration bool
}

// Global ID registry
var globalIDRegistry = &IDRegistry{
	registeredIDs: make(map[string]ErrorID),
	numberIndex:   make(map[string]*list.List),
}

// ID creates a new trusted ErrorID with explicit numbering
// This is the ONLY way to create a trusted ErrorID
//
// WARNING: Must be called at package level (var declaration).
// Calling inside func init() or runtime functions causes unstable numbering.
// Use go:generate or static analysis to verify.
//
// CRITICAL: Since ErrorIDs are a global concept used throughout your entire codebase,
// they should ideally all be defined together in the same var block within a single
// package (e.g., a dedicated errors package). The default global registry cannot
// ensure uniqueness across packages and may incorrectly report gaps that don't
// actually exist due to Go's file compilation ordering. If IDs are scattered across
// multiple files/packages, the registry may see partial sequences and panic on
// apparent gaps that would be filled by later compilation units. Centralize all
// ID definitions in one location to ensure strict sequential numbering and avoid
// false gap detection.
//
// NOTE: The global registry ALWAYS enforces strict sequential numbering (no gaps).
// If you need non-sequential numbering or want to disable gap checking, you must
// create your own isolated IDRegistry via NewIDRegistry() and use registry.ID()
// directly instead of this package-level ID() function. Custom registries allow
// you to control gap checking behavior, but the global registry is strict by design
// to ensure consistency across the entire codebase.
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
//   - Any gap in numbering is detected that is not being filled by current insertion
//
// Example:
//
//	// Centralized in one package - RECOMMENDED
//	var (
//	    AuthInvalidCredentials = fail.ID(0, "AUTH", 0, true, "AuthInvalidCredentials")   // 0_AUTH_0000_S
//	    AuthInvalidPassword    = fail.ID(0, "AUTH", 1, true, "AuthInvalidPassword")      // 0_AUTH_0001_S
//	    AuthCustomError        = fail.ID(0, "AUTH", 0, false, "AuthCustomError")         // 0_AUTH_0000_D
//	    AuthAnotherError       = fail.ID(0, "AUTH", 1, false, "AuthAnotherError")        // 0_AUTH_0001_D
//	    // v0.0.2 - add new ID, must be next in sequence
//	    AuthNewFeature         = fail.ID(0, "AUTH", 2, true, "AuthNewFeature")           // 0_AUTH_0002_S
//	)
func ID(level int, domain string, number int, static bool, name string) ErrorID {
	return globalIDRegistry.ID(level, domain, number, static, name)
}

// ID creates a new trusted ErrorID for this registry
func (r *IDRegistry) ID(level int, domain string, number int, static bool, name string) ErrorID {
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

	groupKey := fmt.Sprintf("%s:%v", domain, static)
	numList, exists := r.numberIndex[groupKey]
	if !exists {
		numList = list.New()
		r.numberIndex[groupKey] = numList
	}

	// Walk the list to find insertion point, check for collision, and detect gaps
	var insertAfter *list.Element
	prevNum := -1
	foundInsertion := false

	for e := numList.Front(); e != nil; e = e.Next() {
		node := e.Value.(numberNode)

		// Check for collision
		if node.number == number {
			panic(fmt.Sprintf("number %d already used in %s (static=%v) by '%s' (%s)",
				number, domain, static, node.name, node.id))
		}

		// Determine expected number at this position
		expected := prevNum + 1

		// Check for gap before this node
		if node.number != expected {
			// Gap detected between prevNum and node.number
			// Are we filling this gap?
			if !foundInsertion && number >= expected && number < node.number {
				// We're filling this gap, continue to find exact insertion point
			} else {
				// Gap that we're not filling - panic immediately
				if prevNum == -1 {
					panic(fmt.Sprintf(
						"fail: ID numbering gap detected in %s (static=%v): missing 0 (started at %d)",
						domain, static, node.number,
					))
				} else {
					panic(fmt.Sprintf(
						"fail: ID numbering gap detected in %s (static=%v): missing %d (between %d and %d)",
						domain, static, expected, prevNum, node.number,
					))
				}
			}
		}

		// Find insertion point (maintain sorted order)
		if !foundInsertion && node.number > number {
			// We found where to insert (before this element)
			foundInsertion = true
			// Don't break - keep walking to check for more gaps
		} else if node.number < number {
			insertAfter = e
		}

		prevNum = node.number
	}

	// Check for gap at the end (after insertAfter)
	if !foundInsertion {
		// We're inserting at the end
		expected := 0
		if insertAfter != nil {
			expected = insertAfter.Value.(numberNode).number + 1
		}

		if number != expected {
			// Gap at the end that we're not filling
			if insertAfter == nil {
				panic(fmt.Sprintf(
					"fail: ID numbering gap detected in %s (static=%v): missing 0 (inserting %d)",
					domain, static, number,
				))
			} else {
				lastNum := insertAfter.Value.(numberNode).number
				panic(fmt.Sprintf(
					"fail: ID numbering gap detected in %s (static=%v): missing %d (between %d and %d)",
					domain, static, expected, lastNum, number,
				))
			}
		}
	}

	id := ErrorID{
		name:         name,
		domain:       domain,
		level:        level,
		isStatic:     static,
		number:       number,
		isRegistered: true,
	}

	newNode := numberNode{number: number, name: name, id: id.String()}
	if insertAfter == nil {
		numList.PushFront(newNode)
	} else {
		numList.InsertAfter(newNode, insertAfter)
	}

	r.registeredIDs[name] = id
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
			"internal id name '%s' already registered as %s",
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

	groupKey := fmt.Sprintf("%s:%v", domain, static)
	numList, exists := r.numberIndex[groupKey]
	if !exists {
		numList = list.New()
		r.numberIndex[groupKey] = numList
	}

	var insertAfter *list.Element
	prevNum := -1
	foundInsertion := false

	for e := numList.Front(); e != nil; e = e.Next() {
		node := e.Value.(numberNode)
		if node.number == number {
			panic(fmt.Sprintf("number %d already used internally by '%s'", number, node.name))
		}

		expected := prevNum + 1
		if node.number != expected {
			if !foundInsertion && number >= expected && number < node.number {
				// filling gap
			} else {
				if prevNum == -1 {
					panic(fmt.Sprintf("fail: internal ID gap: missing 0 (started at %d)", node.number))
				} else {
					panic(fmt.Sprintf("fail: internal ID gap: missing %d (between %d and %d)",
						expected, prevNum, node.number))
				}
			}
		}

		if !foundInsertion && node.number > number {
			foundInsertion = true
		} else if node.number < number {
			insertAfter = e
		}
		prevNum = node.number
	}

	if !foundInsertion {
		expected := 0
		if insertAfter != nil {
			expected = insertAfter.Value.(numberNode).number + 1
		}
		if number != expected {
			if insertAfter == nil {
				panic(fmt.Sprintf("fail: internal ID gap: missing 0 (inserting %d)", number))
			} else {
				lastNum := insertAfter.Value.(numberNode).number
				panic(fmt.Sprintf("fail: internal ID gap: missing %d (between %d and %d)",
					expected, lastNum, number))
			}
		}
	}

	id := ErrorID{
		name:         name,
		domain:       domain,
		level:        level,
		isStatic:     static,
		number:       number,
		isRegistered: true,
	}

	newNode := numberNode{number: number, name: name, id: id.String()}
	if insertAfter == nil {
		numList.PushFront(newNode)
	} else {
		numList.InsertAfter(newNode, insertAfter)
	}

	r.registeredIDs[name] = id
	return id
}

// Reset clears all registered IDs (useful for testing)
func (r *IDRegistry) Reset() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.registeredIDs = make(map[string]ErrorID)
	r.numberIndex = make(map[string]*list.List)
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
			return ids[i].isStatic
		}
		return ids[i].number < ids[j].number
	})

	return ids
}

// NewIDRegistry creates a new isolated ID registry (useful for testing or multi-app)
func NewIDRegistry() *IDRegistry {
	return &IDRegistry{
		registeredIDs: make(map[string]ErrorID),
		numberIndex:   make(map[string]*list.List),
	}
}

// hasPrefix checks if name starts with domain (case-insensitive check of first letters)
func hasPrefix(name, domain string) bool {
	if len(name) < len(domain) {
		return false
	}
	return toLower(name[:len(domain)]) == toLower(domain)
}

// toLower converts string to lowercase (simple ASCII version)
func toLower(s string) string {
	b := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		b[i] = c
	}
	return string(b)
}

// levenshteinDistance calculates the Levenshtein distance between two strings
// This measures how similar two strings are (lower = more similar)
func levenshteinDistance(s1, s2 string) int {
	m, n := len(s1), len(s2)
	if m == 0 {
		return n
	}
	if n == 0 {
		return m
	}

	prev := make([]int, n+1)
	curr := make([]int, n+1)

	for j := 0; j <= n; j++ {
		prev[j] = j
	}

	for i := 1; i <= m; i++ {
		curr[0] = i
		for j := 1; j <= n; j++ {
			cost := 0
			if s1[i-1] != s2[j-1] {
				cost = 1
			}
			curr[j] = minInt(prev[j]+1, curr[j-1]+1, prev[j-1]+cost)
		}
		prev, curr = curr, prev
	}
	return prev[n]
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

// FIXME make so that custom IDRegistries can tolerate gaps if clients wish

// validateNoGaps checks for numbering gaps within each domain+type combination
// Gaps indicate possible mistakes in manual numbering (skipped numbers)
func (r *IDRegistry) validateNoGaps() {
	r.mu.Lock()
	defer r.mu.Unlock()

	for groupKey, numList := range r.numberIndex {
		if numList.Len() == 0 {
			continue
		}

		var domain string
		var static bool
		_, err := fmt.Sscanf(groupKey, "%s:%v", &domain, &static)
		if err != nil {
			panic(fmt.Sprintf("[fail]: invalid group key, error while scanning: %v", err))
		}

		expected := 0
		for e := numList.Front(); e != nil; e = e.Next() {
			node := e.Value.(numberNode)
			if node.number != expected {
				panic(fmt.Sprintf("[fail]: ID numbering gap in %s (static=%v): missing %d"+
					"Hint: IDs must be numbered sequentially starting from 0 within each domain+type combination. "+
					"Gaps indicate skipped numbers or future-proofing, which breaks the stability contract. "+
					"If you want to allow gaps use a custom IDRegistry and a custom error Registry",
					domain, static, expected))
			}
			expected = node.number + 1
		}
	}
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
			return ids[i].isStatic
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
