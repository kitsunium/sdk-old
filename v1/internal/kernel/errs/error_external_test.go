package errs_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/kitsunium/sdk/v1/internal/kernel/errs"
)

func TestDefineAndFields(t *testing.T) {
	tests := []struct {
		name     string
		cfg      errs.Config
		wantPkg  string
		wantCode int
		wantMsg  string
	}{
		{
			name:     "all fields set",
			cfg:      errs.Config{Package: "pkg_a", Code: 1, Message: "boom"},
			wantPkg:  "pkg_a",
			wantCode: 1,
			wantMsg:  "boom",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Cleanup(errs.ResetForTest)
			e := errs.Define(tt.cfg)
			if got := e.Pkg(); got != tt.wantPkg {
				t.Errorf("Pkg() = %q, want %q", got, tt.wantPkg)
			}
			if got := e.Code(); got != tt.wantCode {
				t.Errorf("Code() = %d, want %d", got, tt.wantCode)
			}
			if got := e.Message(); got != tt.wantMsg {
				t.Errorf("Message() = %q, want %q", got, tt.wantMsg)
			}
			if got := e.Error(); got != tt.wantMsg {
				t.Errorf("Error() = %q, want %q", got, tt.wantMsg)
			}
			if e.ID() == 0 {
				t.Error("ID() should be non-zero")
			}
			if got := e.String(); got == "" {
				t.Error("String() should be non-empty")
			}
		})
	}
}

func TestDefineAutoDetectsPackage(t *testing.T) {
	tests := []struct {
		name string
		cfg  errs.Config
	}{
		{name: "no package provided", cfg: errs.Config{Code: 1, Message: "m"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Cleanup(errs.ResetForTest)
			e := errs.Define(tt.cfg)
			if e.Pkg() == "" || e.Pkg() == "unknown" {
				t.Errorf("auto-detect failed, got %q", e.Pkg())
			}
		})
	}
}

func TestDefineDefaultMessage(t *testing.T) {
	tests := []struct {
		name    string
		cfg     errs.Config
		wantMsg string
	}{
		{name: "empty message", cfg: errs.Config{Package: "x", Code: 42}, wantMsg: "error 42"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Cleanup(errs.ResetForTest)
			e := errs.Define(tt.cfg)
			if got := e.Message(); got != tt.wantMsg {
				t.Errorf("Message() = %q, want %q", got, tt.wantMsg)
			}
		})
	}
}

func TestDefineDuplicatePanics(t *testing.T) {
	tests := []struct {
		name string
		cfg1 errs.Config
		cfg2 errs.Config
	}{
		{
			name: "same package and code",
			cfg1: errs.Config{Package: "x", Code: 1, Message: "a"},
			cfg2: errs.Config{Package: "x", Code: 1, Message: "b"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Cleanup(errs.ResetForTest)
			errs.Define(tt.cfg1)
			defer func() {
				if r := recover(); r == nil {
					t.Error("expected panic on duplicate")
				}
			}()
			errs.Define(tt.cfg2)
		})
	}
}

func TestErrIs(t *testing.T) {
	t.Cleanup(errs.ResetForTest)
	e := errs.Define(errs.Config{Package: "x", Code: 2, Message: "m"})
	other := errs.Define(errs.Config{Package: "x", Code: 3, Message: "m"})

	tests := []struct {
		name   string
		target error
		want   bool
	}{
		{name: "self match", target: e, want: true},
		{name: "different entry", target: other, want: false},
		{name: "pointer form", target: &e, want: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := errors.Is(e, tt.target); got != tt.want {
				t.Errorf("errors.Is(e, target) = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestErrIsNilPointer(t *testing.T) {
	t.Cleanup(errs.ResetForTest)
	e := errs.Define(errs.Config{Package: "x", Code: 1, Message: "m"})
	var nilKE *errs.Err
	if e.Is(nilKE) {
		t.Error("Is(nil *Err) should be false")
	}
}

func TestInstanceNewAndNewf(t *testing.T) {
	t.Cleanup(errs.ResetForTest)
	e := errs.Define(errs.Config{Package: "x", Code: 1, Message: "base"})

	tests := []struct {
		name      string
		build     func() *errs.Instance
		wantError string
	}{
		{
			name:      "New uses default message",
			build:     func() *errs.Instance { return e.New() },
			wantError: "[x:1] base",
		},
		{
			name:      "Newf formats message",
			build:     func() *errs.Instance { return e.Newf("code=%d", 7) },
			wantError: "[x:1] code=7",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inst := tt.build()
			if inst.Error() != tt.wantError {
				t.Errorf("Error() = %q, want %q", inst.Error(), tt.wantError)
			}
			if inst.Err().ID() != e.ID() {
				t.Error("Err() ID mismatch")
			}
			if inst.Package() != "x" || inst.Code() != 1 {
				t.Error("Package/Code mismatch")
			}
		})
	}
}

func TestInstanceWrap(t *testing.T) {
	t.Cleanup(errs.ResetForTest)
	e := errs.Define(errs.Config{Package: "x", Code: 1, Message: "base"})
	cause := fmt.Errorf("underlying")

	tests := []struct {
		name      string
		build     func() *errs.Instance
		wantNil   bool
		wantError string
		wantCause error
		wantIs    error
	}{
		{
			name:    "nil cause returns nil",
			build:   func() *errs.Instance { return e.Wrap(nil) },
			wantNil: true,
		},
		{
			name:      "wraps cause",
			build:     func() *errs.Instance { return e.Wrap(cause) },
			wantError: "[x:1] base: underlying",
			wantCause: cause,
			wantIs:    e,
		},
		{
			name:      "Wrapf formats message",
			build:     func() *errs.Instance { return e.Wrapf(cause, "ctx=%s", "abc") },
			wantError: "[x:1] ctx=abc: underlying",
			wantIs:    e,
		},
		{
			name:    "Wrapf nil cause returns nil",
			build:   func() *errs.Instance { return e.Wrapf(nil, "x") },
			wantNil: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inst := tt.build()
			if tt.wantNil {
				if inst != nil {
					t.Error("expected nil")
				}
				return
			}
			if inst.Error() != tt.wantError {
				t.Errorf("Error() = %q, want %q", inst.Error(), tt.wantError)
			}
			if tt.wantCause != nil && inst.Unwrap() != tt.wantCause {
				t.Error("Unwrap mismatch")
			}
			if tt.wantIs != nil && !errors.Is(inst, tt.wantIs) {
				t.Error("errors.Is should match parent Err")
			}
		})
	}
}

func TestInstanceTagsAndDetails(t *testing.T) {
	t.Cleanup(errs.ResetForTest)
	e := errs.Define(errs.Config{Package: "x", Code: 1, Message: "m"})
	inst := e.New().WithTag("env", "prod").WithDetail("size", 42).WithDetail("name", "alice")

	tests := []struct {
		name string
		run  func(t *testing.T)
	}{
		{
			name: "existing tag",
			run: func(t *testing.T) {
				v, ok := inst.Tag("env")
				if !ok || v != "prod" {
					t.Errorf("Tag env = %q,%v", v, ok)
				}
			},
		},
		{
			name: "missing tag",
			run: func(t *testing.T) {
				if _, ok := inst.Tag("missing"); ok {
					t.Error("Tag missing should be absent")
				}
			},
		},
		{
			name: "existing detail",
			run: func(t *testing.T) {
				v, ok := inst.Detail("size")
				if !ok || v != 42 {
					t.Errorf("Detail size = %v,%v", v, ok)
				}
			},
		},
		{
			name: "DetailAs correct type",
			run: func(t *testing.T) {
				size, ok := errs.DetailAs[int](inst, "size")
				if !ok || size != 42 {
					t.Errorf("DetailAs[int] size = %d,%v", size, ok)
				}
			},
		},
		{
			name: "DetailAs wrong type",
			run: func(t *testing.T) {
				if _, ok := errs.DetailAs[string](inst, "size"); ok {
					t.Error("DetailAs[string] on int should fail")
				}
			},
		},
		{
			name: "DetailAs missing key",
			run: func(t *testing.T) {
				if _, ok := errs.DetailAs[int](inst, "missing"); ok {
					t.Error("DetailAs missing should fail")
				}
			},
		},
		{
			name: "DetailAs nil instance",
			run: func(t *testing.T) {
				if _, ok := errs.DetailAs[int](nil, "k"); ok {
					t.Error("DetailAs nil should fail")
				}
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, tt.run)
	}
}

func TestInstanceAsAndUnwrap(t *testing.T) {
	t.Cleanup(errs.ResetForTest)
	e := errs.Define(errs.Config{Package: "x", Code: 1, Message: "m"})
	custom := &customErr{v: 7}
	inst := e.Wrap(custom)

	tests := []struct {
		name string
		run  func(t *testing.T)
	}{
		{
			name: "Unwrap returns cause",
			run: func(t *testing.T) {
				if inst.Unwrap() != custom {
					t.Error("Unwrap mismatch")
				}
			},
		},
		{
			name: "As succeeds",
			run: func(t *testing.T) {
				var target *customErr
				if !inst.As(&target) || target.v != 7 {
					t.Error("As should succeed")
				}
			},
		},
		{
			name: "As on no-cause Instance fails",
			run: func(t *testing.T) {
				var target *customErr
				if e.New().As(&target) {
					t.Error("As on no-cause Instance should be false")
				}
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, tt.run)
	}
}

type customErr struct{ v int }

func (c *customErr) Error() string { return "custom" }

func TestContextRoundTrip(t *testing.T) {
	t.Cleanup(errs.ResetForTest)
	e := errs.Define(errs.Config{Package: "x", Code: 1, Message: "m"})
	inst := e.New()

	tests := []struct {
		name string
		run  func(t *testing.T)
	}{
		{
			name: "round-trip",
			run: func(t *testing.T) {
				ctx := errs.ToContext(t.Context(), inst)
				got, ok := errs.FromContext(ctx)
				if !ok || got != inst {
					t.Error("FromContext failed round-trip")
				}
			},
		},
		{
			name: "empty context returns false",
			run: func(t *testing.T) {
				if _, ok := errs.FromContext(t.Context()); ok {
					t.Error("FromContext empty should be false")
				}
			},
		},
		{
			name: "ToContext with nil inst returns ctx",
			run: func(t *testing.T) {
				ctx := t.Context()
				if errs.ToContext(ctx, nil) != ctx {
					t.Error("ToContext(ctx, nil) should return ctx unchanged")
				}
			},
		},
		{
			name: "FromContext nil ctx returns false",
			run: func(t *testing.T) {
				//nolint:staticcheck
				var nilCtx interface{ Value(any) any }
				_ = nilCtx
				// Test via the exported API
				_, ok := errs.FromContext(nil)
				if ok {
					t.Error("FromContext(nil) should be false")
				}
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, tt.run)
	}
}
