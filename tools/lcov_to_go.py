#!/usr/bin/env python3
"""Convert LCOV coverage format to Go coverage format."""

import sys
import re
from pathlib import Path

def lcov_to_go_coverage(lcov_file, output_file):
    """Convert LCOV format to Go coverage format."""
    
    coverage_data = []
    current_file = None
    
    with open(lcov_file, 'r') as f:
        for line in f:
            line = line.strip()
            
            # Source file
            if line.startswith('SF:'):
                current_file = line[3:]
                # Convert relative path to module path
                if current_file.startswith('pkg/'):
                    current_file = 'github.com/kitsunium/sdk/' + current_file
            
            # Line coverage data
            elif line.startswith('DA:'):
                if current_file:
                    parts = line[3:].split(',')
                    if len(parts) == 2:
                        line_num = parts[0]
                        count = parts[1]
                        
                        # Skip lines with 0 coverage for now, we need block info
                        if count != '0':
                            # For simplicity, treat each line as a single statement block
                            # Format: file:startline.startcol,endline.endcol statements count
                            coverage_data.append(f"{current_file}:{line_num}.1,{line_num}.100 1 {count}")
    
    # Write Go coverage format
    with open(output_file, 'w') as f:
        f.write("mode: set\n")
        for entry in coverage_data:
            f.write(entry + "\n")
    
    print(f"Converted {len(coverage_data)} coverage entries from LCOV to Go format")

if __name__ == "__main__":
    if len(sys.argv) != 3:
        print("Usage: lcov_to_go.py <input_lcov_file> <output_go_coverage_file>")
        sys.exit(1)
    
    lcov_to_go_coverage(sys.argv[1], sys.argv[2])