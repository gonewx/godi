// Package keyednilfix is the nil-key == unkeyed routing fixture (Story 3-3, AC1
// registration side). A nil registration key is symmetric to a nil resolution
// key: it routes to the UNKEYED namespace (AD-17). The generator must normalize
// the nil-keyed marker to a plain unkeyed service — emitting it via the unkeyed
// accessor / GetService path, with NO keyed switch arm. Test INPUT under
// testdata/; never built (AD-7).
package keyednilfix

import "github.com/gonewx/godi"

// IService / Service is registered with a nil key; the generator must treat it as
// an ordinary unkeyed singleton.
type IService interface{ isService() }

type Service struct{}

func (*Service) isService() {}

// NewService is the form-A New{Impl} constructor discovered for the (normalized)
// unkeyed service.
func NewService() *Service { return &Service{} }

// register holds the static DSL marker: a nil-keyed registration that normalizes
// to unkeyed.
func register() {
	var r godi.Registry
	godi.AddKeyedSingleton[IService, Service](&r, nil)
}

var _ = register
