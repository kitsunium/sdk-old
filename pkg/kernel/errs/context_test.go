package errs

import (
	"context"
	"testing"
)

type testContextKey string

func TestFromContext(t *testing.T) {
	ClearRegistry()
	tests := []struct {
		name     string
		setup    func() context.Context
		wantInst bool
		wantNil  bool
	}{
		{
			name: "nil context",
			setup: func() context.Context {
				return nil
			},
			wantInst: false,
			wantNil:  true,
		},
		{
			name: "empty context",
			setup: func() context.Context {
				return context.Background()
			},
			wantInst: false,
			wantNil:  true,
		},
		{
			name: "context with instance",
			setup: func() context.Context {
				err := Define(KConfig{Code: 404}).New()
				return ToContext(context.Background(), err)
			},
			wantInst: true,
			wantNil:  false,
		},
		{
			name: "context with wrong type",
			setup: func() context.Context {
				return context.WithValue(context.Background(), errorContextKey, "not an instance")
			},
			wantInst: false,
			wantNil:  true,
		},
		{
			name: "context with nil instance",
			setup: func() context.Context {
				return context.WithValue(context.Background(), errorContextKey, (*Instance)(nil))
			},
			wantInst: false,
			wantNil:  true,
		},
		{
			name: "nested context",
			setup: func() context.Context {
				err := Define(KConfig{Code: 500}).New()
				ctx := ToContext(context.Background(), err)
				return context.WithValue(ctx, testContextKey("other"), "value")
			},
			wantInst: true,
			wantNil:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := tt.setup()
			inst, ok := FromContext(ctx)

			if ok != tt.wantInst {
				t.Errorf("FromContext() ok = %v, want %v", ok, tt.wantInst)
			}

			if tt.wantNil {
				if inst != nil {
					t.Errorf("FromContext() inst = %v, want nil", inst)
				}
			} else if tt.wantInst && inst == nil {
				t.Errorf("FromContext() inst = nil, want non-nil")
			}

			// Clean up
			if inst != nil {
				inst.Release()
			}
		})
	}
}

func TestToContext(t *testing.T) {
	ClearRegistry()
	tests := []struct {
		name string
		ctx  context.Context
		inst *Instance
	}{
		{
			name: "with background context",
			ctx:  context.Background(),
			inst: Define(KConfig{Code: 404}).New(),
		},
		{
			name: "with TODO context",
			ctx:  context.TODO(),
			inst: Define(KConfig{Code: 500}).New(),
		},
		{
			name: "with valued context",
			ctx:  context.WithValue(context.Background(), testContextKey("key"), "value"),
			inst: Define(KConfig{Code: 401}).New(),
		},
		{
			name: "with nil instance",
			ctx:  context.Background(),
			inst: nil,
		},
		{
			name: "with cancelled context",
			ctx: func() context.Context {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				return ctx
			}(),
			inst: Define(KConfig{Code: 503}).New(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if tt.inst != nil {
					tt.inst.Release()
				}
			}()

			newCtx := ToContext(tt.ctx, tt.inst)

			// Verify instance can be retrieved
			retrieved, ok := FromContext(newCtx)
			if tt.inst != nil {
				if !ok {
					t.Errorf("ToContext() failed to store instance")
				}
				if retrieved != tt.inst {
					t.Errorf("ToContext() stored different instance")
				}
			} else {
				if ok && retrieved != nil {
					t.Errorf("ToContext() stored non-nil for nil instance")
				}
			}

			// Verify original context values are preserved
			if tt.ctx != nil {
				if val := tt.ctx.Value("key"); val != nil {
					if newCtx.Value("key") != val {
						t.Errorf("ToContext() lost original context value")
					}
				}
			}
		})
	}
}

func TestExtractTraceID(t *testing.T) {
	tests := []struct {
		name     string
		ctx      context.Context
		expected string
	}{
		{
			name:     "nil context",
			ctx:      nil,
			expected: "",
		},
		{
			name:     "empty context",
			ctx:      context.Background(),
			expected: "",
		},
		{
			name:     "context with values",
			ctx:      context.WithValue(context.Background(), testContextKey("key"), "value"),
			expected: "",
		},
		{
			name:     "TODO context",
			ctx:      context.TODO(),
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractTraceID(tt.ctx)
			if got != tt.expected {
				t.Errorf("ExtractTraceID() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestExtractSpanID(t *testing.T) {
	tests := []struct {
		name     string
		ctx      context.Context
		expected string
	}{
		{
			name:     "nil context",
			ctx:      nil,
			expected: "",
		},
		{
			name:     "empty context",
			ctx:      context.Background(),
			expected: "",
		},
		{
			name:     "context with values",
			ctx:      context.WithValue(context.Background(), testContextKey("span"), "123"),
			expected: "",
		},
		{
			name: "cancelled context",
			ctx: func() context.Context {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				return ctx
			}(),
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractSpanID(tt.ctx)
			if got != tt.expected {
				t.Errorf("ExtractSpanID() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestContextIntegration(t *testing.T) {
	ClearRegistry()
	// Test full integration flow
	err1 := Define(KConfig{Code: 404, Message: "not found"}).New()
	defer err1.Release()

	// Add to context
	ctx := ToContext(context.Background(), err1)

	// Retrieve from context
	retrieved, ok := FromContext(ctx)
	if !ok {
		t.Fatal("Failed to retrieve error from context")
	}

	if retrieved.Code() != 404 {
		t.Errorf("Retrieved error code = %d, want 404", retrieved.Code())
	}

	// Test with nested contexts
	ctx2 := context.WithValue(ctx, testContextKey("user"), "john")
	retrieved2, ok := FromContext(ctx2)
	if !ok {
		t.Fatal("Failed to retrieve error from nested context")
	}

	if retrieved2 != retrieved {
		t.Error("Retrieved different instance from nested context")
	}

	// Test overwriting
	err2 := Define(KConfig{Code: 500, Message: "internal error"}).New()
	defer err2.Release()

	ctx3 := ToContext(ctx, err2)
	retrieved3, ok := FromContext(ctx3)
	if !ok {
		t.Fatal("Failed to retrieve overwritten error")
	}

	if retrieved3.Code() != 500 {
		t.Errorf("Overwritten error code = %d, want 500", retrieved3.Code())
	}
}

func TestContextConcurrency(t *testing.T) {
	ClearRegistry()
	// Test concurrent access to context
	err := Define(KConfig{Code: 503}).New()
	defer err.Release()

	ctx := ToContext(context.Background(), err)

	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func() {
			inst, ok := FromContext(ctx)
			if !ok || inst == nil {
				t.Error("Failed to retrieve instance concurrently")
			}
			done <- true
		}()
	}

	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestContextKeyUniqueness(t *testing.T) {
	ClearRegistry()
	// Ensure our context key doesn't conflict
	type otherKey int
	const testKey otherKey = 0

	err := Define(KConfig{Code: 400}).New()
	defer err.Release()

	ctx := context.WithValue(context.Background(), testKey, "test value")
	ctx = ToContext(ctx, err)

	// Both values should be retrievable
	if val := ctx.Value(testKey); val != "test value" {
		t.Error("Context key conflict: lost original value")
	}

	if inst, ok := FromContext(ctx); !ok || inst == nil {
		t.Error("Context key conflict: lost error instance")
	}
}
