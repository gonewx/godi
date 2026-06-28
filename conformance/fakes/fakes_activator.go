package fakes

import "sync"

// ---------------------------------------------------------------------------
// ActivatorUtilities Fakes — Story 2.6
//
// Each dotnet class with multiple constructors is mapped to a Go struct with
// multiple factory functions (D-1: Go has no constructor overloading). The
// CtorUsed field records which factory was invoked so tests can assert the
// activator selected the correct constructor.
//
// Fakes that need to be "marked" (D-2: [ActivatorUtilitiesConstructor]) are
// NOT marked here — the test code wraps them with godi.Marked() when building
// the ctors slice, keeping fakes free of activator imports.
// ---------------------------------------------------------------------------

// --- AnotherClass (AC1) ----------------------------------------------------

// AnotherClass has a single constructor that takes an IFakeService dependency.
type AnotherClass struct {
	FakeService IFakeService
}

func NewAnotherClass(svc IFakeService) *AnotherClass {
	return &AnotherClass{FakeService: svc}
}

// --- AnotherClassAcceptingData (AC2) ---------------------------------------

// AnotherClassAcceptingData accepts an IFakeService plus two string data args.
type AnotherClassAcceptingData struct {
	FakeService IFakeService
	One         string
	Two         string
}

func NewAnotherClassAcceptingData(svc IFakeService, one, two string) *AnotherClassAcceptingData {
	return &AnotherClassAcceptingData{FakeService: svc, One: one, Two: two}
}

// --- ClassWithAmbiguousCtors (AC3) -----------------------------------------

// ClassWithAmbiguousCtors has four factory functions with overlapping parameter
// types. The activator's best-match algorithm must select the correct one based
// on the explicit arguments provided.
type ClassWithAmbiguousCtors struct {
	CtorUsed string
	Svc      IFakeService
	Data1    string
	Data2    int
}

func NewClassWithAmbiguousCtorsFromString(data string) *ClassWithAmbiguousCtors {
	return &ClassWithAmbiguousCtors{CtorUsed: "string", Data1: data}
}

func NewClassWithAmbiguousCtorsFromServiceAndString(svc IFakeService, data string) *ClassWithAmbiguousCtors {
	return &ClassWithAmbiguousCtors{CtorUsed: "service_string", Svc: svc, Data1: data}
}

func NewClassWithAmbiguousCtorsFromServiceAndInt(svc IFakeService, data int) *ClassWithAmbiguousCtors {
	return &ClassWithAmbiguousCtors{CtorUsed: "service_int", Svc: svc, Data2: data}
}

func NewClassWithAmbiguousCtorsFromAll(svc IFakeService, data1 string, data2 int) *ClassWithAmbiguousCtors {
	return &ClassWithAmbiguousCtors{CtorUsed: "service_string_int", Svc: svc, Data1: data1, Data2: data2}
}

// --- ClassWithAmbiguousCtorsAndAttribute (AC5) -----------------------------

// ClassWithAmbiguousCtorsAndAttribute has three factory functions. In tests,
// one of them is wrapped with godi.Marked() to indicate the preferred ctor.
type ClassWithAmbiguousCtorsAndAttribute struct {
	CtorUsed string
	Svc      IFakeService
	Data1    string
}

func NewClassWithAmbiguousCtorsAndAttributeFromString(data string) *ClassWithAmbiguousCtorsAndAttribute {
	return &ClassWithAmbiguousCtorsAndAttribute{CtorUsed: "string", Data1: data}
}

func NewClassWithAmbiguousCtorsAndAttributeFromServiceAndString(svc IFakeService, data string) *ClassWithAmbiguousCtorsAndAttribute {
	return &ClassWithAmbiguousCtorsAndAttribute{CtorUsed: "service_string", Svc: svc, Data1: data}
}

func NewClassWithAmbiguousCtorsAndAttributeFromServiceAndOuter(svc IFakeService, _ *AnotherClass) *ClassWithAmbiguousCtorsAndAttribute {
	return &ClassWithAmbiguousCtorsAndAttribute{CtorUsed: "service_outer", Svc: svc}
}

// --- ClassWithMultipleMarkedCtors (AC6) ------------------------------------

// ClassWithMultipleMarkedCtors has two factory functions, both intended to be
// wrapped with godi.Marked() in tests — the activator must error on multiple
// marked constructors.
type ClassWithMultipleMarkedCtors struct {
	CtorUsed string
}

func NewClassWithMultipleMarkedCtorsA(svc IFakeService) *ClassWithMultipleMarkedCtors {
	return &ClassWithMultipleMarkedCtors{CtorUsed: "a"}
}

func NewClassWithMultipleMarkedCtorsB(svc IFakeService, data string) *ClassWithMultipleMarkedCtors {
	return &ClassWithMultipleMarkedCtors{CtorUsed: "b"}
}

// --- ClassWithOptionalArgsCtor (AC8, D-3) ----------------------------------

// ClassWithOptionalArgsCtor simulates dotnet optional constructor parameters
// by hardcoding the default value inside the factory function (Go has no
// optional parameters).
type ClassWithOptionalArgsCtor struct {
	Whatever string
}

func NewClassWithOptionalArgsCtor() *ClassWithOptionalArgsCtor {
	return &ClassWithOptionalArgsCtor{Whatever: "BLARGH"}
}

// --- ClassWithOptionalArgsCtorWithStructs (AC8, D-4) -----------------------

// ClassWithOptionalArgsCtorWithStructs is a simplified Go adaptation of the
// dotnet fixture with 15 optional struct parameters. Reduced to 2 pointer
// fields that default to nil (Go zero value).
type ClassWithOptionalArgsCtorWithStructs struct {
	IntVal    *int
	StringVal *string
}

func NewClassWithOptionalArgsCtorWithStructs() *ClassWithOptionalArgsCtorWithStructs {
	return &ClassWithOptionalArgsCtorWithStructs{}
}

// --- ICreationCountService / CreationCountFakeService (AC13) ---------------

// ICreationCountService is the service interface for CreationCountFakeService,
// used by GetServiceOrCreateInstance tests (AC13).
type ICreationCountService interface {
	iCreationCountService()
	GetInstanceId() int
}

// CreationCountFakeService tracks instance creation via a package-level counter
// protected by a mutex. Each instance receives a monotonically increasing id.
type CreationCountFakeService struct {
	FakeService IFakeService
	InstanceId  int
}

func (*CreationCountFakeService) iCreationCountService() {}
func (c *CreationCountFakeService) GetInstanceId() int   { return c.InstanceId }

var (
	creationCountMu    sync.Mutex
	creationCountTotal int
)

// ResetCreationCount resets the global counter. Call in t.Cleanup.
func ResetCreationCount() {
	creationCountMu.Lock()
	creationCountTotal = 0
	creationCountMu.Unlock()
}

// CreationCount returns the current global instance count.
func CreationCount() int {
	creationCountMu.Lock()
	defer creationCountMu.Unlock()
	return creationCountTotal
}

func NewCreationCountFakeService(svc IFakeService) *CreationCountFakeService {
	creationCountMu.Lock()
	creationCountTotal++
	id := creationCountTotal
	creationCountMu.Unlock()
	return &CreationCountFakeService{FakeService: svc, InstanceId: id}
}

// --- ClassWithThrowingCtor (AC11) ------------------------------------------

// ClassWithThrowingCtor panics during construction (tests panic recovery).
type ClassWithThrowingCtor struct{}

func NewClassWithThrowingCtor(_ IFakeService) *ClassWithThrowingCtor {
	panic("ctor threw an exception")
}

// --- ClassWithThrowingEmptyCtor (AC11, D-8) --------------------------------

// ClassWithThrowingEmptyCtor panics during no-arg construction. Tests the
// activator's panic → error wrapping (dotnet TargetInvocationException inner
// exception capture).
type ClassWithThrowingEmptyCtor struct{}

func NewClassWithThrowingEmptyCtor() *ClassWithThrowingEmptyCtor {
	panic("inner exception message")
}

// --- ClassWithStaticCtor (AC17) --------------------------------------------

// ClassWithStaticCtor is an empty struct representing a type with no
// constructors. When ctors is empty, the activator falls back to reflect.New
// zero-value construction (D-5).
type ClassWithStaticCtor struct{}
