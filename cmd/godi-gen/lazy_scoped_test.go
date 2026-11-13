// Lazy scoped construction (evaluation report §7.5, findings A1/B4). Scoped
// services are built by per-scope accessors on first resolution instead of
// being wired eagerly in a static topological order; each fixture below is
// generated, compiled, and run for real through its committed behavior_test.go.
package main

import "testing"

// TestLazyScoped_SliceDependencyRegisteredAfterConsumer: a scoped []T consumer
// registered before its elements sees built elements, not nil.
func TestLazyScoped_SliceDependencyRegisteredAfterConsumer(t *testing.T) {
	runLazyFixture(t, "lazyscopedslice")
}

// TestLazyScoped_DependencyThroughTransient: a scoped service reaching another
// scoped service through a transient sees it built.
func TestLazyScoped_DependencyThroughTransient(t *testing.T) {
	runLazyFixture(t, "lazyscopedviatransient")
}

// TestLazyScoped_ConstructsOnDemandAndDisposesInReverseResolutionOrder: neither
// NewRoot nor CreateScope builds a scoped service; a scope disposes in reverse
// resolution order; an unrelated failing constructor only fails its own
// resolution, and a retry is not mistaken for a cycle.
func TestLazyScoped_ConstructsOnDemandAndDisposesInReverseResolutionOrder(t *testing.T) {
	runLazyFixture(t, "lazyscopedorder")
}

// TestLazyScoped_KeyedConstructorWithErrorCompiles: a keyed registration with an
// error-returning constructor compiles and resolves.
func TestLazyScoped_KeyedConstructorWithErrorCompiles(t *testing.T) {
	runLazyFixture(t, "lazykeyederr")
}

// TestLazyScoped_KeyedScopedAndTransientConstructorsWithError: keyed scoped and
// keyed transient registrations with error-returning constructors compile and
// panic with the constructor error at the resolution site.
func TestLazyScoped_KeyedScopedAndTransientConstructorsWithError(t *testing.T) {
	runLazyFixture(t, "lazykeyedscopederr")
}

// TestLazyScoped_SharedErrorTransientDependencyCompiles: two singletons sharing
// an error-returning transient dependency compile and build.
func TestLazyScoped_SharedErrorTransientDependencyCompiles(t *testing.T) {
	runLazyFixture(t, "lazyerrtransient")
}

// TestLazyCycle_SingletonFactoriesBecomeNewRootError: a cycle between two
// singleton factories makes NewRoot return godi.ErrCircularDependency.
func TestLazyCycle_SingletonFactoriesBecomeNewRootError(t *testing.T) {
	runLazyFixture(t, "lazycyclesingleton")
}

// TestLazyCycle_ScopedFactoriesPanicOnResolution: a cycle between two scoped
// factories panics with godi.ErrCircularDependency when resolved on a scope,
// while NewRoot and unrelated scoped resolutions succeed.
func TestLazyCycle_ScopedFactoriesPanicOnResolution(t *testing.T) {
	runLazyFixture(t, "lazycyclescoped")
}
