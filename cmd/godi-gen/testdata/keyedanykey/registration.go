// Package keyedanykeyfix is the AnyKey-as-registration-key REJECTION fixture
// (Story 3-3, AC4 / D-3). godi.AnyKey is a query-only sentinel; using it as a
// registration key has no Go mechanism (catch-all registration cannot inject a
// resolve-time key into a form-A New{Impl} or a form-B func(Provider) I), so the
// generator must reject it at generate time with a SPECIFIC message and emit zero
// bytes (NFR2). This is test INPUT under testdata/ — never part of the module
// build (go/types reads it; AD-7 keeps it un-executed).
package keyedanykeyfix

import "github.com/gonewx/godi"

// IService / Service is a trivial form-A service whose only sin is being
// registered under the AnyKey sentinel.
type IService interface{ isService() }

type Service struct{}

func (*Service) isService() {}

// NewService is the form-A New{Impl} constructor — present so the rejection is
// provably about the AnyKey KEY, not a missing constructor.
func NewService() *Service { return &Service{} }

// register holds the static DSL marker the generator analyzes. AnyKey as a key
// must be rejected before any code is emitted.
func register() {
	var r godi.Registry
	godi.AddKeyedSingleton[IService, Service](&r, godi.AnyKey)
}

var _ = register
