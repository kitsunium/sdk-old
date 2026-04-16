// Package errs provides a typed error catalog for the Kitsunium SDK.
//
// See error.go for the full package documentation.
package errs

import "context"

type ctxKey struct{}

// ToContext returns a new context carrying inst.
//
// Params:
//   - ctx: the parent context to enrich
//   - inst: the Instance to store; if nil, ctx is returned unchanged
//
// Returns:
//   - ctx: the enriched context (or original when inst is nil)
func ToContext(ctx context.Context, inst *Instance) (result context.Context) {
	//: return ctx unchanged when there is nothing to propagate
	if inst == nil {
		//: nothing to attach — pass through
		return ctx
	}
	//: embed the instance in a typed key to avoid collisions
	return context.WithValue(ctx, ctxKey{}, inst)
}

// FromContext returns the Instance attached to ctx, if any.
//
// Params:
//   - ctx: the context to inspect; nil returns (nil, false)
//
// Returns:
//   - inst: the attached Instance, or nil
//   - ok: true when an Instance was found
func FromContext(ctx context.Context) (inst *Instance, ok bool) {
	//: nil context cannot carry values
	if ctx == nil {
		//: fast-path: nothing to look up
		return nil, false
	}
	//: extract the typed value using the unexported key
	inst, ok = ctx.Value(ctxKey{}).(*Instance)
	return inst, ok
}
