#!/usr/bin/env python3
"""
Claude Iterative Development Script
Usage: python claude-dev.py instructions.md
"""

import subprocess
import sys
import time
import re
import os
import argparse
import json
from datetime import datetime
from pathlib import Path
from typing import Optional, Dict, Any
import shutil

# ANSI color codes
class Colors:
    BLUE = '\033[94m'
    GREEN = '\033[92m'
    YELLOW = '\033[93m'
    RED = '\033[91m'
    BOLD = '\033[1m'
    END = '\033[0m'

class ClaudeDevRunner:
    def __init__(self, instructions_file: str, config: Dict[str, Any]):
        self.instructions_file = Path(instructions_file)
        self.config = config
        self.log_dir = Path(config['log_dir'])
        self.backup_dir = Path(config['backup_dir'])
        self.iteration = 0
        self.start_time = datetime.now()
        self.timeout_count = 0  # Track consecutive timeouts
        self.error_count = 0    # Track consecutive errors
        self.last_successful_iteration = 0
        
        # Create directories
        self.log_dir.mkdir(exist_ok=True)
        self.backup_dir.mkdir(exist_ok=True)
        
        # Backup original instructions
        self._backup_instructions()
    
    def _print(self, message: str, color: str = ""):
        """Print with optional color"""
        print(f"{color}{message}{Colors.END}")
    
    def _backup_instructions(self):
        """Backup the instructions file"""
        timestamp = datetime.now().strftime("%Y%m%d_%H%M%S")
        backup_path = self.backup_dir / f"instructions_{timestamp}.md"
        shutil.copy2(self.instructions_file, backup_path)
        
        # Keep original if it doesn't exist
        original_path = self.backup_dir / "instructions_original.md"
        if not original_path.exists():
            shutil.copy2(self.instructions_file, original_path)
    
    def _read_instructions(self) -> str:
        """Read the instructions file"""
        return self.instructions_file.read_text(encoding='utf-8')
    
    def _get_prompt(self) -> str:
        """Extract or generate the prompt from instructions"""
        content = self._read_instructions()
        
        # Look for PROMPT DE DÉMARRAGE section
        prompt_match = re.search(r'PROMPT DE DÉMARRAGE[:\s]*\n["\']?(.+?)["\']?\s*$', 
                                 content, re.MULTILINE)
        if prompt_match:
            return prompt_match.group(1).strip('"\'')
        
        # Default prompt
        return (f"Lis {self.instructions_file.name}. "
                f"Exécute la PROCHAINE ACTION. "
                f"Teste. "
                f"Mets à jour le fichier {self.instructions_file.name} avec les résultats. "
                f"Une seule action à la fois.")
    
    def _get_status(self) -> Dict[str, Any]:
        """Parse current status from instructions"""
        content = self._read_instructions()
        status = {
            'iteration': 0,
            'completed_tasks': 0,
            'total_tasks': 0,
            'next_action': 'Unknown',
            'is_complete': False
        }
        
        # Extract iteration number
        iter_match = re.search(r'Itération[:\s]*(\d+)', content, re.IGNORECASE)
        if iter_match:
            status['iteration'] = int(iter_match.group(1))
        
        # Count tasks
        status['total_tasks'] = len(re.findall(r'^- \[[ x]\]', content, re.MULTILINE))
        status['completed_tasks'] = len(re.findall(r'^- \[x\]', content, re.MULTILINE))
        
        # Get next action
        action_match = re.search(r'PROCHAINE ACTION[:\s]*\n.*?Action[:\s]*(.+?)$', 
                                 content, re.MULTILINE | re.IGNORECASE)
        if action_match:
            status['next_action'] = action_match.group(1).strip()
        
        # Check if complete
        complete_patterns = [
            r'\[x\].*Valider performance',
            r'Statut.*Complété',
            r'MISSION.*COMPLÈTE',
            r'Production ready'
        ]
        status['is_complete'] = any(re.search(p, content, re.IGNORECASE) 
                                    for p in complete_patterns)
        
        # Calculate percentage
        if status['total_tasks'] > 0:
            status['percentage'] = (status['completed_tasks'] / status['total_tasks']) * 100
        else:
            status['percentage'] = 0
        
        return status
    
    def _run_claude_code(self, prompt: str) -> tuple[bool, str, str]:
        """Execute claude with the prompt. Returns (success, output, error_type)"""
        timestamp = datetime.now().strftime("%Y%m%d_%H%M%S")
        log_file = self.log_dir / f"iteration_{self.iteration}_{timestamp}.log"
        
        try:
            # Run claude with --print flag for non-interactive mode
            result = subprocess.run(
                ['claude', '--print', prompt],
                capture_output=True,
                text=True,
                timeout=self.config['timeout']
            )
            
            # Save logs
            with open(log_file, 'w') as f:
                f.write(f"=== Iteration {self.iteration} - {timestamp} ===\n")
                f.write(f"Prompt: {prompt}\n")
                f.write(f"Return Code: {result.returncode}\n")
                f.write("\n--- STDOUT ---\n")
                f.write(result.stdout)
                f.write("\n--- STDERR ---\n")
                f.write(result.stderr)
            
            success = result.returncode == 0
            output = result.stdout + result.stderr
            error_type = "none" if success else "command_failed"
            
            return success, output, error_type
            
        except subprocess.TimeoutExpired:
            self._print(f"⏱️ Timeout après {self.config['timeout']} secondes", Colors.RED)
            with open(log_file, 'w') as f:
                f.write(f"=== Iteration {self.iteration} - {timestamp} ===\n")
                f.write(f"TIMEOUT after {self.config['timeout']} seconds\n")
            return False, "Timeout", "timeout"
        except FileNotFoundError:
            self._print("Erreur: claude n'est pas installé", Colors.RED)
            sys.exit(1)
    
    def _add_problem_to_instructions(self, problem_description: str):
        """Add a problem report to the instructions file"""
        content = self._read_instructions()
        
        # Look for PROBLÈMES section or create it
        if "## 🚨 PROBLÈMES CONNUS" not in content:
            # Add section before LOG DES ITÉRATIONS if it exists
            if "## 📝 LOG DES ITÉRATIONS" in content:
                content = content.replace(
                    "## 📝 LOG DES ITÉRATIONS",
                    f"## 🚨 PROBLÈMES CONNUS\n{problem_description}\n\n## 📝 LOG DES ITÉRATIONS"
                )
            else:
                content += f"\n\n## 🚨 PROBLÈMES CONNUS\n{problem_description}"
        else:
            # Add to existing problems section
            content = re.sub(
                r'(## 🚨 PROBLÈMES CONNUS.*?)(\n##|\Z)',
                rf'\1\n{problem_description}\2',
                content,
                flags=re.DOTALL
            )
        
        self.instructions_file.write_text(content, encoding='utf-8')
    
    def _handle_repeated_failures(self) -> str:
        """Generate recovery prompt for repeated failures"""
        self._print("\n⚠️  Détection de problèmes répétés!", Colors.YELLOW + Colors.BOLD)
        
        problem_msg = f"""
### ⚠️ BLOCAGE DÉTECTÉ - Itération {self.iteration}
- **Type**: Timeouts répétés ({self.timeout_count} fois)
- **Dernière réussite**: Itération {self.last_successful_iteration}
- **Action suggérée**: Revenir sur les tâches précédentes ou simplifier l'approche

**IMPORTANT**: Il semble y avoir un blocage. Suggestions:
1. La tâche actuelle est peut-être trop complexe - la décomposer
2. Il y a peut-être une erreur dans le code précédent
3. Les tests ou validations peuvent être en boucle infinie
4. Essayer une approche différente
"""
        
        # Add problem to instructions
        self._add_problem_to_instructions(problem_msg)
        
        # Generate recovery prompt
        recovery_prompt = (
            f"ATTENTION: {self.timeout_count} timeouts consécutifs détectés. "
            f"Il y a un BLOCAGE. "
            f"Analyse le fichier {self.instructions_file.name} et identifie le problème. "
            f"Options: 1) Simplifier la tâche actuelle, 2) Revenir à une tâche précédente, "
            f"3) Corriger une erreur dans le code existant, 4) Marquer la tâche comme bloquée et passer à la suivante. "
            f"Mets à jour le fichier avec la solution choisie."
        )
        
        return recovery_prompt
    
    def run_single_iteration(self) -> bool:
        """Run a single iteration"""
        self.iteration += 1
        self._print(f"\n{'='*60}", Colors.BLUE)
        self._print(f"Itération {self.iteration}", Colors.BLUE + Colors.BOLD)
        self._print(f"{'='*60}", Colors.BLUE)
        
        # Get current status
        status = self._get_status()
        self._display_status(status)
        
        # Check if already complete
        if status['is_complete']:
            self._print("\n✅ Développement déjà complété!", Colors.GREEN)
            return False
        
        # Check if we need recovery after repeated failures
        prompt = self._get_prompt()
        if self.timeout_count >= 3:
            self._print("\n🔄 Mode récupération activé", Colors.YELLOW + Colors.BOLD)
            prompt = self._handle_repeated_failures()
        elif self.timeout_count >= 2:
            self._print("\n⚠️  Attention: 2 timeouts consécutifs", Colors.YELLOW)
            prompt = (f"ATTENTION: La dernière commande a timeout 2 fois. "
                     f"Il y a peut-être un problème. {prompt}")
        
        self._print(f"\n🤖 Prompt: {prompt[:100]}...", Colors.YELLOW)
        
        self._print("\n⚙️  Exécution de claude...", Colors.YELLOW)
        success, output, error_type = self._run_claude_code(prompt)
        
        # Update counters based on result
        if error_type == "timeout":
            self.timeout_count += 1
            self.error_count += 1
            
            if self.timeout_count >= 5:
                self._print("\n❌ Trop de timeouts consécutifs (5). Arrêt forcé.", Colors.RED + Colors.BOLD)
                self._print("💡 Actions suggérées:", Colors.YELLOW)
                self._print("  1. Vérifier si Claude est surchargé", Colors.YELLOW)
                self._print("  2. Augmenter le timeout avec --timeout 600", Colors.YELLOW)
                self._print("  3. Simplifier les tâches dans le fichier instructions", Colors.YELLOW)
                self._print("  4. Vérifier s'il n'y a pas de boucle infinie dans les tests", Colors.YELLOW)
                return False
                
        elif success:
            self._print("✓ Itération complétée avec succès", Colors.GREEN)
            self.timeout_count = 0  # Reset timeout counter
            self.error_count = 0    # Reset error counter
            self.last_successful_iteration = self.iteration
        else:
            self._print("✗ Itération échouée", Colors.RED)
            self.error_count += 1
            
            if self.error_count >= 5 and self.config['stop_on_error']:
                self._print("\n❌ Trop d'erreurs consécutives (5). Arrêt.", Colors.RED)
                return False
        
        # Check new status
        new_status = self._get_status()
        if new_status['is_complete']:
            self._print("\n🎉 Développement complété!", Colors.GREEN + Colors.BOLD)
            return False
        
        # Show warning if stuck on same task
        if self.error_count >= 3:
            self._print("\n⚠️  La même tâche échoue répétitivement", Colors.YELLOW)
            self._print("💡 Considérez de:", Colors.YELLOW)
            self._print("  - Simplifier la tâche", Colors.YELLOW)
            self._print("  - La marquer comme bloquée et continuer", Colors.YELLOW)
            self._print("  - Revoir l'approche", Colors.YELLOW)
        
        return True  # Continue iterations
    
    def run(self):
        """Run the main loop"""
        self._print(f"\n🚀 Démarrage du développement itératif", Colors.BOLD + Colors.GREEN)
        self._print(f"📄 Instructions: {self.instructions_file}")
        self._print(f"🔄 Max iterations: {self.config['max_iterations']}")
        self._print(f"⏱️  Délai entre iterations: {self.config['sleep_between']}s\n")
        
        # Check if claude exists
        if shutil.which('claude') is None:
            self._print("❌ Erreur: claude n'est pas installé", Colors.RED)
            self._print("Installez-le d'abord: npm install -g @anthropic-ai/claude-cli", Colors.YELLOW)
            sys.exit(1)
        
        try:
            while self.iteration < self.config['max_iterations']:
                # Run iteration
                should_continue = self.run_single_iteration()
                
                if not should_continue:
                    break
                
                # Sleep between iterations
                if self.iteration < self.config['max_iterations']:
                    self._print(f"\n⏳ Attente de {self.config['sleep_between']} secondes...", Colors.BLUE)
                    time.sleep(self.config['sleep_between'])
            
            # Final status
            self._print_summary()
            
        except KeyboardInterrupt:
            self._print("\n\n⚠️  Interruption par l'utilisateur", Colors.YELLOW)
            self._print_summary()
    
    def _print_summary(self):
        """Print final summary"""
        duration = datetime.now() - self.start_time
        final_status = self._get_status()
        
        self._print(f"\n{'='*60}", Colors.BOLD)
        self._print("📈 Résumé Final", Colors.BOLD)
        self._print(f"{'='*60}", Colors.BOLD)
        
        self._print(f"⏱️  Durée totale: {duration}")
        self._print(f"🔄 Itérations effectuées: {self.iteration}")
        self._print(f"✅ Itérations réussies: {self.last_successful_iteration}")
        self._print(f"⚠️  Timeouts totaux: {self.timeout_count}")
        self._print(f"✅ Tâches complétées: {final_status['completed_tasks']}/{final_status['total_tasks']}")
        self._print(f"📊 Progression: {final_status['percentage']:.1f}%")
        
        if final_status['is_complete']:
            self._print("\n🎉 SUCCÈS: Développement complété!", Colors.GREEN + Colors.BOLD)
        elif self.timeout_count >= 3:
            self._print("\n⚠️  Développement bloqué par des timeouts", Colors.YELLOW + Colors.BOLD)
            self._print("    Vérifiez le fichier instructions pour les problèmes reportés", Colors.YELLOW)
        else:
            self._print(f"\n⚠️  Développement non terminé", Colors.YELLOW)
            self._print(f"    Prochaine action: {final_status['next_action']}")
        
        self._print(f"\n📁 Logs sauvegardés dans: {self.log_dir}/")

def main():
    parser = argparse.ArgumentParser(
        description="Claude Iterative Development Runner",
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
Exemples:
  %(prog)s instructions.md                    # Run avec fichier par défaut
  %(prog)s kbuffer.md --max-iter 50          # Plus d'itérations
  %(prog)s inst.md --sleep 5 --timeout 120   # Délais personnalisés
  %(prog)s inst.md --watch                   # Mode watch
  %(prog)s inst.md --single                  # Une seule itération
  %(prog)s inst.md --status                  # Afficher le status
        """
    )
    
    parser.add_argument('instructions', 
                       help='Fichier instructions.md')
    
    parser.add_argument('--max-iter', type=int, default=30,
                       help='Nombre maximum d\'itérations (défaut: 30)')
    
    parser.add_argument('--sleep', type=int, default=3,
                       help='Secondes entre les itérations (défaut: 3)')
    
    parser.add_argument('--timeout', type=int, default=300,
                       help='Timeout pour claude en secondes (défaut: 300)')
    
    parser.add_argument('--log-dir', default='.claude-logs',
                       help='Dossier pour les logs (défaut: .claude-logs)')
    
    parser.add_argument('--backup-dir', default='.claude-backups',
                       help='Dossier pour les backups (défaut: .claude-backups)')
    
    parser.add_argument('--stop-on-error', action='store_true',
                       help='Arrêter en cas d\'erreur')
    
    parser.add_argument('--single', action='store_true',
                       help='Exécuter une seule itération')
    
    parser.add_argument('--status', action='store_true',
                       help='Afficher le status actuel et quitter')
    
    parser.add_argument('--watch', action='store_true',
                       help='Mode watch - relancer quand le fichier change')
    
    parser.add_argument('--dry-run', action='store_true',
                       help='Afficher ce qui sera fait sans l\'exécuter')
    
    parser.add_argument('--max-timeouts', type=int, default=5,
                       help='Nombre max de timeouts avant arrêt (défaut: 5)')
    
    parser.add_argument('--recovery-after', type=int, default=3,
                       help='Activer mode récupération après X timeouts (défaut: 3)')
    
    args = parser.parse_args()
    
    # Check if instructions file exists
    if not Path(args.instructions).exists():
        print(f"{Colors.RED}Erreur: Fichier introuvable: {args.instructions}{Colors.END}")
        sys.exit(1)
    
    # Configuration
    config = {
        'max_iterations': 1 if args.single else args.max_iter,
        'sleep_between': args.sleep,
        'timeout': args.timeout,
        'log_dir': args.log_dir,
        'backup_dir': args.backup_dir,
        'stop_on_error': args.stop_on_error,
        'max_timeouts': args.max_timeouts,
        'recovery_after': args.recovery_after
    }
    
    # Create runner
    runner = ClaudeDevRunner(args.instructions, config)
    
    # Handle different modes
    if args.status:
        status = runner._get_status()
        runner._display_status(status)
        
    elif args.dry_run:
        prompt = runner._get_prompt()
        print(f"{Colors.YELLOW}Mode dry-run{Colors.END}")
        print(f"Commande qui serait exécutée:")
        print(f"  claude --print \"{prompt}\"")
        print(f"\nConfiguration:")
        for key, value in config.items():
            print(f"  {key}: {value}")
    
    elif args.watch:
        print(f"{Colors.BLUE}Mode watch activé - surveillance de {args.instructions}{Colors.END}")
        last_mtime = 0
        
        try:
            while True:
                current_mtime = Path(args.instructions).stat().st_mtime
                if current_mtime != last_mtime:
                    last_mtime = current_mtime
                    print(f"{Colors.YELLOW}Changement détecté!{Colors.END}")
                    runner.run_single_iteration()
                time.sleep(2)
        except KeyboardInterrupt:
            print(f"\n{Colors.YELLOW}Watch mode arrêté{Colors.END}")
    
    else:
        # Normal run
        runner.run()

if __name__ == "__main__":
    main()