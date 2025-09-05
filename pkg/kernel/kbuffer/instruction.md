# INSTRUCTIONS TDD - PACKAGE KERNEL

**Projet**: SDK Kitsunium  
**Package**: pkg/kernel/kbuffer  
**Mode**: Test-Driven Development (TDD) avec approche incrémentale  
**Règles**: Suit strictement `.claude/rules/` pour toutes les décisions d'architecture

## 🎯 MISSION

Le package kbuffer est une bibliothèque de gestion mémoire minimaliste pour opérations kernel haute performance.

### Concept Fondamental

Deux composants simples qui travaillent ensemble :

- **Buffer** - Un bloc mémoire fixe réutilisable qui agit comme un tampon circulaire
- **Pool** - Un réservoir de buffers pré-alloués pour éviter le coût des allocations/désallocations

C'est essentiellement un système de recyclage mémoire ultra-léger pour les hot paths du kernel.

## 📚 RÈGLES DE DÉVELOPPEMENT

Ce package suit STRICTEMENT les règles définies dans `.claude/rules/`:

### Architecture (`.claude/rules/01-architecture/`)

- **Interfaces** → `01-interfaces.md`: Contrats d'API et types publics
- **Structs** → `02-structs.md`: Optimisation de layout mémoire
- **Organisation** → `03-file-organization.md`: Une type par fichier, structure plate
- **Patterns** → `04-design-patterns.md`: Patterns architecturaux kernel

### Implémentation (`.claude/rules/02-implementation/`)

- **Safe/Unsafe** → `01-safe-unsafe-pattern.md`: Version safe obligatoire, unsafe si gain >30%
- **Concurrence** → `02-concurrency-detection.md`: Détection runtime en dev
- **Mémoire** → `03-memory-optimization.md`: Cache-line alignment, zero-alloc
- **Erreurs** → `04-error-handling.md`: Patterns d'erreur kernel

### Testing (`.claude/rules/03-testing/`)

- **Unit Tests** → `01-unit-tests.md`: Test pour chaque fichier
- **Benchmarks** → `02-benchmarks.md`: Fichier consolidé `*_bench_test.go`
- **Integration** → `03-integration-tests.md`: Tests end-to-end
- **Coverage** → `04-coverage-requirements.md`: 95% minimum pour kernel

### Conventions (`.claude/rules/04-conventions/`)

- **Naming** → `01-naming-conventions.md`: Conventions de nommage Go
- **Documentation** → `02-documentation-standards.md`: 100% des exports documentés
- **Organisation** → `03-code-organization.md`: Structure du code

### Commandes (`.claude/rules/05-commands/`)

- **Development** → `01-development.md`: Commandes de dev
- **Validation** → `02-validation.md`: Tests et validation
- **Production** → `03-production-builds.md`: Build de production

## 🔄 APPROCHE TDD STRICTE

### ⚠️ ORDRE IMPÉRATIF DES PHASES

**CRITICAL**: Suivre l'ordre exact. Chaque phase valide la précédente.

### Phase 0: Vérification Préalable

1. **TOUJOURS** lire les fichiers existants avant toute création
2. **JAMAIS** redéclarer des fonctions, types ou constantes existants
3. **FORMATER** avec `make fmt` après CHAQUE création de fichier
4. **COMPILER** avec `go build` pour vérifier (pas `make test` car TDD = tests avant code)

### Principes Fondamentaux

1. **Vérifier l'existant** - Toujours checker ce qui existe avec Read avant de créer
2. **Interfaces & Constantes d'abord** - Définir le contrat complet avant toute implémentation
3. **Tests avant code** - Écrire les tests (TDD) qui définissent le comportement attendu
4. **Implémentation minimale** - Juste assez de code pour passer les tests
5. **Thread-safe par défaut** - Toute implémentation de base DOIT être thread-safe
6. **Version unsafe optionnelle** - Seulement si benchmarks montrent >30% de gain
7. **Validation continue** - `make fmt && go build` après chaque fichier (TDD: tests échouent jusqu'à implémentation)
8. **Benchmarks consolidés** - Un seul fichier `*_bench_test.go` pour tous les benchmarks

## 📝 RÈGLES DE FORMATAGE

### Limite de ligne: 150 caractères

- **TOUTES** les lignes de code doivent faire 150 caractères maximum
- Cela inclut les commentaires, les signatures de fonctions, etc.
- Configuré dans `.golangci.yml` et `.editorconfig`

### Comment diviser les lignes longues

```go
// Commentaires longs: divisez sur plusieurs lignes
// Ceci est un commentaire très long qui dépasse la limite de 150 caractères
// et doit donc être divisé sur plusieurs lignes pour respecter la règle

// Signatures de fonctions: mettez les paramètres sur plusieurs lignes
func ProcessComplexData(
    ctx context.Context,
    data []byte,
    options ProcessOptions,
    callback func(result Result) error,
) (*Response, error) {
    // ...
}

// Chaînes longues: utilisez la concaténation
message := "Ceci est un message très long qui dépasse " +
    "la limite de 150 caractères et doit être divisé " +
    "sur plusieurs lignes"
```

## 📂 STRUCTURE KERNEL STANDARD

Selon `.claude/rules/01-architecture/03-file-organization.md`:

```bash
pkg/kernel/kbuffer/
├── interface.go            # [1] TOUTES les interfaces, Config struct, DefaultConfig()
├── constants.go            # [2] TOUTES les constantes du package
├── errors.go              # [3] Types et variables d'erreur (optionnel)
├── options.go             # [4] Options de configuration (optionnel)
├── buffer.go              # [5] Implémentation Buffer safe
├── buffer_test.go         # [6] Tests unitaires Buffer
├── pool.go                # [7] Implémentation Pool (pas de DefaultConfig ici!)
├── pool_test.go           # [8] Tests unitaires Pool
├── unsafe_buffer.go       # [9] Version unsafe (si gain >30%)
├── unsafe_buffer_test.go  # [10] Tests unsafe buffer
├── kbuffer_bench_test.go  # [11] TOUS les benchmarks consolidés
└── BUILD.bazel            # [12] Configuration build
```

### Règles Critiques d'Organisation

1. **Un type par fichier** - JAMAIS plusieurs types dans le même fichier
2. **DefaultConfig() dans interface.go UNIQUEMENT** - Pas de duplication
3. **Benchmarks consolidés** - Tous dans `kbuffer_bench_test.go`
4. **Tests appariés** - Chaque `*.go` a son `*_test.go`

## 📋 PHASES DE DÉVELOPPEMENT TDD

### ✅ Phase 1: Définition des Contrats

**Objectif**: Établir l'API publique complète et les contraintes du système

**Actions OBLIGATOIRES avant création**:

1. Vérifier avec `ls pkg/kernel/kbuffer/` ce qui existe déjà
2. Lire TOUS les fichiers existants avec Read
3. Ne JAMAIS redéclarer ce qui existe déjà

- [ ] Créer interface.go avec toutes les interfaces documentées + DefaultConfig()
- [ ] Créer constants.go avec toutes les constantes documentées
- [ ] Documentation complète des attentes de performance et contraintes
- [ ] Vérifier compilation: `go build ./pkg/kernel/kbuffer`

### ✅ Phase 2: Tests Premier Composant (Buffer)

**Objectif**: Définir le comportement du Buffer via les tests

- [ ] Créer buffer_test.go avec cas nominaux
- [ ] Ajouter cas d'erreur dans buffer_test.go
- [ ] Ajouter benchmarks dans buffer_test.go
- [ ] Implémenter buffer.go pour passer les tests

### ✅ Phase 3: Tests Composants Additionnels (Pool)

**Objectif**: Définir et implémenter le Pool de buffers

- [ ] Créer pool_test.go avec cas nominaux
- [ ] Ajouter cas d'erreur dans pool_test.go
- [ ] Ajouter benchmarks dans pool_test.go
- [ ] Implémenter pool.go pour passer les tests

### ✅ Phase 4: Intégration

**Objectif**: Valider les interactions entre Buffer et Pool

- [ ] Tests d'intégration Buffer/Pool
- [ ] Créer mocks_test.go si nécessaire
- [ ] Valider cohérence du système complet

### ✅ Phase 5: Optimisation

**Objectif**: Améliorer les performances mesurées

- [ ] Profiler avec les benchmarks existants
- [ ] Identifier et optimiser les hot paths
- [ ] Ajouter directives compiler si nécessaire
- [ ] Créer versions unsafe si gains >30%

### ✅ Phase 6: Finalisation

**Objectif**: Préparer pour la production

- [ ] Créer BUILD.bazel
- [ ] Valider coverage > 95%
- [ ] Documentation finale
- [ ] Validation performance globale

## 🎬 PROCHAINE ACTION

**Action**: Vérifier l'existant et corriger les erreurs de compilation

1. Lancer `go build ./pkg/kernel/kbuffer` pour identifier les erreurs
2. Corriger les redéclarations entre fichiers
3. S'assurer que DefaultConfig() est UNIQUEMENT dans interface.go
4. Valider que chaque type est dans son propre fichier

## 📚 RÉFÉRENCES AUX RÈGLES

### Architecture et Organisation

- `.claude/rules/01-architecture/01-interfaces.md` - Design des interfaces
- `.claude/rules/01-architecture/02-structs.md` - Optimisation des structs
- `.claude/rules/01-architecture/03-file-organization.md` - Organisation des fichiers
- `.claude/rules/01-architecture/04-design-patterns.md` - Patterns architecturaux

### Implémentation

- `.claude/rules/02-implementation/01-safe-unsafe-pattern.md` - Pattern safe/unsafe
- `.claude/rules/02-implementation/02-concurrency-detection.md` - Détection concurrence
- `.claude/rules/02-implementation/03-memory-optimization.md` - Optimisation mémoire
- `.claude/rules/02-implementation/04-error-handling.md` - Gestion d'erreurs

### Testing

- `.claude/rules/03-testing/01-unit-tests.md` - Tests unitaires
- `.claude/rules/03-testing/02-benchmarks.md` - Benchmarks
- `.claude/rules/03-testing/03-integration-tests.md` - Tests d'intégration
- `.claude/rules/03-testing/04-coverage-requirements.md` - Couverture de code

## 💻 COMMANDES DE VALIDATION

### Les 2 commandes essentielles

```bash
# 1. Formater le code (limite 150 caractères, gofmt, goimports)
make fmt

# 2. Lancer tous les tests (unitaires, race, coverage, benchmarks)
make test
```

### Workflow TDD (Test-Driven Development)

```bash
# 1. Créer les TESTS d'abord (buffer_test.go)
make fmt
go build ./pkg/kernel/kbuffer  # Compile mais tests échouent (normal!)

# 2. Créer l'IMPLÉMENTATION ensuite (buffer.go)
make fmt
go build ./pkg/kernel/kbuffer  # Doit compiler

# 3. Faire passer les tests
make test  # Maintenant les tests doivent passer ✅

# En TDD: Tests rouges 🔴 → Code → Tests verts ✅
```

### Commandes spécifiques (si besoin)

```bash
# Compilation uniquement
go build ./pkg/kernel/kbuffer

# Tests avec coverage détaillé
go test -v -cover ./pkg/kernel/kbuffer

# Benchmarks uniquement
go test -bench=. -benchmem ./pkg/kernel/kbuffer

# Race detection
go test -race ./pkg/kernel/kbuffer
```

## 🚀 CHECKLIST DE VALIDATION

### Avant de créer un fichier

- [ ] Vérifier qu'il n'existe pas déjà
- [ ] Lire les fichiers existants pour éviter les redéclarations
- [ ] Confirmer l'emplacement selon `.claude/rules/01-architecture/03-file-organization.md`

### Après création de CHAQUE fichier

- [ ] Exécuter `make fmt` pour formater le code
- [ ] Vérifier la compilation : `go build ./pkg/kernel/kbuffer`
- [ ] Si erreurs de compilation → Corriger immédiatement
- [ ] Vérifier qu'il n'y a pas de redéclaration

### Validation TDD (après implémentation)

- [ ] Les tests doivent être écrits AVANT le code (TDD)
- [ ] Une fois le code implémenté → `make test` doit passer
- [ ] Si tests échouent → Corriger le code jusqu'à ce qu'ils passent

### Validation finale du package complet

- [ ] `make fmt` sur tout le package
- [ ] `make test` doit passer à 100%
- [ ] Coverage ≥ 95%
- [ ] Tous les benchmarks dans `kbuffer_bench_test.go`
- [ ] Si version unsafe → Gain documenté >30%

## 📊 MÉTRIQUES DE SUCCÈS

| Métrique     | Cible               | Validation                    |
| ------------ | ------------------- | ----------------------------- |
| Coverage     | ≥95%                | `go test -cover`              |
| Race-free    | 0 races             | `go test -race`               |
| Allocations  | 0/op idéal          | `go test -bench -benchmem`    |
| Gain Unsafe  | >30% pour justifier | Benchmarks comparatifs        |
| Compilation  | 0 erreurs           | `go build`                    |
| Qualité Code | Grade A Codacy      | `codacy-analysis-cli analyze` |
| Code Smells  | 0                   | Analyse Codacy                |
| Complexité   | <10 par fonction    | Analyse Codacy                |

## 🔄 ÉTAT ACTUEL

**Itération**: 0  
**Phase**: Initialisation  
**Composant**: Aucun  
**Prochain**: interface.go

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
