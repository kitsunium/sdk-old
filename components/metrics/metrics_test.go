// Package metrics - unit tests.
package metrics

import (
	"context"
	"math"
	"sync"
	"testing"
)

func TestCounter(t *testing.T) {
	t.Run("starts at zero", func(t *testing.T) {
		c := Counter("hits")
		if c.Value() != 0 {
			t.Fatalf("want 0, got %v", c.Value())
		}
	})

	t.Run("Inc and Add accumulate", func(t *testing.T) {
		c := Counter("hits")
		c.Inc()
		c.Inc()
		c.Add(2.5)
		if c.Value() != 4.5 {
			t.Fatalf("want 4.5, got %v", c.Value())
		}
	})

	t.Run("negative Add is ignored", func(t *testing.T) {
		c := Counter("hits")
		c.Inc()
		c.Add(-5)
		if c.Value() != 1 {
			t.Fatalf("want 1, got %v", c.Value())
		}
	})

	t.Run("NaN Add is ignored", func(t *testing.T) {
		c := Counter("hits")
		c.Add(math.NaN())
		if c.Value() != 0 {
			t.Fatalf("want 0, got %v", c.Value())
		}
	})

	t.Run("Describe carries metadata", func(t *testing.T) {
		c := Counter("hits",
			WithHelp("request count"),
			WithLabel("svc", "api"),
		)
		d := c.Describe()
		if d.Name != "hits" || d.Help != "request count" || d.Labels["svc"] != "api" {
			t.Fatalf("bad descriptor: %+v", d)
		}
		// Mutation of returned labels must not affect the counter.
		d.Labels["svc"] = "mutated"
		if c.Describe().Labels["svc"] != "api" {
			t.Fatalf("descriptor label aliasing detected")
		}
	})

	t.Run("concurrent increments are safe", func(t *testing.T) {
		c := Counter("hits")
		const workers, per = 16, 1000
		var wg sync.WaitGroup
		wg.Add(workers)
		for i := 0; i < workers; i++ {
			go func() {
				defer wg.Done()
				for j := 0; j < per; j++ {
					c.Inc()
				}
			}()
		}
		wg.Wait()
		if c.Value() != float64(workers*per) {
			t.Fatalf("want %d, got %v", workers*per, c.Value())
		}
	})
}

func TestGauge(t *testing.T) {
	t.Run("Set replaces value", func(t *testing.T) {
		g := Gauge("temp")
		g.Set(42.5)
		if g.Value() != 42.5 {
			t.Fatalf("want 42.5, got %v", g.Value())
		}
		g.Set(-1)
		if g.Value() != -1 {
			t.Fatalf("want -1, got %v", g.Value())
		}
	})

	t.Run("Add accepts negative deltas", func(t *testing.T) {
		g := Gauge("temp")
		g.Set(10)
		g.Add(-3)
		if g.Value() != 7 {
			t.Fatalf("want 7, got %v", g.Value())
		}
	})

	t.Run("concurrent Add is safe", func(t *testing.T) {
		g := Gauge("temp")
		const workers, per = 16, 500
		var wg sync.WaitGroup
		wg.Add(workers)
		for i := 0; i < workers; i++ {
			go func() {
				defer wg.Done()
				for j := 0; j < per; j++ {
					g.Add(1)
				}
			}()
		}
		wg.Wait()
		if g.Value() != float64(workers*per) {
			t.Fatalf("want %d, got %v", workers*per, g.Value())
		}
	})
}

func TestHealth(t *testing.T) {
	t.Run("OK result", func(t *testing.T) {
		h := Health("db", func(ctx context.Context) Status { return OK("reachable") })
		s := h.Check(context.Background())
		if s.Code != StatusOK || s.Reason != "reachable" {
			t.Fatalf("want OK/reachable, got %+v", s)
		}
	})

	t.Run("Degraded result", func(t *testing.T) {
		h := Health("cache", func(ctx context.Context) Status { return Degraded("slow") })
		s := h.Check(context.Background())
		if s.Code != StatusDegraded || s.Reason != "slow" {
			t.Fatalf("want Degraded/slow, got %+v", s)
		}
	})

	t.Run("Down result", func(t *testing.T) {
		h := Health("queue", func(ctx context.Context) Status { return Down("connection refused") })
		s := h.Check(context.Background())
		if s.Code != StatusDown || s.Reason != "connection refused" {
			t.Fatalf("want Down/connection refused, got %+v", s)
		}
	})

	t.Run("nil check defaults to OK", func(t *testing.T) {
		h := Health("noop", nil)
		if s := h.Check(context.Background()); s.Code != StatusOK {
			t.Fatalf("want OK, got %+v", s)
		}
	})

	t.Run("nil ctx is tolerated", func(t *testing.T) {
		called := false
		h := Health("x", func(ctx context.Context) Status {
			if ctx == nil {
				t.Fatal("ctx was nil inside check")
			}
			called = true
			return OK("")
		})
		_ = h.Check(nil) //nolint:staticcheck // intentional nil
		if !called {
			t.Fatal("check was not invoked")
		}
	})

	t.Run("Describe carries metadata", func(t *testing.T) {
		h := Health("db", nil, WithHelp("database liveness"), WithLabels(map[string]string{"region": "eu"}))
		d := h.Describe()
		if d.Name != "db" || d.Help != "database liveness" || d.Labels["region"] != "eu" {
			t.Fatalf("bad descriptor: %+v", d)
		}
	})
}

func TestStatusCode_String(t *testing.T) {
	cases := map[StatusCode]string{
		StatusOK:       "ok",
		StatusDegraded: "degraded",
		StatusDown:     "down",
		StatusCode(99): "unknown",
	}
	for code, want := range cases {
		if got := code.String(); got != want {
			t.Fatalf("StatusCode(%d).String() = %q, want %q", code, got, want)
		}
	}
}
