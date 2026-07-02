package activator

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/gonewx/godi"
)

type ctor struct {
	fn reflect.Value
	ft reflect.Type
}

// CreateInstance constructs an instance of T using the best-matching constructor
// from ctors. Parameters not covered by explicit args are resolved from provider
// via GetService. This is the ONLY place in godi that uses reflect.Value.Call
// (Tier C, AD-13).
func CreateInstance[T any](provider godi.Provider, ctors []any, args ...any) (result T, err error) {
	defer func() {
		if r := recover(); r != nil {
			var zero T
			result = zero
			switch v := r.(type) {
			case error:
				err = v
			case string:
				err = errors.New(v)
			default:
				err = fmt.Errorf("%v", v)
			}
		}
	}()

	if provider == nil {
		var zero T
		return zero, fmt.Errorf("activator: provider must not be nil")
	}

	if len(ctors) == 0 {
		return zeroConstruct[T]()
	}

	marked, unmarked, mErr := partitionCtors(ctors)
	if mErr != nil {
		var zero T
		return zero, mErr
	}

	candidates := unmarked
	if len(marked) > 0 {
		candidates = marked
	}

	selected, paramValues, sErr := selectBestMatch(provider, candidates, args, reflect.TypeFor[T]())
	if sErr != nil {
		var zero T
		return zero, sErr
	}

	return invoke[T](selected.fn, paramValues)
}

func zeroConstruct[T any]() (T, error) {
	var zero T
	t := reflect.TypeFor[T]()
	if t.Kind() == reflect.Interface {
		return zero, fmt.Errorf("activator: cannot create instance of interface type %s", t)
	}
	if t.Kind() == reflect.Pointer && t.Elem().Kind() == reflect.Struct {
		return reflect.New(t.Elem()).Interface().(T), nil
	}
	return reflect.New(t).Elem().Interface().(T), nil
}

func partitionCtors(ctors []any) (marked, unmarked []ctor, err error) {
	for _, c := range ctors {
		if mc, ok := c.(godi.MarkedConstructor); ok {
			fv := reflect.ValueOf(mc.Fn)
			if !fv.IsValid() || fv.Kind() != reflect.Func {
				return nil, nil, fmt.Errorf("activator: MarkedConstructor.Fn is not a function")
			}
			marked = append(marked, ctor{fn: fv, ft: fv.Type()})
		} else {
			fv := reflect.ValueOf(c)
			if !fv.IsValid() || fv.Kind() != reflect.Func {
				return nil, nil, fmt.Errorf("activator: ctor element is not a function")
			}
			unmarked = append(unmarked, ctor{fn: fv, ft: fv.Type()})
		}
	}
	if len(marked) > 1 {
		return nil, nil, fmt.Errorf("activator: multiple constructors marked as preferred (%d found)", len(marked))
	}
	return marked, unmarked, nil
}

type matchResult struct {
	c               ctor
	explicitMatched int
}

func selectBestMatch(provider godi.Provider, candidates []ctor, args []any, targetType reflect.Type) (*ctor, []reflect.Value, error) {
	if len(candidates) == 0 {
		return nil, nil, fmt.Errorf("activator: no constructors available")
	}

	argTypes := make([]reflect.Type, len(args))
	for i, a := range args {
		if a != nil {
			argTypes[i] = reflect.TypeOf(a)
		}
	}

	var matches []matchResult
	var lastMissing reflect.Type

	for _, c := range candidates {
		explicitCount, missingType, ok := canMatch(provider, c.ft, argTypes)
		if ok {
			matches = append(matches, matchResult{
				c:               c,
				explicitMatched: explicitCount,
			})
		} else if missingType != nil {
			lastMissing = missingType
		}
	}

	if len(matches) == 0 {
		if lastMissing != nil {
			return nil, nil, fmt.Errorf("activator: unable to resolve service for type %s while activating %s", lastMissing, targetType)
		}
		if len(args) > 0 {
			return nil, nil, fmt.Errorf("activator: no constructor accepts all provided arguments")
		}
		return nil, nil, fmt.Errorf("activator: unable to resolve service while attempting to activate %s", targetType)
	}

	best := pickBest(matches)
	if best == nil {
		// dotnet: "Multiple constructors for type '{0}' were found with length {1}."
		return nil, nil, fmt.Errorf("activator: multiple constructors for type %s were found with length %d",
			targetType, bestMatchLength(matches))
	}

	if best.explicitMatched < len(args) {
		return nil, nil, fmt.Errorf("activator: no constructor accepts all provided arguments")
	}

	paramValues, missingType, rErr := resolveMatch(provider, best.c.ft, args, argTypes)
	if rErr != nil {
		return nil, nil, fmt.Errorf("activator: unable to resolve service for type %s while activating %s", missingType, targetType)
	}

	return &best.c, paramValues, nil
}

func canMatch(provider godi.Provider, ft reflect.Type, argTypes []reflect.Type) (explicitCount int, missingType reflect.Type, ok bool) {
	numIn := ft.NumIn()
	usedArgs := make([]bool, len(argTypes))

	for i := 0; i < numIn; i++ {
		paramType := ft.In(i)
		matched := false

		for j, at := range argTypes {
			if usedArgs[j] || at == nil {
				continue
			}
			if at.AssignableTo(paramType) {
				usedArgs[j] = true
				explicitCount++
				matched = true
				break
			}
		}

		if !matched {
			if !provider.IsService(paramType) {
				return 0, paramType, false
			}
		}
	}

	return explicitCount, nil, true
}

func resolveMatch(provider godi.Provider, ft reflect.Type, args []any, argTypes []reflect.Type) ([]reflect.Value, reflect.Type, error) {
	numIn := ft.NumIn()
	paramValues := make([]reflect.Value, numIn)
	usedArgs := make([]bool, len(args))

	for i := 0; i < numIn; i++ {
		paramType := ft.In(i)
		matched := false

		for j, at := range argTypes {
			if usedArgs[j] || at == nil {
				continue
			}
			if at.AssignableTo(paramType) {
				paramValues[i] = reflect.ValueOf(args[j])
				usedArgs[j] = true
				matched = true
				break
			}
		}

		if !matched {
			svc := provider.GetService(paramType)
			if svc != nil {
				paramValues[i] = reflect.ValueOf(svc)
			} else {
				return nil, paramType, fmt.Errorf("activator: unable to resolve service for type %s", paramType)
			}
		}
	}

	return paramValues, nil, nil
}

func pickBest(matches []matchResult) *matchResult {
	best := &matches[0]
	ambiguous := false

	for i := 1; i < len(matches); i++ {
		m := &matches[i]
		mIn := m.c.ft.NumIn()
		bIn := best.c.ft.NumIn()
		if mIn > bIn {
			best = m
			ambiguous = false
		} else if mIn == bIn {
			if m.explicitMatched > best.explicitMatched {
				best = m
				ambiguous = false
			} else if m.explicitMatched == best.explicitMatched {
				ambiguous = true
			}
		}
	}

	if ambiguous {
		return nil
	}
	return best
}

// bestMatchLength reports the parameter count of the longest satisfiable ctor
// among matches — the "best length" at which the ambiguity arose. Mirrors
// dotnet's MultipleCtorsFoundWithBestLength message which names this length.
func bestMatchLength(matches []matchResult) int {
	best := 0
	for i := range matches {
		if n := matches[i].c.ft.NumIn(); n > best {
			best = n
		}
	}
	return best
}

func invoke[T any](fn reflect.Value, paramValues []reflect.Value) (T, error) {
	var zero T
	out := fn.Call(paramValues)
	if len(out) == 0 {
		return zero, fmt.Errorf("activator: constructor returned no values")
	}

	result, ok := out[0].Interface().(T)
	if !ok {
		return zero, fmt.Errorf("activator: constructor return type %s is not assignable to %s",
			out[0].Type(), reflect.TypeFor[T]())
	}

	if len(out) >= 2 {
		if errVal := out[1].Interface(); errVal != nil {
			if e, eok := errVal.(error); eok {
				return zero, e
			}
		}
	}

	return result, nil
}
