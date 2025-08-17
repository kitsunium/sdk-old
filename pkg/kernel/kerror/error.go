package kerror

import (
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
)

// KConfig represents configuration for defining a KError
type KConfig struct {
	Package string // Package name (auto-detected if empty)
	Code    int    // Error code (HTTP status code, also used as exit code)
	Message string // Optional message (auto-filled from HTTP status if empty)
}

// KError represents a unique error identifier that can be used as a constant
type KError struct {
	id      uint32 // Unique identifier (auto-incremented)
	pkg     string // Package name
	code    int    // Error code (HTTP status code, also used as exit code)
	message string // Default message
}

// Global variables are defined in registry.go to avoid duplication
var errorCounter uint32

// getCallerPackage returns the package name of the caller with caching
func getCallerPackage() string {
	// Skip 2 frames: getCallerPackage and Define
	pc, _, _, ok := runtime.Caller(2)
	if !ok {
		return GetConfig().DefaultPackage
	}

	// Check cache first
	if cached, found := callerPackageCache.Load(pc); found {
		return cached.(string)
	}

	fn := runtime.FuncForPC(pc)
	if fn == nil {
		return GetConfig().DefaultPackage
	}

	// Get the full function name
	fullName := fn.Name()

	var pkg string
	lastSlash := strings.LastIndexByte(fullName, '/')
	if lastSlash < 0 {
		if dot := strings.IndexByte(fullName, '.'); dot >= 0 {
			pkg = fullName[:dot]
		} else {
			pkg = fullName
		}
	} else {
		remaining := fullName[lastSlash+1:]
		if dot := strings.IndexByte(remaining, '.'); dot >= 0 {
			pkg = remaining[:dot]
		} else {
			pkg = remaining
		}
	}

	// Store in cache
	callerPackageCache.Store(pc, pkg)
	return pkg
}

// Define creates a new error constant for a package using a config struct
func Define(config KConfig) KError {
	// Auto-detect package if not provided
	pkg := config.Package
	if pkg == "" {
		pkg = getCallerPackage()
	}

	// Get or create package map using sync.Map for better concurrency
	pkgMapInterface, _ := registryByPkgCode.LoadOrStore(pkg, &sync.Map{})
	pkgMap := pkgMapInterface.(*sync.Map)

	// Generate ID first
	id := atomic.AddUint32(&errorCounter, 1)

	// Get message with caching for HTTP status text
	message := config.Message
	if message == "" {
		if cached, found := httpStatusTextCache.Load(config.Code); found {
			message = cached.(string)
		} else {
			message = http.StatusText(config.Code)
			if message == "" {
				message = fmt.Sprintf("error %d", config.Code)
			}
			httpStatusTextCache.Store(config.Code, message)
		}
	}

	err := &KError{
		id:      id,
		pkg:     pkg,
		code:    config.Code,
		message: message,
	}

	// Check for duplicates and store atomically
	if actual, loaded := pkgMap.LoadOrStore(config.Code, err); loaded {
		existing := actual.(*KError)
		panic(fmt.Sprintf("kerror: duplicate error code %d in package %s (already defined with ID %d)",
			config.Code, pkg, existing.id))
	}

	// Store in optimized registry
	registryByID.Store(id, err)

	// Record metrics if enabled
	if GetConfig().EnableMetrics {
		recordErrorDefinition(pkg, config.Code)
	}

	return *err
}

// ID returns the unique identifier
func (e KError) ID() uint32 {
	return e.id
}

// Package returns the package name
func (e KError) Package() string {
	return e.pkg
}

// Code returns the error code (HTTP status code)
func (e KError) Code() int {
	return e.code
}

// Message returns the default message
func (e KError) Message() string {
	return e.message
}

// Error implements the error interface
func (e KError) Error() string {
	return e.message
}

// Is implements errors.Is for comparison
func (e KError) Is(target error) bool {
	if t, ok := target.(KError); ok {
		return e.id == t.id
	}
	if t, ok := target.(*KError); ok {
		return t != nil && e.id == t.id
	}
	return false
}

// MarshalJSON implements json.Marshaler
func (e KError) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		ID      uint32 `json:"id"`
		Package string `json:"package"`
		Code    int    `json:"code"`
		Message string `json:"message"`
	}{
		ID:      e.id,
		Package: e.pkg,
		Code:    e.code,
		Message: e.message,
	})
}

// UnmarshalJSON implements json.Unmarshaler
func (e *KError) UnmarshalJSON(data []byte) error {
	var temp struct {
		ID      uint32 `json:"id"`
		Package string `json:"package"`
		Code    int    `json:"code"`
		Message string `json:"message"`
	}
	
	if err := json.Unmarshal(data, &temp); err != nil {
		return err
	}
	
	e.id = temp.ID
	e.pkg = temp.Package
	e.code = temp.Code
	e.message = temp.Message
	
	return nil
}

// String implements fmt.Stringer
func (e KError) String() string {
	return fmt.Sprintf("[%s:%d] %s (ID:%d)", e.pkg, e.code, e.message, e.id)
}