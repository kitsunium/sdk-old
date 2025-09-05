# INSTRUCTIONS TDD - PACKAGE KERNEL

**Projet**: SDK Kitsunium  
**Package**: pkg/kernel/kbuffer  
**Mode**: Test-Driven Development (TDD) avec approche incrémentale  
**Règles**: Suit strictement `.claude/rules/` pour toutes les décisions d'architecture

## 🚨 RÈGLES CRITIQUES

**OBLIGATOIRE**: Lire `.claude/rules/00-critical-architecture.md` - Ces règles ont priorité absolue sur toute autre règle.

## 🎯 MISSION DÉTAILLÉE

### Vue d'ensemble

Le package kbuffer est un système de gestion mémoire kernel ultra-optimisé conçu pour éliminer complètement les allocations dynamiques dans les chemins critiques. Il fournit des primitives de bas
niveau pour la manipulation directe de la mémoire avec une performance maximale.

### Architecture Technique

#### Buffer - Tampon Mémoire Haute Performance

- **Bloc mémoire contigu pré-alloué** de taille fixe (configurable)
- **Réutilisable à l'infini** sans allocation/désallocation
- **Opérations zero-copy** pour lectures/écritures
- **Accès direct via unsafe** pour éviter les vérifications de boundaries
- **Support du mode circulaire** pour streaming continu
- **Thread-safe** dans la version de base (mutex minimal)
- **Version unsafe lock-free** disponible si benchmarks montrent >30% de gain

#### Pool - Gestionnaire de Buffers

- **Réservoir pré-alloué** de N buffers (taille configurable)
- **Acquisition/libération O(1)** via free-list
- **Zero allocation** après initialisation
- **Statistiques internes minimales** (uniquement compteurs atomiques)
- **Stratégie LIFO** pour maximiser la localité cache
- **Protection contre les fuites** via finalizers optionnels en dev
- **Support de pools hiérarchiques** pour différentes tailles

### Cas d'Usage Cibles

1. **Parsers haute fréquence** - Traitement de millions de messages/sec
2. **Proxy/Gateway** - Transfert de données avec latence minimale
3. **Serialization** - Marshaling/Unmarshaling sans allocation
4. **Network I/O** - Buffers réutilisables pour sockets
5. **File I/O** - Lecture/écriture par blocs optimisés
6. **Message queues** - Ring buffers pour IPC haute performance

### Objectifs de Performance

- **Latence**: <10ns pour Get/Put sur Pool (P99)
- **Throughput**: >100M ops/sec sur CPU moderne
- **Allocations**: ZÉRO après warm-up
- **Cache misses**: Minimisés via pool LIFO et alignment
- **Contention**: Lock-free dans version unsafe pour scaling linéaire

## 📚 RÈGLES DE DÉVELOPPEMENT

**Voir `.claude/rules/` pour toutes les règles détaillées.**

## 🔄 APPROCHE TDD STRICTE

**Voir `.claude/rules/03-testing/` pour les règles de tests et TDD.**

## 📋 PHASES DE DÉVELOPPEMENT TDD

### ✅ Phase 1: Fondations du Package

**Objectif**: Définir les types de base et constantes

- [ ] Créer constants.go avec DefaultBufferSize, MaxBufferSize, DefaultPoolSize
- [ ] Exécuter `make fmt` sur constants.go
- [ ] Créer errors.go avec ErrBufferFull, ErrPoolEmpty, ErrInvalidSize
- [ ] Exécuter `make fmt` sur errors.go
- [ ] Créer interface.go avec interfaces Buffer/Pool + DefaultConfig()
- [ ] Exécuter `make fmt` sur interface.go
- [ ] Créer types.go avec Config{Size, PoolSize} et Options
- [ ] Exécuter `make fmt` sur types.go
- [ ] Exécuter `go build ./...` pour vérifier la compilation

### ✅ Phase 2: Buffer - Élément Atomique

**Objectif**: Implémenter le Buffer (plus petit élément réutilisable)

- [ ] Créer buffer_test.go avec TestNewBuffer et TestBufferCapacity
- [ ] Exécuter `go test ./...` et vérifier que les tests échouent
- [ ] Créer buffer.go avec struct buffer et NewBuffer(), Cap() minimaux
- [ ] Exécuter `make fmt` sur buffer.go
- [ ] Exécuter `go test ./...` et vérifier que TestNewBuffer et TestBufferCapacity passent
- [ ] Ajouter TestBufferReadWrite dans buffer_test.go
- [ ] Exécuter `go test ./...` et vérifier échec de TestBufferReadWrite
- [ ] Implémenter Read(), Write(), Len(), Bytes() dans buffer.go
- [ ] Exécuter `go test ./...` jusqu'à ce que TestBufferReadWrite passe
- [ ] Ajouter TestBufferReset dans buffer_test.go
- [ ] Exécuter `go test ./...` et vérifier échec de TestBufferReset
- [ ] Implémenter Reset(), Clear() dans buffer.go
- [ ] Exécuter `go test ./...` jusqu'à ce que tous les tests passent
- [ ] Ajouter BenchmarkBuffer dans buffer_test.go
- [ ] Exécuter `go test -bench . -benchmem` et vérifier 0 allocs/op

### ✅ Phase 3: Pool - Gestionnaire de Buffers

**Objectif**: Implémenter le Pool qui gère plusieurs Buffers

- [ ] Créer pool_test.go avec TestNewPool
- [ ] Exécuter `go test ./...` et vérifier que TestNewPool échoue
- [ ] Créer pool.go avec struct pool et NewPool(size int) minimal
- [ ] Exécuter `make fmt` sur pool.go
- [ ] Exécuter `go test ./...` et vérifier que TestNewPool passe
- [ ] Ajouter TestPoolGetPut dans pool_test.go
- [ ] Exécuter `go test ./...` et vérifier échec de TestPoolGetPut
- [ ] Implémenter Get(), Put(*buffer), Len(), Cap() dans pool.go
- [ ] Exécuter `go test ./...` jusqu'à ce que TestPoolGetPut passe
- [ ] Ajouter TestPoolConcurrent avec 100 goroutines dans pool_test.go
- [ ] Exécuter `go test -race ./...` et vérifier 0 race détectée
- [ ] Si races détectées, corriger pool.go avec synchronisation appropriée
- [ ] Ajouter BenchmarkPoolGetPut dans pool_test.go
- [ ] Exécuter `go test -bench GetPut` et vérifier latence <10ns

### ✅ Phase 4: Intégration et Benchmarks Consolidés

**Objectif**: Valider le système complet et consolider les benchmarks

- [ ] Créer integration_test.go avec TestRealWorldScenario (parsing JSON de 10MB)
- [ ] Exécuter `go test ./...` et vérifier que TestRealWorldScenario passe
- [ ] Ajouter BenchmarkBufferVsSlice dans buffer_test.go comparant Buffer vs []byte
- [ ] Exécuter `go test -bench BufferVsSlice` et vérifier gain >30%
- [ ] Ajouter BenchmarkPoolConcurrent dans pool_test.go avec 1000 goroutines
- [ ] Exécuter `go test -bench PoolConcurrent` et mesurer throughput >100M ops/sec
- [ ] Exécuter `go test -bench . -benchmem` et vérifier 0 allocs sur tous les chemins critiques
- [ ] Exécuter `make test` pour validation complète

### ✅ Phase 5: Optimisation et Version Unsafe

**Objectif**: Optimiser et créer version unsafe si gains significatifs

- [ ] Exécuter `go test -bench . -cpuprofile cpu.prof`
- [ ] Analyser avec `go tool pprof cpu.prof` et identifier les hotspots
- [ ] Exécuter `go build -gcflags="-m -m" ./...` pour identifier les escape allocations
- [ ] Si hotspot identifié, créer unsafe_buffer.go avec UnsafeBuffer
- [ ] Exécuter `make fmt` sur unsafe_buffer.go
- [ ] Créer unsafe_buffer_test.go avec tests adaptés pour UnsafeBuffer
- [ ] Exécuter `go test ./...` et vérifier que les tests unsafe passent
- [ ] Exécuter `go test -bench . -benchmem` pour comparer Safe vs Unsafe
- [ ] Si gain >30%, ajouter build tag `// +build unsafe` dans unsafe_buffer.go
- [ ] Documenter les risques et gains dans unsafe_buffer.go

### ✅ Phase 6: Finalisation et Documentation

**Objectif**: Package production-ready avec documentation complète

- [ ] Créer README.md avec sections: Purpose, API Reference, Usage Examples, Performance, Thread Safety
- [ ] Créer BUILD.bazel avec go_library et go_test targets
- [ ] Exécuter `bazel build //pkg/kernel/kbuffer:all` si Bazel disponible
- [ ] Exécuter `bazel test //pkg/kernel/kbuffer:all` si Bazel disponible
- [ ] Exécuter `make fmt` et vérifier aucune modification
- [ ] Exécuter `make test` et vérifier 100% PASS, 0 races, coverage ≥99%
- [ ] Créer examples/basic_usage.go avec exemple complet
- [ ] Exécuter `go run examples/basic_usage.go` pour validation
- [ ] Mettre à jour instruction.md section "État Actuel" avec résultats finaux

## 🎬 PROCHAINE ACTION

**Action**: Implémenter le package kbuffer selon les phases TDD

## 💻 COMMANDES DE VALIDATION

### Commandes essentielles

```bash
# 1. Formater le code (limite 150 caractères, gofmt, goimports)
make fmt

# 2. Lancer tous les tests (unitaires, race, coverage, benchmarks)
make test
```

## 📊 MÉTRIQUES DE SUCCÈS

| Métrique    | Cible               | Validation                 |
| ----------- | ------------------- | -------------------------- |
| Coverage    | ≥99% (kernel)       | `go test -cover`           |
| Allocations | 0/op obligatoire    | `go test -bench -benchmem` |
| Race-free   | 0 races             | `go test -race`            |
| Gain Unsafe | >30% pour justifier | Benchmarks comparatifs     |
| README.md   | 100% complet        | Documentation API complète |

## 🔄 ÉTAT ACTUEL

**Itération**: 0  
**Phase**: Initialisation  
**Composant**: Aucun  
**Prochain**: -

## 📝 TEMPLATE LOG D'ITÉRATION

```markdown
### Itération N - YYYY-MM-DD HH:MM

- **Phase TDD**: [Contracts|Tests|Implementation|Optimization]
- **Composant**: [interface|constants|buffer]
- **Versions implémentées**: [Safe only|Safe + Unsafe]
- **Actions**:
  - Créé: [fichiers créés]
  - Tests: X ajoutés (Y pass, Z fail)
  - Coverage: XX% sur buffer
- **Performance Safe**:
  - Read: Xns, Y allocs/op
  - Thread-safe: ✓ (race detector clean)
- **Performance Unsafe** (si applicable):
  - Read: Xns, Y allocs/op
  - Gain vs Safe: +XX%
  - Panic multithread: ✓ testé
- **Validation Bazel**:
  - Dev mode: [PASS|FAIL]
  - Prod mode: [PASS|FAIL]
- **Décision**: [Garder Safe only|Ajouter Unsafe car gain >30%]
- **Prochain**: [Action suivante]
```

## 📝 LOG DES ITÉRATIONS TDD

---
