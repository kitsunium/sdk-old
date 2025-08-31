# INSTRUCTIONS DÉVELOPPEMENT - PACKAGE KBUFFER

**Projet**: SDK Kitsunium  
**Package**: pkg/kernel/kbuffer  
**Mode**: Développement Itératif Autonome

## 🎯 MISSION

Créer/Améliorer le package un gestion de pool de buffers haute performance en Go
avec zéro allocation après initialisation.

## 📂 CONTEXTE PROJET

```
sdk/
└── pkg/
    └── kernel/
        └── kbuffer/          # <-- FOCUS ICI
```

## 🔄 ÉTAT ACTUEL

<!-- [AUTO-UPDATE] Cette section est mise à jour automatiquement à chaque itération -->

**Itération**: 0  
**Statut**: Non démarré  
**Dernière action**: Initialisation du fichier instructions

## 📋 TÂCHES RESTANTES

<!-- [AUTO-UPDATE] Cocher les tâches au fur et à mesure -->

- [ ] Créer la structure de base du package
- [ ] Implémenter BufferPool avec sync.Pool
- [ ] Ajouter méthode Get() pour obtenir un buffer
- [ ] Ajouter méthode Put() pour remettre un buffer
- [ ] Implémenter Reset() pour nettoyer les buffers
- [ ] Ajouter support pour différentes tailles de buffers
- [ ] Créer tests unitaires basiques
- [ ] Ajouter tests de concurrence
- [ ] Implémenter benchmarks
- [ ] Optimiser pour zéro allocation
- [ ] Ajouter métriques de monitoring (hits/misses)
- [ ] Créer documentation GoDoc
- [ ] Écrire README avec exemples
- [ ] Valider performance < 10ns par opération

## 🎬 PROCHAINE ACTION

<!-- [AUTO-UPDATE] Une seule action claire et atomique -->

**Action**: Analyser le package si il existe en profondeur pour identifier
itérer sur toute les taches et voir si elle sont déjà remplis ou non.

## 💻 COMMANDES DE VALIDATION

```bash
# À exécuter après chaque modification
Make test
```

## ✅ CRITÈRES DE SUCCÈS

- Tests passent: `PASS`
- Coverage > 80%
- Benchmarks: 0 allocs/op après warm-up
- Get/Put < 10ns

## 📝 LOG DES ITÉRATIONS

<!-- [AUTO-UPDATE] Ajouter une entrée à chaque itération -->

### Itération 0 - [DATE]

- **Action**: Initialisation
- **Résultat**: Fichier instructions créé
- **Tests**: N/A
- **Prochain**: Créer structure de base

---

## 🤖 INSTRUCTIONS POUR CLAUDE CODE

### À CHAQUE ITÉRATION

1. **LIRE** l'état actuel et la prochaine action
2. **IMPLÉMENTER** uniquement la prochaine action (une seule chose à la fois)
3. **TESTER** avec les commandes de validation
4. **METTRE À JOUR** ce fichier:
   - Cocher la tâche complétée dans TÂCHES RESTANTES
   - Incrémenter le numéro d'itération dans ÉTAT ACTUEL
   - Définir la nouvelle PROCHAINE ACTION
   - Ajouter une entrée dans LOG DES ITÉRATIONS avec les résultats
5. **COMMIT** avec message descriptif:
   `feat(kbuffer): [description de l'action]`

### RÈGLES IMPORTANTES

- **UNE SEULE ACTION** par itération (petit pas incrémental)
- **TOUJOURS TESTER** avant de valider
- **NE PAS OPTIMISER** prématurément
- **DOCUMENTER** au fur et à mesure
- Si une action échoue, l'indiquer dans le log et définir une action corrective

### EXEMPLE DE MISE À JOUR APRÈS ITÉRATION

```markdown
## 🔄 ÉTAT ACTUEL

**Itération**: 1  
**Statut**: Structure de base créée  
**Dernière action**: Création BufferPool interface et struct de base

## 📋 TÂCHES RESTANTES

- [x] Créer la structure de base du package
- [ ] Implémenter BufferPool avec sync.Pool ...

## 🎬 PROCHAINE ACTION

**Action**: Implémenter BufferPool avec sync.Pool backend

## 📝 LOG DES ITÉRATIONS

### Itération 1 - 2025-01-26 14:30

- **Action**: Créer structure de base
- **Résultat**: Fichiers kbuffer.go créé avec interface et struct
- **Tests**: Compilation OK
- **Prochain**: Ajouter sync.Pool
```

## 🚨 PROBLÈMES CONNUS

<!-- [AUTO-UPDATE] Lister les problèmes rencontrés et leurs solutions -->

- Aucun pour le moment

## 💡 DÉCISIONS D'ARCHITECTURE

<!-- [AUTO-UPDATE] Noter les choix importants -->

- Utiliser sync.Pool pour la gestion sous-jacente
- Buffers de taille fixe par pool
- Reset automatique au Put()

## 📊 MÉTRIQUES ACTUELLES

<!-- [AUTO-UPDATE] Après chaque benchmark -->

**Get()**: - ns/op  
**Put()**: - ns/op  
**Allocations**: - allocs/op  
**Coverage**: 0%

---

**PROMPT DE DÉMARRAGE**: "Lis instructions.md. Exécute la PROCHAINE ACTION.
Teste. Mets à jour le fichier instructions.md avec les résultats. Une seule
action à la fois."
