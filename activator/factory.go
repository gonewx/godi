package activator

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/gonewx/godi"
)

type paramSource int

const (
	fromProvider paramSource = iota
	fromArg
)

type paramPlan struct {
	source   paramSource
	argIndex int
}

// CreateFactory pre-compiles a best-match constructor selection against the
// declared argTypes and returns a factory function. The factory, when called
// with a Provider and matching args, resolves remaining parameters from the
// provider and invokes the selected constructor (AC12).
//
// argTypes declares the types of explicit arguments that will be passed to the
// factory at call time. All argTypes must be consumed by the selected ctor;
// if multiple ctors are equally viable, CreateFactory returns an error.
func CreateFactory[T any](ctors []any, argTypes ...reflect.Type) (func(godi.Provider, ...any) (T, error), error) {
	if len(ctors) == 0 {
		return func(_ godi.Provider, _ ...any) (T, error) {
			return zeroConstruct[T]()
		}, nil
	}

	marked, unmarked, err := partitionCtors(ctors)
	if err != nil {
		return nil, err
	}

	candidates := unmarked
	if len(marked) > 0 {
		candidates = marked
	}

	selected, plan, selErr := selectFactoryMatch(candidates, argTypes)
	if selErr != nil {
		return nil, selErr
	}

	fn := selected.fn
	factory := func(provider godi.Provider, args ...any) (result T, err error) {
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

		paramValues := make([]reflect.Value, len(plan))
		for i, p := range plan {
			switch p.source {
			case fromArg:
				if p.argIndex >= len(args) {
					var zero T
					return zero, fmt.Errorf("activator: expected argument at index %d but only %d provided", p.argIndex, len(args))
				}
				paramValues[i] = reflect.ValueOf(args[p.argIndex])
			case fromProvider:
				if provider == nil {
					var zero T
					return zero, fmt.Errorf("activator: provider must not be nil (maps ArgumentNullException)")
				}
				paramType := fn.Type().In(i)
				svc := provider.GetService(paramType)
				if svc == nil {
					var zero T
					return zero, fmt.Errorf("activator: unable to resolve service for type %s", paramType)
				}
				paramValues[i] = reflect.ValueOf(svc)
			}
		}
		return invoke[T](fn, paramValues)
	}

	return factory, nil
}

// GetServiceOrCreateInstance tries to resolve T from the provider first. If the
// service is registered, it returns the provider's instance. If not registered,
// it falls through to CreateInstance (AC13).
func GetServiceOrCreateInstance[T any](provider godi.Provider, ctors []any, args ...any) (T, error) {
	if v, ok := godi.Get[T](provider); ok {
		return v, nil
	}
	return CreateInstance[T](provider, ctors, args...)
}

type factoryMatchResult struct {
	c               ctor
	plan            []paramPlan
	explicitMatched int
}

func selectFactoryMatch(candidates []ctor, argTypes []reflect.Type) (*ctor, []paramPlan, error) {
	if len(candidates) == 0 {
		return nil, nil, fmt.Errorf("activator: no constructors available")
	}

	var match *factoryMatchResult

	for _, c := range candidates {
		plan, explicitUsed, ok := tryFactoryMatch(c.ft, argTypes)
		if ok && explicitUsed == len(argTypes) {
			if match != nil {
				return nil, nil, fmt.Errorf("activator: ambiguous constructors — multiple matches for the given argument types")
			}
			match = &factoryMatchResult{
				c:               c,
				plan:            plan,
				explicitMatched: explicitUsed,
			}
		}
	}

	if match == nil {
		if len(argTypes) > 0 {
			return nil, nil, fmt.Errorf("activator: no constructor accepts all provided argument types")
		}
		return nil, nil, fmt.Errorf("activator: unable to resolve constructor")
	}

	return &match.c, match.plan, nil
}

func tryFactoryMatch(ft reflect.Type, argTypes []reflect.Type) ([]paramPlan, int, bool) {
	numIn := ft.NumIn()
	plan := make([]paramPlan, numIn)
	usedArgs := make([]bool, len(argTypes))
	explicitCount := 0

	for i := 0; i < numIn; i++ {
		paramType := ft.In(i)
		matched := false

		for j, at := range argTypes {
			if usedArgs[j] {
				continue
			}
			if at.AssignableTo(paramType) {
				plan[i] = paramPlan{source: fromArg, argIndex: j}
				usedArgs[j] = true
				explicitCount++
				matched = true
				break
			}
		}

		if !matched {
			plan[i] = paramPlan{source: fromProvider}
		}
	}

	return plan, explicitCount, true
}
