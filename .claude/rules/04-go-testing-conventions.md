# 🧪 GO TESTING CONVENTIONS

## 📋 RÈGLES OBLIGATOIRES POUR TOUS LES TESTS GO

### 1. STRUCTURE DES PACKAGES DE TEST

#### Tests Black-Box (Recommandé par défaut)
```go
// Fichier: buffer_test.go
package kbuffer_test  // ✅ OBLIGATOIRE - toujours suffixer avec _test

import (
    "testing"
    "github.com/kitsunium/sdk/pkg/kernel/kbuffer"  // Import explicite du package testé
)

func TestNewBuffer(t *testing.T) {
    buf := kbuffer.NewBuffer(1024)  // ✅ Utiliser le nom du package
    // ...
}
```

#### Tests White-Box (Uniquement si accès aux membres privés nécessaire)
```go
// Fichier: buffer_internal_test.go  
package kbuffer  // ✅ Même package - uniquement pour tester des fonctions privées

func TestInternalFunction(t *testing.T) {
    result := internalFunction()  // ✅ Accès direct aux fonctions privées
    // ...
}
```

### 2. CONVENTIONS DE NOMMAGE

#### Tests Unitaires
```go
func TestFunctionName(t *testing.T)           // ✅ Test simple
func TestStructMethod(t *testing.T)           // ✅ Test de méthode  
func TestFunctionName_ErrorCase(t *testing.T) // ✅ Test de cas d'erreur
func TestFunctionName_EdgeCase(t *testing.T)  // ✅ Test de cas limite
```

#### Benchmarks
```go
func BenchmarkFunctionName(b *testing.B)              // ✅ Benchmark simple
func BenchmarkStructMethod(b *testing.B)              // ✅ Benchmark de méthode
func BenchmarkFunctionName_LargeInput(b *testing.B)   // ✅ Benchmark avec variation
```

#### Tests d'Exemples
```go
func ExampleFunctionName()       // ✅ Exemple simple
func ExampleStruct_Method()      // ✅ Exemple de méthode
func ExampleFunctionName_option() // ✅ Exemple avec variation
```

### 3. STRUCTURE STANDARD DES TESTS

#### Pattern Table-Driven Tests (Obligatoire pour > 2 cas)
```go
func TestNewBuffer(t *testing.T) {
    tests := []struct {
        name    string
        size    int
        wantErr bool
        want    interface{}
    }{
        {
            name: "valid size",
            size: 1024,
            wantErr: false,
            want: 1024,
        },
        {
            name: "zero size", 
            size: 0,
            wantErr: true,
            want: nil,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result, err := kbuffer.NewBuffer(tt.size)
            
            // Vérification d'erreur d'abord
            if (err != nil) != tt.wantErr {
                t.Errorf("NewBuffer() error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            
            // Puis vérification du résultat si pas d'erreur
            if !tt.wantErr && result.Cap() != tt.want {
                t.Errorf("NewBuffer() = %v, want %v", result.Cap(), tt.want)
            }
        })
    }
}
```

### 4. BENCHMARKS PERFORMANCE-ORIENTED

#### Structure Standard des Benchmarks
```go
func BenchmarkBuffer(b *testing.B) {
    // Setup une seule fois avant la mesure
    testData := make([]byte, 1024)
    buf := kbuffer.NewBuffer(kbuffer.DefaultBufferSize)
    
    b.ResetTimer()  // ✅ OBLIGATOIRE - exclut le setup du timing
    
    for i := 0; i < b.N; i++ {
        buf.Write(testData)
        buf.Reset()
    }
    
    // Optionnel: reporter des métriques custom
    b.SetBytes(1024)  // Pour calculer MB/sec automatiquement
}
```

#### Benchmarks Comparatifs (Buffer vs Standard Library)
```go
func BenchmarkBufferVsSlice(b *testing.B) {
    data := make([]byte, 1024)
    
    b.Run("Buffer", func(b *testing.B) {
        buf := kbuffer.NewBuffer(1024)
        b.ResetTimer()
        for i := 0; i < b.N; i++ {
            buf.Write(data)
            buf.Reset()
        }
    })
    
    b.Run("Slice", func(b *testing.B) {
        b.ResetTimer()
        for i := 0; i < b.N; i++ {
            slice := make([]byte, 0, 1024)
            slice = append(slice, data...)
            _ = slice[:0]  // "reset"
        }
    })
}
```

### 5. TESTS DE CONCURRENCE (Obligatoire pour code thread-safe)

```go
func TestConcurrentAccess(t *testing.T) {
    const numGoroutines = 100
    const opsPerGoroutine = 1000
    
    pool := kbuffer.NewPool(50)
    
    var wg sync.WaitGroup
    wg.Add(numGoroutines)
    
    // Canal pour collecter les erreurs
    errors := make(chan error, numGoroutines*10)
    
    for i := 0; i < numGoroutines; i++ {
        go func(id int) {
            defer wg.Done()
            
            for j := 0; j < opsPerGoroutine; j++ {
                buf, err := pool.Get()
                if err != nil {
                    errors <- fmt.Errorf("goroutine %d: Get() failed: %v", id, err)
                    return
                }
                
                // Utiliser le buffer
                data := []byte(fmt.Sprintf("data-%d-%d", id, j))
                buf.Write(data)
                
                // Le remettre dans le pool
                pool.Put(buf)
            }
        }(i)
    }
    
    wg.Wait()
    close(errors)
    
    // Vérifier qu'aucune erreur n'est survenue
    for err := range errors {
        t.Errorf("Concurrent test error: %v", err)
    }
}
```

### 6. VALIDATION DES PERFORMANCES (Pour packages kernel/)

#### Allocation Checking (OBLIGATOIRE pour pkg/kernel/*)
```go
func BenchmarkZeroAlloc(b *testing.B) {
    buf := kbuffer.NewBuffer(1024)
    data := make([]byte, 512)
    
    b.ResetTimer()
    b.ReportAllocs()  // ✅ OBLIGATOIRE - report le nombre d'allocations
    
    for i := 0; i < b.N; i++ {
        buf.Write(data)
        buf.Read(make([]byte, 512))
        buf.Reset()
    }
    
    // Le test échoue si allocs/op > 0
    // Vérification automatique via 'go test -bench . -benchmem'
}
```

### 7. HELPERS ET UTILITAIRES

#### Helpers pour réduire la duplication
```go
// testHelper.go (dans le même package de test)
func newTestBuffer(t *testing.T, size int) kbuffer.Buffer {
    t.Helper()  // ✅ OBLIGATOIRE pour les fonctions helper
    
    buf, err := kbuffer.NewBuffer(size)
    if err != nil {
        t.Fatalf("Failed to create test buffer: %v", err)
    }
    return buf
}

func fillBuffer(t *testing.T, buf kbuffer.Buffer, data []byte) {
    t.Helper()
    
    n, err := buf.Write(data)
    if err != nil {
        t.Fatalf("Failed to write to buffer: %v", err)
    }
    if n != len(data) {
        t.Fatalf("Partial write: got %d, want %d", n, len(data))
    }
}
```

## 🎯 VALIDATION AUTOMATIQUE

### Commandes de validation obligatoires
```bash
# Tests standard
go test ./...

# Tests avec race detection (obligatoire pour concurrence)
go test -race ./...

# Benchmarks avec allocation tracking
go test -bench . -benchmem ./...

# Coverage (≥99% pour kernel packages)
go test -cover ./...
```

### Métriques de succès pour pkg/kernel/*
- **Coverage**: ≥99% obligatoire
- **Allocations**: 0/op sur tous les chemins critiques  
- **Race conditions**: 0 détectée
- **Benchmarks**: Gain mesurable vs standard library