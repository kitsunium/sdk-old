package kerror

import (
	"fmt"
	"sync"
	"sync/atomic"
)

// Global registry variables
var (
	// Optimized registries using sync.Map for better concurrent performance
	registryByID      sync.Map // map[uint32]*KError
	registryByPkgCode sync.Map // map[string]*sync.Map (package -> code -> *KError)

	// Cache for HTTP status text to avoid repeated lookups
	httpStatusTextCache sync.Map

	// Cache for caller package names
	callerPackageCache sync.Map
)

// GetError retrieves a registered error by ID
func GetError(id uint32) (*KError, bool) {
	if val, ok := registryByID.Load(id); ok {
		return val.(*KError), true
	}
	return nil, false
}

// GetErrorByPackageCode retrieves a registered error by package and code
func GetErrorByPackageCode(pkg string, code int) (*KError, bool) {
	if pkgMapInterface, ok := registryByPkgCode.Load(pkg); ok {
		pkgMap := pkgMapInterface.(*sync.Map)
		if val, ok := pkgMap.Load(code); ok {
			return val.(*KError), true
		}
	}
	return nil, false
}

// ListErrors returns all registered errors
func ListErrors() []KError {
	var errors []KError

	registryByID.Range(func(key, value any) bool {
		if err, ok := value.(*KError); ok {
			errors = append(errors, *err)
		}
		return true
	})

	return errors
}

// ListPackageCodes returns all error codes defined for a specific package
func ListPackageCodes(pkg string) []int {
	if pkgMapInterface, ok := registryByPkgCode.Load(pkg); ok {
		pkgMap := pkgMapInterface.(*sync.Map)
		var codes []int
		pkgMap.Range(func(key, value any) bool {
			codes = append(codes, key.(int))
			return true
		})
		return codes
	}
	return nil
}

// ListPackages returns all packages that have defined errors
func ListPackages() []string {
	var packages []string

	registryByPkgCode.Range(func(key, value any) bool {
		packages = append(packages, key.(string))
		return true
	})

	return packages
}

// ValidatePackageCode checks if a code is already used in a package
func ValidatePackageCode(pkg string, code int) error {
	if pkgMapInterface, ok := registryByPkgCode.Load(pkg); ok {
		pkgMap := pkgMapInterface.(*sync.Map)
		if val, ok := pkgMap.Load(code); ok {
			existing := val.(*KError)
			return fmt.Errorf("code %d already used in package %s (ID: %d)", code, pkg, existing.id)
		}
	}
	return nil
}

// ClearRegistry clears the error registry (useful for testing)
func ClearRegistry() {
	// Clear registries
	registryByID = sync.Map{}
	registryByPkgCode = sync.Map{}
	httpStatusTextCache = sync.Map{}
	callerPackageCache = sync.Map{}

	atomic.StoreUint32(&errorCounter, 0)
}
