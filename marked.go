package godi

// MarkedConstructor wraps a constructor function to signal that it is the
// preferred candidate when the activator performs best-match disambiguation
// (the Go equivalent of dotnet's [ActivatorUtilitiesConstructor] attribute).
//
// The wrapped Fn must be a function value whose signature the activator can
// inspect via reflect. Place a MarkedConstructor in the ctors slice passed to
// activator.CreateInstance; the activator unwraps it and gives it priority over
// unmarked candidates (AD-9 / D-2).
type MarkedConstructor struct{ Fn any }

// Marked wraps fn as a MarkedConstructor for activator best-match selection.
func Marked(fn any) MarkedConstructor { return MarkedConstructor{Fn: fn} }
