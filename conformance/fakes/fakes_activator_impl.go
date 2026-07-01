// Package fakes — Story 6.5 increment: ActivatorUtilities impl-layer
// best-match fixtures, ported from dotnet DI.Tests/ActivatorUtilitiesTests.cs.
//
// These fixtures mirror the dotnet A/B/C/S/Y/Z marker types and the ClassWithABC
// constructor-chain family used to exercise the activator's best-match selection
// in fine detail (longest-available ctor, default-ctor degeneracy, marked-ctor
// priority, same-length ambiguity). They are DISTINCT from the Story 2.6
// Specification.Tests fixtures in fakes_activator.go (AnotherClass,
// ClassWithAmbiguousCtors, ...) — see Story 6.5 D-1.
//
// NOT build-tagged: fixtures are compiled dependencies of the red-phase tests.
// dotnet source: DI.Tests/ActivatorUtilitiesTests.cs lines 745-1051 (v10.0.9).
package fakes

// -- marker types (dotnet: internal class A { } ... Z { }) ------------------

// A is an empty marker service. dotnet: internal class A.
type A struct{}

// B is an empty marker service. dotnet: internal class B.
type B struct{}

// C is an empty marker service. dotnet: internal class C.
type C struct{}

// S is an empty marker service. dotnet: internal class S.
type S struct{}

// Y is an empty marker service. dotnet: internal class Y.
type Y struct{}

// Z is an empty marker service. dotnet: internal class Z.
type Z struct{}

// -- ClassWithA (single ctor) ------------------------------------------------

// ClassWithA has exactly one constructor taking A. dotnet: ClassWithA.
type ClassWithA struct {
	A *A
}

// NewClassWithA is the sole ctor. dotnet: ClassWithA(A a).
func NewClassWithA(a *A) *ClassWithA {
	return &ClassWithA{A: a}
}

// -- ClassWithABC (4-ctor chain: () / (A) / (A,B) / (A,B,C)) -----------------

// ClassWithABC is the base of the best-match constructor family. It exposes A/B/C
// fields so the selected ctor is observable through non-nil field inspection
// (not reflection). dotnet: ClassWithABC with ctors (), (A), (A,B), (A,B,C).
type ClassWithABC struct {
	A *A
	B *B
	C *C
}

// NewClassWithABC0 is the parameterless ctor. dotnet: ClassWithABC().
func NewClassWithABC0() *ClassWithABC {
	return &ClassWithABC{}
}

// NewClassWithABC1 takes only A. dotnet: ClassWithABC(A a).
func NewClassWithABC1(a *A) *ClassWithABC {
	return &ClassWithABC{A: a}
}

// NewClassWithABC2 takes A and B. dotnet: ClassWithABC(A a, B b).
func NewClassWithABC2(a *A, b *B) *ClassWithABC {
	return &ClassWithABC{A: a, B: b}
}

// NewClassWithABC3 takes A, B and C. dotnet: ClassWithABC(A a, B b, C c).
func NewClassWithABC3(a *A, b *B, c *C) *ClassWithABC {
	return &ClassWithABC{A: a, B: b, C: c}
}

// -- ClassWithABCS (longest-available ctor: (A,B,C,S) / (A,C,S)) -------------

// ClassWithABCS extends the family with an S field and two ctors of different
// length. dotnet: ClassWithABCS(A,B,C,S) and ClassWithABCS(A,C,S).
type ClassWithABCS struct {
	A *A
	B *B
	C *C
	S *S
}

// NewClassWithABCS4 is the 4-param (longest) ctor. dotnet: ClassWithABCS(A,B,C,S).
func NewClassWithABCS4(a *A, b *B, c *C, s *S) *ClassWithABCS {
	return &ClassWithABCS{A: a, B: b, C: c, S: s}
}

// NewClassWithABCS3 is the 3-param ctor (no B). dotnet: ClassWithABCS(A,C,S).
func NewClassWithABCS3(a *A, c *C, s *S) *ClassWithABCS {
	return &ClassWithABCS{A: a, C: c, S: s}
}

// -- ClassWithABCSZ (5-param single ctor) ------------------------------------

// ClassWithABCSZ has a single 5-parameter ctor — dotnet uses it to probe the
// fixed-vs-Span invoker reflection threshold (a .NET impl detail; Go has no such
// split, so #21 is adapted). dotnet: ClassWithABCSZ(A,B,C,S,Z).
type ClassWithABCSZ struct {
	A *A
	B *B
	C *C
	S *S
	Z *Z
}

// NewClassWithABCSZ5 is the sole 5-param ctor. dotnet: ClassWithABCSZ(A,B,C,S,Z).
func NewClassWithABCSZ5(a *A, b *B, c *C, s *S, z *Z) *ClassWithABCSZ {
	return &ClassWithABCSZ{A: a, B: b, C: c, S: s, Z: z}
}

// -- ABCS derived types (different definition orders, #4 Theory×3) -----------
//
// dotnet ABCS has two ctors (A,B,C) and (B,C,S); ABCS1/2/3 rewire the base call
// with different parameter orders. The godi port models each derived type as a
// distinct struct exposing A/B/C/S, with two factory funcs mirroring the two
// ctors. The test registers S in the provider and passes B, C explicitly, so the
// (B,C,S) ctor is the longest satisfiable → A stays nil.

// ABCS1 mirrors dotnet ABCS1 (ctors in order (A,B,C), (B,C,S)).
type ABCS1 struct {
	A *A
	B *B
	C *C
	S *S
}

// NewABCS1FromABC is the (A,B,C) ctor. dotnet: ABCS1(A a, B b, C c).
func NewABCS1FromABC(a *A, b *B, c *C) *ABCS1 { return &ABCS1{A: a, B: b, C: c} }

// NewABCS1FromBCS is the (B,C,S) ctor. dotnet: ABCS1(B b, C c, S s).
func NewABCS1FromBCS(b *B, c *C, s *S) *ABCS1 { return &ABCS1{B: b, C: c, S: s} }

// ABCS2 mirrors dotnet ABCS2 (ctors (B,A,C), (B,S,C)).
type ABCS2 struct {
	A *A
	B *B
	C *C
	S *S
}

// NewABCS2FromBAC is the (B,A,C) ctor. dotnet: ABCS2(B b, A a, C c).
func NewABCS2FromBAC(b *B, a *A, c *C) *ABCS2 { return &ABCS2{A: a, B: b, C: c} }

// NewABCS2FromBSC is the (B,S,C) ctor. dotnet: ABCS2(B b, S s, C c).
func NewABCS2FromBSC(b *B, s *S, c *C) *ABCS2 { return &ABCS2{B: b, C: c, S: s} }

// ABCS3 mirrors dotnet ABCS3 (ctors (B,S,C), (A,B,C)).
type ABCS3 struct {
	A *A
	B *B
	C *C
	S *S
}

// NewABCS3FromBSC is the (B,S,C) ctor. dotnet: ABCS3(B b, S s, C c).
func NewABCS3FromBSC(b *B, s *S, c *C) *ABCS3 { return &ABCS3{B: b, C: c, S: s} }

// NewABCS3FromABC is the (A,B,C) ctor. dotnet: ABCS3(A a, B b, C c).
func NewABCS3FromABC(a *A, b *B, c *C) *ABCS3 { return &ABCS3{A: a, B: b, C: c} }

// -- default-ctor-order variants (#9 Theory×2) -------------------------------

// ClassWithABC_DefaultConstructorFirst declares the parameterless ctor first.
// With no resolvable deps, best-match must degenerate to the default ctor
// regardless of declaration order. dotnet: ClassWithABC_DefaultConstructorFirst.
type ClassWithABC_DefaultConstructorFirst struct {
	A *A
	B *B
	C *C
}

// NewClassWithABC_DefaultFirst0 is the parameterless ctor (declared first).
func NewClassWithABC_DefaultFirst0() *ClassWithABC_DefaultConstructorFirst {
	return &ClassWithABC_DefaultConstructorFirst{}
}

// NewClassWithABC_DefaultFirst1 takes A.
func NewClassWithABC_DefaultFirst1(a *A) *ClassWithABC_DefaultConstructorFirst {
	return &ClassWithABC_DefaultConstructorFirst{A: a}
}

// NewClassWithABC_DefaultFirst2 takes A, B.
func NewClassWithABC_DefaultFirst2(a *A, b *B) *ClassWithABC_DefaultConstructorFirst {
	return &ClassWithABC_DefaultConstructorFirst{A: a, B: b}
}

// NewClassWithABC_DefaultFirst3 takes A, B, C.
func NewClassWithABC_DefaultFirst3(a *A, b *B, c *C) *ClassWithABC_DefaultConstructorFirst {
	return &ClassWithABC_DefaultConstructorFirst{A: a, B: b, C: c}
}

// ClassWithABC_DefaultConstructorLast declares the parameterless ctor last.
// dotnet: ClassWithABC_DefaultConstructorLast.
type ClassWithABC_DefaultConstructorLast struct {
	A *A
	B *B
	C *C
}

// NewClassWithABC_DefaultLast3 takes A, B, C (declared first).
func NewClassWithABC_DefaultLast3(a *A, b *B, c *C) *ClassWithABC_DefaultConstructorLast {
	return &ClassWithABC_DefaultConstructorLast{A: a, B: b, C: c}
}

// NewClassWithABC_DefaultLast2 takes A, B.
func NewClassWithABC_DefaultLast2(a *A, b *B) *ClassWithABC_DefaultConstructorLast {
	return &ClassWithABC_DefaultConstructorLast{A: a, B: b}
}

// NewClassWithABC_DefaultLast1 takes A.
func NewClassWithABC_DefaultLast1(a *A) *ClassWithABC_DefaultConstructorLast {
	return &ClassWithABC_DefaultConstructorLast{A: a}
}

// NewClassWithABC_DefaultLast0 is the parameterless ctor (declared last).
func NewClassWithABC_DefaultLast0() *ClassWithABC_DefaultConstructorLast {
	return &ClassWithABC_DefaultConstructorLast{}
}

// -- same-length ambiguity (#19) ---------------------------------------------

// ClassWithABC_MultipleCtorsWithSameLength has two length-1 ctors: (A) and (B).
// With both A and B registered, best-match sees two equally-long satisfiable
// ctors and must report an ambiguity error whose message names the type and the
// ctor length. dotnet: ClassWithABC_MultipleCtorsWithSameLength.
type ClassWithABC_MultipleCtorsWithSameLength struct {
	A *A
	B *B
}

// NewClassWithABC_SameLen0 is the parameterless ctor.
func NewClassWithABC_SameLen0() *ClassWithABC_MultipleCtorsWithSameLength {
	return &ClassWithABC_MultipleCtorsWithSameLength{}
}

// NewClassWithABC_SameLenA is a length-1 ctor taking A.
func NewClassWithABC_SameLenA(a *A) *ClassWithABC_MultipleCtorsWithSameLength {
	return &ClassWithABC_MultipleCtorsWithSameLength{A: a}
}

// NewClassWithABC_SameLenB is a length-1 ctor taking B (same length as A ctor).
func NewClassWithABC_SameLenB(b *B) *ClassWithABC_MultipleCtorsWithSameLength {
	return &ClassWithABC_MultipleCtorsWithSameLength{B: b}
}

// -- attribute-picks-ctor variants (#11 Theory×2) ----------------------------
//
// dotnet marks the (A,B,C) ctor with [ActivatorUtilitiesConstructor]; the godi
// port wraps the corresponding factory with godi.Marked() at the test site.
// The First/Last variants only differ in declaration order to prove order does
// not affect selection.

// ClassWithABC_FirstConstructorWithAttribute has its (A,B,C) ctor marked, declared first.
type ClassWithABC_FirstConstructorWithAttribute struct {
	A *A
	B *B
	C *C
}

// NewClassWithABC_FirstAttr3 is the (A,B,C) ctor (marked at test site, declared first).
func NewClassWithABC_FirstAttr3(a *A, b *B, c *C) *ClassWithABC_FirstConstructorWithAttribute {
	return &ClassWithABC_FirstConstructorWithAttribute{A: a, B: b, C: c}
}

// NewClassWithABC_FirstAttr2 is the (B,C) ctor (A stays nil).
func NewClassWithABC_FirstAttr2(b *B, c *C) *ClassWithABC_FirstConstructorWithAttribute {
	return &ClassWithABC_FirstConstructorWithAttribute{B: b, C: c}
}

// ClassWithABC_LastConstructorWithAttribute has its (A,B,C) ctor marked, declared last.
type ClassWithABC_LastConstructorWithAttribute struct {
	A *A
	B *B
	C *C
}

// NewClassWithABC_LastAttr2 is the (B,C) ctor (declared first, A stays nil).
func NewClassWithABC_LastAttr2(b *B, c *C) *ClassWithABC_LastConstructorWithAttribute {
	return &ClassWithABC_LastConstructorWithAttribute{B: b, C: c}
}

// NewClassWithABC_LastAttr3 is the (A,B,C) ctor (marked at test site, declared last).
func NewClassWithABC_LastAttr3(a *A, b *B, c *C) *ClassWithABC_LastConstructorWithAttribute {
	return &ClassWithABC_LastConstructorWithAttribute{A: a, B: b, C: c}
}

// -- ambiguous (A)/(B) with no marker (#13/#14 Fails group) ------------------

// ClassWithA_And_B has two length-1 ctors (A) and (B), neither marked. Resolving
// with both A and B registered is ambiguous → error. dotnet: ClassWithA_And_B.
type ClassWithA_And_B struct {
	A *A
	B *B
}

// NewClassWithA_And_B_FromA is the (A) ctor.
func NewClassWithA_And_B_FromA(a *A) *ClassWithA_And_B { return &ClassWithA_And_B{A: a} }

// NewClassWithA_And_B_FromB is the (B) ctor (same length as A ctor).
func NewClassWithA_And_B_FromB(b *B) *ClassWithA_And_B { return &ClassWithA_And_B{B: b} }

// ClassWithB_And_A is the reversed-declaration-order twin of ClassWithA_And_B.
// dotnet: ClassWithB_And_A.
type ClassWithB_And_A struct {
	A *A
	B *B
}

// NewClassWithB_And_A_FromB is the (B) ctor (declared first).
func NewClassWithB_And_A_FromB(b *B) *ClassWithB_And_A { return &ClassWithB_And_A{B: b} }

// NewClassWithB_And_A_FromA is the (A) ctor (declared last).
func NewClassWithB_And_A_FromA(a *A) *ClassWithB_And_A { return &ClassWithB_And_A{A: a} }

// -- attribute breaks same-length ambiguity (#15/#16 Passes group) -----------

// ClassWithA_And_B_Attr models dotnet's
// ClassWithA_And_B_ActivatorUtilitiesConstructorAttribute: two length-1 ctors
// (A)[marked] and (B). Marking (A) resolves the ambiguity → ctor(A) chosen.
type ClassWithA_And_B_Attr struct {
	A *A
	B *B
}

// NewClassWithA_And_B_Attr_FromA is the (A) ctor (marked at test site, declared first).
func NewClassWithA_And_B_Attr_FromA(a *A) *ClassWithA_And_B_Attr { return &ClassWithA_And_B_Attr{A: a} }

// NewClassWithA_And_B_Attr_FromB is the (B) ctor.
func NewClassWithA_And_B_Attr_FromB(b *B) *ClassWithA_And_B_Attr { return &ClassWithA_And_B_Attr{B: b} }

// ClassWithB_And_A_Attr is the reversed-order twin — (B) declared first, (A)
// marked and declared last. dotnet:
// ClassWithB_And_A_ActivatorUtilitiesConstructorAttribute.
type ClassWithB_And_A_Attr struct {
	A *A
	B *B
}

// NewClassWithB_And_A_Attr_FromB is the (B) ctor (declared first).
func NewClassWithB_And_A_Attr_FromB(b *B) *ClassWithB_And_A_Attr { return &ClassWithB_And_A_Attr{B: b} }

// NewClassWithB_And_A_Attr_FromA is the (A) ctor (marked at test site, declared last).
func NewClassWithB_And_A_Attr_FromA(a *A) *ClassWithB_And_A_Attr { return &ClassWithB_And_A_Attr{A: a} }

// -- attribute overrides longest rule (#17/#18 Ignores group) ----------------

// ClassWithA_And_AB_Attr models dotnet's
// ClassWithA_And_AB_ActivatorUtilitiesConstructorAttribute: (A)[marked] and
// (A,B). Because the marked ctor forms the sole candidate set, best-match picks
// (A) even though (A,B) is longer → B stays nil.
type ClassWithA_And_AB_Attr struct {
	A *A
	B *B
}

// NewClassWithA_And_AB_Attr_FromA is the (A) ctor (marked at test site, declared first).
func NewClassWithA_And_AB_Attr_FromA(a *A) *ClassWithA_And_AB_Attr {
	return &ClassWithA_And_AB_Attr{A: a}
}

// NewClassWithA_And_AB_Attr_FromAB is the longer (A,B) ctor (unmarked).
func NewClassWithA_And_AB_Attr_FromAB(a *A, b *B) *ClassWithA_And_AB_Attr {
	return &ClassWithA_And_AB_Attr{A: a, B: b}
}

// ClassWithAB_And_A_Attr is the reversed-order twin — (A,B) declared first, (A)
// marked and declared last. dotnet:
// ClassWithAB_And_A_ActivatorUtilitiesConstructorAttribute.
type ClassWithAB_And_A_Attr struct {
	A *A
	B *B
}

// NewClassWithAB_And_A_Attr_FromAB is the longer (A,B) ctor (unmarked, declared first).
func NewClassWithAB_And_A_Attr_FromAB(a *A, b *B) *ClassWithAB_And_A_Attr {
	return &ClassWithAB_And_A_Attr{A: a, B: b}
}

// NewClassWithAB_And_A_Attr_FromA is the (A) ctor (marked at test site, declared last).
func NewClassWithAB_And_A_Attr_FromA(a *A) *ClassWithAB_And_A_Attr {
	return &ClassWithAB_And_A_Attr{A: a}
}
