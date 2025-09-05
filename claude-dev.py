#!/usr/bin/env python3
"""
Claude TDD Development Script with Smart Phase Detection
Usage: python claude-tdd.py instructions.md
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
from typing import Optional, Dict, Any, Tuple
import shutil
import threading

# ANSI color codes
class Colors:
    BLUE = '\033[94m'
    GREEN = '\033[92m'
    YELLOW = '\033[93m'
    RED = '\033[91m'
    CYAN = '\033[96m'
    MAGENTA = '\033[95m'
    BOLD = '\033[1m'
    END = '\033[0m'

class TDDPhase:
    """TDD Phase detection and management"""
    INTERFACE = "Interface Definition"
    TEST_WRITING = "Test Writing"
    IMPLEMENTATION = "Implementation"
    IMPLEMENTATION_UNSAFE = "Implementation Unsafe"
    BENCHMARKING = "Benchmarking"
    BENCHMARKING_COMPARISON = "Benchmarking Safe vs Unsafe"
    REFACTORING = "Refactoring"
    VALIDATION = "Validation"

class ClaudeTDDRunner:
    def __init__(self, instructions_file: str, config: Dict[str, Any]):
        self.original_instructions_file = Path(instructions_file)
        self.config = config
        self.log_dir = Path(config['log_dir'])
        self.backup_dir = Path(config['backup_dir'])
        self.iteration = 0
        self.start_time = datetime.now()
        self.timeout_count = 0
        self.error_count = 0
        self.last_successful_iteration = 0
        self.current_phase = TDDPhase.INTERFACE
        self.global_timeout_multiplier = config.get('timeout_multiplier', 5)
        self.spinner_active = False
        self.spinner_thread = None
        
        # Create directories
        self.log_dir.mkdir(exist_ok=True)
        self.backup_dir.mkdir(exist_ok=True)
        
        # Setup working files
        self._setup_working_files()
        
        # Now set the working file as the instructions file
        self.instructions_file = self.working_file
        self.package_info = self._extract_package_info()
        
        # Backup original instructions
        self._backup_instructions()
    
    def _extract_package_info(self) -> Dict[str, str]:
        """Extract package information from instructions"""
        content = self._read_instructions()
        info = {
            'name': 'unknown',
            'path': 'pkg/kernel/unknown',
            'types': []
        }
        
        # Extract package path
        path_match = re.search(r'\*\*Package\*\*:\s*([^\n]+)', content)
        if path_match:
            path = path_match.group(1).strip()
            # Handle placeholders like pkg/kernel/${PACKAGE_NAME}
            if '${PACKAGE_NAME}' in path:
                # Try to extract from file path
                if 'kbuffer' in str(self.instructions_file):
                    info['path'] = 'pkg/kernel/kbuffer'
                    info['name'] = 'kbuffer'
            elif not path.startswith('${'):
                info['path'] = path
                info['name'] = Path(path).name
        
        # If still using placeholders, try to extract from file path
        if info['name'] == 'unknown':
            # Extract from file path
            if 'kbuffer' in str(self.instructions_file).lower():
                info['name'] = 'kbuffer'
                info['path'] = 'pkg/kernel/kbuffer'
            elif 'kcache' in str(self.instructions_file).lower():
                info['name'] = 'kcache'
                info['path'] = 'pkg/kernel/kcache'
        
        # Extract types from structure section
        # Look for ${TYPE1}, ${TYPE2} pattern first
        type_pattern = r'\$\{TYPE(\d+)\}'
        type_matches = re.findall(type_pattern, content)
        
        # If we have placeholder types, try to guess from context
        if type_matches and info['name'] == 'kbuffer':
            info['types'] = ['buffer', 'pool']
        elif type_matches and info['name'] == 'kcache':
            info['types'] = ['cache', 'entry']
        else:
            # Look for actual Go files mentioned
            go_files = re.findall(r'(\w+)\.go\b', content)
            for filename in go_files:
                if filename not in ['interface', 'constants', 'global', 'mocks']:
                    if filename not in info['types'] and not filename.endswith('_test'):
                        info['types'].append(filename)
        
        # If no types found, use defaults based on package
        if not info['types']:
            if info['name'] == 'kbuffer':
                info['types'] = ['buffer', 'pool']
            elif info['name'] == 'kcache':
                info['types'] = ['cache', 'entry']
        
        return info
    
    def _print(self, message: str, color: str = ""):
        """Print with optional color"""
        print(f"{color}{message}{Colors.END}")
    
    def _setup_working_files(self):
        """Setup working files - keep original intact and work on a copy"""
        # Create permanent original copy
        self.original_backup = self.backup_dir / "instruction_original.md"
        if not self.original_backup.exists():
            shutil.copy2(self.original_instructions_file, self.original_backup)
            self._print(f"✅ Copie originale sauvegardée: {self.original_backup}", Colors.GREEN)
        
        # Create or continue with working copy
        self.working_file = self.backup_dir / "instruction_work.md"
        if self.working_file.exists():
            # Continue with existing work
            self._print(f"📝 Reprise du fichier de travail existant: {self.working_file}", Colors.CYAN)
            # Get the last iteration number from existing backups
            existing_backups = sorted(self.backup_dir.glob("instruction_iter_*.md"))
            if existing_backups:
                last_backup = existing_backups[-1].stem
                last_iter = int(last_backup.split('_')[-1])
                self.iteration = last_iter
                self._print(f"  ↳ Reprise à l'itération {self.iteration + 1}", Colors.BLUE)
        else:
            # First run - copy from original
            shutil.copy2(self.original_instructions_file, self.working_file)
            self._print(f"📝 Nouveau fichier de travail créé: {self.working_file}", Colors.CYAN)
    
    def _backup_instructions(self):
        """Backup the current state of instructions file for this iteration"""
        if self.iteration > 0:  # Only backup after iterations, not at init
            iteration_backup = self.backup_dir / f"instruction_iter_{self.iteration:03d}.md"
            shutil.copy2(self.instructions_file, iteration_backup)
            self._print(f"  💾 Sauvegarde itération {self.iteration}: {iteration_backup.name}", Colors.BLUE)
    
    def _read_instructions(self) -> str:
        """Read the instructions file"""
        return self.instructions_file.read_text(encoding='utf-8')
    
    def _detect_tdd_phase(self) -> Tuple[str, int]:
        """
        Detect current TDD phase and return appropriate timeout.
        Returns: (phase_name, timeout_seconds)
        """
        content = self._read_instructions()
        
        # Check for TDD phase markers (multiplied by global timeout multiplier)
        phase_patterns = [
            (r'Phase.*Interface.*Definition|Créer\s+interface\.go', TDDPhase.INTERFACE, 60 * self.global_timeout_multiplier),
            (r'Phase.*Test.*Writing|Créer.*_test\.go|Tests?\s+unitaires', TDDPhase.TEST_WRITING, 90 * self.global_timeout_multiplier),
            (r'Version\s+Unsafe|NewUnsafe|Implémenter.*unsafe', TDDPhase.IMPLEMENTATION_UNSAFE, 120 * self.global_timeout_multiplier),
            (r'Phase.*Implementation|Implémenter.*\.go', TDDPhase.IMPLEMENTATION, 90 * self.global_timeout_multiplier),
            (r'Benchmark.*Safe.*vs.*Unsafe|Comparaison.*performance', TDDPhase.BENCHMARKING_COMPARISON, 90 * self.global_timeout_multiplier),
            (r'Phase.*Refactoring|Optimiser', TDDPhase.REFACTORING, 120 * self.global_timeout_multiplier),
            (r'Benchmark|bench.*test', TDDPhase.BENCHMARKING, 60 * self.global_timeout_multiplier),
            (r'Phase.*Validation|Coverage.*95%', TDDPhase.VALIDATION, 120 * self.global_timeout_multiplier),
        ]
        
        for pattern, phase, timeout in phase_patterns:
            if re.search(pattern, content, re.IGNORECASE | re.MULTILINE):
                self.current_phase = phase
                return (phase, timeout)
        
        # Check specific action patterns for safe/unsafe operations (multiplied by global timeout multiplier)
        action_patterns = [
            (r'go\s+test.*-race', "Race Detection Test", 60 * self.global_timeout_multiplier),
            (r'go\s+test.*-bench=.*Safe\|Unsafe', "Safe vs Unsafe Benchmark", 90 * self.global_timeout_multiplier),
            (r'bazel\s+test.*--config=dev', "Bazel Dev Test", 90 * self.global_timeout_multiplier),
            (r'bazel\s+test.*--config=prod', "Bazel Prod Test", 90 * self.global_timeout_multiplier),
            (r'bazel\s+run.*--config=benchmark', "Bazel Benchmark", 120 * self.global_timeout_multiplier),
            (r'go\s+test.*-short', "Quick Test", 30 * self.global_timeout_multiplier),
            (r'go\s+test.*-bench=Benchmark\w+\s', "Targeted Benchmark", 45 * self.global_timeout_multiplier),
            (r'go\s+test.*-cover', "Coverage Test", 60 * self.global_timeout_multiplier),
        ]
        
        for pattern, phase, timeout in action_patterns:
            if re.search(pattern, content, re.IGNORECASE):
                return (phase, timeout)
        
        # Default TDD timeout (multiplied by global timeout multiplier)
        return ("TDD Development", 60 * self.global_timeout_multiplier)
    
    def _get_prompt(self) -> str:
        """Generate TDD-aware prompt"""
        content = self._read_instructions()
        
        # Use relative path for Claude (it can only access files in current directory)
        # Get relative path from current working directory
        try:
            instructions_path = str(self.instructions_file.relative_to(Path.cwd()))
        except ValueError:
            # If not relative to cwd, use absolute path as fallback
            instructions_path = str(self.instructions_file.absolute())
        
        # Look for TDD prompt at the end of the file
        # Find the prompt section
        if 'PROMPT DE DÉMARRAGE TDD' in content:
            # Split at the prompt marker
            parts = content.split('PROMPT DE DÉMARRAGE TDD')
            if len(parts) > 1:
                # Get everything after the marker
                prompt_section = parts[-1].strip()
                # Remove the colon and leading whitespace
                if prompt_section.startswith(':'):
                    prompt_section = prompt_section[1:].strip()
                # Extract content between quotes (handle both regular and typographic quotes)
                prompt = None
                quote_chars = ['"', '"', '"', "'", "'", "'"]
                for open_quote in quote_chars:
                    if prompt_section.startswith(open_quote):
                        # Find matching close quote
                        for close_quote in quote_chars:
                            if close_quote in prompt_section[1:]:
                                end_quote = prompt_section.index(close_quote, 1)
                                prompt = prompt_section[1:end_quote]
                                break
                        if prompt:
                            break
                
                if not prompt:
                    # No quotes found, take everything until the end
                    prompt = prompt_section.split('\n\n')[0] if '\n\n' in prompt_section else prompt_section
                
                # Replace any reference to instruction.md with the actual working file path
                prompt = prompt.replace('instruction.md', instructions_path)
                prompt = prompt.replace('instructions.md', instructions_path)
                return prompt
        
        # Generate phase-specific prompt based on package info
        package_info = self.package_info
        package_name = package_info.get('name', 'kbuffer')
        
        # Use absolute path for Claude
        instructions_path = str(self.instructions_file.absolute())
        
        phase_prompts = {
            TDDPhase.INTERFACE: (
                f"Lis {instructions_path}. "
                f"Package {package_name}. "
                f"Crée interface.go avec TOUTES les interfaces documentées. "
                f"Documentation complète avec requirements de performance."
            ),
            TDDPhase.TEST_WRITING: (
                f"Lis {instructions_path}. "
                f"Package {package_name}. "
                f"Écris les tests pour le composant actuel. "
                f"D'abord cas nominaux, puis erreurs. "
                f"Utilise testify. Tests courts et ciblés."
            ),
            TDDPhase.IMPLEMENTATION: (
                f"Lis {instructions_path}. "
                f"Package {package_name}. "
                f"Implémente le MINIMUM pour faire passer les tests. "
                f"Pas d'optimisation prématurée."
            ),
            TDDPhase.REFACTORING: (
                f"Lis {instructions_path}. "
                f"Package {package_name}. "
                f"Optimise le code. Garde les tests verts. "
                f"Focus sur les hot paths identifiés."
            ),
            TDDPhase.BENCHMARKING: (
                f"Lis {instructions_path}. "
                f"Package {package_name}. "
                f"Exécute UN benchmark spécifique. "
                f"Utilise -bench=BenchmarkNomSpécifique. "
                f"Timeout court, résultats rapides."
            ),
        }
        
        if self.current_phase in phase_prompts:
            return phase_prompts[self.current_phase]
        
        # Default TDD prompt
        return (f"Lis {instructions_path}. "
                f"Package {package_name}. "
                f"Exécute la PROCHAINE ACTION en mode TDD. "
                f"Tests courts, ciblés, rapides. "
                f"Mets à jour le fichier avec les résultats.")
    
    def _get_status(self) -> Dict[str, Any]:
        """Parse current TDD status"""
        content = self._read_instructions()
        status = {
            'iteration': 0,
            'phase': 'Unknown',
            'completed_tasks': 0,
            'total_tasks': 0,
            'next_action': 'Unknown',
            'is_complete': False,
            'current_component': None
        }
        
        # Extract iteration
        iter_match = re.search(r'Itération[:\s]*(\d+)', content, re.IGNORECASE)
        if iter_match:
            status['iteration'] = int(iter_match.group(1))
        
        # Extract TDD phase
        phase_match = re.search(r'Phase[:\s]*([^\n]+)', content, re.IGNORECASE)
        if phase_match:
            status['phase'] = phase_match.group(1).strip()
        
        # Extract component being worked on
        comp_match = re.search(r'Composant[:\s]*([^\n]+)', content, re.IGNORECASE)
        if comp_match:
            component = comp_match.group(1).strip()
            # Skip if it's a placeholder or partial match
            if not component.startswith('$') and len(component) > 2:
                status['current_component'] = component
        
        # Count TDD tasks
        status['total_tasks'] = len(re.findall(r'^- \[[ x]\]', content, re.MULTILINE))
        status['completed_tasks'] = len(re.findall(r'^- \[x\]', content, re.MULTILINE))
        
        # Get next action
        action_match = re.search(r'PROCHAINE ACTION[:\s]*\n.*?Action[:\s]*(.+?)$', 
                                content, re.MULTILINE | re.IGNORECASE)
        if action_match:
            next_action = action_match.group(1).strip()
            # Replace common placeholders with actual values
            if '${NEXT_ACTION}' in next_action:
                # Try to find actual action from task list
                task_match = re.search(r'^- \[ \] (.+?)$', content, re.MULTILINE)
                if task_match:
                    next_action = task_match.group(1)
                else:
                    next_action = "Créer interface.go avec toutes les interfaces"
            status['next_action'] = next_action
        
        # Check completion
        status['is_complete'] = (
            status['total_tasks'] > 0 and 
            status['completed_tasks'] == status['total_tasks']
        )
        
        # Calculate percentage
        if status['total_tasks'] > 0:
            status['percentage'] = (status['completed_tasks'] / status['total_tasks']) * 100
        else:
            status['percentage'] = 0
        
        return status
    
    def _display_status(self, status: Dict[str, Any]):
        """Display TDD-aware status"""
        self._print("\n📊 TDD Status:", Colors.BOLD)
        self._print(f"  Itération: {status['iteration']}")
        self._print(f"  Phase TDD: {self.current_phase}", Colors.CYAN)
        if status['current_component']:
            self._print(f"  Composant: {status['current_component']}", Colors.MAGENTA)
        self._print(f"  Progression: {status['completed_tasks']}/{status['total_tasks']} ({status['percentage']:.1f}%)")
        self._print(f"  Action: {status['next_action'][:60]}...")
        
        # Progress bar
        bar_length = 40
        filled = int(bar_length * status['percentage'] / 100)
        bar = '█' * filled + '░' * (bar_length - filled)
        self._print(f"  [{bar}] {status['percentage']:.1f}%")
        
        # Show timeout for current phase
        phase, timeout = self._detect_tdd_phase()
        self._print(f"  Timeout: {timeout}s pour {phase}", Colors.BLUE)
    
    def _spinner(self):
        """Show spinner while Claude is thinking"""
        spinner_chars = ['⠋', '⠙', '⠹', '⠸', '⠼', '⠴', '⠦', '⠧']
        i = 0
        while self.spinner_active:
            print(f'\r  {spinner_chars[i % len(spinner_chars)]} Claude réfléchit...', end='', flush=True)
            time.sleep(0.1)
            i += 1
        print('\r' + ' ' * 30 + '\r', end='')  # Clear spinner line
    
    def _run_claude_code(self, prompt: str) -> tuple[bool, str, str]:
        """Execute claude with TDD-aware timeout"""
        timestamp = datetime.now().strftime("%Y%m%d_%H%M%S")
        log_file = self.log_dir / f"tdd_{self.iteration}_{timestamp}.log"
        
        # Get phase-specific timeout
        phase, timeout = self._detect_tdd_phase()
        
        # Apply global timeout override if specified
        if hasattr(self.config, 'global_timeout') and self.config.get('global_timeout'):
            timeout = self.config['global_timeout']
        
        self._print(f"  🧪 TDD: {phase} (timeout: {timeout}s)", Colors.CYAN)
        
        try:
            # Start spinner
            self.spinner_active = True
            self.spinner_thread = threading.Thread(target=self._spinner, daemon=True)
            self.spinner_thread.start()
            
            # Execute claude with pipe input (without --print as it causes timeouts)
            result = subprocess.run(
                ['claude'],
                input=prompt,
                capture_output=True,
                text=True,
                timeout=timeout
            )
            result_stdout = result.stdout
            result_stderr = result.stderr
            return_code = result.returncode
        
        finally:
            # Stop spinner
            self.spinner_active = False
            if self.spinner_thread:
                self.spinner_thread.join(timeout=0.5)
        
        try:
            
            # Save comprehensive logs
            with open(log_file, 'w', encoding='utf-8') as f:
                f.write(f"{'='*60}\n")
                f.write(f"TDD Iteration {self.iteration} - {timestamp}\n")
                f.write(f"{'='*60}\n\n")
                f.write(f"Phase: {phase}\n")
                f.write(f"Timeout: {timeout}s\n")
                f.write(f"Working File: {self.instructions_file}\n")
                f.write(f"Return Code: {return_code}\n")
                f.write(f"\n{'='*30} PROMPT {'='*30}\n")
                f.write(prompt)
                f.write(f"\n\n{'='*30} OUTPUT {'='*30}\n")
                if result_stdout:
                    f.write("--- STDOUT ---\n")
                    f.write(result_stdout)
                if result_stderr:
                    f.write("\n--- STDERR ---\n")
                    f.write(result_stderr)
                f.write(f"\n{'='*60}\n")
            
            success = return_code == 0
            output = result_stdout + result_stderr
            error_type = "none" if success else "command_failed"
            
            # Check for specific results
            self._analyze_output(output, phase)
            
            return success, output, error_type
            
        except subprocess.TimeoutExpired:
            # Stop spinner if running
            self.spinner_active = False
            if self.spinner_thread:
                self.spinner_thread.join(timeout=0.5)
            
            self._print(f"\n⏱️ Timeout après {timeout}s en phase {phase}", Colors.YELLOW)
            
            # TDD-specific timeout handling
            if phase == TDDPhase.BENCHMARKING_COMPARISON:
                self._print("💡 Conseil: Réduisez -benchtime ou ciblez moins de benchmarks", Colors.CYAN)
            elif phase == TDDPhase.BENCHMARKING:
                self._print("💡 Conseil: Utilisez -bench=BenchmarkSpecifique au lieu de -bench=.", Colors.CYAN)
            elif phase == TDDPhase.TEST_WRITING:
                self._print("💡 Conseil: Divisez les tests en plus petites unités", Colors.CYAN)
            elif phase == TDDPhase.INTERFACE:
                self._print("💡 Claude semble ne pas répondre. Vérifiez:", Colors.CYAN)
                self._print("   1. Que Claude Code est bien configuré", Colors.CYAN)
                self._print("   2. Que le fichier existe: " + str(self.instructions_file.absolute()), Colors.CYAN)
                self._print("   3. Essayez manuellement: claude --print \"test\"", Colors.CYAN)
            
            with open(log_file, 'w', encoding='utf-8') as f:
                f.write(f"{'='*60}\n")
                f.write(f"TDD Iteration {self.iteration} - TIMEOUT\n")
                f.write(f"{'='*60}\n\n")
                f.write(f"Phase: {phase}\n")
                f.write(f"Timeout after: {timeout}s\n")
                f.write(f"Working File: {self.instructions_file.absolute()}\n")
                f.write(f"\n{'='*30} PROMPT {'='*30}\n")
                f.write(prompt)
                f.write(f"\n{'='*60}\n")
            
            return False, "Timeout", "timeout"
            
        except FileNotFoundError:
            self._print("❌ Erreur: claude n'est pas installé", Colors.RED)
            sys.exit(1)
    
    def _analyze_output(self, output: str, phase: str):
        """Analyze command output for specific patterns"""
        
        # Check for test results
        if "PASS" in output and phase in [TDDPhase.TEST_WRITING, TDDPhase.IMPLEMENTATION]:
            self._print("  ✅ Tests passent!", Colors.GREEN)
        elif "FAIL" in output and phase == TDDPhase.TEST_WRITING:
            self._print("  🔴 Tests échouent (normal en TDD)", Colors.YELLOW)
        
        # Check for race conditions
        if "DATA RACE" in output:
            self._print("  ⚠️ RACE CONDITION détectée!", Colors.RED + Colors.BOLD)
            self._print("    La version Safe n'est PAS thread-safe!", Colors.RED)
        
        # Check for panic detection in unsafe tests
        if "panic: concurrent access detected" in output:
            self._print("  ✅ Détection multithread fonctionne sur Unsafe", Colors.GREEN)
        
        # Analyze benchmark comparisons
        if phase == TDDPhase.BENCHMARKING_COMPARISON:
            self._analyze_benchmark_comparison(output)
    
    def _analyze_benchmark_comparison(self, output: str):
        """Analyze benchmark output to compare Safe vs Unsafe"""
        import re
        
        # Pattern: BenchmarkXXX_Safe-8    1000000    1050 ns/op    48 B/op    1 allocs/op
        pattern = r'Benchmark(\w+)_(Safe|Unsafe)[^\n]*\s+(\d+)\s+(\d+\.?\d*)\s+ns/op'
        matches = re.findall(pattern, output)
        
        if not matches:
            return
        
        # Group by operation name
        benchmarks = {}
        for name, variant, iterations, ns_per_op in matches:
            if name not in benchmarks:
                benchmarks[name] = {}
            benchmarks[name][variant] = float(ns_per_op)
        
        # Calculate performance gains
        self._print("\n  📊 Comparaison Safe vs Unsafe:", Colors.CYAN + Colors.BOLD)
        
        for op_name, results in benchmarks.items():
            if 'Safe' in results and 'Unsafe' in results:
                safe_time = results['Safe']
                unsafe_time = results['Unsafe']
                
                if unsafe_time > 0:
                    gain = ((safe_time - unsafe_time) / safe_time) * 100
                    
                    self._print(f"    {op_name}:", Colors.CYAN)
                    self._print(f"      Safe:   {safe_time:.1f} ns/op", Colors.BLUE)
                    self._print(f"      Unsafe: {unsafe_time:.1f} ns/op", Colors.BLUE)
                    
                    if gain > 30:
                        self._print(f"      Gain:   {gain:.1f}% ✅ (>30%, justifie Unsafe)", Colors.GREEN + Colors.BOLD)
                    elif gain > 0:
                        self._print(f"      Gain:   {gain:.1f}% ⚠️ (<30%, garde Safe only)", Colors.YELLOW)
                    else:
                        self._print(f"      Gain:   {gain:.1f}% ❌ (Unsafe plus lent!)", Colors.RED)
    
    def _suggest_recovery(self) -> str:
        """Generate TDD-specific recovery suggestions"""
        phase = self.current_phase
        
        suggestions = {
            TDDPhase.TEST_WRITING: (
                "Tests trop lourds. Actions: "
                "1) Utiliser -short pour tests rapides, "
                "2) Diviser en sous-tests avec t.Run(), "
                "3) Ajouter t.Parallel() partout"
            ),
            TDDPhase.BENCHMARKING: (
                "Benchmarks timeout. Actions: "
                "1) Cibler UN benchmark: -bench=BenchmarkNom_Safe, "
                "2) Réduire -benchtime=10ms pour tests rapides, "
                "3) Séparer Safe et Unsafe"
            ),
            TDDPhase.BENCHMARKING_COMPARISON: (
                "Comparaison trop longue. Actions: "
                "1) Benchmark une opération à la fois, "
                "2) Utiliser -benchtime=1s au lieu de 10s, "
                "3) Profiler pour identifier le bottleneck"
            ),
            TDDPhase.IMPLEMENTATION: (
                "Implémentation Safe complexe. Actions: "
                "1) Commencer simple avec mutex, "
                "2) Optimiser après si nécessaire, "
                "3) Valider avec race detector"
            ),
            TDDPhase.IMPLEMENTATION_UNSAFE: (
                "Version Unsafe complexe. Actions: "
                "1) Réutiliser logique Safe (méthode xxxImpl), "
                "2) Ajouter panic detection simple, "
                "3) Benchmark pour valider gain >30%"
            ),
        }
        
        if phase in suggestions:
            return suggestions[phase]
        
        return "Simplifier l'action courante et la diviser en étapes plus petites."
    
    def _run_bazel_command(self, command: str) -> tuple[bool, str]:
        """Execute Bazel command and return success status and output"""
        self._print(f"  🏗️ Bazel: {command}", Colors.BLUE)
        
        try:
            result = subprocess.run(
                command,
                shell=True,
                capture_output=True,
                text=True,
                timeout=120 * self.global_timeout_multiplier  # Bazel commands get more time
            )
            
            success = result.returncode == 0
            output = result.stdout + result.stderr
            
            # Check for specific Bazel patterns
            if "PASSED" in output:
                self._print("    ✅ Bazel tests PASSED", Colors.GREEN)
            elif "FAILED" in output:
                self._print("    ❌ Bazel tests FAILED", Colors.RED)
            
            if "--config=prod" in command and "unsafe_no_check" in command:
                self._print("    🚀 Mode PRODUCTION (unsafe_no_check activé)", Colors.YELLOW)
            elif "--config=dev" in command:
                self._print("    🛡️ Mode DEV (safety checks actifs)", Colors.CYAN)
            
            return success, output
            
        except subprocess.TimeoutExpired:
            self._print(f"    ⏱️ Bazel timeout après {120 * self.global_timeout_multiplier}s", Colors.RED)
            return False, "Timeout"
        except Exception as e:
            self._print(f"    ❌ Erreur Bazel: {e}", Colors.RED)
            return False, str(e)
    
    def run_single_iteration(self) -> bool:
        """Run a single TDD iteration"""
        self.iteration += 1
        self._print(f"\n{'='*60}", Colors.CYAN)
        self._print(f"🧪 TDD Itération {self.iteration}", Colors.CYAN + Colors.BOLD)
        self._print(f"{'='*60}", Colors.CYAN)
        
        # Backup current state before iteration
        self._backup_instructions()
        
        # Get status
        status = self._get_status()
        self._display_status(status)
        
        # Check completion
        if status['is_complete']:
            self._print("\n✅ Développement TDD complété!", Colors.GREEN)
            return False
        
        # Generate prompt
        prompt = self._get_prompt()
        
        # Handle timeout recovery
        if self.timeout_count >= 2:
            self._print("\n⚠️ Timeouts répétés détectés", Colors.YELLOW)
            recovery = self._suggest_recovery()
            self._print(f"💡 {recovery}", Colors.CYAN)
            prompt = f"ATTENTION: Timeouts répétés. {recovery}. {prompt}"
        
        # Show more of the prompt for debugging
        prompt_display = prompt[:200] if len(prompt) > 200 else prompt
        self._print(f"\n🤖 Prompt envoyé à Claude:", Colors.YELLOW)
        self._print(f"   {prompt_display}{'...' if len(prompt) > 200 else ''}", Colors.CYAN)
        self._print(f"\n⚙️ Claude réfléchit... (timeout: {self._detect_tdd_phase()[1]}s)", Colors.YELLOW)
        
        success, output, error_type = self._run_claude_code(prompt)
        
        # Update counters
        if error_type == "timeout":
            self.timeout_count += 1
            if self.timeout_count >= 3:
                self._print("\n⚠️ 3 timeouts - Passage forcé à l'action suivante", Colors.YELLOW)
                # Force update to move to next action
                self._force_next_action()
                self.timeout_count = 0
        elif success:
            self._print("\n✅ Itération TDD complétée avec succès", Colors.GREEN)
            # Show a summary of what was done
            if output and len(output) > 0:
                lines = output.split('\n')
                if len(lines) > 3:
                    self._print("📝 Résumé de l'action:", Colors.CYAN)
                    for line in lines[:5]:  # Show first 5 lines
                        if line.strip():
                            self._print(f"   {line[:80]}{'...' if len(line) > 80 else ''}", Colors.BLUE)
            self.timeout_count = 0
            self.error_count = 0
            self.last_successful_iteration = self.iteration
        else:
            self._print("\n❌ Itération échouée", Colors.RED)
            if output and "error" in output.lower():
                self._print(f"   Erreur: {output[:100]}...", Colors.RED)
            self.error_count += 1
        
        return True
    
    def _force_next_action(self):
        """Force progression to next action after repeated timeouts"""
        content = self._read_instructions()
        
        # Add timeout note
        timeout_note = f"""
### ⏱️ Timeout Forcé - Itération {self.iteration}
- Phase: {self.current_phase}
- Action skippée après 3 timeouts
- Suggestion: Revoir cette action plus tard avec approche différente
"""
        
        # Find and update next action
        if "## 📝 LOG DES ITÉRATIONS TDD" in content:
            content = content.replace(
                "## 📝 LOG DES ITÉRATIONS TDD",
                f"{timeout_note}\n\n## 📝 LOG DES ITÉRATIONS TDD"
            )
        
        self.instructions_file.write_text(content, encoding='utf-8')
    
    def run(self):
        """Run the TDD development loop"""
        self._print(f"\n🧪 Démarrage TDD Development", Colors.BOLD + Colors.CYAN)
        self._print(f"📄 Instructions originales: {self.original_instructions_file}")
        self._print(f"📝 Fichier de travail: {self.working_file.name}")
        self._print(f"💾 Sauvegardes dans: {self.backup_dir}/")
        self._print(f"🔄 Max iterations: {self.config['max_iterations']}")
        self._print(f"⏱️ Timeouts adaptatifs par phase TDD")
        self._print(f"⏳ Délai entre iterations: {self.config['sleep_between']}s\n")
        
        # Check claude
        if shutil.which('claude') is None:
            self._print("❌ Erreur: claude n'est pas installé", Colors.RED)
            self._print("Installez: npm install -g @anthropic-ai/claude-cli", Colors.YELLOW)
            sys.exit(1)
        
        try:
            iterations_to_run = self.config['max_iterations']
            iterations_done = 0
            
            while iterations_done < iterations_to_run:
                should_continue = self.run_single_iteration()
                iterations_done += 1
                
                if not should_continue:
                    break
                
                # Sleep between iterations
                if iterations_done < iterations_to_run:
                    self._print(f"\n⏳ Pause {self.config['sleep_between']}s...", Colors.BLUE)
                    time.sleep(self.config['sleep_between'])
            
            self._print_summary()
            
        except KeyboardInterrupt:
            self._print("\n\n⚠️ Interruption utilisateur", Colors.YELLOW)
            self._print_summary()
    
    def _print_summary(self):
        """Print TDD summary"""
        duration = datetime.now() - self.start_time
        final_status = self._get_status()
        
        self._print(f"\n{'='*60}", Colors.BOLD)
        self._print("📈 Résumé TDD", Colors.BOLD + Colors.CYAN)
        self._print(f"{'='*60}", Colors.BOLD)
        
        self._print(f"⏱️ Durée: {duration}")
        self._print(f"🔄 Itérations: {self.iteration}")
        self._print(f"✅ Succès: {self.last_successful_iteration}")
        self._print(f"⚠️ Timeouts: {self.timeout_count}")
        self._print(f"🧪 Phase finale: {self.current_phase}")
        self._print(f"📊 Progression: {final_status['percentage']:.1f}%")
        
        if final_status['is_complete']:
            self._print("\n🎉 SUCCÈS: TDD complété!", Colors.GREEN + Colors.BOLD)
        else:
            self._print(f"\n⚠️ TDD en cours", Colors.YELLOW)
            self._print(f"    Prochaine: {final_status['next_action']}")
        
        self._print(f"\n📁 Fichiers:", Colors.BOLD)
        self._print(f"  • Logs: {self.log_dir}/")
        if self.iteration > 0:
            latest_log = sorted(self.log_dir.glob("tdd_*.log"))[-1] if list(self.log_dir.glob("tdd_*.log")) else None
            if latest_log:
                self._print(f"    → Dernier: {latest_log.name}", Colors.BLUE)
        self._print(f"  • Sauvegardes: {self.backup_dir}/")
        self._print(f"    → Original: instruction_original.md")
        self._print(f"    → Travail: instruction_work.md")
        if self.iteration > 0:
            self._print(f"    → Itérations: instruction_iter_001.md à instruction_iter_{self.iteration:03d}.md")
        
        self._print(f"\n💡 Commandes utiles:", Colors.BOLD)
        self._print(f"  • Voir le dernier log: cat {self.log_dir}/tdd_*.log | tail -50")
        self._print(f"  • Nettoyer: claude-dev {self.original_instructions_file} --cleanup")
        self._print(f"  • Reprendre: claude-dev {self.original_instructions_file}")

def main():
    parser = argparse.ArgumentParser(
        description="Claude TDD Development Runner with Safe/Unsafe Strategy",
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
Approche TDD avec Safe/Unsafe et timeouts adaptatifs (×5 par défaut):
  - Interface: 300s (définition des contrats)
  - Tests: 450s (écriture des tests avec concurrence)  
  - Implementation Safe: 450s (version thread-safe)
  - Implementation Unsafe: 600s (si gain >30%%)
  - Benchmarks comparatifs: 450s (Safe vs Unsafe)
  
Stratégie Safe/Unsafe:
  - Version Safe OBLIGATOIRE (thread-safe avec mutex/atomic)
  - Version Unsafe OPTIONNELLE (si gain >30%% en benchmark)
  - Protection multithread en DEV (panic si concurrent)
  - Performance MAX en PROD (unsafe_no_check activé)
  
Exemples:
  %(prog)s instructions.md              # TDD standard
  %(prog)s inst.md --quick              # Tests rapides (-short)
  %(prog)s inst.md --single             # Une itération
  %(prog)s inst.md --timeout 120        # Override timeout global à 120s
  %(prog)s inst.md --benchmark-compare  # Force comparaison Safe/Unsafe
  %(prog)s inst.md --bazel-dev          # Test avec Bazel mode dev
  %(prog)s inst.md --bazel-prod         # Test avec Bazel mode prod
        """
    )
    
    parser.add_argument('instructions', 
                       help='Fichier instructions.md TDD')
    
    parser.add_argument('--max-iter', type=int, default=50,
                       help='Maximum iterations (défaut: 50)')
    
    parser.add_argument('--sleep', type=int, default=2,
                       help='Secondes entre iterations (défaut: 2)')
    
    parser.add_argument('--quick', action='store_true',
                       help='Mode rapide avec -short pour tests')
    
    parser.add_argument('--phase', choices=['interface', 'test', 'impl', 'impl-unsafe', 'refactor', 'bench', 'bench-compare'],
                       help='Forcer une phase TDD spécifique')
    
    parser.add_argument('--benchmark-compare', action='store_true',
                       help='Force comparaison benchmarks Safe vs Unsafe')
    
    parser.add_argument('--bazel-dev', action='store_true',
                       help='Execute tests Bazel en mode dev (avec safety checks)')
    
    parser.add_argument('--bazel-prod', action='store_true',
                       help='Execute tests Bazel en mode prod (unsafe_no_check)')
    
    parser.add_argument('--log-dir', default='.tdd-logs',
                       help='Dossier logs (défaut: .tdd-logs)')
    
    parser.add_argument('--backup-dir', default='.tdd-backups',
                       help='Dossier backups (défaut: .tdd-backups)')
    
    parser.add_argument('--single', action='store_true',
                       help='Une seule itération')
    
    parser.add_argument('--status', action='store_true',
                       help='Afficher status et quitter')
    
    parser.add_argument('--analyze-bench', metavar='FILE',
                       help='Analyser un fichier de benchmarks pour décision Safe/Unsafe')
    
    parser.add_argument('--timeout', type=int, metavar='SECONDS',
                       help='Override global des timeouts (en secondes)')
    
    parser.add_argument('--cleanup', action='store_true',
                       help='Nettoyer et repartir de zéro (supprime instruction_work.md et les sauvegardes)')
    
    args = parser.parse_args()
    
    # Check file
    if not Path(args.instructions).exists():
        print(f"{Colors.RED}Erreur: {args.instructions} introuvable{Colors.END}")
        sys.exit(1)
    
    # Configuration
    config = {
        'max_iterations': 1 if args.single else args.max_iter,
        'sleep_between': args.sleep,
        'log_dir': args.log_dir,
        'backup_dir': args.backup_dir,
        'quick_mode': args.quick,
        'forced_phase': args.phase,
        'benchmark_compare': args.benchmark_compare,
        'bazel_dev': args.bazel_dev,
        'bazel_prod': args.bazel_prod,
        'stop_on_error': False,
        'timeout_multiplier': 5,  # Default multiplier
        'global_timeout': args.timeout
    }
    
    # Handle cleanup mode first (before creating runner)
    if args.cleanup:
        backup_dir = Path(config['backup_dir'])
        if backup_dir.exists():
            # Remove working file and iteration backups
            working_file = backup_dir / "instruction_work.md"
            if working_file.exists():
                working_file.unlink()
                print(f"{Colors.GREEN}✅ Supprimé: {working_file.name}{Colors.END}")
            
            # Remove iteration backups
            iteration_files = list(backup_dir.glob("instruction_iter_*.md"))
            for f in iteration_files:
                f.unlink()
            if iteration_files:
                print(f"{Colors.GREEN}✅ Supprimé: {len(iteration_files)} sauvegardes d'itération{Colors.END}")
            
            # Keep the original
            original = backup_dir / "instruction_original.md"
            if original.exists():
                print(f"{Colors.CYAN}📝 Conservé: {original.name} (fichier original){Colors.END}")
            
            print(f"{Colors.GREEN}🧹 Nettoyage terminé - prêt à repartir de zéro{Colors.END}")
        else:
            print(f"{Colors.YELLOW}⚠️ Aucun fichier de travail à nettoyer{Colors.END}")
        sys.exit(0)
    
    # Create runner
    runner = ClaudeTDDRunner(args.instructions, config)
    
    # Handle special modes
    if args.status:
        status = runner._get_status()
        runner._display_status(status)
        
    elif args.analyze_bench:
        # Analyze benchmark file for Safe/Unsafe decision
        with open(args.analyze_bench, 'r') as f:
            content = f.read()
        runner._analyze_benchmark_comparison(content)
        
    elif args.bazel_dev:
        # Run Bazel tests in dev mode
        package_path = runner.package_info['path']
        cmd = f"bazel test --config=dev //{package_path}:all"
        success, output = runner._run_bazel_command(cmd)
        if success:
            print(f"{Colors.GREEN}✅ Bazel DEV tests passed (safety checks active){Colors.END}")
        else:
            print(f"{Colors.RED}❌ Bazel DEV tests failed{Colors.END}")
            
    elif args.bazel_prod:
        # Run Bazel tests in prod mode
        package_path = runner.package_info['path']
        cmd = f"bazel test --config=prod //{package_path}:all"
        success, output = runner._run_bazel_command(cmd)
        if success:
            print(f"{Colors.GREEN}✅ Bazel PROD tests passed (unsafe_no_check active){Colors.END}")
        else:
            print(f"{Colors.RED}❌ Bazel PROD tests failed{Colors.END}")
            
    elif args.benchmark_compare:
        # Force benchmark comparison phase
        runner.current_phase = TDDPhase.BENCHMARKING_COMPARISON
        runner.run_single_iteration()
        
    else:
        # Normal TDD run
        runner.run()

if __name__ == "__main__":
    main()