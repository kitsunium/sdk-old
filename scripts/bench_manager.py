#!/usr/bin/env python3

import argparse
import json
import os
import re
import sqlite3
import subprocess
import sys
from datetime import datetime
from typing import Dict, List, Optional, Tuple

# ANSI color codes
RED = "\033[0;31m"
GREEN = "\033[0;32m"
YELLOW = "\033[1;33m"
BLUE = "\033[0;34m"
CYAN = "\033[0;36m"
NC = "\033[0m"  # No Color


class BenchmarkManager:
    def __init__(self, db_path: str = "benchmarks.sqlite"):
        self.db_path = db_path
        self.conn = None
        self.init_database()

    def init_database(self):
        """Initialize SQLite database with benchmark results schema."""
        self.conn = sqlite3.connect(self.db_path)
        cursor = self.conn.cursor()
        
        # Create benchmarks table
        cursor.execute("""
            CREATE TABLE IF NOT EXISTS benchmarks (
                id INTEGER PRIMARY KEY AUTOINCREMENT,
                timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
                commit_hash TEXT NOT NULL,
                branch TEXT,
                package TEXT NOT NULL,
                test_name TEXT NOT NULL,
                iterations INTEGER,
                ns_per_op REAL,
                mb_per_sec REAL,
                bytes_per_op INTEGER,
                allocs_per_op INTEGER,
                raw_output TEXT
            )
        """)
        
        # Create index for efficient queries
        cursor.execute("""
            CREATE INDEX IF NOT EXISTS idx_benchmark_lookup 
            ON benchmarks(package, test_name, commit_hash)
        """)
        
        self.conn.commit()

    def get_current_commit(self) -> Tuple[str, str, bool]:
        """Get current git commit hash, branch, and dirty status."""
        try:
            commit_hash = subprocess.check_output(
                ["git", "rev-parse", "HEAD"], 
                text=True
            ).strip()
            
            branch = subprocess.check_output(
                ["git", "rev-parse", "--abbrev-ref", "HEAD"], 
                text=True
            ).strip()
            
            # Check if working directory is dirty (has uncommitted changes)
            try:
                subprocess.check_output(
                    ["git", "diff-index", "--quiet", "HEAD", "--"],
                    stderr=subprocess.DEVNULL
                )
                is_dirty = False
            except subprocess.CalledProcessError:
                is_dirty = True
            
            return commit_hash, branch, is_dirty
        except subprocess.CalledProcessError:
            print(f"{RED}Error: Not in a git repository{NC}")
            sys.exit(1)
    
    def get_main_commit(self) -> Optional[str]:
        """Get the most recent commit hash from main branch with saved benchmarks."""
        cursor = self.conn.cursor()
        cursor.execute("""
            SELECT commit_hash 
            FROM benchmarks 
            WHERE branch = 'main' 
            GROUP BY commit_hash
            ORDER BY timestamp DESC 
            LIMIT 1
        """)
        result = cursor.fetchone()
        return result[0] if result else None

    def parse_benchmark_output(self, output: str, package: str) -> List[Dict]:
        """Parse benchmark output from Go test format."""
        results = []
        
        # Pattern for benchmark results
        # Example: BenchmarkBuffer_Write/size_64-10    393359812    3.013 ns/op    21238.17 MB/s    0 B/op    0 allocs/op
        pattern = re.compile(
            r'^(Benchmark\S+)(?:-(\d+))?\s+'  # Benchmark name and optional CPU count
            r'(\d+)\s+'                        # Iterations
            r'([\d.]+)\s+ns/op'               # Nanoseconds per operation
            r'(?:\s+([\d.]+)\s+MB/s)?'        # Optional MB/s
            r'(?:\s+(\d+)\s+B/op)?'           # Optional bytes per operation
            r'(?:\s+(\d+)\s+allocs/op)?'      # Optional allocations per operation
        )
        
        for line in output.split('\n'):
            match = pattern.match(line.strip())
            if match:
                test_name = match.group(1)
                iterations = int(match.group(3))
                ns_per_op = float(match.group(4))
                mb_per_sec = float(match.group(5)) if match.group(5) else None
                bytes_per_op = int(match.group(6)) if match.group(6) else 0
                allocs_per_op = int(match.group(7)) if match.group(7) else 0
                
                results.append({
                    'package': package,
                    'test_name': test_name,
                    'iterations': iterations,
                    'ns_per_op': ns_per_op,
                    'mb_per_sec': mb_per_sec,
                    'bytes_per_op': bytes_per_op,
                    'allocs_per_op': allocs_per_op,
                    'raw_output': line
                })
        
        return results

    def run_benchmarks(self, targets: Optional[List[str]] = None) -> Dict[str, str]:
        """Run Bazel benchmarks and collect results."""
        if targets is None:
            # Get all benchmark targets
            try:
                targets_output = subprocess.check_output(
                    ["bazel", "query", 'attr(tags, "bench", //...)'],
                    stderr=subprocess.DEVNULL,
                    text=True
                ).strip()
                targets = targets_output.split('\n') if targets_output else []
            except subprocess.CalledProcessError:
                print(f"{RED}Error: Failed to query benchmark targets{NC}")
                return {}
        
        results = {}
        
        for target in targets:
            if not target:
                continue
                
            # Extract package from target (e.g., //pkg/kernel/kbuffer:test -> pkg/kernel/kbuffer)
            package = target.replace('//', '').split(':')[0]
            
            print(f"{CYAN}Running benchmark: {target}{NC}")
            
            try:
                # Run benchmark
                output = subprocess.check_output(
                    [
                        "bazel", "run", target, "--",
                        "-test.bench=.", "-test.benchmem",
                        "-test.benchtime=100ms", "-test.run=^$"
                    ],
                    stderr=subprocess.STDOUT,
                    text=True
                )
                
                # Filter out Bazel output
                filtered_lines = []
                for line in output.split('\n'):
                    if not any(skip in line for skip in ['exec ', 'Executing tests', '---', 'Computing', 'Loading', 'Analyzing', 'INFO:', 'Target', 'goos:', 'goarch:', 'cpu:']):
                        if line.strip() and line.startswith('Benchmark'):
                            filtered_lines.append(line)
                
                results[package] = '\n'.join(filtered_lines)
                
            except subprocess.CalledProcessError as e:
                print(f"{RED}Error running benchmark {target}: {e}{NC}")
                continue
        
        return results

    def save_results(self, results: Dict[str, str], commit_hash: str, branch: str, is_dirty: bool = False):
        """Save benchmark results to database."""
        cursor = self.conn.cursor()
        
        # Use "current" as commit hash for uncommitted changes
        save_hash = "current" if is_dirty else commit_hash
        
        # Delete existing results for this commit (replace instead of append)
        cursor.execute("DELETE FROM benchmarks WHERE commit_hash = ?", (save_hash,))
        
        for package, output in results.items():
            if not output:
                continue
                
            parsed_results = self.parse_benchmark_output(output, package)
            
            for result in parsed_results:
                cursor.execute("""
                    INSERT INTO benchmarks (
                        commit_hash, branch, package, test_name,
                        iterations, ns_per_op, mb_per_sec, 
                        bytes_per_op, allocs_per_op, raw_output
                    ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
                """, (
                    save_hash, branch, result['package'], result['test_name'],
                    result['iterations'], result['ns_per_op'], result['mb_per_sec'],
                    result['bytes_per_op'], result['allocs_per_op'], result['raw_output']
                ))
        
        self.conn.commit()
        print(f"{GREEN}✓ Results saved to {self.db_path}{NC}")

    def compare_commits(self, base_commit: str, current_commit: str):
        """Compare benchmark results between two specific commits."""
        cursor = self.conn.cursor()
        
        # Validate both commits exist in database
        cursor.execute("""
            SELECT COUNT(DISTINCT commit_hash) 
            FROM benchmarks 
            WHERE commit_hash LIKE ? OR commit_hash LIKE ?
        """, (base_commit + '%', current_commit + '%'))
        
        count = cursor.fetchone()[0]
        if count < 2:
            print(f"{RED}❌ Error: Both commits must have saved benchmark results{NC}")
            print(f"\nChecking commits:")
            
            # Check base commit
            cursor.execute("SELECT commit_hash FROM benchmarks WHERE commit_hash LIKE ? LIMIT 1", (base_commit + '%',))
            base_result = cursor.fetchone()
            if base_result:
                print(f"  {GREEN}✓{NC} Base commit {base_commit} found: {base_result[0][:8]}")
            else:
                print(f"  {RED}✗{NC} Base commit {base_commit} not found")
            
            # Check current commit  
            cursor.execute("SELECT commit_hash FROM benchmarks WHERE commit_hash LIKE ? LIMIT 1", (current_commit + '%',))
            current_result = cursor.fetchone()
            if current_result:
                print(f"  {GREEN}✓{NC} Current commit {current_commit} found: {current_result[0][:8]}")
            else:
                print(f"  {RED}✗{NC} Current commit {current_commit} not found")
            
            print(f"\n{YELLOW}Run 'make bench-list' to see available commits{NC}")
            print(f"{YELLOW}Run 'make bench-save' to save current benchmark results{NC}")
            sys.exit(1)
        
        # Get full commit hashes
        cursor.execute("SELECT DISTINCT commit_hash FROM benchmarks WHERE commit_hash LIKE ? LIMIT 1", (base_commit + '%',))
        base_full = cursor.fetchone()[0]
        
        cursor.execute("SELECT DISTINCT commit_hash FROM benchmarks WHERE commit_hash LIKE ? LIMIT 1", (current_commit + '%',))
        current_full = cursor.fetchone()[0]
        
        # Compare results
        print(f"\n{BLUE}{'='*80}{NC}")
        print(f"{BLUE}Benchmark Comparison{NC}")
        print(f"{BLUE}Base:    {base_full[:8]}{NC}")
        print(f"{BLUE}Current: {current_full[:8]}{NC}")
        print(f"{BLUE}{'='*80}{NC}\n")
        
        # Get all benchmarks from both commits
        cursor.execute("""
            SELECT DISTINCT package, test_name
            FROM benchmarks
            WHERE commit_hash IN (?, ?)
            ORDER BY package, test_name
        """, (base_full, current_full))
        
        all_tests = cursor.fetchall()
        current_package = None
        
        for package, test_name in all_tests:
            # Print package header when it changes
            if package != current_package:
                if current_package is not None:
                    print()  # Space between packages
                print(f"{CYAN}Package: {package}{NC}")
                print("-" * 100)
                current_package = package
            
            # Get base results
            cursor.execute("""
                SELECT ns_per_op, mb_per_sec, bytes_per_op, allocs_per_op
                FROM benchmarks
                WHERE commit_hash = ? AND package = ? AND test_name = ?
                ORDER BY timestamp DESC
                LIMIT 1
            """, (base_full, package, test_name))
            base_result = cursor.fetchone()
            
            # Get current results
            cursor.execute("""
                SELECT ns_per_op, mb_per_sec, bytes_per_op, allocs_per_op
                FROM benchmarks
                WHERE commit_hash = ? AND package = ? AND test_name = ?
                ORDER BY timestamp DESC
                LIMIT 1
            """, (current_full, package, test_name))
            current_result = cursor.fetchone()
            
            # Format test name
            test_display = test_name.replace('Benchmark', '')
            if len(test_display) > 35:
                test_display = test_display[:32] + "..."
            
            print(f"  {test_display:35}", end=" ")
            
            if base_result and current_result:
                base_ns, base_mb, base_bytes, base_allocs = base_result
                curr_ns, curr_mb, curr_bytes, curr_allocs = current_result
                
                # Show ns/op comparison
                ns_change = ((curr_ns - base_ns) / base_ns) * 100 if base_ns else 0
                ns_color = GREEN if ns_change < 0 else RED if ns_change > 0 else NC
                ns_symbol = "↓" if ns_change < 0 else "↑" if ns_change > 0 else "="
                
                print(f"{base_ns:7.2f} → {curr_ns:7.2f} ns/op ", end="")
                print(f"{ns_color}{ns_symbol}{abs(ns_change):5.1f}%{NC}", end="  ")
                
                # Show MB/s comparison if available
                if base_mb and curr_mb:
                    mb_change = ((curr_mb - base_mb) / base_mb) * 100
                    mb_color = GREEN if mb_change > 0 else RED if mb_change < 0 else NC
                    mb_symbol = "↑" if mb_change > 0 else "↓" if mb_change < 0 else "="
                    
                    print(f"{base_mb:8.1f} → {curr_mb:8.1f} MB/s ", end="")
                    print(f"{mb_color}{mb_symbol}{abs(mb_change):5.1f}%{NC}", end="")
                
                # Show allocation changes if different
                if base_allocs != curr_allocs:
                    alloc_diff = curr_allocs - base_allocs
                    alloc_color = GREEN if alloc_diff < 0 else RED
                    print(f"  {alloc_color}allocs: {base_allocs} → {curr_allocs}{NC}", end="")
                
            elif base_result and not current_result:
                # Test removed in current
                print(f"{RED}[REMOVED]{NC} - was {base_result[0]:.2f} ns/op", end="")
                
            elif not base_result and current_result:
                # Test added in current
                curr_ns, curr_mb, curr_bytes, curr_allocs = current_result
                print(f"{YELLOW}[NEW]{NC} - {curr_ns:.2f} ns/op", end="")
                if curr_mb:
                    print(f"  {curr_mb:.1f} MB/s", end="")
                if curr_allocs:
                    print(f"  {curr_allocs} allocs/op", end="")
            
            print()  # New line
        
        print()
        
        print(f"{BLUE}{'='*80}{NC}")
        print(f"{GREEN}✓ Comparison complete and results saved{NC}")

    def list_commits(self, limit: int = 25):
        """List commits with saved benchmarks."""
        cursor = self.conn.cursor()
        cursor.execute("""
            SELECT DISTINCT 
                b.commit_hash, 
                b.branch, 
                b.timestamp, 
                COUNT(*) as bench_count,
                GROUP_CONCAT(DISTINCT b.package) as packages
            FROM benchmarks b
            GROUP BY b.commit_hash
            ORDER BY b.timestamp DESC
            LIMIT ?
        """, (limit,))
        
        rows = cursor.fetchall()
        
        if not rows:
            print(f"{RED}❌ No benchmark results found{NC}")
            print(f"{YELLOW}Run 'make bench-save' to save benchmark results{NC}")
            return
        
        print(f"\n{BLUE}{'='*80}{NC}")
        print(f"{BLUE}Saved Benchmark Results (Last {min(len(rows), limit)} commits){NC}")
        print(f"{BLUE}{'='*80}{NC}\n")
        
        print(f"{'Commit':10} {'Branch':20} {'Date':16} {'Time':8} {'Tests':>6}  Packages")
        print("-" * 80)
        
        for i, (commit, branch, timestamp, count, packages) in enumerate(rows):
            dt = datetime.fromisoformat(timestamp)
            
            # Special handling for "current" commit
            if commit == "current":
                commit_display = f"{YELLOW}current{NC}  "
                color = YELLOW
            else:
                # Color code recent commits
                if i == 0:
                    color = GREEN  # Most recent
                elif i < 5:
                    color = CYAN   # Recent
                else:
                    color = ""     # Older
                commit_display = f"{color}{commit[:8]}{NC}"
            
            # Truncate branch name if too long
            if len(branch) > 18:
                branch = branch[:15] + "..."
            
            # Format packages list
            pkg_list = packages.split(',') if packages else []
            pkg_display = ', '.join([p.split('/')[-1] for p in pkg_list[:3]])
            if len(pkg_list) > 3:
                pkg_display += f" (+{len(pkg_list)-3} more)"
            
            print(f"{commit_display}  {branch:20} {dt.strftime('%Y-%m-%d'):16} {dt.strftime('%H:%M'):8} {count:>6}  {pkg_display}")
        
        print(f"\n{CYAN}ℹ Usage: make bench/compare BASE_COMMIT CURRENT_COMMIT{NC}")
        print(f"{CYAN}Example: make bench/compare {rows[-1][0][:8]} {rows[0][0][:8]}{NC}\n")

    def close(self):
        """Close database connection."""
        if self.conn:
            self.conn.close()


def main():
    parser = argparse.ArgumentParser(description='Benchmark Manager for Bazel')
    subparsers = parser.add_subparsers(dest='command', help='Commands')
    
    # Save command
    save_parser = subparsers.add_parser('save', help='Run benchmarks and save results')
    save_parser.add_argument('--targets', nargs='*', help='Specific targets to benchmark')
    
    # Compare command  
    compare_parser = subparsers.add_parser('compare', help='Compare benchmark results')
    compare_parser.add_argument('base', nargs='?', help='Base commit hash (default: main branch)')
    compare_parser.add_argument('current', nargs='?', help='Current commit hash (default: current HEAD)')
    
    # List command
    list_parser = subparsers.add_parser('list', help='List saved benchmark results')
    list_parser.add_argument('--limit', type=int, default=25, help='Number of commits to display (default: 25)')
    
    # Database path option
    parser.add_argument('--db', default='benchmarks.sqlite', help='Path to SQLite database')
    
    args = parser.parse_args()
    
    if not args.command:
        parser.print_help()
        sys.exit(1)
    
    manager = BenchmarkManager(args.db)
    
    try:
        if args.command == 'save':
            commit_hash, branch, is_dirty = manager.get_current_commit()
            if is_dirty:
                print(f"{YELLOW}▶ Running benchmarks for uncommitted changes (current) on {branch}{NC}")
            else:
                print(f"{YELLOW}▶ Running benchmarks for commit: {commit_hash[:8]} ({branch}){NC}")
            results = manager.run_benchmarks(args.targets)
            if results:
                manager.save_results(results, commit_hash, branch, is_dirty)
            else:
                print(f"{RED}No benchmark results collected{NC}")
        
        elif args.command == 'compare':
            # Determine which commits to compare
            if args.base is None:
                # No arguments: compare current with main
                commit_hash, _, is_dirty = manager.get_current_commit()
                current_hash = "current" if is_dirty else commit_hash
                base_hash = manager.get_main_commit()
                if base_hash is None:
                    print(f"{RED}Error: No benchmarks found for main branch{NC}")
                    sys.exit(1)
            elif args.current is None:
                # One argument: compare current with specified commit
                commit_hash, _, is_dirty = manager.get_current_commit()
                current_hash = "current" if is_dirty else commit_hash
                base_hash = args.base
            else:
                # Two arguments: compare two specified commits
                base_hash = args.base
                current_hash = args.current
            
            manager.compare_commits(base_hash, current_hash)
        
        elif args.command == 'list':
            manager.list_commits(args.limit)
    
    finally:
        manager.close()


if __name__ == '__main__':
    main()