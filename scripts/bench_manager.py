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

try:
    from tabulate import tabulate
    HAS_TABULATE = True
except ImportError:
    HAS_TABULATE = False
    print("Warning: tabulate not installed. Install with: pip install tabulate")
    print("Falling back to basic formatting.\n")

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
        
        self._create_tables(cursor)
        self._create_indexes(cursor)
        self.conn.commit()

    def _create_tables(self, cursor):
        """Create database tables."""
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

    def _create_indexes(self, cursor):
        """Create database indexes for efficient queries."""
        cursor.execute("""
            CREATE INDEX IF NOT EXISTS idx_benchmark_lookup 
            ON benchmarks(package, test_name, commit_hash)
        """)

    def get_current_commit(self) -> Tuple[str, str, bool]:
        """Get current git commit hash, branch, and dirty status."""
        try:
            # Use static commands directly for security
            commit_hash = subprocess.check_output(
                ["git", "rev-parse", "HEAD"],
                text=True
            ).strip()
            
            branch = subprocess.check_output(
                ["git", "rev-parse", "--abbrev-ref", "HEAD"],
                text=True
            ).strip()
            
            is_dirty = self._check_dirty_status()
            return commit_hash, branch, is_dirty
        except subprocess.CalledProcessError:
            print(f"{RED}Error: Not in a git repository{NC}")
            sys.exit(1)

    def _check_dirty_status(self) -> bool:
        """Check if working directory has uncommitted changes."""
        try:
            subprocess.check_output(
                ["git", "diff-index", "--quiet", "HEAD", "--"],
                stderr=subprocess.DEVNULL
            )
            return False
        except subprocess.CalledProcessError:
            return True
    
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
        pattern = self._get_benchmark_pattern()
        
        for line in output.split('\n'):
            match = pattern.match(line.strip())
            if match:
                result = self._extract_benchmark_data(match, package, line)
                results.append(result)
        
        return results

    def _get_benchmark_pattern(self) -> re.Pattern:
        """Get regex pattern for parsing benchmark output."""
        return re.compile(
            r'^(Benchmark\S+?)(?:-(\d+))?\s+'  # Benchmark name and optional CPU count (non-greedy)
            r'(\d+)\s+'                         # Iterations
            r'([\d.]+)\s+ns/op'                # Nanoseconds per operation
            r'(?:\s+([\d.]+)\s+MB/s)?'         # Optional MB/s
            r'(?:\s+(\d+)\s+B/op)?'            # Optional bytes per operation
            r'(?:\s+(\d+)\s+allocs/op)?'       # Optional allocations per operation
        )

    def _extract_benchmark_data(self, match: re.Match, package: str, line: str) -> Dict:
        """Extract benchmark data from regex match."""
        return {
            'package': package,
            'test_name': match.group(1),
            'iterations': int(match.group(3)),
            'ns_per_op': float(match.group(4)),
            'mb_per_sec': float(match.group(5)) if match.group(5) else None,
            'bytes_per_op': int(match.group(6)) if match.group(6) else 0,
            'allocs_per_op': int(match.group(7)) if match.group(7) else 0,
            'raw_output': line
        }

    def run_benchmarks(self, targets: Optional[List[str]] = None) -> Dict[str, str]:
        """Run Bazel benchmarks and collect results."""
        if targets is None:
            targets = self._get_benchmark_targets()
        
        results = {}
        for target in targets:
            if not target:
                continue
            
            package = self._extract_package_from_target(target)
            output = self._run_single_benchmark(target)
            
            if output is not None:
                results[package] = self._filter_benchmark_output(output)
        
        return results

    def _get_benchmark_targets(self) -> List[str]:
        """Query Bazel for all benchmark targets."""
        try:
            output = subprocess.check_output(
                ["bazel", "query", 'attr(tags, "bench", //...)'],
                stderr=subprocess.DEVNULL,
                text=True
            ).strip()
            return output.split('\n') if output else []
        except subprocess.CalledProcessError:
            print(f"{RED}Error: Failed to query benchmark targets{NC}")
            return []

    def _extract_package_from_target(self, target: str) -> str:
        """Extract package name from Bazel target."""
        return target.replace('//', '').split(':')[0]

    def _run_single_benchmark(self, target: str) -> Optional[str]:
        """Run a single benchmark target."""
        print(f"{CYAN}Running benchmark: {target}{NC}")
        
        try:
            return subprocess.check_output(
                [
                    "bazel", "run", "--config=perf", target, "--",
                    "-test.bench=.", "-test.benchmem",
                    "-test.benchtime=10ms", "-test.run=^$"
                ],
                stderr=subprocess.STDOUT,
                text=True
            )
        except subprocess.CalledProcessError as e:
            print(f"{RED}Error running benchmark {target}: {e}{NC}")
            return None

    def _filter_benchmark_output(self, output: str) -> str:
        """Filter Bazel output to keep only benchmark results."""
        skip_patterns = ['exec ', 'Executing tests', '---', 'Computing', 
                        'Loading', 'Analyzing', 'INFO:', 'Target', 
                        'goos:', 'goarch:', 'cpu:']
        
        filtered_lines = []
        for line in output.split('\n'):
            if not any(skip in line for skip in skip_patterns):
                if line.strip() and line.startswith('Benchmark'):
                    filtered_lines.append(line)
        
        return '\n'.join(filtered_lines)

    def save_results(self, results: Dict[str, str], commit_hash: str, 
                    branch: str, is_dirty: bool = False, preserve_history: bool = False):
        """Save benchmark results to database."""
        cursor = self.conn.cursor()
        
        save_hash = "current" if is_dirty else commit_hash
        
        # Only delete existing results if not preserving history
        if not preserve_history:
            self._delete_existing_results(cursor, save_hash, is_dirty)
        else:
            # In preserve_history mode, only delete 'current' entries
            cursor.execute("DELETE FROM benchmarks WHERE commit_hash = 'current'")
        
        self._insert_new_results(cursor, results, save_hash, branch)
        
        self.conn.commit()
        print(f"{GREEN}✓ Results saved to {self.db_path}{NC}")

    def _delete_existing_results(self, cursor, save_hash: str, is_dirty: bool):
        """Delete existing results for this commit."""
        cursor.execute("DELETE FROM benchmarks WHERE commit_hash = ?", (save_hash,))
        
        if not is_dirty:
            cursor.execute("DELETE FROM benchmarks WHERE commit_hash = 'current'")

    def _insert_new_results(self, cursor, results: Dict[str, str], 
                           save_hash: str, branch: str):
        """Insert new benchmark results into database."""
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

    def compare_commits(self, base_commit: str, current_commit: str):
        """Compare benchmark results between two specific commits."""
        cursor = self.conn.cursor()
        
        if not self._validate_commits(cursor, base_commit, current_commit):
            return
        
        base_full, current_full = self._get_full_commit_hashes(cursor, base_commit, current_commit)
        
        self._print_comparison_header(base_full, current_full)
        self._compare_all_benchmarks(cursor, base_full, current_full)
        self._print_comparison_footer()

    def _validate_commits(self, cursor, base_commit: str, current_commit: str) -> bool:
        """Validate that both commits exist in database."""
        cursor.execute("""
            SELECT COUNT(DISTINCT commit_hash) 
            FROM benchmarks 
            WHERE commit_hash LIKE ? OR commit_hash LIKE ?
        """, (base_commit + '%', current_commit + '%'))
        
        count = cursor.fetchone()[0]
        if count < 2:
            self._print_commit_validation_error(cursor, base_commit, current_commit)
            return False
        return True

    def _print_commit_validation_error(self, cursor, base_commit: str, current_commit: str):
        """Print error message for missing commits."""
        print(f"{RED}❌ Error: Both commits must have saved benchmark results{NC}")
        print(f"\nChecking commits:")
        
        self._check_single_commit(cursor, base_commit, "Base")
        self._check_single_commit(cursor, current_commit, "Current")
        
        print(f"\n{YELLOW}Run 'make bench/list' to see available commits{NC}")
        print(f"{YELLOW}Run 'make bench/save' to save current benchmark results{NC}")
        sys.exit(1)

    def _check_single_commit(self, cursor, commit: str, label: str):
        """Check if a single commit exists in database."""
        cursor.execute("SELECT commit_hash FROM benchmarks WHERE commit_hash LIKE ? LIMIT 1", 
                      (commit + '%',))
        result = cursor.fetchone()
        if result:
            print(f"  {GREEN}✓{NC} {label} commit {commit} found: {result[0][:8]}")
        else:
            print(f"  {RED}✗{NC} {label} commit {commit} not found")

    def _get_full_commit_hashes(self, cursor, base_commit: str, current_commit: str) -> Tuple[str, str]:
        """Get full commit hashes from partial hashes."""
        cursor.execute("SELECT DISTINCT commit_hash FROM benchmarks WHERE commit_hash LIKE ? LIMIT 1", 
                      (base_commit + '%',))
        base_full = cursor.fetchone()[0]
        
        cursor.execute("SELECT DISTINCT commit_hash FROM benchmarks WHERE commit_hash LIKE ? LIMIT 1", 
                      (current_commit + '%',))
        current_full = cursor.fetchone()[0]
        
        return base_full, current_full

    def _print_comparison_header(self, base_full: str, current_full: str):
        """Print comparison header."""
        print(f"\n{BLUE}{'='*80}{NC}")
        print(f"{BLUE}Benchmark Comparison{NC}")
        print(f"{BLUE}Base:    {base_full[:8]}{NC}")
        print(f"{BLUE}Current: {current_full[:8]}{NC}")
        print(f"{BLUE}{'='*80}{NC}\n")

    def _print_comparison_footer(self):
        """Print comparison footer."""
        print(f"\n{BLUE}{'='*80}{NC}")
        print(f"{GREEN}✓ Comparison complete{NC}")

    def _compare_all_benchmarks(self, cursor, base_full: str, current_full: str):
        """Compare all benchmarks between two commits."""
        all_tests = self._get_all_tests(cursor, base_full, current_full)
        current_package = None
        
        if HAS_TABULATE:
            self._compare_with_tabulate(cursor, base_full, current_full, all_tests)
        else:
            for package, test_name in all_tests:
                if package != current_package:
                    self._print_package_header(package, current_package)
                    current_package = package
                
                self._compare_single_benchmark(cursor, base_full, current_full, package, test_name)
    
    def _compare_with_tabulate(self, cursor, base_full: str, current_full: str, all_tests):
        """Compare benchmarks using tabulate for better formatting."""
        current_package = None
        table_data = []
        
        for package, test_name in all_tests:
            if package != current_package:
                if current_package is not None and table_data:
                    # Print the table for the previous package
                    headers = ["Test", "Base (ns/op)", "Current (ns/op)", "Change", "MB/s", "Allocs/op"]
                    print(tabulate(table_data, headers=headers, tablefmt="simple"))
                    print()
                    table_data = []
                
                # Print new package header
                print(f"\n{CYAN}Package: {package}{NC}")
                print("-" * 120)
                current_package = package
            
            # Get benchmark results
            test_name_base = re.sub(r'-\d+$', '', test_name)
            base_result = self._get_benchmark_result_flexible(cursor, base_full, package, test_name_base)
            current_result = self._get_benchmark_result_flexible(cursor, current_full, package, test_name_base)
            
            if base_result and current_result:
                base_ns, base_mb, base_bytes, base_allocs = base_result
                curr_ns, curr_mb, curr_bytes, curr_allocs = current_result
                
                # Format base and current values with colors
                base_ns_str = f"{CYAN}{base_ns:.2f}{NC}"
                curr_ns_str = f"{YELLOW}{curr_ns:.2f}{NC}"
                
                # Calculate change
                ns_change = ((curr_ns - base_ns) / base_ns) * 100 if base_ns else 0
                if ns_change < -5:
                    change_color = GREEN
                    change_symbol = "↓"
                elif ns_change > 5:
                    change_color = RED
                    change_symbol = "↑"
                else:
                    change_color = ""
                    change_symbol = "="
                change_str = f"{change_color}{change_symbol}{abs(ns_change):.1f}%{NC}"
                
                # Format MB/s with both values
                mb_str = ""
                if base_mb and curr_mb:
                    mb_change = ((curr_mb - base_mb) / base_mb) * 100
                    mb_color = GREEN if mb_change > 5 else RED if mb_change < -5 else ""
                    mb_symbol = "↑" if mb_change > 0 else "↓" if mb_change < 0 else "="
                    mb_str = f"{CYAN}{base_mb:.1f}{NC} → {YELLOW}{curr_mb:.1f}{NC} {mb_color}{mb_symbol}{abs(mb_change):.1f}%{NC}"
                elif curr_mb:
                    mb_str = f"- → {YELLOW}{curr_mb:.1f}{NC}"
                
                # Format allocations with both values
                alloc_str = ""
                if base_allocs > 0 or curr_allocs > 0:
                    if base_allocs != curr_allocs:
                        alloc_color = GREEN if curr_allocs < base_allocs else RED if curr_allocs > base_allocs else ""
                        alloc_str = f"{CYAN}{base_allocs}{NC} → {YELLOW}{curr_allocs}{NC}"
                        if base_allocs > 0:
                            alloc_change = ((curr_allocs - base_allocs) / base_allocs) * 100
                            alloc_symbol = "↑" if alloc_change > 0 else "↓" if alloc_change < 0 else "="
                            alloc_str += f" {alloc_color}{alloc_symbol}{abs(alloc_change):.1f}%{NC}"
                    else:
                        alloc_str = f"{curr_allocs}"
                
                test_display = test_name_base.replace('Benchmark', '')
                table_data.append([test_display, base_ns_str, curr_ns_str, change_str, mb_str, alloc_str])
            
            elif base_result and not current_result:
                test_display = test_name_base.replace('Benchmark', '')
                table_data.append([test_display, base_result[0], "-", f"{RED}[REMOVED]{NC}", "", ""])
            
            elif not base_result and current_result:
                test_display = test_name_base.replace('Benchmark', '')
                curr_ns, curr_mb, curr_bytes, curr_allocs = current_result
                mb_str = f"{curr_mb:.1f} MB/s" if curr_mb else ""
                alloc_str = f"{curr_allocs} allocs" if curr_allocs else ""
                table_data.append([test_display, "-", curr_ns, f"{YELLOW}[NEW]{NC}", mb_str, alloc_str])
        
        # Print the last package's table
        if table_data:
            headers = ["Test", "Base (ns/op)", "Current (ns/op)", "Change", "MB/s", "Allocs/op"]
            print(tabulate(table_data, headers=headers, tablefmt="simple"))

    def _get_all_tests(self, cursor, base_full: str, current_full: str) -> List[Tuple[str, str]]:
        """Get all test names from both commits, normalized without CPU suffix."""
        cursor.execute("""
            SELECT DISTINCT package, test_name
            FROM benchmarks
            WHERE commit_hash IN (?, ?)
        """, (base_full, current_full))
        
        # Normalize test names by removing CPU suffix
        normalized_tests = {}
        for package, test_name in cursor.fetchall():
            # Remove CPU suffix (e.g., -2, -4, -8, etc.)
            normalized_name = re.sub(r'-\d+$', '', test_name)
            key = (package, normalized_name)
            if key not in normalized_tests:
                normalized_tests[key] = True
        
        # Sort and return unique normalized tests
        return sorted(normalized_tests.keys())

    def _print_package_header(self, package: str, current_package: Optional[str]):
        """Print package header when it changes."""
        if current_package is not None:
            print()  # Space between packages
        print(f"{CYAN}Package: {package}{NC}")
        print("-" * 100)

    def _compare_single_benchmark(self, cursor, base_full: str, current_full: str, 
                                 package: str, test_name: str):
        """Compare a single benchmark between two commits."""
        # Strip CPU suffix for comparison
        test_name_base = re.sub(r'-\d+$', '', test_name)
        
        # Try to find matching benchmark with any CPU count
        base_result = self._get_benchmark_result_flexible(cursor, base_full, package, test_name_base)
        current_result = self._get_benchmark_result_flexible(cursor, current_full, package, test_name_base)
        
        test_display = self._format_test_name(test_name_base)
        print(f"  {test_display:35}", end=" ")
        
        if base_result and current_result:
            self._print_comparison(base_result, current_result)
        elif base_result and not current_result:
            self._print_removed_test(base_result)
        elif not base_result and current_result:
            self._print_new_test(current_result)
        
        print()  # New line

    def _get_benchmark_result(self, cursor, commit: str, package: str, test_name: str) -> Optional[Tuple]:
        """Get benchmark result for a specific test."""
        cursor.execute("""
            SELECT ns_per_op, mb_per_sec, bytes_per_op, allocs_per_op
            FROM benchmarks
            WHERE commit_hash = ? AND package = ? AND test_name = ?
            ORDER BY timestamp DESC
            LIMIT 1
        """, (commit, package, test_name))
        return cursor.fetchone()
    
    def _get_benchmark_result_flexible(self, cursor, commit: str, package: str, test_name_base: str) -> Optional[Tuple]:
        """Get benchmark result for a test, ignoring CPU suffix."""
        # First try exact match
        cursor.execute("""
            SELECT ns_per_op, mb_per_sec, bytes_per_op, allocs_per_op, test_name
            FROM benchmarks
            WHERE commit_hash = ? AND package = ? AND test_name = ?
            ORDER BY timestamp DESC
            LIMIT 1
        """, (commit, package, test_name_base))
        result = cursor.fetchone()
        
        if result:
            return result[:-1]  # Return without test_name
        
        # Try with any CPU suffix
        cursor.execute("""
            SELECT ns_per_op, mb_per_sec, bytes_per_op, allocs_per_op, test_name
            FROM benchmarks
            WHERE commit_hash = ? AND package = ? AND (test_name = ? OR test_name LIKE ?)
            ORDER BY timestamp DESC
            LIMIT 1
        """, (commit, package, test_name_base, test_name_base + '-%'))
        result = cursor.fetchone()
        
        if result:
            return result[:-1]  # Return without test_name
        
        return None

    def _format_test_name(self, test_name: str) -> str:
        """Format test name for display."""
        test_display = test_name.replace('Benchmark', '')
        if len(test_display) > 35:
            test_display = test_display[:32] + "..."
        return test_display

    def _print_comparison(self, base_result: Tuple, current_result: Tuple):
        """Print comparison between two benchmark results."""
        base_ns, base_mb, base_bytes, base_allocs = base_result
        curr_ns, curr_mb, curr_bytes, curr_allocs = current_result
        
        self._print_ns_comparison(base_ns, curr_ns)
        
        if base_mb and curr_mb:
            self._print_mb_comparison(base_mb, curr_mb)
        
        if base_allocs != curr_allocs:
            self._print_alloc_comparison(base_allocs, curr_allocs)

    def _print_ns_comparison(self, base_ns: float, curr_ns: float):
        """Print nanoseconds per operation comparison."""
        ns_change = ((curr_ns - base_ns) / base_ns) * 100 if base_ns else 0
        ns_color = GREEN if ns_change < 0 else RED if ns_change > 0 else NC
        ns_symbol = "↓" if ns_change < 0 else "↑" if ns_change > 0 else "="
        
        print(f"{base_ns:7.2f} → {curr_ns:7.2f} ns/op ", end="")
        print(f"{ns_color}{ns_symbol}{abs(ns_change):5.1f}%{NC}", end="  ")

    def _print_mb_comparison(self, base_mb: float, curr_mb: float):
        """Print MB/s comparison."""
        mb_change = ((curr_mb - base_mb) / base_mb) * 100
        mb_color = GREEN if mb_change > 0 else RED if mb_change < 0 else NC
        mb_symbol = "↑" if mb_change > 0 else "↓" if mb_change < 0 else "="
        
        print(f"{base_mb:8.1f} → {curr_mb:8.1f} MB/s ", end="")
        print(f"{mb_color}{mb_symbol}{abs(mb_change):5.1f}%{NC}", end="")

    def _print_alloc_comparison(self, base_allocs: int, curr_allocs: int):
        """Print allocation comparison."""
        alloc_diff = curr_allocs - base_allocs
        alloc_color = GREEN if alloc_diff < 0 else RED
        print(f"  {alloc_color}allocs: {base_allocs} → {curr_allocs}{NC}", end="")

    def _print_removed_test(self, base_result: Tuple):
        """Print message for removed test."""
        print(f"{RED}[REMOVED]{NC} - was {base_result[0]:.2f} ns/op", end="")

    def _print_new_test(self, current_result: Tuple):
        """Print message for new test."""
        curr_ns, curr_mb, curr_bytes, curr_allocs = current_result
        print(f"{YELLOW}[NEW]{NC} - {curr_ns:.2f} ns/op", end="")
        if curr_mb:
            print(f"  {curr_mb:.1f} MB/s", end="")
        if curr_allocs:
            print(f"  {curr_allocs} allocs/op", end="")

    def list_commits(self, limit: int = 25):
        """List commits with saved benchmarks."""
        cursor = self.conn.cursor()
        rows = self._get_commits_list(cursor, limit)
        
        if not rows:
            self._print_no_results_message()
            return
        
        if HAS_TABULATE:
            self._print_commits_with_tabulate(rows, limit)
        else:
            self._print_commits_header(rows, limit)
            self._print_commits_table(rows)
        
        self._print_usage_hint(rows)
    
    def _print_commits_with_tabulate(self, rows: List[Tuple], limit: int):
        """Print commits list using tabulate."""
        print(f"\n{BLUE}{'='*80}{NC}")
        print(f"{BLUE}Saved Benchmark Results (Last {min(len(rows), limit)} commits){NC}")
        print(f"{BLUE}{'='*80}{NC}\n")
        
        table_data = []
        for i, row in enumerate(rows):
            commit, branch, timestamp, count, packages = row
            dt = datetime.fromisoformat(timestamp)
            
            # Format commit
            if commit == "current":
                commit_display = f"{YELLOW}current{NC}"
            else:
                color = GREEN if i == 0 else CYAN if i < 5 else ""
                commit_display = f"{color}{commit[:8]}{NC}"
            
            # Format branch
            branch_display = branch[:18] + "..." if len(branch) > 18 else branch
            
            # Format packages
            pkg_list = packages.split(',') if packages else []
            pkg_display = ', '.join([p.split('/')[-1] for p in pkg_list[:3]])
            if len(pkg_list) > 3:
                pkg_display += f" (+{len(pkg_list)-3} more)"
            
            table_data.append([
                commit_display,
                branch_display,
                dt.strftime('%Y-%m-%d'),
                dt.strftime('%H:%M'),
                count,
                pkg_display
            ])
        
        headers = ["Commit", "Branch", "Date", "Time", "Tests", "Packages"]
        print(tabulate(table_data, headers=headers, tablefmt="simple"))

    def _get_commits_list(self, cursor, limit: int) -> List[Tuple]:
        """Get list of commits with benchmarks."""
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
        return cursor.fetchall()

    def _print_no_results_message(self):
        """Print message when no results found."""
        print(f"{RED}❌ No benchmark results found{NC}")
        print(f"{YELLOW}Run 'make bench/save' to save benchmark results{NC}")

    def _print_commits_header(self, rows: List[Tuple], limit: int):
        """Print commits list header."""
        print(f"\n{BLUE}{'='*80}{NC}")
        print(f"{BLUE}Saved Benchmark Results (Last {min(len(rows), limit)} commits){NC}")
        print(f"{BLUE}{'='*80}{NC}\n")
        
        print(f"{'Commit':10} {'Branch':20} {'Date':16} {'Time':8} {'Tests':>6}  Packages")
        print("-" * 80)

    def _print_commits_table(self, rows: List[Tuple]):
        """Print table of commits."""
        for i, row in enumerate(rows):
            self._print_single_commit_row(i, row)

    def _print_single_commit_row(self, index: int, row: Tuple):
        """Print a single row in the commits table."""
        commit, branch, timestamp, count, packages = row
        dt = datetime.fromisoformat(timestamp)
        
        commit_display = self._format_commit_display(commit, index)
        branch_display = self._format_branch_display(branch)
        pkg_display = self._format_packages_display(packages)
        
        print(f"{commit_display}  {branch_display:20} {dt.strftime('%Y-%m-%d'):16} "
              f"{dt.strftime('%H:%M'):8} {count:>6}  {pkg_display}")

    def _format_commit_display(self, commit: str, index: int) -> str:
        """Format commit hash for display."""
        if commit == "current":
            return f"{YELLOW}current{NC}  "
        
        color = self._get_commit_color(index)
        return f"{color}{commit[:8]}{NC}"

    def _get_commit_color(self, index: int) -> str:
        """Get color for commit based on index."""
        if index == 0:
            return GREEN  # Most recent
        elif index < 5:
            return CYAN   # Recent
        else:
            return ""     # Older

    def _format_branch_display(self, branch: str) -> str:
        """Format branch name for display."""
        if len(branch) > 18:
            return branch[:15] + "..."
        return branch

    def _format_packages_display(self, packages: str) -> str:
        """Format packages list for display."""
        pkg_list = packages.split(',') if packages else []
        pkg_display = ', '.join([p.split('/')[-1] for p in pkg_list[:3]])
        if len(pkg_list) > 3:
            pkg_display += f" (+{len(pkg_list)-3} more)"
        return pkg_display

    def _print_usage_hint(self, rows: List[Tuple]):
        """Print usage hint at the end of list."""
        print(f"\n{CYAN}ℹ Usage: make bench/compare BASE_COMMIT CURRENT_COMMIT{NC}")
        if len(rows) >= 2:
            print(f"{CYAN}Example: make bench/compare {rows[-1][0][:8]} {rows[0][0][:8]}{NC}\n")

    def close(self):
        """Close database connection."""
        if self.conn:
            self.conn.close()


def create_parser() -> argparse.ArgumentParser:
    """Create command line argument parser."""
    parser = argparse.ArgumentParser(description='Benchmark Manager for Bazel')
    subparsers = parser.add_subparsers(dest='command', help='Commands')
    
    # Save command
    save_parser = subparsers.add_parser('save', help='Run benchmarks and save results')
    save_parser.add_argument('--targets', nargs='*', help='Specific targets to benchmark')
    save_parser.add_argument('--preserve-history', action='store_true', 
                           help='Preserve existing benchmark history (for CI/CD)')
    
    # Compare command  
    compare_parser = subparsers.add_parser('compare', help='Compare benchmark results')
    compare_parser.add_argument('base', nargs='?', help='Base commit hash (default: main branch)')
    compare_parser.add_argument('current', nargs='?', help='Current commit hash (default: current HEAD)')
    
    # List command
    list_parser = subparsers.add_parser('list', help='List saved benchmark results')
    list_parser.add_argument('--limit', type=int, default=25, help='Number of commits to display')
    
    # Database path option
    parser.add_argument('--db', default='benchmarks.sqlite', help='Path to SQLite database')
    
    return parser


def handle_save_command(manager: BenchmarkManager, args):
    """Handle save command."""
    commit_hash, branch, is_dirty = manager.get_current_commit()
    
    if is_dirty:
        print(f"{YELLOW}▶ Running benchmarks for uncommitted changes (current) on {branch}{NC}")
    else:
        print(f"{YELLOW}▶ Running benchmarks for commit: {commit_hash[:8]} ({branch}){NC}")
    
    if args.preserve_history:
        print(f"{CYAN}ℹ Preserving benchmark history (CI/CD mode){NC}")
    
    results = manager.run_benchmarks(args.targets)
    
    if results:
        manager.save_results(results, commit_hash, branch, is_dirty, 
                           preserve_history=args.preserve_history)
    else:
        print(f"{RED}No benchmark results collected{NC}")


def handle_compare_command(manager: BenchmarkManager, args):
    """Handle compare command."""
    base_hash, current_hash = determine_compare_commits(manager, args)
    manager.compare_commits(base_hash, current_hash)


def determine_compare_commits(manager: BenchmarkManager, args) -> Tuple[str, str]:
    """Determine which commits to compare based on arguments."""
    if args.base is None:
        # No arguments: compare current with main
        return get_current_vs_main(manager)
    elif args.current is None:
        # One argument: compare current with specified commit
        return get_current_vs_specified(manager, args.base)
    else:
        # Two arguments: compare two specified commits
        return args.base, args.current


def get_current_vs_main(manager: BenchmarkManager) -> Tuple[str, str]:
    """Get current commit and main branch commit for comparison."""
    commit_hash, _, is_dirty = manager.get_current_commit()
    current_hash = "current" if is_dirty else commit_hash
    base_hash = manager.get_main_commit()
    
    if base_hash is None:
        print(f"{RED}Error: No benchmarks found for main branch{NC}")
        sys.exit(1)
    
    return base_hash, current_hash


def get_current_vs_specified(manager: BenchmarkManager, base: str) -> Tuple[str, str]:
    """Get current commit and specified commit for comparison."""
    commit_hash, _, is_dirty = manager.get_current_commit()
    current_hash = "current" if is_dirty else commit_hash
    return base, current_hash


def main():
    parser = create_parser()
    args = parser.parse_args()
    
    if not args.command:
        parser.print_help()
        sys.exit(1)
    
    manager = BenchmarkManager(args.db)
    
    try:
        if args.command == 'save':
            handle_save_command(manager, args)
        elif args.command == 'compare':
            handle_compare_command(manager, args)
        elif args.command == 'list':
            manager.list_commits(args.limit)
    finally:
        manager.close()


if __name__ == '__main__':
    main()