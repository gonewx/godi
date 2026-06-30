package fakes

// Struct (value-type) service fakes for Story 6.1 (SPCT-18..21). dotnet has
// struct services — value types boxed into `object` on resolve. The godi analog
// is a Go struct whose interface methods have VALUE receivers, so the struct
// value (not a pointer) satisfies the interface and is boxed into `any` when
// resolved. These verify the provider's reflect.Type keying and []any
// collections behave for value types exactly as for pointer types (D-3).

// IStructFakeService is a value-type service marker. The sealed value-receiver
// method ties implementations to this package (mirrors IFakeService) and forces
// the struct value (not *struct) to be the interface-satisfying type.
type IStructFakeService interface {
	isStructFakeService()
	StructTag() string
}

// StructFakeService is a VALUE-type service: value receivers mean the struct
// value satisfies IStructFakeService, so resolving boxes the struct into `any`.
type StructFakeService struct {
	Tag string
}

func (StructFakeService) isStructFakeService() {}
func (s StructFakeService) StructTag() string  { return s.Tag }

// NewStructFakeService is the form-A constructor — it returns the value type
// (not a pointer), the canonical struct-service shape.
func NewStructFakeService() StructFakeService { return StructFakeService{Tag: "struct-fake"} }

// IStructFakeMultipleService is a value-type service registered multiple times,
// verifying struct elements keep their identity inside a []any collection (SPCT-21).
type IStructFakeMultipleService interface {
	isStructFakeMultipleService()
	MultiTag() string
}

type StructFakeOne struct{}

func (StructFakeOne) isStructFakeMultipleService() {}
func (StructFakeOne) MultiTag() string             { return "struct-one" }

func NewStructFakeOne() StructFakeOne { return StructFakeOne{} }

type StructFakeTwo struct{}

func (StructFakeTwo) isStructFakeMultipleService() {}
func (StructFakeTwo) MultiTag() string             { return "struct-two" }

func NewStructFakeTwo() StructFakeTwo { return StructFakeTwo{} }

// IStructConsumer depends on the value-type IStructFakeService (SPCT-20: a struct
// service injected as a constructor dependency). It is a pointer type so it can
// carry the boxed struct dependency.
type IStructConsumer interface {
	isStructConsumer()
	Dependency() IStructFakeService
}

type StructConsumer struct {
	dep IStructFakeService
}

func (*StructConsumer) isStructConsumer()                {}
func (c *StructConsumer) Dependency() IStructFakeService { return c.dep }

// NewStructConsumer is the form-A constructor; its signature exposes the struct
// dependency (read left-to-right by the generator, AD-8).
func NewStructConsumer(dep IStructFakeService) *StructConsumer { return &StructConsumer{dep: dep} }
