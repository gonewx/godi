package testutil

// Option mutates a value of type T in place. It is the building block of the
// data-factory pattern: start from sensible defaults, then apply per-test
// overrides — the idiomatic Go alternative to fixture factories with overrides
// (replaces faker-style builders called out in the framework-setup step).
type Option[T any] func(*T)

// Build constructs a T from its zero value, applying opts left to right.
//
//	svc := testutil.Build(
//	    func(s *Service) { s.Name = "auth" },
//	    func(s *Service) { s.Lifetime = Singleton },
//	)
func Build[T any](opts ...Option[T]) T {
	var v T
	for _, opt := range opts {
		opt(&v)
	}
	return v
}

// BuildFrom is Build starting from a provided base instead of the zero value.
// Useful for deriving a variant fixture from a shared baseline.
func BuildFrom[T any](base T, opts ...Option[T]) T {
	for _, opt := range opts {
		opt(&base)
	}
	return base
}
