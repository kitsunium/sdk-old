# Benchmark Stabilization Guide

## Problème identifié
Les benchmarks montrent des variations importantes (jusqu'à 300%) entre deux exécutions du même code, notamment sur:
- `Buffer_Concurrent`: 0.43ns vs 1.77ns (↑316%)
- `Workload_SmallWrites`: 138.1ns vs 272.2ns (↑97%)
- Tests concurrents en général

## Causes principales

### 1. **Tests concurrents instables**
Les benchmarks concurrents (`BenchmarkBuffer_Concurrent`, `LRU_ConcurrentGet`) sont très sensibles:
- Scheduling du CPU
- Garbage collection
- Contention entre goroutines
- État du système

### 2. **Configuration système**
- Multiple CPU cores actifs
- Throttling thermique
- Autres processus en arrière-plan
- Turbo Boost / fréquence dynamique

## Solutions recommandées

### 1. **Stabilisation des benchmarks Go**

```go
// Ajouter dans les benchmarks critiques:
func BenchmarkBuffer_Concurrent(b *testing.B) {
    // Forcer un seul CPU pour la stabilité
    runtime.GOMAXPROCS(1)
    defer runtime.GOMAXPROCS(runtime.NumCPU())
    
    // Forcer GC avant le benchmark
    runtime.GC()
    
    // Augmenter le temps minimum d'exécution
    b.SetParallelism(1)  // Limiter le parallélisme
    
    b.RunParallel(func(pb *testing.PB) {
        // ... benchmark code
    })
}
```

### 2. **Configuration Bazel pour benchmarks stables**

Créer `.bazelrc.benchmark`:
```bash
# Configuration pour benchmarks stables
build:benchmark --test_output=all
build:benchmark --test_arg=-test.cpu=1
build:benchmark --test_arg=-test.benchtime=10s
build:benchmark --test_arg=-test.count=3
build:benchmark --test_env=GOMAXPROCS=1
build:benchmark --local_test_jobs=1
build:benchmark --jobs=1
```

### 3. **Script de benchmark avec moyennes**

```python
# Modifier bench_manager.py pour faire plusieurs runs
def run_benchmark_stable(self, target: str, runs: int = 5) -> Dict:
    """Run benchmark multiple times and return median values."""
    results = []
    for i in range(runs):
        print(f"  Run {i+1}/{runs}...")
        output = self.run_single_benchmark(target)
        if output:
            results.append(self.parse_benchmark_output(output))
    
    # Calculer la médiane pour chaque métrique
    return self.compute_median_results(results)
```

### 4. **Makefile amélioré**

```makefile
# Mode benchmark stable (plus lent mais plus fiable)
bench/stable:
	@echo "$(YELLOW)▶ Running stable benchmarks (5 runs, 1 CPU)...$(NC)"
	@GOMAXPROCS=1 python3 scripts/bench_manager.py save --stable --runs=5
```

### 5. **Isolation système (macOS)**

```bash
# Désactiver Turbo Boost temporairement
sudo pmset -a disablesleep 1
sudo nvram boot-args="serverperfmode=1"

# Limiter les processus en arrière-plan
sudo nice -n -20 make bench/stable
```

## Recommandations

### Pour des benchmarks fiables:

1. **Court terme** (rapide):
   - Utiliser `GOMAXPROCS=1` pour limiter à 1 CPU
   - Augmenter `-benchtime=10s` pour plus de stabilité
   - Faire 3-5 runs et prendre la médiane

2. **Moyen terme** (recommandé):
   - Implémenter le mode `bench/stable` dans le Makefile
   - Modifier les benchmarks concurrents pour être déterministes
   - Ajouter des warmup runs avant les mesures

3. **Long terme** (idéal):
   - Séparer benchmarks "micro" (performance pure) et "macro" (comportement réel)
   - Utiliser des benchmarks statistiques avec intervalles de confiance
   - CI dédié pour benchmarks sur machine isolée

## Commande recommandée immédiatement

```bash
# Pour des résultats plus stables tout de suite:
GOMAXPROCS=1 bazel test //pkg/... \
  --test_arg=-test.bench=. \
  --test_arg=-test.benchtime=10s \
  --test_arg=-test.count=3 \
  --test_output=all \
  --test_env=GOMAXPROCS=1 \
  --local_test_jobs=1
```

## Benchmarks à stabiliser en priorité

1. **Très instables** (à corriger):
   - `Buffer_Concurrent`
   - `LRU_ConcurrentGet/concurrency=*`
   - `ComparisonConcurrentReadHeavy/*`
   - `Workload_SmallWrites`

2. **Moyennement instables** (à surveiller):
   - `Pool_Concurrent`
   - `ShardedCache_ConcurrentSet`
   - Tests avec allocation mémoire

3. **Stables** (référence):
   - `Buffer_Write` (sans concurrence)
   - `Result/*` (très simple)
   - Tests séquentiels