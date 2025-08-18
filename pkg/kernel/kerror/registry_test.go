package kerror

import (
	"fmt"
	"sort"
	"sync"
	"sync/atomic"
	"testing"
)

func TestGetError(t *testing.T) {
	ClearRegistry()
	
	// Define some errors
	err1 := Define(KConfig{Code: 404})
	err2 := Define(KConfig{Code: 500})
	
	tests := []struct {
		name    string
		id      uint32
		want    *KError
		wantOk  bool
	}{
		{
			name:   "existing error 1",
			id:     err1.ID(),
			want:   &err1,
			wantOk: true,
		},
		{
			name:   "existing error 2",
			id:     err2.ID(),
			want:   &err2,
			wantOk: true,
		},
		{
			name:   "non-existent error",
			id:     999999,
			want:   nil,
			wantOk: false,
		},
		{
			name:   "zero ID",
			id:     0,
			want:   nil,
			wantOk: false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := GetError(tt.id)
			
			if ok != tt.wantOk {
				t.Errorf("GetError() ok = %v, want %v", ok, tt.wantOk)
			}
			
			if tt.wantOk {
				if got == nil {
					t.Error("GetError() returned nil for existing error")
				} else if got.id != tt.want.id {
					t.Errorf("GetError() returned wrong error: got ID %d, want %d", got.id, tt.want.id)
				}
			} else {
				if got != nil {
					t.Error("GetError() should return nil for non-existent error")
				}
			}
		})
	}
}

func TestGetErrorByPackageCode(t *testing.T) {
	ClearRegistry()
	
	// Define errors with specific packages
	err1 := Define(KConfig{Package: "pkg1", Code: 404})
	err2 := Define(KConfig{Package: "pkg1", Code: 500})
	err3 := Define(KConfig{Package: "pkg2", Code: 404})
	
	tests := []struct {
		name   string
		pkg    string
		code   int
		want   *KError
		wantOk bool
	}{
		{
			name:   "pkg1/404",
			pkg:    "pkg1",
			code:   404,
			want:   &err1,
			wantOk: true,
		},
		{
			name:   "pkg1/500",
			pkg:    "pkg1",
			code:   500,
			want:   &err2,
			wantOk: true,
		},
		{
			name:   "pkg2/404",
			pkg:    "pkg2",
			code:   404,
			want:   &err3,
			wantOk: true,
		},
		{
			name:   "non-existent package",
			pkg:    "nonexistent",
			code:   404,
			want:   nil,
			wantOk: false,
		},
		{
			name:   "non-existent code",
			pkg:    "pkg1",
			code:   999,
			want:   nil,
			wantOk: false,
		},
		{
			name:   "empty package",
			pkg:    "",
			code:   404,
			want:   nil,
			wantOk: false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := GetErrorByPackageCode(tt.pkg, tt.code)
			
			if ok != tt.wantOk {
				t.Errorf("GetErrorByPackageCode() ok = %v, want %v", ok, tt.wantOk)
			}
			
			if tt.wantOk {
				if got == nil {
					t.Error("GetErrorByPackageCode() returned nil for existing error")
				} else if got.id != tt.want.id {
					t.Errorf("GetErrorByPackageCode() returned wrong error: got ID %d, want %d", got.id, tt.want.id)
				}
			} else {
				if got != nil {
					t.Error("GetErrorByPackageCode() should return nil for non-existent error")
				}
			}
		})
	}
}

func TestListErrors(t *testing.T) {
	ClearRegistry()
	
	// Empty registry
	errors := ListErrors()
	if len(errors) != 0 {
		t.Error("ListErrors() should return empty slice for empty registry")
	}
	
	// Define some errors
	err1 := Define(KConfig{Code: 404})
	err2 := Define(KConfig{Code: 500})
	err3 := Define(KConfig{Code: 503})
	
	errors = ListErrors()
	if len(errors) != 3 {
		t.Errorf("ListErrors() returned %d errors, want 3", len(errors))
	}
	
	// Verify all errors are present
	ids := map[uint32]bool{
		err1.ID(): false,
		err2.ID(): false,
		err3.ID(): false,
	}
	
	for _, e := range errors {
		if _, ok := ids[e.ID()]; ok {
			ids[e.ID()] = true
		}
	}
	
	for id, found := range ids {
		if !found {
			t.Errorf("Error with ID %d not found in ListErrors()", id)
		}
	}
}

func TestListPackageCodes(t *testing.T) {
	ClearRegistry()
	
	// Empty package
	codes := ListPackageCodes("nonexistent")
	if codes != nil {
		t.Error("ListPackageCodes() should return nil for non-existent package")
	}
	
	// Define errors
	Define(KConfig{Package: "pkg1", Code: 404})
	Define(KConfig{Package: "pkg1", Code: 500})
	Define(KConfig{Package: "pkg1", Code: 503})
	Define(KConfig{Package: "pkg2", Code: 400})
	
	// List pkg1 codes
	codes = ListPackageCodes("pkg1")
	if len(codes) != 3 {
		t.Errorf("ListPackageCodes(pkg1) returned %d codes, want 3", len(codes))
	}
	
	// Sort for consistent comparison
	sort.Ints(codes)
	expected := []int{404, 500, 503}
	for i, code := range expected {
		if i >= len(codes) || codes[i] != code {
			t.Errorf("Expected code %d at index %d", code, i)
		}
	}
	
	// List pkg2 codes
	codes = ListPackageCodes("pkg2")
	if len(codes) != 1 {
		t.Errorf("ListPackageCodes(pkg2) returned %d codes, want 1", len(codes))
	}
	if codes[0] != 400 {
		t.Errorf("ListPackageCodes(pkg2) returned %d, want 400", codes[0])
	}
}

func TestListPackages(t *testing.T) {
	ClearRegistry()
	
	// Empty registry
	packages := ListPackages()
	if len(packages) != 0 {
		t.Error("ListPackages() should return empty slice for empty registry")
	}
	
	// Define errors in different packages
	Define(KConfig{Package: "pkg1", Code: 404})
	Define(KConfig{Package: "pkg2", Code: 500})
	Define(KConfig{Package: "pkg3", Code: 503})
	Define(KConfig{Package: "pkg1", Code: 400}) // Another in pkg1
	
	packages = ListPackages()
	if len(packages) != 3 {
		t.Errorf("ListPackages() returned %d packages, want 3", len(packages))
	}
	
	// Sort for consistent comparison
	sort.Strings(packages)
	expected := []string{"pkg1", "pkg2", "pkg3"}
	for i, pkg := range expected {
		if i >= len(packages) || packages[i] != pkg {
			t.Errorf("Expected package %s at index %d", pkg, i)
		}
	}
}

func TestValidatePackageCode(t *testing.T) {
	ClearRegistry()
	
	// No errors defined yet
	err := ValidatePackageCode("pkg1", 404)
	if err != nil {
		t.Error("ValidatePackageCode() should return nil for unused code")
	}
	
	// Define an error
	existing := Define(KConfig{Package: "pkg1", Code: 404})
	
	// Check duplicate
	err = ValidatePackageCode("pkg1", 404)
	if err == nil {
		t.Error("ValidatePackageCode() should return error for duplicate code")
	}
	if err != nil {
		expectedMsg := fmt.Sprintf("code 404 already used in package pkg1 (ID: %d)", existing.ID())
		if err.Error() != expectedMsg {
			t.Errorf("Error message = %s, want %s", err.Error(), expectedMsg)
		}
	}
	
	// Different code in same package is OK
	err = ValidatePackageCode("pkg1", 500)
	if err != nil {
		t.Error("ValidatePackageCode() should return nil for different code")
	}
	
	// Same code in different package is OK
	err = ValidatePackageCode("pkg2", 404)
	if err != nil {
		t.Error("ValidatePackageCode() should return nil for different package")
	}
}

func TestClearRegistry(t *testing.T) {
	// Define some errors
	Define(KConfig{Code: 404})
	Define(KConfig{Code: 500})
	
	// Verify they exist
	errors := ListErrors()
	if len(errors) == 0 {
		t.Fatal("Should have errors before clear")
	}
	
	// Clear
	ClearRegistry()
	
	// Verify cleared
	errors = ListErrors()
	if len(errors) != 0 {
		t.Error("Registry not cleared")
	}
	
	// Verify counter reset
	if atomic.LoadUint32(&errorCounter) != 0 {
		t.Error("Error counter not reset")
	}
	
	// Verify can define new errors
	err := Define(KConfig{Code: 404})
	if err.ID() != 1 {
		t.Errorf("First error after clear should have ID 1, got %d", err.ID())
	}
}

func TestRegistryConcurrency(t *testing.T) {
	ClearRegistry()
	
	var wg sync.WaitGroup
	
	// Concurrent definitions
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			Define(KConfig{
				Package: fmt.Sprintf("pkg%d", n),
				Code:    n * 100,
			})
		}(i)
	}
	
	// Concurrent reads
	for i := 0; i < 10; i++ {
		wg.Add(4)
		
		go func() {
			defer wg.Done()
			_ = ListErrors()
		}()
		
		go func() {
			defer wg.Done()
			_ = ListPackages()
		}()
		
		go func(n int) {
			defer wg.Done()
			_, _ = GetError(uint32(n))
		}(i)
		
		go func(n int) {
			defer wg.Done()
			_, _ = GetErrorByPackageCode(fmt.Sprintf("pkg%d", n), n*100)
		}(i)
	}
	
	wg.Wait()
	
	// Verify all errors were registered
	errors := ListErrors()
	if len(errors) != 10 {
		t.Errorf("Expected 10 errors, got %d", len(errors))
	}
	
	packages := ListPackages()
	if len(packages) != 10 {
		t.Errorf("Expected 10 packages, got %d", len(packages))
	}
}

func TestRegistryWithAutoPackage(t *testing.T) {
	ClearRegistry()
	
	// Define without explicit package
	err := Define(KConfig{Code: 404})
	
	// Should be able to retrieve by auto-detected package
	pkg := err.Package()
	if pkg == "" {
		t.Fatal("Package should be auto-detected")
	}
	
	retrieved, ok := GetErrorByPackageCode(pkg, 404)
	if !ok {
		t.Error("Should be able to retrieve by auto-detected package")
	}
	if retrieved.ID() != err.ID() {
		t.Error("Retrieved wrong error")
	}
}

func TestRegistrySyncMapOperations(t *testing.T) {
	ClearRegistry()
	
	// Test LoadOrStore behavior
	err1 := Define(KConfig{Package: "test", Code: 100})
	
	// Try to get the same package map
	if pkgMapInterface, ok := registryByPkgCode.Load("test"); !ok {
		t.Error("Package map should exist")
	} else {
		pkgMap := pkgMapInterface.(*sync.Map)
		if val, ok := pkgMap.Load(100); !ok {
			t.Error("Error should exist in package map")
		} else {
			stored := val.(*KError)
			if stored.ID() != err1.ID() {
				t.Error("Wrong error in package map")
			}
		}
	}
}

func TestCacheOperations(t *testing.T) {
	ClearRegistry()
	
	// Clear caches
	callerPackageCache = sync.Map{}
	
	// Define error to populate caches
	_ = Define(KConfig{Code: 200})
	
	// Check caller package cache
	cached := false
	callerPackageCache.Range(func(key, value interface{}) bool {
		cached = true
		return false
	})
	if !cached {
		t.Error("Caller package should be cached")
	}
	
	// Verify caches are cleared
	ClearRegistry()
	
	count := 0
	callerPackageCache.Range(func(key, value interface{}) bool {
		count++
		return true
	})
	if count != 0 {
		t.Error("Caller package cache not cleared")
	}
}

func TestListPackageCodesEdgeCases(t *testing.T) {
	ClearRegistry()
	
	// Test with many codes
	for i := 0; i < 100; i++ {
		Define(KConfig{Package: "many", Code: i})
	}
	
	codes := ListPackageCodes("many")
	if len(codes) != 100 {
		t.Errorf("Should have 100 codes, got %d", len(codes))
	}
	
	// Test with duplicate attempts (should panic in Define, but test retrieval)
	func() {
		defer func() {
			recover() // Ignore panic
		}()
		Define(KConfig{Package: "dup", Code: 200})
		Define(KConfig{Package: "dup", Code: 200}) // Will panic
	}()
	
	// Should still be able to list
	dupCodes := ListPackageCodes("dup")
	if len(dupCodes) != 1 {
		t.Error("Should have exactly one code despite duplicate attempt")
	}
}

func TestRegistryStressTest(t *testing.T) {
	ClearRegistry()
	
	// Stress test with many goroutines
	var wg sync.WaitGroup
	const numGoroutines = 100
	const opsPerGoroutine = 100
	
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(base int) {
			defer wg.Done()
			
			// Define errors
			for j := 0; j < opsPerGoroutine; j++ {
				code := base*1000 + j
				Define(KConfig{
					Package: fmt.Sprintf("stress%d", base),
					Code:    code,
				})
				
				// Immediately try to retrieve
				if _, ok := GetErrorByPackageCode(fmt.Sprintf("stress%d", base), code); !ok {
					t.Errorf("Failed to retrieve just-defined error")
				}
			}
			
			// List operations
			_ = ListErrors()
			_ = ListPackages()
			_ = ListPackageCodes(fmt.Sprintf("stress%d", base))
		}(i)
	}
	
	wg.Wait()
	
	// Verify total count
	errors := ListErrors()
	expected := numGoroutines * opsPerGoroutine
	if len(errors) != expected {
		t.Errorf("Expected %d errors, got %d", expected, len(errors))
	}
}

func TestEmptyPackageHandling(t *testing.T) {
	ClearRegistry()
	
	// Store in a map with empty key
	pkgMapInterface, _ := registryByPkgCode.LoadOrStore("", &sync.Map{})
	pkgMap := pkgMapInterface.(*sync.Map)
	
	testErr := &KError{
		id:      999,
		pkg:     "",
		code:    404,
		message: "test",
	}
	
	pkgMap.Store(404, testErr)
	registryByID.Store(uint32(999), testErr)
	
	// Should be able to retrieve
	retrieved, ok := GetErrorByPackageCode("", 404)
	if !ok {
		t.Error("Should handle empty package name")
	}
	if retrieved.ID() != 999 {
		t.Error("Retrieved wrong error for empty package")
	}
	
	// Should appear in listings
	codes := ListPackageCodes("")
	if len(codes) != 1 || codes[0] != 404 {
		t.Error("ListPackageCodes should work with empty package")
	}
}