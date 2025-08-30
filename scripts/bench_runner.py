#!/usr/bin/env python3
"""
Benchmark runner with SQLite storage and commit comparison.
Supports both single-core and multi-core benchmarks with git clone for isolation.
"""

import argparse
import json
import os
import re
import sqlite3
import subprocess
import sys
import tempfile
import time
from datetime import datetime
from pathlib import Path
from typing import Dict, List, Optional, Tuple

# ANSI color codes
class Colors:
    RED = '\033[0;31m'
    GREEN = '\033[0;32m'
    YELLOW = '\033[1;33m'
    BLUE = '\033[0;34m'
    MAGENTA = '\033[0;35m'
    CYAN = '\033[0;36m'
    NC = '\033[0m'  # No Color
    BOLD = '\033[1m'

class BenchmarkDB:
    """SQLite database for benchmark results."""
    
    def __init__(self, db_path: str = "benchmarks.sqlite"):
        self.db_path = db_path
        self.conn = sqlite3.connect(db_path)
        self.conn.row_factory = sqlite3.Row
        self._init_schema()
    
    def _init_schema(self):
        """Initialize database schema."""
        self.conn.executescript("""
            CREATE TABLE IF NOT EXISTS benchmarks (
                id INTEGER PRIMARY KEY AUTOINCREMENT,
                commit_hash TEXT NOT NULL,
                commit_date TEXT NOT NULL,
                branch TEXT,
                package TEXT NOT NULL,
                benchmark TEXT NOT NULL,
                mode TEXT NOT NULL,
                cores INTEGER DEFAULT 1,
                iterations INTEGER,
                ns_per_op REAL,
                mb_per_sec REAL,
                allocs_per_op INTEGER,
                bytes_per_op INTEGER,
                ops_per_sec REAL,
                custom_metrics TEXT,
                run_date TEXT NOT NULL,
                go_version TEXT,
                os TEXT,
                arch TEXT,
                UNIQUE(commit_hash, package, benchmark, mode)
            );
            
            CREATE INDEX IF NOT EXISTS idx_commit ON benchmarks(commit_hash);
            CREATE INDEX IF NOT EXISTS idx_benchmark ON benchmarks(package, benchmark);
            CREATE INDEX IF NOT EXISTS idx_mode ON benchmarks(mode);
            CREATE INDEX IF NOT EXISTS idx_date ON benchmarks(run_date);
            
            CREATE TABLE IF NOT EXISTS commits (
                commit_hash TEXT PRIMARY KEY,
                commit_date TEXT NOT NULL,
                author TEXT,
                message TEXT,
                branch TEXT
            );
        """)
        self.conn.commit()
    
    def save_benchmark(self, result: Dict):
        """Save a benchmark result."""
        try:
            # Save commit info if not exists
            self.conn.execute("""
                INSERT OR IGNORE INTO commits (commit_hash, commit_date, author, message, branch)
                VALUES (?, ?, ?, ?, ?)
            """, (
                result['commit_hash'],
                result['commit_date'],
                result.get('author', ''),
                result.get('message', ''),
                result.get('branch', '')
            ))
            
            # Save benchmark result
            self.conn.execute("""
                INSERT OR REPLACE INTO benchmarks (
                    commit_hash, commit_date, branch, package, benchmark,
                    mode, cores, iterations, ns_per_op, mb_per_sec,
                    allocs_per_op, bytes_per_op, ops_per_sec, custom_metrics,
                    run_date, go_version, os, arch
                ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
            """, (
                result['commit_hash'],
                result['commit_date'],
                result.get('branch', ''),
                result['package'],
                result['benchmark'],
                result['mode'],
                result.get('cores', 1),
                result.get('iterations'),
                result.get('ns_per_op'),
                result.get('mb_per_sec'),
                result.get('allocs_per_op'),
                result.get('bytes_per_op'),
                result.get('ops_per_sec'),
                json.dumps(result.get('custom_metrics', {})),
                result['run_date'],
                result.get('go_version', ''),
                result.get('os', ''),
                result.get('arch', '')
            ))
            self.conn.commit()
        except sqlite3.Error as e:
            print(f"{Colors.RED}Error saving benchmark: {e}{Colors.NC}")
    
    def get_results(self, commit_hash: str, mode: Optional[str] = None) -> List[sqlite3.Row]:
        """Get benchmark results for a commit."""
        query = "SELECT * FROM benchmarks WHERE commit_hash = ?"
        params = [commit_hash]
        if mode:
            query += " AND mode = ?"
            params.append(mode)
        query += " ORDER BY package, benchmark"
        return self.conn.execute(query, params).fetchall()
    
    def get_commits(self) -> List[sqlite3.Row]:
        """Get all commits with benchmarks."""
        return self.conn.execute("""
            SELECT DISTINCT c.*, COUNT(b.id) as benchmark_count
            FROM commits c
            JOIN benchmarks b ON c.commit_hash = b.commit_hash
            GROUP BY c.commit_hash
            ORDER BY c.commit_date DESC
        """).fetchall()
    
    def close(self):
        """Close database connection."""
        self.conn.close()

class BenchmarkRunner:
    """Runs benchmarks with support for git cloning and parallel execution."""
    
    def __init__(self, db: BenchmarkDB):
        self.db = db
        self.repo_url = self._get_repo_url()
    
    def _get_repo_url(self):
        """Get the current repository URL."""
        try:
            result = subprocess.run(
                ["git", "config", "--get", "remote.origin.url"],
                capture_output=True, text=True, check=True
            )
            return result.stdout.strip()
        except:
            # Fallback to local path
            return os.getcwd()
    
    def _get_commit_info(self, commit: str = "HEAD") -> Dict:
        """Get commit information."""
        try:
            # Get commit hash
            hash_result = subprocess.run(
                ["git", "rev-parse", commit],
                capture_output=True, text=True, check=True
            )
            commit_hash = hash_result.stdout.strip()[:8]
            
            # Get commit date
            date_result = subprocess.run(
                ["git", "show", "-s", "--format=%ci", commit],
                capture_output=True, text=True, check=True
            )
            commit_date = date_result.stdout.strip()
            
            # Get branch
            branch_result = subprocess.run(
                ["git", "rev-parse", "--abbrev-ref", "HEAD"],
                capture_output=True, text=True
            )
            branch = branch_result.stdout.strip() if branch_result.returncode == 0 else ""
            
            # Get author and message
            info_result = subprocess.run(
                ["git", "show", "-s", "--format=%an|%s", commit],
                capture_output=True, text=True, check=True
            )
            author, message = info_result.stdout.strip().split('|', 1)
            
            return {
                'commit_hash': commit_hash,
                'commit_date': commit_date,
                'branch': branch,
                'author': author,
                'message': message
            }
        except subprocess.CalledProcessError as e:
            print(f"{Colors.RED}Error getting commit info: {e}{Colors.NC}")
            return {}
    
    def _parse_benchmark_output(self, output: str, mode: str, cores: int = 1) -> List[Dict]:
        """Parse benchmark output from go test."""
        results = []
        package = ""
        
        # Regex patterns for parsing
        pkg_pattern = r'^pkg:\s+(.+)$'
        bench_pattern = r'^Benchmark(\S+?)(?:-\d+)?\s+(\d+)\s+(\d+(?:\.\d+)?)\s+ns/op'
        alloc_pattern = r'(\d+)\s+B/op\s+(\d+)\s+allocs/op'
        mb_pattern = r'(\d+(?:\.\d+)?)\s+MB/s'
        
        # Try to extract package from output
        for line in output.split('\n'):
            if line.startswith('pkg:'):
                package = line.split(':', 1)[1].strip()
                break
        
        for line in output.split('\n'):
            
            # Check for benchmark result
            bench_match = re.match(bench_pattern, line)
            if bench_match:
                result = {
                    'benchmark': bench_match.group(1),
                    'iterations': int(bench_match.group(2)),
                    'ns_per_op': float(bench_match.group(3)),
                    'package': package,
                    'mode': mode,
                    'cores': cores
                }
                
                # Calculate ops/sec
                if result['ns_per_op'] > 0:
                    result['ops_per_sec'] = 1_000_000_000 / result['ns_per_op']
                
                # Check for allocations
                alloc_match = re.search(alloc_pattern, line)
                if alloc_match:
                    result['bytes_per_op'] = int(alloc_match.group(1))
                    result['allocs_per_op'] = int(alloc_match.group(2))
                
                # Check for MB/s
                mb_match = re.search(mb_pattern, line)
                if mb_match:
                    result['mb_per_sec'] = float(mb_match.group(1))
                
                results.append(result)
        
        return results
    
    def run_benchmarks(self, commit: str = "HEAD", use_clone: bool = False) -> bool:
        """Run benchmarks for a specific commit."""
        commit_info = self._get_commit_info(commit)
        if not commit_info:
            return False
        
        print(f"{Colors.CYAN}Running benchmarks for commit {commit_info['commit_hash']}{Colors.NC}")
        print(f"  {Colors.BLUE}Date: {commit_info['commit_date']}{Colors.NC}")
        print(f"  {Colors.BLUE}Message: {commit_info['message']}{Colors.NC}")
        
        # Setup working directory (either clone or current)
        if use_clone and commit != "HEAD":
            tmpdir = tempfile.mkdtemp()
            try:
                clone_dir = os.path.join(tmpdir, "repo")
                print(f"{Colors.YELLOW}Cloning repository to {clone_dir}...{Colors.NC}")
                
                # Clone repo
                subprocess.run(
                    ["git", "clone", self.repo_url, clone_dir],
                    check=True, capture_output=True
                )
                
                # Checkout commit
                subprocess.run(
                    ["git", "checkout", commit],
                    cwd=clone_dir, check=True, capture_output=True
                )
                
                work_dir = clone_dir
            except Exception as e:
                import shutil
                shutil.rmtree(tmpdir, ignore_errors=True)
                raise e
        else:
            work_dir = os.getcwd()
            tmpdir = None
        
        try:
            # Get system info
            go_version = subprocess.run(
                ["go", "version"], capture_output=True, text=True
            ).stdout.strip()
            
            os_info = os.uname()
            
            # Find benchmark packages
            result = subprocess.run(
                ["bazel", "query", '//pkg/...'],
                cwd=work_dir, capture_output=True, text=True
            )
            
            if result.returncode != 0:
                print(f"{Colors.RED}Failed to find benchmark targets{Colors.NC}")
                return False
            
            all_targets = [t for t in result.stdout.strip().split('\n') if t.startswith('//') and ':bench' in t]
            
            # Separate single and multi targets
            single_targets = [t for t in all_targets if not t.endswith(':bench_multi')]
            multi_targets = [t for t in all_targets if t.endswith(':bench_multi')]
            
            if not single_targets and not multi_targets:
                print(f"{Colors.YELLOW}No benchmark targets found{Colors.NC}")
                return False
            
            print(f"{Colors.GREEN}Found {len(single_targets)} single-core and {len(multi_targets)} multi-core benchmark targets{Colors.NC}")
            
            # Get number of cores
            try:
                cores = os.cpu_count() or 4
            except:
                cores = 4
            
            run_date = datetime.now().isoformat()
            
            # Run single-core benchmarks
            if single_targets:
                print(f"\n{Colors.CYAN}━━━ Single-Core Benchmarks ━━━{Colors.NC}")
                for i, target in enumerate(single_targets, 1):
                    pkg_name = target.split(':')[0].split('/')[-1]
                    print(f"  [{i}/{len(single_targets)}] Running {pkg_name} (single-core)...")
                    
                    try:
                    # Run benchmark using bazel run when in clone mode
                        env = {**os.environ, "GOMAXPROCS": "1", "PAGER": "cat"}
                        if use_clone and commit != "HEAD":
                            # Use bazel run in clone mode
                            result = subprocess.run(
                                ["bazel", "run", target, "--", "-test.bench=.", "-test.benchtime=100ms"],
                                cwd=work_dir, capture_output=True, text=True,
                                env=env,
                                timeout=60  # 1 minute timeout
                            )
                        else:
                            # Build and run directly when in local mode
                            subprocess.run(
                                ["bazel", "build", target],
                                cwd=work_dir, capture_output=True, check=False
                            )
                            
                            # Get binary path
                            binary_result = subprocess.run(
                                ["bazel", "cquery", "--output=files", target],
                                cwd=work_dir, capture_output=True, text=True
                            )
                            
                            if binary_result.returncode != 0:
                                print(f"  {Colors.RED}✗{Colors.NC} {pkg_name} - could not find binary")
                                continue
                            
                            binary_path = binary_result.stdout.strip()
                            if not os.path.isabs(binary_path):
                                binary_path = os.path.join(work_dir, binary_path)
                            
                            result = subprocess.run(
                                [binary_path, "-test.bench=.", "-test.benchtime=100ms"],
                                cwd=work_dir, capture_output=True, text=True,
                                env=env,
                                timeout=60  # 1 minute timeout
                            )
                        
                        if result.returncode == 0:
                            benchmarks = self._parse_benchmark_output(result.stdout, "single", 1)
                            for bench in benchmarks:
                                bench.update(commit_info)
                                bench['run_date'] = run_date
                                bench['go_version'] = go_version
                                bench['os'] = os_info.sysname
                                bench['arch'] = os_info.machine
                                self.db.save_benchmark(bench)
                            print(f"  {Colors.GREEN}✓{Colors.NC} {pkg_name} complete ({len(benchmarks)} benchmarks)")
                        else:
                            print(f"  {Colors.RED}✗{Colors.NC} {pkg_name} failed")
                    except subprocess.TimeoutExpired:
                        print(f"  {Colors.RED}✗{Colors.NC} {pkg_name} timed out")
            
            # Run multi-core benchmarks
            if multi_targets:
                print(f"\n{Colors.CYAN}━━━ Multi-Core Benchmarks ({cores} cores) ━━━{Colors.NC}")
                for i, target in enumerate(multi_targets, 1):
                    pkg_name = target.split(':')[0].split('/')[-1]
                    print(f"  [{i}/{len(multi_targets)}] Running {pkg_name} ({cores} cores)...")
                    
                    try:
                    # Run benchmark using bazel run when in clone mode
                        env = {**os.environ, "GOMAXPROCS": str(cores), "PAGER": "cat"}
                        if use_clone and commit != "HEAD":
                            # Use bazel run in clone mode
                            result = subprocess.run(
                                ["bazel", "run", target, "--", "-test.bench=.", "-test.benchtime=100ms"],
                                cwd=work_dir, capture_output=True, text=True,
                                env=env,
                                timeout=60  # 1 minute timeout
                            )
                        else:
                            # Build and run directly when in local mode
                            subprocess.run(
                                ["bazel", "build", target],
                                cwd=work_dir, capture_output=True, check=False
                            )
                            
                            # Get binary path
                            binary_result = subprocess.run(
                                ["bazel", "cquery", "--output=files", target],
                                cwd=work_dir, capture_output=True, text=True
                            )
                            
                            if binary_result.returncode != 0:
                                print(f"  {Colors.RED}✗{Colors.NC} {pkg_name} - could not find binary")
                                continue
                            
                            binary_path = binary_result.stdout.strip()
                            if not os.path.isabs(binary_path):
                                binary_path = os.path.join(work_dir, binary_path)
                            
                            result = subprocess.run(
                                [binary_path, "-test.bench=.", "-test.benchtime=100ms"],
                                cwd=work_dir, capture_output=True, text=True,
                                env=env,
                                timeout=60  # 1 minute timeout
                            )
                        
                        if result.returncode == 0:
                            benchmarks = self._parse_benchmark_output(result.stdout, "multi", cores)
                            for bench in benchmarks:
                                bench.update(commit_info)
                                bench['run_date'] = run_date
                                bench['go_version'] = go_version
                                bench['os'] = os_info.sysname
                                bench['arch'] = os_info.machine
                                self.db.save_benchmark(bench)
                            print(f"  {Colors.GREEN}✓{Colors.NC} {pkg_name} complete ({len(benchmarks)} benchmarks)")
                        else:
                            print(f"  {Colors.RED}✗{Colors.NC} {pkg_name} failed")
                    except subprocess.TimeoutExpired:
                        print(f"  {Colors.RED}✗{Colors.NC} {pkg_name} timed out")
            
            print(f"\n{Colors.GREEN}✓ Benchmark results saved to {self.db.db_path}{Colors.NC}")
            return True
        finally:
            # Cleanup temp directory if we created one
            if tmpdir:
                import shutil
                shutil.rmtree(tmpdir, ignore_errors=True)

class BenchmarkComparator:
    """Compare benchmark results between commits."""
    
    def __init__(self, db: BenchmarkDB):
        self.db = db
    
    def _format_change(self, old_val: float, new_val: float) -> str:
        """Format performance change as percentage with color."""
        if not old_val or not new_val:
            return "N/A"
        
        change = ((new_val - old_val) / old_val) * 100
        
        # For ns/op and bytes, negative is better
        # For ops/sec and MB/s, positive is better
        if abs(change) < 1:
            color = Colors.NC
            symbol = "→"
        elif change < 0:
            color = Colors.GREEN
            symbol = "↓"
        else:
            color = Colors.RED
            symbol = "↑"
        
        return f"{color}{symbol} {abs(change):.1f}%{Colors.NC}"
    
    def _format_speedup(self, single_ns: float, multi_ns: float, cores: int) -> str:
        """Format parallel speedup."""
        if not single_ns or not multi_ns:
            return "N/A"
        
        speedup = single_ns / multi_ns
        efficiency = (speedup / cores) * 100
        
        if speedup >= cores * 0.8:  # 80% efficiency
            color = Colors.GREEN
        elif speedup >= cores * 0.5:  # 50% efficiency
            color = Colors.YELLOW
        else:
            color = Colors.RED
        
        return f"{color}{speedup:.2f}x ({efficiency:.0f}% eff){Colors.NC}"
    
    def compare_commits(self, commit1: str, commit2: str):
        """Compare benchmarks between two commits."""
        # Get commit info
        info1 = self.db.conn.execute(
            "SELECT * FROM commits WHERE commit_hash LIKE ?",
            (f"{commit1}%",)
        ).fetchone()
        
        info2 = self.db.conn.execute(
            "SELECT * FROM commits WHERE commit_hash LIKE ?",
            (f"{commit2}%",)
        ).fetchone()
        
        if not info1 or not info2:
            print(f"{Colors.RED}One or both commits not found in database{Colors.NC}")
            return
        
        print(f"\n{Colors.BOLD}Benchmark Comparison{Colors.NC}")
        print(f"{Colors.CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━{Colors.NC}")
        print(f"Base:    {info1['commit_hash']} - {info1['message'][:50]}")
        print(f"Compare: {info2['commit_hash']} - {info2['message'][:50]}")
        print()
        
        # Get all benchmarks
        results1 = self.db.get_results(info1['commit_hash'])
        results2 = self.db.get_results(info2['commit_hash'])
        
        # Group by package and benchmark
        bench_map1 = {}
        bench_map2 = {}
        
        for r in results1:
            key = (r['package'], r['benchmark'], r['mode'])
            bench_map1[key] = r
        
        for r in results2:
            key = (r['package'], r['benchmark'], r['mode'])
            bench_map2[key] = r
        
        # Group results by package
        packages = {}
        all_keys = set(bench_map1.keys()) | set(bench_map2.keys())
        
        for key in all_keys:
            pkg, bench, mode = key
            if pkg not in packages:
                packages[pkg] = {}
            if bench not in packages[pkg]:
                packages[pkg][bench] = {}
            packages[pkg][bench][mode] = key
        
        # Print comparison by package
        for pkg in sorted(packages.keys()):
            pkg_name = pkg.split('/')[-1] if pkg else "root"
            print(f"\n{Colors.BOLD}📦 {pkg_name}{Colors.NC}")
            print("─" * 85)
            
            # Print header with better formatting
            print(f"{'Benchmark':<35} {'Single-Core':<20} {'Multi-Core':<20} {'Scaling':<10}")
            print("─" * 85)
            
            for bench_name in sorted(packages[pkg].keys()):
                modes = packages[pkg][bench_name]
                
                # Get single and multi results
                single_key = modes.get('single')
                multi_key = modes.get('multi')
                
                single_r1 = bench_map1.get(single_key) if single_key else None
                single_r2 = bench_map2.get(single_key) if single_key else None
                multi_r1 = bench_map1.get(multi_key) if multi_key else None
                multi_r2 = bench_map2.get(multi_key) if multi_key else None
                
                # Format single-core results
                single_str = ""
                if single_r1 and single_r2:
                    change = self._format_change(single_r1['ns_per_op'], single_r2['ns_per_op'])
                    single_str = f"{single_r2['ns_per_op']:.1f}ns {change}"
                elif single_r2:
                    single_str = f"{single_r2['ns_per_op']:.1f}ns NEW"
                elif single_r1:
                    single_str = "REMOVED"
                else:
                    single_str = "-"
                
                # Format multi-core results
                multi_str = ""
                if multi_r1 and multi_r2:
                    change = self._format_change(multi_r1['ns_per_op'], multi_r2['ns_per_op'])
                    multi_str = f"{multi_r2['ns_per_op']:.1f}ns {change}"
                elif multi_r2:
                    multi_str = f"{multi_r2['ns_per_op']:.1f}ns NEW"
                elif multi_r1:
                    multi_str = "REMOVED"
                else:
                    multi_str = "-"
                
                # Calculate scaling efficiency if both modes exist
                scaling_str = ""
                if single_r2 and multi_r2 and multi_r2['cores'] > 1:
                    speedup = single_r2['ns_per_op'] / multi_r2['ns_per_op']
                    efficiency = (speedup / multi_r2['cores']) * 100
                    if efficiency > 80:
                        scaling_str = f"{Colors.GREEN}{efficiency:.0f}%{Colors.NC}"
                    elif efficiency > 50:
                        scaling_str = f"{Colors.YELLOW}{efficiency:.0f}%{Colors.NC}"
                    else:
                        scaling_str = f"{Colors.RED}{efficiency:.0f}%{Colors.NC}"
                
                # Only print if there's something to show
                if single_str != "-" or multi_str != "-":
                    bench_display = bench_name[:34]
                    print(f"{bench_display:<35} {single_str:<20} {multi_str:<20} {scaling_str:<10}")
    
    def compare_parallel_scaling(self, commit: str):
        """Compare single vs multi-core performance for a commit."""
        info = self.db.conn.execute(
            "SELECT * FROM commits WHERE commit_hash LIKE ?",
            (f"{commit}%",)
        ).fetchone()
        
        if not info:
            print(f"{Colors.RED}Commit not found in database{Colors.NC}")
            return
        
        print(f"\n{Colors.BOLD}Parallel Scaling Analysis{Colors.NC}")
        print(f"{Colors.CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━{Colors.NC}")
        print(f"Commit: {info['commit_hash']} - {info['message'][:50]}")
        print()
        
        # Get results
        single_results = self.db.get_results(info['commit_hash'], 'single')
        multi_results = self.db.get_results(info['commit_hash'], 'multi')
        
        if not single_results or not multi_results:
            print(f"{Colors.YELLOW}Missing single or multi-core results{Colors.NC}")
            return
        
        # Get cores from multi results
        cores = multi_results[0]['cores'] if multi_results else 4
        
        # Group by package and benchmark
        single_map = {(r['package'], r['benchmark']): r for r in single_results}
        multi_map = {(r['package'], r['benchmark']): r for r in multi_results}
        
        # Compare
        packages = {}
        for key in set(single_map.keys()) & set(multi_map.keys()):
            pkg, bench = key
            if pkg not in packages:
                packages[pkg] = []
            packages[pkg].append(bench)
        
        for pkg in sorted(packages.keys()):
            print(f"\n{Colors.BOLD}{pkg.split('/')[-1]}{Colors.NC}")
            print("─" * 80)
            
            # Print header
            print(f"{'Benchmark':<30} {'Single-core':<15} {f'Multi ({cores} cores)':<15} {'Speedup':<20}")
            print("─" * 80)
            
            for bench in sorted(packages[pkg]):
                single = single_map[(pkg, bench)]
                multi = multi_map[(pkg, bench)]
                
                speedup = self._format_speedup(single['ns_per_op'], multi['ns_per_op'], cores)
                
                print(f"{bench[:29]:<30} "
                      f"{single['ns_per_op']:.1f} ns{'':<5} "
                      f"{multi['ns_per_op']:.1f} ns{'':<5} "
                      f"{speedup:<20}")

def main():
    parser = argparse.ArgumentParser(description='Benchmark Runner and Analyzer')
    subparsers = parser.add_subparsers(dest='command', help='Commands')
    
    # Run command
    run_parser = subparsers.add_parser('run', help='Run benchmarks')
    run_parser.add_argument('commit', nargs='?', default='HEAD',
                           help='Commit to benchmark (default: HEAD)')
    run_parser.add_argument('--clone', action='store_true',
                           help='Clone to temp directory for isolation')
    run_parser.add_argument('--db', default='benchmarks.sqlite',
                           help='Database file (default: benchmarks.sqlite)')
    
    # Compare command
    compare_parser = subparsers.add_parser('compare', help='Compare benchmarks')
    compare_parser.add_argument('commit1', help='First commit (base)')
    compare_parser.add_argument('commit2', nargs='?', help='Second commit (default: HEAD)')
    compare_parser.add_argument('--db', default='benchmarks.sqlite',
                           help='Database file')
    
    # Scaling command
    scaling_parser = subparsers.add_parser('scaling', help='Analyze parallel scaling')
    scaling_parser.add_argument('commit', nargs='?', default='HEAD',
                               help='Commit to analyze')
    scaling_parser.add_argument('--db', default='benchmarks.sqlite',
                               help='Database file')
    
    # List command
    list_parser = subparsers.add_parser('list', help='List saved benchmarks')
    list_parser.add_argument('--db', default='benchmarks.sqlite',
                           help='Database file')
    
    args = parser.parse_args()
    
    if not args.command:
        parser.print_help()
        return 1
    
    # Initialize database
    db = BenchmarkDB(args.db)
    
    try:
        if args.command == 'run':
            runner = BenchmarkRunner(db)
            success = runner.run_benchmarks(args.commit, args.clone)
            return 0 if success else 1
        
        elif args.command == 'compare':
            commit2 = args.commit2 or 'HEAD'
            comparator = BenchmarkComparator(db)
            comparator.compare_commits(args.commit1, commit2)
            return 0
        
        elif args.command == 'scaling':
            comparator = BenchmarkComparator(db)
            comparator.compare_parallel_scaling(args.commit)
            return 0
        
        elif args.command == 'list':
            commits = db.get_commits()
            if not commits:
                print(f"{Colors.YELLOW}No benchmarks found in database{Colors.NC}")
                return 0
            
            print(f"\n{Colors.BOLD}Saved Benchmarks{Colors.NC}")
            print(f"{Colors.CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━{Colors.NC}")
            print(f"{'Commit':<10} {'Date':<20} {'Benchmarks':<12} {'Message':<40}")
            print("─" * 80)
            
            for commit in commits:
                date = commit['commit_date'][:19]  # Remove timezone
                msg = commit['message'][:39]
                print(f"{commit['commit_hash']:<10} {date:<20} "
                      f"{commit['benchmark_count']:<12} {msg:<40}")
            
            return 0
    
    finally:
        db.close()

if __name__ == '__main__':
    sys.exit(main())