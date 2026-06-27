// Package godi is the public dependency-injection DSL and API for the godi
// module. Users register services here with marker functions such as
// AddSingleton; the generator (cmd/godi-gen) reads those calls statically via
// go/packages + go/types and never executes them (AD-7). The DSL body is a pure
// static marker — it builds no runtime registry representation.
//
// Story 1.1 scope: only the minimal seed (Registry + AddSingleton) needed to
// prove the AD-7 static-analysis path end to end. The fuller DSL (AddScoped /
// AddTransient / AddKeyed), the Provider interface, and Get[T] land in Story 1.2
// and the capability epics.
package godi
