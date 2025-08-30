package kerror

import (
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/kitsunium/sdk/pkg/kernel/kcache"
)

// Global registry variables
var (
	// Optimized registries using high-performance kcache
	registryByID      kcache.Cache
	registryByPkgCode kcache.Cache // Maps package to another cache of error codes

	// Cache for caller package names
	callerPackageCache kcache.Cache

	// Lists for iteration support (since kcache doesn't support iteration)
	allErrors    []*KError        // All registered errors
	packageList  []string         // All packages with errors
	packageCodes map[string][]int // Map of package to error codes
	registryMu   sync.RWMutex     // Protects the lists above

	// Initialize caches once
	cacheInit sync.Once
)

// init ensures caches are initialized at package load time
func init() {
	initCaches()
}

// initCaches initializes all caches with optimal settings
func initCaches() {
	cacheInit.Do(func() {
		// Use sharded cache for better concurrency
		registryByID = kcache.NewSafeShardedCache(10000, 32)
		registryByPkgCode = kcache.NewSafeShardedCache(1000, 16)
		callerPackageCache = kcache.NewSafeShardedCache(1000, 16)

		// Initialize lists for iteration support
		allErrors = make([]*KError, 0, 100)
		packageList = make([]string, 0, 10)
		packageCodes = make(map[string][]int)
	})
}

// GetError retrieves a registered error by ID.
func GetError(id uint32) (*KError, bool) {
	initCaches()
	if v, ok := registryByID.Get(id); ok {
		if err, ok := v.(*KError); ok {
			return err, true
		}
	}
	return nil, false
}

// GetErrorByPackageCode retrieves a registered error by package and code.
func GetErrorByPackageCode(pkg string, code int) (*KError, bool) {
	initCaches()
	if v, ok := registryByPkgCode.Get(pkg); ok {
		if pkgCache, ok := v.(kcache.Cache); ok {
			if v2, ok := pkgCache.Get(code); ok {
				if err, ok := v2.(*KError); ok {
					return err, true
				}
			}
		}
	}
	return nil, false
}

// ListErrors returns all registered errors.
func ListErrors() []KError {
	initCaches()
	registryMu.RLock()
	defer registryMu.RUnlock()

	// Return a copy of all errors
	result := make([]KError, len(allErrors))
	for i, err := range allErrors {
		if err != nil {
			result[i] = *err
		}
	}
	return result
}

// ListPackageCodes returns all error codes defined for a specific package.
func ListPackageCodes(pkg string) []int {
	initCaches()
	registryMu.RLock()
	defer registryMu.RUnlock()

	codes, exists := packageCodes[pkg]
	if !exists {
		return nil
	}

	// Return a copy of the codes
	result := make([]int, len(codes))
	copy(result, codes)
	return result
}

// ListPackages returns all packages that have defined errors.
func ListPackages() []string {
	initCaches()
	registryMu.RLock()
	defer registryMu.RUnlock()

	// Return a copy of the package list
	result := make([]string, len(packageList))
	copy(result, packageList)
	return result
}

// ValidatePackageCode checks if a code is already used in a package.
func ValidatePackageCode(pkg string, code int) error {
	initCaches()
	if v, ok := registryByPkgCode.Get(pkg); ok {
		if pkgCache, ok := v.(kcache.Cache); ok {
			if v2, ok := pkgCache.Get(code); ok {
				if existing, ok := v2.(*KError); ok {
					return fmt.Errorf("code %d already used in package %s (ID: %d)", code, pkg, existing.id)
				}
			}
		}
	}
	return nil
}

// ClearRegistry clears the error registry (useful for testing).
func ClearRegistry() {
	initCaches()

	// Clear caches
	registryByID.Clear()
	registryByPkgCode.Clear()
	callerPackageCache.Clear()

	// Clear lists
	registryMu.Lock()
	allErrors = allErrors[:0]
	packageList = packageList[:0]
	for k := range packageCodes {
		delete(packageCodes, k)
	}
	registryMu.Unlock()

	atomic.StoreUint32(&errorCounter, 0)
}
