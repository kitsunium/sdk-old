// Package contract holds architecture-level enforcement and conformance
// tests for the Kitsunium SDK layering rules.
//
// The canonical dependency order is:
//
//	adapters/    ─┐
//	              ├─▶ ports/
//	components/  ─┘   (interfaces, no impl)
//	   │
//	   └─▶ internal/core/ ─▶ internal/kernel/ ─▶ stdlib
//
// Violations are rejected by arch_test.go on every CI run via a
// go/ast import walk. See CLAUDE.md hierarchy for the full contract.
package contract
