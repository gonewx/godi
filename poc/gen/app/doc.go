//go:generate go run ../gen container_gen.go

// Package app is a sample application whose entire DI container is generated
// from spec.Registry() by godipoc/gen. The hand-written part is only runtime.go
// (the disposer); everything in container_gen.go is emitted.
//
// Regenerate with:  go generate ./app
package app
