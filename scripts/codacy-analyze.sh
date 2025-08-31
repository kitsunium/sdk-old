#!/bin/bash
# Script pour lancer l'analyse de code pour projet Go/Shell

set -e

# Couleurs pour output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

echo -e "${CYAN}🔍 Code Analysis for Go/Shell Project${NC}"
echo "========================================="

# Créer le dossier de sortie si nécessaire
mkdir -p .codacy/out

# Analyse Go avec golangci-lint (incluant gosec)
echo -e "\n${YELLOW}🔧 Analyse Go avec golangci-lint...${NC}"
if command -v golangci-lint &> /dev/null; then
    if golangci-lint run --out-format json > .codacy/out/golangci.json 2>&1; then
        echo -e "${GREEN}✅ Analyse Go terminée sans erreurs${NC}"
    else
        echo -e "${YELLOW}⚠️  Problèmes détectés dans le code Go${NC}"
        golangci-lint run --out-format colored-line-number || true
    fi
else
    echo -e "${RED}❌ golangci-lint n'est pas installé${NC}"
    echo "Installation: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"
fi

# Analyse des scripts Shell avec shellcheck
echo -e "\n${YELLOW}🐚 Analyse Shell avec shellcheck...${NC}"
if command -v shellcheck &> /dev/null; then
    shell_files=$(find . -name "*.sh" -not -path "./vendor/*" -not -path "./bazel-*/*" 2>/dev/null)
    if [ -n "$shell_files" ]; then
        if shellcheck -f json $shell_files > .codacy/out/shellcheck.json 2>&1; then
            echo -e "${GREEN}✅ Analyse Shell terminée sans erreurs${NC}"
        else
            echo -e "${YELLOW}⚠️  Problèmes détectés dans les scripts Shell${NC}"
            shellcheck $shell_files || true
        fi
    else
        echo "Aucun fichier shell trouvé"
    fi
else
    echo -e "${YELLOW}⚠️  shellcheck n'est pas installé${NC}"
    echo "Installation: brew install shellcheck (macOS) ou apt-get install shellcheck (Linux)"
fi

# Analyse de sécurité avec Codacy/Semgrep
echo -e "\n${YELLOW}🔒 Analyse de sécurité avec semgrep...${NC}"
if command -v codacy-cli &> /dev/null; then
    if codacy-cli analyze --tool semgrep --format sarif -o .codacy/out/semgrep.sarif; then
        echo -e "${GREEN}✅ Analyse de sécurité terminée${NC}"
    else
        echo -e "${YELLOW}⚠️  Problèmes de sécurité potentiels détectés${NC}"
    fi
else
    # Fallback sur semgrep direct si codacy-cli n'est pas disponible
    if command -v semgrep &> /dev/null; then
        echo "Utilisation de semgrep directement..."
        semgrep --config=auto --json -o .codacy/out/semgrep.json . || true
        echo -e "${GREEN}✅ Analyse de sécurité terminée${NC}"
    else
        echo -e "${YELLOW}⚠️  Ni codacy-cli ni semgrep ne sont installés${NC}"
    fi
fi

# Analyse spécifique de sécurité Go avec gosec
echo -e "\n${YELLOW}🔐 Analyse de sécurité Go avec gosec...${NC}"
if command -v gosec &> /dev/null; then
    if gosec -fmt json -out .codacy/out/gosec.json ./... 2>/dev/null; then
        echo -e "${GREEN}✅ Analyse gosec terminée${NC}"
    else
        echo -e "${YELLOW}⚠️  Vulnérabilités potentielles détectées${NC}"
        gosec -fmt text ./... 2>&1 | head -50 || true
    fi
else
    echo -e "${YELLOW}ℹ️  gosec est intégré dans golangci-lint${NC}"
fi

# Résumé
echo -e "\n${GREEN}📊 Analyse complète terminée!${NC}"
echo "Rapports générés dans .codacy/out/"
echo ""
echo "Pour voir les résultats:"
echo "  - Rapports JSON: ls -la .codacy/out/*.json"
echo "  - Rapport SARIF: cat .codacy/out/*.sarif"
echo ""
echo "Commandes directes:"
echo "  - golangci-lint run               # Analyse Go complète"
echo "  - shellcheck scripts/*.sh          # Vérifier les scripts shell"
echo "  - gosec ./...                      # Analyse de sécurité Go"
echo "  - semgrep --config=auto .          # Analyse de sécurité générale"

# Stats si disponibles
if [ -f .codacy/out/golangci.json ]; then
    issues_count=$(grep -c '"Issue"' .codacy/out/golangci.json 2>/dev/null || echo "0")
    echo -e "\n${CYAN}📈 Statistiques:${NC}"
    echo "  - Issues Go détectées: $issues_count"
fi