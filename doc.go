// Package godi is the public dependency-injection DSL and API for the godi
// module. Users register services here with marker functions such as
// AddSingleton; the generator (cmd/godi-gen) reads those calls statically via
// go/packages + go/types and never executes them (AD-7). The DSL body is a pure
// static marker — it builds no runtime registry representation.
//
// Story 1.1 seeded the minimal DSL (Registry + AddSingleton) needed to prove the
// AD-7 static-analysis path end to end. Story 1.2 added the shared Provider /
// Scope contract (AD-16, see provider.go) and the free generic convenience layer
// Get[T] / GetRequired[T] (AD-5, see get.go). The fuller DSL (AddScoped /
// AddTransient / AddKeyed) lands in the capability epics.
package godi
