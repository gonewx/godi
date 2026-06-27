package godi

import "reflect"

// Provider is the single shared runtime contract pinned by AD-16. It is the ONLY
// surface the two providers (the reflect-free runtime oracle and the codegen
// provider) and every generated container have in common, so its method set and
// semantics are fixed here once and never forked.
//
// The method face mirrors dotnet's IServiceProvider family. Unmet-lookup
// semantics are part of the contract: GetService / GetKeyedService /
// GetKeyedServices on an unregistered type return nil (never panic). The
// generic convenience layer Get[T] (see get.go) reports the miss as
// (zero, false); only GetRequired[T] panics on a miss.
//
// Provider is deliberately NON-generic: an interface method may not declare type
// parameters (AD-5), so the ergonomic Get[T] / GetRequired[T] live as free
// generic functions over Provider rather than as methods on it. Interface-typed
// services are resolved through generated typed accessors in a real container;
// the free Get[T] resolves them through GetService plus a value assertion (D2).
//
// The behavioral guarantees documented per method (idempotent Close,
// resolve-after-dispose consistency, registration-order enumeration, nil-key
// routing) are PINNED here as contract but NOT implemented in this story; the
// runtime provider (Story 1.3) and the codegen provider (Story 1.4) honor them.
type Provider interface {
	// GetService returns the service registered for t, or nil if none is
	// registered. A miss is nil, never a panic (AD-16).
	GetService(t reflect.Type) any
	// IsService reports whether t can be resolved by this provider (CAP-6).
	IsService(t reflect.Type) bool
	// GetKeyedService returns the service registered for (t, key), or nil if
	// none. A nil key routes to the unkeyed service (AD-17).
	GetKeyedService(t reflect.Type, key any) any
	// GetKeyedServices returns every service registered for (t, key) in
	// registration order (CAP-4 / CAP-5).
	GetKeyedServices(t reflect.Type, key any) []any
	// CreateScope opens a child resolution scope; the scope exposes the same
	// resolution surface as its parent, including nested CreateScope (AD-1 / AD-16).
	CreateScope() Scope
	// Close releases this provider (or scope) and the disposables it owns. It is
	// idempotent: calling it more than once is safe (AD-16).
	Close() error
}

// Scope is a child resolution scope. It exposes exactly the same resolution
// surface as its parent Provider and adds no method of its own (D3): a scope IS
// a Provider, so dotnet's IServiceScope.Dispose is covered by Provider.Close and
// IServiceScope.ServiceProvider by the scope being a Provider itself (AD-16).
type Scope interface {
	Provider
}
