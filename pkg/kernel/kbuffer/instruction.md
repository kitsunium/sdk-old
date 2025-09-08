# INSTRUCTIONS TDD - PACKAGE KERNEL

**Projet**: SDK Kitsunium  
**Package**: pkg/kernel/kbuffer  
**Mode**: Test-Driven Development (TDD) avec approche incrémentale  
**Règles**: Suit strictement `.claude/rules/` pour toutes les décisions d'architecture

## 🚨 RÈGLES CRITIQUES

**OBLIGATOIRE**: Lire `.claude/rules/00-critical-architecture.md` - Ces règles ont priorité absolue.

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
