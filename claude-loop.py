#!/usr/bin/env python3
"""
Generic TDD Loop Runner - Works with any instruction file
Sets working directory to the instruction file's directory
Preserves original by working on a copy
"""

import subprocess
import sys
import time
import argparse
from pathlib import Path
from datetime import datetime
import shutil
import json
import re
import os

class Colors:
    GREEN = '\033[92m'
    YELLOW = '\033[93m'
    RED = '\033[91m'
    CYAN = '\033[96m'
    BOLD = '\033[1m'
    END = '\033[0m'

class TDDLoop:
    def __init__(self, instructions_path: str, config: dict):
        # Store full path to original instruction file
        self.original_file = Path(instructions_path).resolve()
        
        if not self.original_file.exists():
            raise FileNotFoundError(f"Instruction file not found: {self.original_file}")
        
        # Set working directory to the instruction file's directory
        self.working_dir = self.original_file.parent
        self.original_dir = Path.cwd()
        os.chdir(self.working_dir)
        
        # Create working copy name (original.md -> original.working.md)
        stem = self.original_file.stem
        suffix = self.original_file.suffix
        self.instructions_file = self.original_file.parent / f"{stem}.working{suffix}"
        
        # Create working copy if it doesn't exist
        if not self.instructions_file.exists():
            shutil.copy2(self.original_file, self.instructions_file)
            print(f"✨ Created working copy: {self.instructions_file.name}")
        else:
            print(f"📂 Using existing working copy: {self.instructions_file.name}")
        
        # Store relative path from working dir (just the filename)
        self.instructions_name = self.instructions_file.name
        
        # Configuration
        self.iteration = 0
        self.max_iterations = config.get('max_iterations', 500)
        self.timeout = config.get('timeout', 1800)
        self.sleep_between = config.get('sleep_between', 2)
        self.debug = config.get('debug', False)
        
        # Create backup directory in working dir
        self.backup_dir = Path('.tdd-backups')
        self.backup_dir.mkdir(exist_ok=True)
        
        # Load initial state
        self.load_state()
        
        print(f"📁 Working directory: {self.working_dir}")
        print(f"📄 Original file: {self.original_file.name} (preserved)")
        print(f"📝 Working copy: {self.instructions_name}")
    
    def __del__(self):
        """Restore original directory when done"""
        if hasattr(self, 'original_dir'):
            os.chdir(self.original_dir)
    
    def load_state(self):
        """Load current state from instruction file"""
        content = self.instructions_file.read_text()
        
        # Extract iteration number
        iter_match = re.search(r'\*\*Itération\*\*:\s*(\d+)', content)
        if iter_match:
            self.iteration = int(iter_match.group(1))
    
    def get_existing_files(self) -> dict:
        """Scan working directory for all files, organized by directory"""
        files_by_dir = {}
        
        # Walk through working directory
        for root, dirs, files in os.walk('.'):
            # Skip hidden directories
            dirs[:] = [d for d in dirs if not d.startswith('.')]
            
            # Skip if no files
            if not files:
                continue
                
            # Get relative path
            rel_path = Path(root).relative_to('.')
            
            # Skip hidden files and instruction files
            visible_files = [f for f in files 
                           if not f.startswith('.') 
                           and f != self.instructions_name
                           and f != self.original_file.name]
            
            if visible_files:
                files_by_dir[str(rel_path)] = visible_files
        
        return files_by_dir
    
    def extract_tasks(self) -> tuple:
        """Extract unchecked and checked tasks"""
        content = self.instructions_file.read_text()
        unchecked = re.findall(r'- \[ \] (.+)', content)
        checked = re.findall(r'- \[x\] (.+)', content)
        return unchecked, checked
    
    def update_instruction_file(self, files_created: list, commands_run: list, message: str):
        """Update instruction file with progress"""
        content = self.instructions_file.read_text()
        original_content = content
        
        # Increment iteration
        self.iteration += 1
        content = re.sub(
            r'(\*\*Itération\*\*):\s*\d+',
            f'\\1: {self.iteration}',
            content
        )
        
        # Check off tasks based on created files - MORE STRICT MATCHING
        for file_path in files_created:
            filename = Path(file_path).name
            
            # Only check off tasks that explicitly mention creating this specific file
            patterns = [
                # Match "Créer filename" exactly
                rf'(- \[)[ ](]\s*Créer {re.escape(filename)}[^]]*)',
                # Match "Create filename" exactly
                rf'(- \[)[ ](]\s*Create {re.escape(filename)}[^]]*)',
            ]
            
            for pattern in patterns:
                content = re.sub(pattern, r'\1x\2', content, count=1)  # Only replace first match
        
        # Check off tasks based on commands run
        for cmd_output in commands_run:
            cmd = cmd_output.split(':')[0]
            
            # Check off tasks that mention executing this command
            patterns = [
                rf'(- \[)[ ](]\s*[^]]*exécuter[^]]*{re.escape(cmd)}[^]]*)',
                rf'(- \[)[ ](]\s*[^]]*Exécuter[^]]*{re.escape(cmd)}[^]]*)',
                rf'(- \[)[ ](]\s*[^]]*run[^]]*{re.escape(cmd)}[^]]*)',
            ]
            
            for pattern in patterns:
                content = re.sub(pattern, r'\1x\2', content, flags=re.IGNORECASE, count=1)
        
        # Add iteration log
        log_entry = f"""
### Itération {self.iteration} - {datetime.now().strftime('%Y-%m-%d %H:%M')}
- **Fichiers créés**: {', '.join(files_created) if files_created else 'Aucun'}
- **Message**: {message}
"""
        
        # Add or update log section
        if "## 📝 LOG DES ITÉRATIONS" in content:
            content = re.sub(
                r'(## 📝 LOG DES ITÉRATIONS[^\n]*\n)',
                f'\\1{log_entry}',
                content
            )
        elif "## LOG DES ITÉRATIONS" in content:
            content = re.sub(
                r'(## LOG DES ITÉRATIONS[^\n]*\n)',
                f'\\1{log_entry}',
                content
            )
        else:
            content += f"\n\n## 📝 LOG DES ITÉRATIONS\n{log_entry}"
        
        # Save if changed
        if content != original_content:
            # Backup current version
            backup_file = self.backup_dir / f'iter_{self.iteration:03d}_{self.instructions_name}'
            shutil.copy2(self.instructions_file, backup_file)
            
            # Update file
            self.instructions_file.write_text(content)
            print(f"    📝 Updated {self.instructions_name}", Colors.GREEN)
    
    def generate_prompt(self) -> str:
        """Generate prompt for Claude"""
        content = self.instructions_file.read_text()
        existing_files = self.get_existing_files()
        unchecked_tasks, checked_tasks = self.extract_tasks()
        
        # Debug mode
        if self.debug:
            print(f"🔍 Debug: {len(unchecked_tasks)} tâches non cochées")
            if unchecked_tasks:
                print(f"   Prochaines: {unchecked_tasks[:3]}")
        
        # Format existing files
        files_list = []
        for dir_path, files in existing_files.items():
            if dir_path == '.':
                files_list.extend(files)
            else:
                files_list.extend([f"{dir_path}/{f}" for f in files])
        
        prompt = f"""Tu dois suivre EXACTEMENT les instructions dans {self.instructions_name}.
Ton répertoire de travail actuel est: {self.working_dir}

FICHIERS EXISTANTS DANS LE PROJET:
{chr(10).join(files_list) if files_list else 'Aucun fichier créé'}

TÂCHES À FAIRE (non cochées):
{chr(10).join(unchecked_tasks[:10]) if unchecked_tasks else 'Toutes les tâches sont complétées'}

TÂCHES DÉJÀ FAITES (cochées):
{chr(10).join(checked_tasks[:5]) if checked_tasks else 'Aucune tâche complétée'}

INSTRUCTION CRITIQUE:
1. Lis attentivement {self.instructions_name} pour comprendre le projet
2. Suis STRICTEMENT les règles dans .claude/rules/ (priorité absolue)
3. Identifie la PROCHAINE tâche non cochée à réaliser
4. Exécute EXACTEMENT UNE tâche à la fois (création de fichier OU exécution de commande)
5. Si la tâche demande d'exécuter une commande (make fmt, go test, etc.), exécute-la
6. Utilise les chemins relatifs depuis le répertoire de travail

DÉCISION:
- Si il reste des tâches non cochées → exécute la prochaine tâche
- Si TOUTES les tâches sont cochées (et seulement dans ce cas) → retourne status:"completed"

Réponds UNIQUEMENT avec un JSON valide (pas de texte avant ou après):
{{
  "files": {{
    "chemin/relatif/fichier.ext": "contenu complet du fichier"
  }},
  "commands": {{
    "commande": "sortie de la commande (ou 'OK' si succès)"
  }},
  "status": "in_progress" ou "completed",
  "message": "Description de ce qui a été fait"
}}

NOTE: Le champ "files" est pour créer des fichiers, "commands" pour exécuter des commandes.
"""
        return prompt
    
    def run_claude(self, prompt: str) -> tuple:
        """Execute Claude with the prompt"""
        try:
            result = subprocess.run(
                ['claude'],
                input=prompt,
                capture_output=True,
                text=True,
                timeout=self.timeout,
                cwd=self.working_dir  # Ensure Claude runs in working dir
            )
            return result.returncode == 0, result.stdout
        except subprocess.TimeoutExpired:
            return False, f"Timeout after {self.timeout}s"
        except FileNotFoundError:
            return False, "Claude CLI not found. Please install it first."
        except Exception as e:
            return False, str(e)
    
    def parse_response(self, output: str) -> tuple:
        """Parse Claude's JSON response"""
        try:
            # Extract JSON from output
            json_match = re.search(r'\{.*\}', output, re.DOTALL)
            if not json_match:
                return [], [], "error", "No JSON found in response"
            
            data = json.loads(json_match.group())
            
            files_created = []
            for file_path, content in data.get('files', {}).items():
                # Create file relative to working directory
                path = Path(file_path)
                path.parent.mkdir(parents=True, exist_ok=True)
                path.write_text(content)
                files_created.append(file_path)
            
            commands_run = []
            for cmd, output in data.get('commands', {}).items():
                commands_run.append(f"{cmd}: {output[:100]}..." if len(output) > 100 else f"{cmd}: {output}")
                print(f"    ⚡ Executed: {cmd}", Colors.CYAN)
            
            status = data.get('status', 'in_progress')
            message = data.get('message', '')
            
            return files_created, commands_run, status, message
            
        except json.JSONDecodeError as e:
            return [], [], "error", f"Invalid JSON: {e}"
        except Exception as e:
            return [], [], "error", str(e)
    
    def save_iteration_debug(self, iteration: int, prompt: str, output: str):
        """Save iteration data for debugging"""
        iter_dir = self.backup_dir / f'iter_{iteration:03d}'
        iter_dir.mkdir(exist_ok=True)
        
        (iter_dir / 'prompt.txt').write_text(prompt)
        (iter_dir / 'output.txt').write_text(output)
        
        # Try to extract and save parsed JSON
        json_match = re.search(r'\{.*\}', output, re.DOTALL)
        if json_match:
            try:
                data = json.loads(json_match.group())
                (iter_dir / 'parsed.json').write_text(
                    json.dumps(data, indent=2, ensure_ascii=False)
                )
            except:
                pass
    
    def check_completion(self) -> bool:
        """Check if all tasks are complete"""
        unchecked, _ = self.extract_tasks()
        return len(unchecked) == 0
    
    def run(self):
        """Main execution loop"""
        print(f"\n🔄 TDD Loop Started", Colors.BOLD + Colors.CYAN)
        print(f"⚙️  Configuration:")
        print(f"    Max iterations: {self.max_iterations}")
        print(f"    Timeout: {self.timeout}s")
        print(f"    Sleep: {self.sleep_between}s")
        print(f"    Starting iteration: {self.iteration}\n")
        
        try:
            while self.iteration < self.max_iterations:
                # Check completion
                if self.check_completion():
                    print("\n✅ All tasks completed!", Colors.GREEN + Colors.BOLD)
                    break
                
                print(f"\n{'='*60}", Colors.CYAN)
                print(f"📍 Iteration {self.iteration + 1}/{self.max_iterations}", Colors.BOLD)
                print(f"{'='*60}", Colors.CYAN)
                
                # Show current state
                unchecked, checked = self.extract_tasks()
                print(f"📊 Progress: {len(checked)} done, {len(unchecked)} remaining")
                
                # Generate and run prompt
                prompt = self.generate_prompt()
                print("⏳ Running Claude...", Colors.YELLOW)
                
                success, output = self.run_claude(prompt)
                
                # Save for debugging
                self.save_iteration_debug(self.iteration + 1, prompt, output)
                
                if success:
                    files_created, commands_run, status, message = self.parse_response(output)
                    
                    # Verify completion claim
                    if status == "completed":
                        unchecked, _ = self.extract_tasks()
                        if len(unchecked) > 0:
                            print(f"    ⚠️ Claude dit 'completed' mais {len(unchecked)} tâches restent!", Colors.YELLOW)
                            status = "in_progress"
                        else:
                            print("🎯 Project completed!", Colors.GREEN + Colors.BOLD)
                            break
                    
                    if files_created or commands_run:
                        for file in files_created:
                            print(f"    ✅ Created: {file}", Colors.GREEN)
                        
                        self.update_instruction_file(files_created, commands_run, message)
                        
                        if message:
                            print(f"    💬 {message[:100]}", Colors.CYAN)
                    else:
                        print(f"    ⚠️ No files created or commands run", Colors.YELLOW)
                        if message:
                            print(f"    💬 {message}", Colors.YELLOW)
                else:
                    print(f"❌ Error: {output[:200]}", Colors.RED)
                
                # Sleep before next iteration
                if not self.check_completion():
                    print(f"💤 Waiting {self.sleep_between}s...", Colors.CYAN)
                    time.sleep(self.sleep_between)
                    
        except KeyboardInterrupt:
            print("\n\n⚠️ Interrupted by user", Colors.YELLOW)
        finally:
            # Final summary
            print(f"\n{'='*60}", Colors.BOLD)
            print("📊 Final Summary")
            print(f"{'='*60}")
            print(f"  Total iterations: {self.iteration}")
            
            unchecked, checked = self.extract_tasks()
            print(f"  Tasks completed: {len(checked)}")
            print(f"  Tasks remaining: {len(unchecked)}")
            
            existing = self.get_existing_files()
            total_files = sum(len(files) for files in existing.values())
            print(f"  Files created: {total_files}")
            
            print(f"\n💾 Backups saved in: {self.backup_dir.absolute()}/")

def print_colored(msg: str, color: str = ""):
    """Print with color"""
    print(f"{color}{msg}{Colors.END}")

def main():
    parser = argparse.ArgumentParser(
        description="Generic TDD Loop Runner with Command Execution",
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
Examples:
  %(prog)s path/to/instruction.md
  %(prog)s ../project/specs.md --max 20
  %(prog)s docs/tdd.md --timeout 90 --sleep 3
        """
    )
    
    parser.add_argument('instruction_file', 
                       help='Path to instruction file (any .md file)')
    parser.add_argument('--max', type=int, default=500, 
                       help='Max iterations (default: 500)')
    parser.add_argument('--timeout', type=int, default=1800, 
                       help='Timeout per iteration in seconds (default: 1800)')
    parser.add_argument('--sleep', type=int, default=2, 
                       help='Sleep between iterations (default: 2)')
    parser.add_argument('--debug', action='store_true',
                       help='Enable debug mode for verbose output')
    
    args = parser.parse_args()
    
    config = {
        'max_iterations': args.max,
        'timeout': args.timeout,
        'sleep_between': args.sleep,
        'debug': args.debug
    }
    
    try:
        loop = TDDLoop(args.instruction_file, config)
        loop.run()
    except FileNotFoundError as e:
        print_colored(f"❌ Error: {e}", Colors.RED)
        sys.exit(1)
    except Exception as e:
        print_colored(f"❌ Unexpected error: {e}", Colors.RED)
        sys.exit(1)

if __name__ == "__main__":
    main()