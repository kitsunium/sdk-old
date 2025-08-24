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

    def get_current_commit(self) -> Tuple[str, str]:
        """Get current git commit hash and branch."""
        try:
            commit_hash = subprocess.check_output(
                ["git", "rev-parse", "HEAD"], 
                text=True
            ).strip()
            
            branch = subprocess.check_output(
                ["git", "rev-parse", "--abbrev-ref", "HEAD"], 
                text=True
            ).strip()
            
            return commit_hash, branch
        except subprocess.CalledProcessError:
            print(f"{RED}Error: Not in a git repository{NC}")
            sys.exit(1)

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
                        "-test.benchtime=1s", "-test.run=^$"
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

    def save_results(self, results: Dict[str, str], commit_hash: str, branch: str):
        """Save benchmark results to database."""
        cursor = self.conn.cursor()
        
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
                    commit_hash, branch, result['package'], result['test_name'],
                    result['iterations'], result['ns_per_op'], result['mb_per_sec'],
                    result['bytes_per_op'], result['allocs_per_op'], result['raw_output']
                ))
        
        self.conn.commit()
        print(f"{GREEN}✓ Results saved to {self.db_path}{NC}")

    def compare_results(self, targets: Optional[List[str]] = None, base_commit: Optional[str] = None):
        """Run benchmarks and compare with previous results."""
        current_commit, current_branch = self.get_current_commit()
        
        # Run current benchmarks
        print(f"{YELLOW}▶ Running benchmarks for current commit: {current_commit[:8]}{NC}")
        current_results = self.run_benchmarks(targets)
        
        if not current_results:
            print(f"{RED}No benchmark results to compare{NC}")
            return
        
        # Get base commit for comparison
        if base_commit is None:
            # Try to find the latest saved results
            cursor = self.conn.cursor()
            cursor.execute("""
                SELECT DISTINCT commit_hash 
                FROM benchmarks 
                WHERE commit_hash != ?
                ORDER BY timestamp DESC 
                LIMIT 1
            """, (current_commit,))
            row = cursor.fetchone()
            base_commit = row[0] if row else None
        
        if not base_commit:
            print(f"{YELLOW}No previous results found for comparison{NC}")
            print(f"{YELLOW}Saving current results for future comparisons{NC}")
            self.save_results(current_results, current_commit, current_branch)
            return
        
        # Compare results
        print(f"\n{BLUE}{'='*80}{NC}")
        print(f"{BLUE}Benchmark Comparison{NC}")
        print(f"{BLUE}Base:    {base_commit[:8]}{NC}")
        print(f"{BLUE}Current: {current_commit[:8]} ({current_branch}){NC}")
        print(f"{BLUE}{'='*80}{NC}\n")
        
        cursor = self.conn.cursor()
        
        for package, output in current_results.items():
            current_benchmarks = self.parse_benchmark_output(output, package)
            
            print(f"{CYAN}Package: {package}{NC}")
            print("-" * 70)
            
            for bench in current_benchmarks:
                # Get baseline result
                cursor.execute("""
                    SELECT ns_per_op, mb_per_sec, bytes_per_op, allocs_per_op
                    FROM benchmarks
                    WHERE commit_hash = ? AND package = ? AND test_name = ?
                    ORDER BY timestamp DESC
                    LIMIT 1
                """, (base_commit, package, bench['test_name']))
                
                baseline = cursor.fetchone()
                
                if baseline:
                    base_ns, base_mb, base_bytes, base_allocs = baseline
                    
                    # Calculate changes
                    ns_change = ((bench['ns_per_op'] - base_ns) / base_ns) * 100 if base_ns else 0
                    
                    # Format test name
                    test_display = bench['test_name'].replace('Benchmark', '')
                    if len(test_display) > 40:
                        test_display = test_display[:37] + "..."
                    
                    print(f"  {test_display:40}", end=" ")
                    
                    # Show ns/op with change
                    ns_color = GREEN if ns_change < 0 else RED if ns_change > 0 else NC
                    ns_symbol = "↓" if ns_change < 0 else "↑" if ns_change > 0 else "="
                    print(f"{bench['ns_per_op']:8.2f} ns/op ", end="")
                    print(f"{ns_color}{ns_symbol}{abs(ns_change):6.1f}%{NC}", end="  ")
                    
                    # Show MB/s if available
                    if bench['mb_per_sec'] and base_mb:
                        mb_change = ((bench['mb_per_sec'] - base_mb) / base_mb) * 100
                        mb_color = GREEN if mb_change > 0 else RED if mb_change < 0 else NC
                        mb_symbol = "↑" if mb_change > 0 else "↓" if mb_change < 0 else "="
                        print(f"{bench['mb_per_sec']:8.1f} MB/s ", end="")
                        print(f"{mb_color}{mb_symbol}{abs(mb_change):5.1f}%{NC}", end="  ")
                    
                    # Show allocations if changed
                    if bench['allocs_per_op'] != base_allocs:
                        alloc_diff = bench['allocs_per_op'] - base_allocs
                        alloc_color = GREEN if alloc_diff < 0 else RED
                        alloc_symbol = "+" if alloc_diff > 0 else ""
                        print(f"{alloc_color}allocs: {alloc_symbol}{alloc_diff}{NC}", end="")
                    
                    print()  # New line
                    
                else:
                    # New benchmark
                    test_display = bench['test_name'].replace('Benchmark', '')
                    if len(test_display) > 40:
                        test_display = test_display[:37] + "..."
                    print(f"  {test_display:40} {YELLOW}[NEW]{NC}")
                    print(f"    {bench['ns_per_op']:.2f} ns/op", end="")
                    if bench['mb_per_sec']:
                        print(f"  {bench['mb_per_sec']:.1f} MB/s", end="")
                    if bench['bytes_per_op']:
                        print(f"  {bench['bytes_per_op']} B/op", end="")
                    if bench['allocs_per_op']:
                        print(f"  {bench['allocs_per_op']} allocs/op", end="")
                    print()
            
            print()
        
        # Save current results
        self.save_results(current_results, current_commit, current_branch)
        
        print(f"{BLUE}{'='*80}{NC}")
        print(f"{GREEN}✓ Comparison complete and results saved{NC}")

    def list_commits(self):
        """List all commits with saved benchmarks."""
        cursor = self.conn.cursor()
        cursor.execute("""
            SELECT DISTINCT commit_hash, branch, timestamp, COUNT(*) as bench_count
            FROM benchmarks
            GROUP BY commit_hash
            ORDER BY timestamp DESC
        """)
        
        rows = cursor.fetchall()
        
        if not rows:
            print(f"{YELLOW}No benchmark results found{NC}")
            return
        
        print(f"{BLUE}Saved benchmark results:{NC}")
        print("-" * 80)
        
        for commit, branch, timestamp, count in rows:
            dt = datetime.fromisoformat(timestamp)
            print(f"  {commit[:8]}  {branch:15}  {dt.strftime('%Y-%m-%d %H:%M')}  {count} benchmarks")

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
    compare_parser = subparsers.add_parser('compare', help='Run benchmarks and compare with previous results')
    compare_parser.add_argument('--base', help='Base commit to compare against')
    compare_parser.add_argument('--targets', nargs='*', help='Specific targets to benchmark')
    
    # List command
    list_parser = subparsers.add_parser('list', help='List saved benchmark results')
    
    # Database path option
    parser.add_argument('--db', default='benchmarks.sqlite', help='Path to SQLite database')
    
    args = parser.parse_args()
    
    if not args.command:
        parser.print_help()
        sys.exit(1)
    
    manager = BenchmarkManager(args.db)
    
    try:
        if args.command == 'save':
            commit_hash, branch = manager.get_current_commit()
            print(f"{YELLOW}▶ Running benchmarks for commit: {commit_hash[:8]} ({branch}){NC}")
            results = manager.run_benchmarks(args.targets)
            if results:
                manager.save_results(results, commit_hash, branch)
            else:
                print(f"{RED}No benchmark results collected{NC}")
        
        elif args.command == 'compare':
            manager.compare_results(args.targets, args.base)
        
        elif args.command == 'list':
            manager.list_commits()
    
    finally:
        manager.close()


if __name__ == '__main__':
    main()