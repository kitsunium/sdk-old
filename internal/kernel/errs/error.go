// Package errs provides a typed error catalog for the Kitsunium SDK.
//
// Catalog entries are registered once via Define and carry a unique ID,
// a package name, a code, and a default message. Each entry can be turned
// into a concrete Instance at the call-site to attach a cause, tags, and
// typed details, while remaining comparable via errors.Is.
package errs

import (
	"fmt"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
)

// Config is the input passed to Define.
//
// Package may be left empty to let Define auto-detect from the call stack.
// Message may be left empty; Define will produce "error <Code>" as default.
type Config struct {
	Package string
	Code    int
	Message string
}

// Err is a catalog entry. It is comparable by ID and safe to pass by value.
//
// Create entries via Define; do not construct Err literals directly.
type Err struct {
	id      uint32
	pkg     string
	code    int
	message string
}

// initialMapCapacity is the preallocated capacity hint for inline maps.
const initialMapCapacity = 2

// callerSkip is the depth passed to runtime.Caller inside Define.
const callerSkip = 2

var (
	// errorCounter is the monotonic ID source for catalog entries.
	errorCounter uint32
	// registry maps (pkg, code) pairs to their catalog entry pointers.
	registry sync.Map
	// defineMu serialises duplicate-detection during Define.
	defineMu sync.Mutex
)

// Define registers a new catalog entry and returns its Err handle.
//
// It panics on a duplicate (package, code) pair — duplicates indicate a
// programmer error, not a runtime condition.
//
// Params:
//   - config: the registration parameters (Package, Code, Message)
//
// Returns:
//   - entry: the registered catalog handle
func Define(config Config) (entry Err) {
	pkg := config.Package
	//: fall back to automatic package detection when caller omits it
	if pkg == "" {
		//: walk up callerSkip frames to find the caller's package
		pkg = callerPackage(callerSkip)
	}

	defineMu.Lock()
	defer defineMu.Unlock()

	key := registryKey{pkg: pkg, code: config.Code}
	//: reject programmer errors at startup rather than silently shadowing
	if existing, ok := registry.Load(key); ok {
		existingEntry, _ := existing.(*Err)
		//: panic is intentional — duplicate catalog codes are always a bug
		if existingEntry != nil {
			panic(fmt.Sprintf("errs: duplicate error code %d in package %s (existing id=%d)", config.Code, pkg, existingEntry.id))
		}
	}

	message := config.Message
	//: provide a sensible default so callers can omit Message
	if message == "" {
		//: format matches the convention used in error identifiers
		message = fmt.Sprintf("error %d", config.Code)
	}

	id := atomic.AddUint32(&errorCounter, 1)
	reg := &Err{id: id, pkg: pkg, code: config.Code, message: message}
	registry.Store(key, reg)
	return *reg
}

type registryKey struct {
	pkg  string
	code int
}

// ID returns the globally unique identifier.
//
// Returns:
//   - id: monotonically assigned at Define time
func (e Err) ID() (id uint32) {
	//: expose the immutable identifier for comparison
	return e.id
}

// Pkg returns the owning package name.
//
// Returns:
//   - pkg: the package that registered this entry
func (e Err) Pkg() (pkg string) {
	//: expose for logging and structured output
	return e.pkg
}

// Code returns the package-scoped error code.
//
// Returns:
//   - code: the integer code assigned at registration
func (e Err) Code() (code int) {
	//: expose for structured output
	return e.code
}

// Message returns the default message.
//
// Returns:
//   - msg: the default human-readable description
func (e Err) Message() (msg string) {
	//: expose for display without formatting
	return e.message
}

// Error implements the error interface.
//
// Returns:
//   - msg: the default message (same as Message)
func (e Err) Error() (msg string) {
	//: satisfy the error interface using the catalog message
	return e.message
}

// Is reports whether target is the same catalog entry.
//
// Params:
//   - target: the error to compare against
//
// Returns:
//   - match: true when target refers to the same catalog ID
func (e Err) Is(target error) (match bool) {
	//: compare by catalog ID, accepting both value and pointer forms
	switch t := target.(type) {
	//: match value form
	case Err:
		return e.id == t.id
	//: match pointer form (nil pointer never matches)
	case *Err:
		return t != nil && e.id == t.id
	}
	//: unrecognised type — no match
	return false
}

// String returns "[pkg:code] message".
//
// Returns:
//   - s: human-readable representation of the catalog entry
func (e Err) String() (s string) {
	//: format matches Instance.Error for consistent log output
	return fmt.Sprintf("[%s:%d] %s", e.pkg, e.code, e.message)
}

// callerPackage returns the package name of the Nth caller up the stack.
//
// Params:
//   - skip: the number of stack frames to skip
//
// Returns:
//   - pkg: the package name, or "unknown" when unavailable
func callerPackage(skip int) (pkg string) {
	pc, _, _, ok := runtime.Caller(skip)
	//: no PC available — degrade gracefully
	if !ok {
		//: sentinel value that callers can detect
		return "unknown"
	}
	fn := runtime.FuncForPC(pc)
	//: no function info — degrade gracefully
	if fn == nil {
		//: sentinel value that callers can detect
		return "unknown"
	}
	full := fn.Name()
	//: strip the module path prefix (everything before the last slash)
	if slash := strings.LastIndexByte(full, '/'); slash >= 0 {
		full = full[slash+1:]
	}
	//: extract the package segment (before the first dot)
	if pkg, _, ok := strings.Cut(full, "."); ok {
		return pkg
	}
	return full
}

// clearRegistry resets the registry. Test-only.
func clearRegistry() {
	defineMu.Lock()
	defer defineMu.Unlock()
	//: iterate and delete all entries to allow tests to re-register codes
	registry.Range(func(k, _ any) bool {
		registry.Delete(k)
		//: continue ranging
		return true
	})
	//: reset the ID counter so tests get predictable IDs
	atomic.StoreUint32(&errorCounter, 0)
}
