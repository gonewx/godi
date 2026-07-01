package godi

import "context"

// AsyncDisposable is the structural, opt-in async-release contract — the
// asymmetric-name counterpart of io.Closer (D-1). A Go type may not carry two
// methods named Close with different signatures, so a service that must
// support BOTH synchronous and asynchronous release (dotnet's
// SyncAsyncDisposable, the prefer-async corpus) could not name its async method
// Close(ctx). The async method therefore takes an independent name, CloseAsync.
//
// Its context parameter is the Go reading of "async": Go has no
// ValueTask/async-await, so the value dotnet's DisposeAsync provides ("await
// without blocking a thread") maps to a cancellable / deadline-aware graceful
// release — that is what ctx carries here. This is a deliberate Go idiom, not a
// defect (see conformance/ASYNC_DIVERGENCE.md).
//
// Like io.Closer, AsyncDisposable is discovered by type assertion and is NOT
// part of the Provider/Scope interface — AD-16's seven-method face is unchanged
// (D-2). A service opts in structurally: implement io.Closer and it is released
// via Close; implement AsyncDisposable and it is released via CloseAsync;
// implement both and async is preferred (prefer-async, D-4).
type AsyncDisposable interface {
	CloseAsync(ctx context.Context) error
}

// AsyncScope is the value-semantics wrapper mapping dotnet's AsyncServiceScope
// readonly struct. It wraps a Scope and layers async-release delegation on top
// WITHOUT changing the Scope interface (D-2/D-5): Close() always forwards to the
// wrapped scope's synchronous Close, while CloseAsync() smart-delegates —
// preferring the wrapped scope's CloseAsync when it implements AsyncDisposable,
// otherwise falling back to Close.
type AsyncScope struct {
	scope Scope
}

// NewAsyncScope wraps s. A nil scope panics, mapping dotnet's
// ThrowsIfServiceScopeIsNull (ArgumentNullException) — the AsyncScope contract
// requires a real scope to delegate to.
func NewAsyncScope(s Scope) AsyncScope {
	if s == nil {
		panic("godi: NewAsyncScope requires a non-nil scope")
	}
	return AsyncScope{scope: s}
}

// Provider returns the wrapped scope (a Scope IS a Provider), mapping dotnet's
// AsyncServiceScope.ServiceProvider property.
func (a AsyncScope) Provider() Provider {
	return a.scope
}

// Close ALWAYS forwards to the wrapped scope's synchronous Close — even when the
// wrapped scope also supports async release (maps dotnet AsyncServiceScope.
// Dispose; AsyncServiceScopeTests 3/5).
func (a AsyncScope) Close() error {
	return a.scope.Close()
}

// CloseAsync smart-delegates: it prefers the wrapped scope's CloseAsync when the
// scope implements AsyncDisposable (AsyncServiceScopeTests 6), otherwise falls
// back to the wrapped scope's synchronous Close (AsyncServiceScopeTests 4).
// Mirrors dotnet AsyncServiceScope.DisposeAsync's `if (_serviceScope is
// IAsyncDisposable ad) return ad.DisposeAsync(); _serviceScope.Dispose();`.
func (a AsyncScope) CloseAsync(ctx context.Context) error {
	if ad, ok := a.scope.(AsyncDisposable); ok {
		return ad.CloseAsync(ctx)
	}
	return a.scope.Close()
}

// CreateAsyncScope opens a child scope on p and wraps it in an AsyncScope,
// mapping dotnet's CreateAsyncScope extension method
// (NewAsyncScope(p.CreateScope())).
func CreateAsyncScope(p Provider) AsyncScope {
	return NewAsyncScope(p.CreateScope())
}
