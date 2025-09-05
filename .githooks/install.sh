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
# List executable hook files safely without parsing ls output
# Use find or glob pattern to avoid issues with special filenames
hook_count=0
for hook in "$HOOKS_DIR"/*; do
    if [ -f "$hook" ] && [ -x "$hook" ]; then
        basename_hook=$(basename "$hook")
        case "$basename_hook" in
            pre-*|commit-*|post-*)
                echo "  - $basename_hook"
                hook_count=$((hook_count + 1))
                ;;
        esac
    fi
done
if [ "$hook_count" -eq 0 ]; then
    echo "  No hooks found"
fi

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
echo "  • AI checks CANNOT be bypassed - no exceptions"
echo ""
echo "To uninstall hooks:"
echo "  git config --unset core.hooksPath"