package container

import "godipoc/lifecycle/domain"

// Get is an OPTIONAL generic ergonomic facade over the typed accessors — the
// codegen-emittable, reflection-free equivalent of dotnet's GetService<T>().
//
// A generator emits this type switch from the registration set; resolution is a
// plain type switch — NO reflect, NO map, NO interface-generic-method (which Go
// forbids). Unknown T returns ok=false. This shows the B2 world can still offer
// `Get[T]()` ergonomics on top of zero-reflection direct wiring.
func Get[T any](s *Scope) (T, bool) {
	var zero T
	switch any(zero).(type) {
	case *domain.Logger:
		return any(s.root.Logger()).(T), true
	case *domain.DbConnection:
		return any(s.DbConnection()).(T), true
	case *domain.AuditSink:
		return any(s.AuditSink()).(T), true
	case *domain.UserService:
		return any(s.NewUserService()).(T), true
	}
	return zero, false
}
