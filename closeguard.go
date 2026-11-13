package godi

import "sync/atomic"

// CloseGuard is the shared close-once state of a provider or scope.
//
// Both providers use it: the runtime root and scope, and every generated Root
// and scope, which hold it as a `guard godi.CloseGuard` field. Close and
// CloseAsync share one guard (first-caller-wins), the resolution entry points
// check Closed. It lives in the root package so generated code reaches it
// through the godi import it already has, without importing sync or
// sync/atomic (mirrors ProviderClosedError). Its zero value is an open guard;
// it must not be copied after first use.
type CloseGuard struct {
	closed atomic.Bool
	err    atomic.Pointer[error]
}

// Closed reports whether Close has started.
func (g *CloseGuard) Closed() bool {
	return g.closed.Load()
}

// Close runs release exactly once. The first caller marks the guard closed, runs
// release and caches its error. Any other caller — including a re-entrant call
// made from inside release — returns the cached error, or nil if release has not
// finished yet. It never blocks.
func (g *CloseGuard) Close(release func() error) error {
	if !g.closed.CompareAndSwap(false, true) {
		if p := g.err.Load(); p != nil {
			return *p
		}
		return nil
	}
	err := release()
	g.err.Store(&err)
	return err
}
