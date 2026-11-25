package godi

import (
	"errors"
	"reflect"
)

// Provider is the single shared runtime contract pinned by AD-16 (method face
// revised 6 → 7 in Story 3.4, see below). It is the ONLY surface the two
// providers (the reflect-free runtime oracle and the codegen provider) and every
// generated container have in common, so its method set and semantics are fixed
// here once and never forked.
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
//
// Concurrency: once a provider is built (Build or NewRoot has returned),
// resolving singletons from the root provider, resolving transients that do not
// depend — directly or indirectly — on a scoped service, and calling
// CreateScope are safe from multiple goroutines; disposal tracking and the
// closed state are synchronized. Scoped instances are not: use each Scope from
// one goroutine at a time, and do not let a root provider that is shared across
// goroutines resolve anything scoped — a scoped service itself, a transient
// that depends on one, or GetKeyedServices(t, nil) / GetKeyedServices(t,
// AnyKey) over a type that has scoped registrations. Call Close only after
// resolution has finished.
type Provider interface {
	// GetService returns the service registered for t, or nil if none is
	// registered. A miss is nil, never a panic (AD-16). Resolving after Close
	// is not a miss: it panics with ErrProviderClosed.
	GetService(t reflect.Type) any
	// IsService reports whether t can be resolved by this provider (CAP-6).
	IsService(t reflect.Type) bool
	// IsKeyedService reports whether (t, key) can be resolved by this provider
	// (CAP-5/6). It is the keyed counterpart of IsService and, like it, is a PURE
	// QUERY that never panics: an unregistered (type, key) pair returns false.
	//
	// Key routing mirrors GetKeyedService (AD-17), with one deliberate divergence:
	//   - A nil key routes to the unkeyed namespace — IsKeyedService(t, nil) ==
	//     IsService(t) (null == unkeyed).
	//   - AnyKey returns false and NEVER panics. This is the key contrast with
	//     GetKeyedService(t, AnyKey), which always panics (AnyKey is a query-only
	//     sentinel for single-value resolution): a question about resolvability must
	//     not itself throw. AnyKey is never accepted as a registration key (both
	//     providers reject it), so an AnyKey query can never hit — hence false. The
	//     dotnet catch-all hit-row (a concrete-key query returning true because an
	//     AnyKey registration exists) is N/A in godi — there is no AnyKey
	//     registration, a Go mechanism wall (AD-17, Story 3.3 D-3).
	IsKeyedService(t reflect.Type, key any) bool
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

// ErrProviderClosed is the sentinel a provider (root or scope) panics with when a
// service is resolved — or a child scope is created — AFTER Close. It is a package
// var, not a freshly built error, so a recovering caller can match it with
// errors.Is(v, godi.ErrProviderClosed) regardless of which provider raised it.
var ErrProviderClosed = errors.New("godi: provider is closed (resolve-after-dispose, AD-16)")

// ProviderClosedError returns the ErrProviderClosed sentinel that a provider (root
// or scope) panics with when a service is resolved — or a child scope is created —
// AFTER Close. dotnet throws ObjectDisposedException; godi panics with this single
// shared sentinel from BOTH providers so the resolve-after-dispose consistency
// PINNED in the Provider contract above is identical across the runtime oracle and
// generated code (AD-16) and recoverable via errors.Is. Generated code calls
// godi.ProviderClosedError() rather than building its own error, avoiding an extra
// import in the generated file (mirrors AnyKeySingleResolveError). Pure queries
// (IsService / IsKeyedService) deliberately do NOT use this guard — they stay total
// and never panic (AD-16).
func ProviderClosedError() error {
	return ErrProviderClosed
}

// Scope is a child resolution scope. It exposes exactly the same resolution
// surface as its parent Provider and adds no method of its own (D3): a scope IS
// a Provider, so dotnet's IServiceScope.Dispose is covered by Provider.Close and
// IServiceScope.ServiceProvider by the scope being a Provider itself (AD-16).
type Scope interface {
	Provider
}
