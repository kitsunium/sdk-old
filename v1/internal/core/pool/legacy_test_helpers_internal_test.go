package pool

import (
	"runtime"
	"strings"
	"sync/atomic"
)

// testingSkipSafetyCheck is a no-op toggle kept for backward compatibility
// with tests written when an unsafe-buffer goroutine checker existed. The
// checker has been removed; the variable is preserved so legacy tests
// compile. It has no runtime effect.
var testingSkipSafetyCheck bool

// getCurrentGID returns a stable, process-local identifier derived from the
// Go runtime goroutine stack. Used only by legacy tests; not a real GID.
func getCurrentGID() uint32 {
	var buf [64]byte
	n := runtime.Stack(buf[:], false)
	s := string(buf[:n])
	// "goroutine N [..." — the number after the prefix is the runtime GID.
	s = strings.TrimPrefix(s, "goroutine ")
	if sp := strings.IndexByte(s, ' '); sp > 0 {
		s = s[:sp]
	}
	var id uint32
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c < '0' || c > '9' {
			break
		}
		id = id*10 + uint32(c-'0')
	}
	if id == 0 {
		id = atomic.AddUint32(&legacyGIDFallback, 1)
	}
	return id
}

var legacyGIDFallback uint32
