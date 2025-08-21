#!/bin/bash
# Script d'installation des hooks Git
# Objectif : configurer Git pour utiliser les hooks dans .githooks/

# Active le mode strict :
# -e : arrête le script si une commande échoue
set -e

# Affiche un message de bienvenue
echo "🔧 Installing Git hooks..."

# Récupère le répertoire du script
# $0 : nom du script en cours d'exécution
# dirname : extrait le répertoire du chemin
# cd && pwd : se déplace dans le répertoire et affiche le chemin absolu
HOOKS_DIR="$(cd "$(dirname "$0")" && pwd)"

# Configure Git pour utiliser notre répertoire de hooks personnalisé
# core.hooksPath : paramètre Git qui définit où chercher les hooks
# Au lieu du répertoire par défaut .git/hooks, Git utilisera .githooks/
git config core.hooksPath "$HOOKS_DIR"

# Affiche le chemin configuré pour confirmation
echo "✅ Git hooks path set to: $HOOKS_DIR"

# Rend tous les fichiers de hooks exécutables
# chmod +x : ajoute la permission d'exécution
# *.sh : tous les fichiers .sh
# pre-* commit-* : tous les hooks (pre-commit, pre-push, commit-msg, etc.)
echo "📝 Making hook scripts executable..."
chmod +x "$HOOKS_DIR"/*.sh 2>/dev/null || true  # Ignore les erreurs si pas de .sh
chmod +x "$HOOKS_DIR"/pre-* 2>/dev/null || true  # Hooks pre-*
chmod +x "$HOOKS_DIR"/commit-* 2>/dev/null || true  # Hooks commit-*
chmod +x "$HOOKS_DIR"/post-* 2>/dev/null || true  # Hooks post-* (si présents)

# Liste les hooks installés pour confirmation
echo ""
echo "📋 Installed hooks:"
# ls -la : liste détaillée avec permissions
# grep -E : expression régulière étendue
# Cherche les fichiers exécutables qui sont des hooks Git
ls -la "$HOOKS_DIR" | grep -E "^-..x.*\s+(pre-|commit-|post-)" | awk '{print "  - " $NF}' || echo "  No hooks found"

# Instructions finales pour l'utilisateur
echo ""
echo "✨ Git hooks installation complete!"
echo ""
echo "The following hooks are now active:"
echo "  • pre-commit: Runs formatting and tests before each commit"
echo "  • commit-msg: Validates commit message format (conventional commits)"
echo "  • pre-push: Blocks AI-related content from being pushed"
echo ""
echo "To bypass hooks temporarily (not recommended):"
echo "  • Skip pre-commit: git commit --no-verify"
echo "  • Skip pre-push with AI: ALLOW_AI_MENTIONS=1 git push"
echo ""
echo "To uninstall hooks:"
echo "  git config --unset core.hooksPath"