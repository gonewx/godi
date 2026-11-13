// Package godi is the public dependency-injection DSL and API for the godi
// module. Users register services here with marker functions such as
// AddSingleton; the generator (cmd/godi-gen) reads those calls statically via
// go/packages + go/types and never executes them (AD-7). The DSL body is a pure
// static marker — it builds no runtime registry representation.
//
// The DSL (dsl.go) covers AddSingleton / AddScoped / AddTransient together with
// their Factory (form B) and Keyed variants. The shared Provider / Scope contract
// lives in provider.go (AD-16), and the typed convenience functions in get.go and
// keyed.go (AD-5).
package godi
