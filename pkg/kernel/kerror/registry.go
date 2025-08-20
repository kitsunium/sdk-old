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
	registryByID      kcache.Cache[uint32, *KError]
	registryByPkgCode kcache.Cache[string, kcache.Cache[int, *KError]]

	// Cache for caller package names
	callerPackageCache kcache.Cache[uintptr, string]

	// Initialize caches once
	cacheInit sync.Once
)

// initCaches initializes all caches with optimal settings
func initCaches() {
	cacheInit.Do(func() {
		// Use AtomicCache for read-heavy workloads
		registryByID = kcache.NewAtomicCache[uint32, *KError](10000)
		registryByPkgCode = kcache.NewShardedLRU[string, kcache.Cache[int, *KError]](1000, 64)
		callerPackageCache = kcache.NewAtomicCache[uintptr, string](1000)
	})
}

// GetError retrieves a registered error by ID.
func GetError(id uint32) (*KError, bool) {
	initCaches()
	return registryByID.Get(id)
}

// GetErrorByPackageCode retrieves a registered error by package and code.
func GetErrorByPackageCode(pkg string, code int) (*KError, bool) {
	initCaches()
	if pkgCache, ok := registryByPkgCode.Get(pkg); ok {
		return pkgCache.Get(code)
	}
	return nil, false
}

// ListErrors returns all registered errors.
func ListErrors() []KError {
	initCaches()
	var errors []KError

	if atomicCache, ok := registryByID.(*kcache.AtomicCache[uint32, *KError]); ok {
		atomicCache.Range(func(id uint32, err *KError) bool {
			if err != nil {
				errors = append(errors, *err)
			}
			return true
		})
	}

	return errors
}

// ListPackageCodes returns all error codes defined for a specific package.
func ListPackageCodes(pkg string) []int {
	initCaches()
	var codes []int

	if pkgCache, ok := registryByPkgCode.Get(pkg); ok {
		if atomicCache, ok := pkgCache.(*kcache.AtomicCache[int, *KError]); ok {
			atomicCache.Range(func(code int, err *KError) bool {
				codes = append(codes, code)
				return true
			})
		}
	}

	return codes
}

// ListPackages returns all packages that have defined errors.
func ListPackages() []string {
	initCaches()
	var packages []string

	if shardedCache, ok := registryByPkgCode.(*kcache.ShardedLRU[string, kcache.Cache[int, *KError]]); ok {
		shardedCache.Range(func(pkg string, _ kcache.Cache[int, *KError]) bool {
			packages = append(packages, pkg)
			return true
		})
	}

	return packages
}

// ValidatePackageCode checks if a code is already used in a package.
func ValidatePackageCode(pkg string, code int) error {
	initCaches()
	if pkgCache, ok := registryByPkgCode.Get(pkg); ok {
		if existing, ok := pkgCache.Get(code); ok {
			return fmt.Errorf("code %d already used in package %s (ID: %d)", code, pkg, existing.id)
		}
	}
	return nil
}

// ClearRegistry clears the error registry (useful for testing).
func ClearRegistry() {
	initCaches()
	// Clear registries
	registryByID.Clear()
	registryByPkgCode.Clear()
	callerPackageCache.Clear()

	atomic.StoreUint32(&errorCounter, 0)
}
