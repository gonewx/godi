// Package fakes holds the test fakes ported incrementally from the dotnet
// conformance corpus (52 fakes), migrated per the epic that consumes them.
//
// Current state (Story 1.5): the Walking Skeleton's single-service slice is
// in place — IFakeService (sealed marker interface), FakeService, and the
// form-A constructor NewFakeService. The full FakeService surface
// (Value/Disposed/...) migrates with Epic 2 as construction-injection and
// dispose cases land.
package fakes
