// Package runtime is the reflect-free provider (the compliance oracle) plus a
// thin LIFO disposer. It builds services with closures and zero reflective
// construction (AD-1 / AD-13 / AD-14); the generated container depends on its
// disposer but never on the generator (AD-2).
//
// Story 1.3 status: the minimal singleton slice has landed — a form-B closure
// registers a singleton (NewBuilder / AddSingleton[T] / Build), every resolve
// shares one cached instance, an unregistered type misses honestly, and Close is
// idempotent. Scoped/transient lifetimes, IEnumerable, keyed services and the
// LIFO disposer remain Epic 2/3 (Story 2.3 for the disposer).
package runtime
