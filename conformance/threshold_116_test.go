//go:build atdd_redphase

// ATDD scaffold for Story 4.5 (AC2) — one-level gate assertion:
// 116 dotnet specification methods × godi coverage mapping.
// Remove the atdd_redphase build tag to activate after dev phase.

package conformance

import "testing"

type coverageStatus string

const (
	covered coverageStatus = "covered"
	adapted coverageStatus = "adapted"
	na      coverageStatus = "na"
)

type methodMapping struct {
	DotnetMethod string
	DotnetFile   string
	GodiTest     string
	Status       coverageStatus
	Note         string
}

// dotnet116MethodMap declares the authoritative mapping from each of the
// 116 dotnet specification test methods to their godi equivalents.
//
// Grouped by the 5 dotnet source files:
//   - DependencyInjectionSpecificationTests (41)
//   - ActivatorUtilitiesTests (20)
//   - KeyedDependencyInjectionSpecificationTests (45)
//   - ServiceProviderIsServiceSpecificationTests (5)
//   - ServiceProviderIsKeyedServiceSpecificationTests (5)
var dotnet116MethodMap = []methodMapping{
	// ── DependencyInjectionSpecificationTests (41 methods) ──────────────

	{DotnetMethod: "ServicesRegisteredWithImplementationTypeCanBeResolved", DotnetFile: "DISpec", GodiTest: "spec_test.go: resolves IFakeService to non-nil with correct dynamic type", Status: covered},
	{DotnetMethod: "ServicesRegisteredWithImplementationType_ReturnDifferentInstancesPerResolution_ForTransientServices", DotnetFile: "DISpec", GodiTest: "spec_lifetimes: transient: every resolve returns different instance", Status: covered},
	{DotnetMethod: "ServicesRegisteredWithImplementationType_ReturnSameInstancesPerResolution_ForSingletons", DotnetFile: "DISpec", GodiTest: "spec_test.go: TestConformance_SingletonSharedNotReconstructed", Status: covered},
	{DotnetMethod: "ServiceInstanceCanBeResolved", DotnetFile: "DISpec", GodiTest: "spec_test.go: resolves IFakeService (form-B closure = instance registration equivalent)", Status: adapted, Note: "godi form-B closure replaces raw instance registration"},
	{DotnetMethod: "TransientServiceCanBeResolvedFromProvider", DotnetFile: "DISpec", GodiTest: "spec_lifetimes: transient: resolvable from root provider", Status: covered},
	{DotnetMethod: "TransientServiceCanBeResolvedFromScope", DotnetFile: "DISpec", GodiTest: "spec_lifetimes: transient: resolvable from scope, differs from root and each other", Status: covered},
	{DotnetMethod: "NonSingletonService_WithInjectedProvider_ResolvesScopeProvider", DotnetFile: "DISpec", GodiTest: "", Status: na, Note: "godi does not support Provider self-injection (deferred 2-2 D-3, Epic 6)"},
	{DotnetMethod: "SingletonServiceCanBeResolvedFromScope", DotnetFile: "DISpec", GodiTest: "spec_lifetimes: singleton: shared across scopes", Status: covered},
	{DotnetMethod: "SingleServiceCanBeIEnumerableResolved", DotnetFile: "DISpec", GodiTest: "spec_multiimpl: single registration can be collection-resolved (DISpec:199)", Status: covered},
	{DotnetMethod: "MultipleServiceCanBeIEnumerableResolved", DotnetFile: "DISpec", GodiTest: "spec_multiimpl: multiple registrations resolved as collection (DISpec:216)", Status: covered},
	{DotnetMethod: "RegistrationOrderIsPreservedWhenServicesAreIEnumerableResolved", DotnetFile: "DISpec", GodiTest: "spec_multiimpl: registration order preserved in collection (DISpec:234)", Status: covered},
	{DotnetMethod: "OuterServiceCanHaveOtherServicesInjected", DotnetFile: "DISpec", GodiTest: "spec_multiimpl: OuterService injected with single + multi slice (DISpec:261)", Status: covered},
	{DotnetMethod: "FactoryServicesCanBeCreatedByGetService", DotnetFile: "DISpec", GodiTest: "spec_test.go: resolves IFakeService (form-B = factory equivalent)", Status: adapted, Note: "godi form-B closure = dotnet factory registration"},
	{DotnetMethod: "FactoryServicesAreCreatedAsPartOfCreatingObjectGraph", DotnetFile: "DISpec", GodiTest: "spec_test.go: resolves IFakeConsumer with injected IFakeService dependency", Status: adapted, Note: "godi form-B closure with dependency injection"},
	{DotnetMethod: "LastServiceReplacesPreviousServices", DotnetFile: "DISpec", GodiTest: "spec_multiimpl: GetService returns last-wins (DISpec:354)", Status: covered},
	{DotnetMethod: "SingletonServiceCanBeResolved", DotnetFile: "DISpec", GodiTest: "spec_test.go: resolves IFakeService to non-nil", Status: covered},
	{DotnetMethod: "ServiceProviderRegistersServiceScopeFactory", DotnetFile: "DISpec", GodiTest: "spec_lifetimes: scoped: root ≠ scope (Provider.CreateScope replaces IScopeFactory)", Status: adapted, Note: "godi uses Provider.CreateScope() not separate IScopeFactory"},
	{DotnetMethod: "ServiceScopeFactoryIsSingleton", DotnetFile: "DISpec", GodiTest: "spec_lifetimes: (Provider.CreateScope is on root, inherently singleton)", Status: adapted, Note: "CreateScope on Provider interface makes this implicit"},
	{DotnetMethod: "ScopedServiceCanBeResolved", DotnetFile: "DISpec", GodiTest: "spec_lifetimes: scoped: root ≠ scope, scope internal same", Status: covered},
	{DotnetMethod: "NestedScopedServiceCanBeResolved", DotnetFile: "DISpec", GodiTest: "spec_lifetimes: scoped: nested scope outer ≠ inner", Status: covered},
	{DotnetMethod: "ScopedServices_FromCachedScopeFactory_CanBeResolvedAndDisposed", DotnetFile: "DISpec", GodiTest: "spec_dispose: scope.Close releases scoped + transient disposables", Status: adapted, Note: "godi Provider.CreateScope+Close covers this scenario"},
	{DotnetMethod: "ScopesAreFlatNotHierarchical", DotnetFile: "DISpec", GodiTest: "spec_lifetimes: scoped: nested scope outer ≠ inner (flat = parent isolation)", Status: covered},
	{DotnetMethod: "ServiceProviderIsDisposable", DotnetFile: "DISpec", GodiTest: "spec_dispose: provider satisfies Close contract (ServiceProviderIsDisposable)", Status: covered},
	{DotnetMethod: "DisposingScopeDisposesService", DotnetFile: "DISpec", GodiTest: "spec_dispose: scope.Close releases scoped (DisposingScopeDisposesService)", Status: covered},
	{DotnetMethod: "SelfResolveThenDispose", DotnetFile: "DISpec", GodiTest: "", Status: na, Note: "Provider self-injection not supported (deferred 2-2 D-3, Epic 6)"},
	{DotnetMethod: "SafelyDisposeNestedProviderReferences", DotnetFile: "DISpec", GodiTest: "", Status: na, Note: "Provider self-injection not supported (deferred 2-2 D-3, Epic 6)"},
	{DotnetMethod: "SingletonServicesComeFromRootProvider", DotnetFile: "DISpec", GodiTest: "spec_lifetimes: singleton: shared across scopes + TestConformance_SingletonSharedNotReconstructed", Status: covered},
	{DotnetMethod: "NestedScopedServiceCanBeResolvedWithNoFallbackProvider", DotnetFile: "DISpec", GodiTest: "spec_lifetimes: scoped: nested scope outer ≠ inner", Status: covered},
	{DotnetMethod: "OpenGenericServicesCanBeResolved", DotnetFile: "DISpec", GodiTest: "spec_opengeneric: closed [IFakeService] resolves (DISpec:648)", Status: covered},
	{DotnetMethod: "ConstrainedOpenGenericServicesCanBeResolved", DotnetFile: "DISpec", GodiTest: "spec_opengeneric_constrained: satisfying [IPocoClass] returns 2 impls (DISpec:665)", Status: covered},
	{DotnetMethod: "ConstrainedOpenGenericServicesReturnsEmptyWithNoMatches", DotnetFile: "DISpec", GodiTest: "spec_opengeneric_constrained: unregistered closed type returns nil (DISpec:688)", Status: covered},
	{DotnetMethod: "InterfaceConstrainedOpenGenericServicesCanBeResolved", DotnetFile: "DISpec", GodiTest: "spec_opengeneric_constrained: interface constraint (DISpec:702)", Status: covered},
	{DotnetMethod: "AbstractClassConstrainedOpenGenericServicesCanBeResolved", DotnetFile: "DISpec", GodiTest: "spec_opengeneric_constrained: abstract-class constraint (DISpec:725)", Status: covered},
	{DotnetMethod: "ClosedServicesPreferredOverOpenGenericServices", DotnetFile: "DISpec", GodiTest: "spec_opengeneric_slotorder: closed preferred via last-wins (DISpec:748)", Status: covered},
	{DotnetMethod: "ResolvingEnumerableContainingOpenGenericServiceUsesCorrectSlot", DotnetFile: "DISpec", GodiTest: "spec_opengeneric_slotorder: enumerable of [IGenSlotA] has count=2 (DISpec:766)", Status: covered},
	{DotnetMethod: "AttemptingToResolveNonexistentServiceReturnsNull", DotnetFile: "DISpec", GodiTest: "spec_isservice: Get[T] unregistered returns (zero, false) (DISpec:787)", Status: covered},
	{DotnetMethod: "NonexistentServiceCanBeIEnumerableResolved", DotnetFile: "DISpec", GodiTest: "spec_multiimpl: unregistered type returns nil (DISpec:801)", Status: covered},
	{DotnetMethod: "ServiceContainerPicksConstructorWithLongestMatches", DotnetFile: "DISpec", GodiTest: "activator: TestActivator_LongestAvailableConstructor", Status: adapted, Note: "godi splits ctor selection into activator pkg (AD-9)"},
	{DotnetMethod: "DisposesInReverseOrderOfCreation", DotnetFile: "DISpec", GodiTest: "spec_dispose: LIFO + TestCodegen_PathGamma_LIFO_StrictOnB2", Status: covered},
	{DotnetMethod: "ResolvesMixedOpenClosedGenericsAsEnumerable", DotnetFile: "DISpec", GodiTest: "spec_opengeneric_slotorder: mixed enumerable of [IGenSlotB] (DISpec:908)", Status: covered},
	{DotnetMethod: "ResolvesDifferentInstancesForServiceWhenResolvingEnumerable", DotnetFile: "DISpec", GodiTest: "spec_multiimpl: TestConformanceMultiImpl_RegistrationOrderSequence", Status: covered},

	// ── ActivatorUtilitiesTests (20 methods) ────────────────────────────

	{DotnetMethod: "TypeActivatorEnablesYouToCreateAnyTypeWithServicesEvenWhenNotInIocContainer", DotnetFile: "ActivatorUtils", GodiTest: "activator: TestActivator_BasicConstruction", Status: covered},
	{DotnetMethod: "TypeActivatorAcceptsAnyNumberOfAdditionalConstructorParametersToProvide", DotnetFile: "ActivatorUtils", GodiTest: "activator: TestActivator_MixedParams", Status: covered},
	{DotnetMethod: "TypeActivatorWorksWithStaticCtor", DotnetFile: "ActivatorUtils", GodiTest: "activator: TestActivator_BasicConstruction (Go has no static ctors; ClassWithStaticCtor has init() equivalent)", Status: adapted, Note: "Go init() is not a constructor; test verifies construction still works"},
	{DotnetMethod: "TypeActivatorWorksWithCtorWithOptionalArgs", DotnetFile: "ActivatorUtils", GodiTest: "activator: TestActivator_OptionalArgsDefaults", Status: covered},
	{DotnetMethod: "TypeActivatorWorksWithCtorWithOptionalArgs_WithStructDefaults", DotnetFile: "ActivatorUtils", GodiTest: "activator: TestActivator_OptionalArgsStructDefaults", Status: covered},
	{DotnetMethod: "TypeActivatorCanDisambiguateConstructorsWithUniqueArguments", DotnetFile: "ActivatorUtils", GodiTest: "activator: TestActivator_DisambiguateByUniqueArgs", Status: covered},
	{DotnetMethod: "TypeActivatorRequiresPublicConstructor", DotnetFile: "ActivatorUtils", GodiTest: "activator: TestActivator_NoConstructorsError", Status: adapted, Note: "Go exported/unexported not equivalent to C# access modifiers; test verifies no-ctor error"},
	{DotnetMethod: "TypeActivatorRequiresAllArgumentsCanBeAccepted", DotnetFile: "ActivatorUtils", GodiTest: "activator: TestActivator_SurplusExplicitArgsError", Status: covered},
	{DotnetMethod: "TypeActivatorRethrowsOriginalExceptionFromConstructor", DotnetFile: "ActivatorUtils", GodiTest: "activator: TestActivator_CtorPanicPropagation", Status: covered},
	{DotnetMethod: "TypeActivatorCreateFactoryDoesNotAllowForAmbiguousConstructorMatches", DotnetFile: "ActivatorUtils", GodiTest: "activator: TestActivator_CreateFactoryAmbiguousError", Status: covered},
	{DotnetMethod: "TypeActivatorCreateInstanceUsesLongestAvailableConstructor", DotnetFile: "ActivatorUtils", GodiTest: "activator: TestActivator_LongestAvailableConstructor", Status: covered},
	{DotnetMethod: "TypeActivatorUsesMarkedConstructor", DotnetFile: "ActivatorUtils", GodiTest: "activator: TestActivator_MarkedConstructorPriority", Status: covered},
	{DotnetMethod: "TypeActivatorThrowsOnMultipleMarkedCtors", DotnetFile: "ActivatorUtils", GodiTest: "activator: TestActivator_MultipleMarkedCtorsError", Status: covered},
	{DotnetMethod: "TypeActivatorThrowsWhenMarkedCtorDoesntAcceptArguments", DotnetFile: "ActivatorUtils", GodiTest: "activator: TestActivator_MarkedCtorRejectsMismatchedArgs", Status: covered},
	{DotnetMethod: "GetServiceOrCreateInstanceRegisteredServiceTransient", DotnetFile: "ActivatorUtils", GodiTest: "activator: TestActivator_GetServiceOrCreateInstance_Transient", Status: covered},
	{DotnetMethod: "GetServiceOrCreateInstanceRegisteredServiceSingleton", DotnetFile: "ActivatorUtils", GodiTest: "activator: TestActivator_GetServiceOrCreateInstance_Singleton", Status: covered},
	{DotnetMethod: "GetServiceOrCreateInstanceUnregisteredService", DotnetFile: "ActivatorUtils", GodiTest: "activator: TestActivator_GetServiceOrCreateInstance_Unregistered", Status: covered},
	{DotnetMethod: "UnRegisteredServiceAsConstructorParameterThrowsException", DotnetFile: "ActivatorUtils", GodiTest: "activator: TestActivator_UnregisteredDependencyError", Status: covered},
	{DotnetMethod: "CreateInstance_WithAbstractTypeAndPublicConstructor_ThrowsCorrectException", DotnetFile: "ActivatorUtils", GodiTest: "activator: TestActivator_InterfaceTypeError", Status: adapted, Note: "Go has interfaces not abstract classes; test verifies interface-type rejection"},
	{DotnetMethod: "CreateInstance_CapturesInnerException_OfTargetInvocationException", DotnetFile: "ActivatorUtils", GodiTest: "activator: TestActivator_InnerExceptionCapture", Status: covered},

	// ── KeyedDependencyInjectionSpecificationTests (45 methods) ─────────

	{DotnetMethod: "CombinationalRegistration", DotnetFile: "KeyedDISpec", GodiTest: "spec_keyed_anykey: combinational truth table holds (CombinationalRegistration adapted)", Status: covered},
	{DotnetMethod: "ResolveKeyedService", DotnetFile: "KeyedDISpec", GodiTest: "spec_keyed: resolve distinct keys returns distinct impls (ResolveKeyedService)", Status: covered},
	{DotnetMethod: "ResolveNullKeyedService", DotnetFile: "KeyedDISpec", GodiTest: "spec_keyed_anykey: nil key routes to unkeyed service (ResolveNonKeyedService)", Status: covered},
	{DotnetMethod: "ResolveNonKeyedService", DotnetFile: "KeyedDISpec", GodiTest: "spec_keyed_anykey: nil key routes to unkeyed service", Status: covered},
	{DotnetMethod: "ResolveKeyedOpenGenericService", DotnetFile: "KeyedDISpec", GodiTest: "spec_opengeneric_isservice: open-generic-iskeyed materialized closed type + correct key", Status: covered},
	{DotnetMethod: "ResolveKeyedServices", DotnetFile: "KeyedDISpec", GodiTest: "spec_keyed: collection resolve returns all impls in registration order (ResolveKeyedServices)", Status: covered},
	{DotnetMethod: "ResolveKeyedServicesAnyKey", DotnetFile: "KeyedDISpec", GodiTest: "spec_keyed_anykey: GetKeyedServices(AnyKey) returns all concrete-keyed, excludes unkeyed", Status: covered},
	{DotnetMethod: "ResolveKeyedServicesAnyKeyWithAnyKeyRegistration", DotnetFile: "KeyedDISpec", GodiTest: "", Status: na, Note: "Go mechanism wall: no key-receiving factory DSL for catch-all AnyKey registration (Story 3-3)"},
	{DotnetMethod: "ResolveKeyedServicesAnyKeyConsistency", DotnetFile: "KeyedDISpec", GodiTest: "spec_keyed_anykey: GetKeyedService(AnyKey) always panics, root", Status: covered},
	{DotnetMethod: "ResolveKeyedServicesAnyKeyConsistencyWithAnyKeyRegistration", DotnetFile: "KeyedDISpec", GodiTest: "", Status: na, Note: "Go mechanism wall: no AnyKey catch-all registration (Story 3-3)"},
	{DotnetMethod: "ResolveWithAnyKeyQuery_Constructor", DotnetFile: "KeyedDISpec", GodiTest: "", Status: na, Note: "Go mechanism wall: AnyKey-query IEnumerable reference caching not applicable (Story 3-3)"},
	{DotnetMethod: "ResolveWithAnyKeyQuery_Constructor_Duplicates", DotnetFile: "KeyedDISpec", GodiTest: "", Status: na, Note: "Go mechanism wall: AnyKey-query IEnumerable reference caching (Story 3-3)"},
	{DotnetMethod: "ResolveWithAnyKeyQuery_InstanceProvided", DotnetFile: "KeyedDISpec", GodiTest: "", Status: na, Note: "Go mechanism wall: AnyKey-query IEnumerable reference caching (Story 3-3)"},
	{DotnetMethod: "ResolveWithAnyKeyQuery_InstanceProvided_Duplicates", DotnetFile: "KeyedDISpec", GodiTest: "", Status: na, Note: "Go mechanism wall: AnyKey-query IEnumerable reference caching (Story 3-3)"},
	{DotnetMethod: "ResolveKeyedServicesAnyKeyOrdering", DotnetFile: "KeyedDISpec", GodiTest: "spec_keyed_anykey: GetKeyedServices(AnyKey) preserves registration order", Status: covered},
	{DotnetMethod: "ResolveKeyedGenericServices", DotnetFile: "KeyedDISpec", GodiTest: "spec_opengeneric_isservice: open-generic-iskeyed tests cover keyed generic resolution", Status: covered},
	{DotnetMethod: "ResolveKeyedServiceSingletonInstance", DotnetFile: "KeyedDISpec", GodiTest: "spec_keyed: singleton keyed resolves correct type (ResolveKeyedServiceSingletonInstance/Type)", Status: covered},
	{DotnetMethod: "ResolveKeyedServiceSingletonInstanceWithKeyInjection", DotnetFile: "KeyedDISpec", GodiTest: "spec_keyed_injection: closure capture injects key value (ResolveKeyedServiceWithKeyInjection)", Status: covered},
	{DotnetMethod: "ResolveKeyedServiceSingletonInstanceWithAnyKey", DotnetFile: "KeyedDISpec", GodiTest: "", Status: na, Note: "Go mechanism wall: AnyKey catch-all registration (Story 3-3)"},
	{DotnetMethod: "ResolveKeyedServicesSingletonInstanceWithAnyKey", DotnetFile: "KeyedDISpec", GodiTest: "", Status: na, Note: "Go mechanism wall: AnyKey catch-all registration (Story 3-3)"},
	{DotnetMethod: "ResolveKeyedServiceSingletonInstanceWithKeyedParameter", DotnetFile: "KeyedDISpec", GodiTest: "spec_keyed_injection: form-B closure injects keyed dependency (ResolveKeyedServiceWithKeyedParameter)", Status: covered},
	{DotnetMethod: "ResolveKeyedServiceWithKeyedParameter_MissingRegistration_SecondParameter", DotnetFile: "KeyedDISpec", GodiTest: "spec_keyed_injection: missing keyed dependency — factory receives nil (MissingRegistration)", Status: covered},
	{DotnetMethod: "ResolveKeyedServiceWithKeyedParameter_MissingRegistration_FirstParameter", DotnetFile: "KeyedDISpec", GodiTest: "spec_keyed_injection: missing keyed dependency — factory receives nil (MissingRegistration)", Status: covered},
	{DotnetMethod: "ResolveKeyedServiceWithKeyedParameter_MissingRegistrationButWithDefaults", DotnetFile: "KeyedDISpec", GodiTest: "spec_keyed_injection: missing keyed dependency — factory receives nil (Go zero-value fallback)", Status: adapted, Note: "Go zero-value fallback replaces C# default parameter values"},
	{DotnetMethod: "ResolveKeyedServiceWithKeyedParameter_MissingRegistrationButWithUnkeyedService", DotnetFile: "KeyedDISpec", GodiTest: "spec_keyed_injection: missing keyed dependency — godi form-B closure controls injection", Status: adapted, Note: "godi closure injection replaces attribute-based keyed parameter injection"},
	{DotnetMethod: "CreateServiceWithKeyedParameter", DotnetFile: "KeyedDISpec", GodiTest: "activator: TestActivator_CreateInstanceWithKeyedParameter", Status: covered},
	{DotnetMethod: "ResolveKeyedServiceSingletonFactory", DotnetFile: "KeyedDISpec", GodiTest: "spec_keyed: singleton factory — two resolves return same instance (ResolveKeyedServiceSingletonFactory)", Status: covered},
	{DotnetMethod: "ResolveKeyedServiceSingletonFactoryWithAnyKey", DotnetFile: "KeyedDISpec", GodiTest: "", Status: na, Note: "Go mechanism wall: AnyKey catch-all factory registration (Story 3-3)"},
	{DotnetMethod: "ResolveKeyedServiceSingletonFactoryWithAnyKeyIgnoreWrongType", DotnetFile: "KeyedDISpec", GodiTest: "", Status: na, Note: "Go mechanism wall: AnyKey catch-all factory (Story 3-3)"},
	{DotnetMethod: "ResolveKeyedServiceSingletonType", DotnetFile: "KeyedDISpec", GodiTest: "spec_keyed: singleton keyed resolves correct type", Status: covered},
	{DotnetMethod: "ResolveKeyedServiceTransientFactory", DotnetFile: "KeyedDISpec", GodiTest: "spec_keyed: transient factory — different instances (ResolveKeyedServiceTransientFactory)", Status: covered},
	{DotnetMethod: "ResolveKeyedServiceTransientType", DotnetFile: "KeyedDISpec", GodiTest: "spec_keyed: transient keyed — two resolves return different instances (ResolveKeyedServiceTransientType)", Status: covered},
	{DotnetMethod: "ResolveKeyedServiceTransientTypeWithAnyKey", DotnetFile: "KeyedDISpec", GodiTest: "", Status: na, Note: "Go mechanism wall: AnyKey catch-all registration (Story 3-3)"},
	{DotnetMethod: "ResolveKeyedSingletonFromInjectedServiceProvider", DotnetFile: "KeyedDISpec", GodiTest: "spec_keyed_injection: injected provider resolves keyed service dynamically (FromInjectedServiceProvider)", Status: covered},
	{DotnetMethod: "ResolveKeyedTransientFromInjectedServiceProvider", DotnetFile: "KeyedDISpec", GodiTest: "spec_keyed_injection: injected provider resolves keyed service dynamically", Status: covered},
	{DotnetMethod: "ResolveKeyedSingletonFromScopeServiceProvider", DotnetFile: "KeyedDISpec", GodiTest: "spec_keyed_injection: scope provider injection — singleton accessible from scope", Status: covered},
	{DotnetMethod: "ResolveKeyedScopedFromScopeServiceProvider", DotnetFile: "KeyedDISpec", GodiTest: "spec_keyed_injection: scoped via scope — same scope shared, cross scope isolated", Status: covered},
	{DotnetMethod: "ResolveKeyedTransientFromScopeServiceProvider", DotnetFile: "KeyedDISpec", GodiTest: "spec_keyed_injection: transient via scope — two scope resolves return different instances", Status: covered},
	{DotnetMethod: "ResolveRequiredKeyedServiceThrowsIfNotFound", DotnetFile: "KeyedDISpec", GodiTest: "spec_keyed: GetRequiredKeyed panics for unregistered key (ResolveRequiredKeyedServiceThrowsIfNotFound)", Status: covered},
	{DotnetMethod: "ResolveKeyedServiceThrowsIfNotSupported", DotnetFile: "KeyedDISpec", GodiTest: "", Status: na, Note: "godi always supports keyed services; no IKeyedServiceProvider gate"},
	{DotnetMethod: "SimpleServiceKeyedResolution", DotnetFile: "KeyedDISpec", GodiTest: "spec_keyed: resolve distinct keys returns distinct impls", Status: covered},
	{DotnetMethod: "ResolveKeyedServiceWithFromServiceKeyAttribute", DotnetFile: "KeyedDISpec", GodiTest: "", Status: na, Note: "Go has no attributes; FromServiceKey injection uses godi form-B closure capture instead (deferred)"},
	{DotnetMethod: "ResolveKeyedServiceWithFromServiceKeyAttribute_NotFound", DotnetFile: "KeyedDISpec", GodiTest: "", Status: na, Note: "Go has no attributes; FromServiceKey (deferred)"},
	{DotnetMethod: "ResolveKeyedServiceWithFromServiceKeyAttribute_NotFound_WithUnkeyed", DotnetFile: "KeyedDISpec", GodiTest: "", Status: na, Note: "Go has no attributes; FromServiceKey (deferred)"},
	{DotnetMethod: "ResolveUnkeyedServiceWithFromServiceKeyAttributeWithNullKey", DotnetFile: "KeyedDISpec", GodiTest: "", Status: na, Note: "Go has no attributes; FromServiceKey (deferred)"},

	// ── ServiceProviderIsServiceSpecificationTests (5 methods) ──────────

	{DotnetMethod: "ExplicitServiceRegisterationWithIsService", DotnetFile: "IsServiceSpec", GodiTest: "spec_isservice: registered interface=true, impl type=false (IsServiceSpec:17)", Status: covered},
	{DotnetMethod: "OpenGenericsWithIsService", DotnetFile: "IsServiceSpec", GodiTest: "spec_opengeneric_isservice: materialized closed type IsService=true (touchpoint 9)", Status: covered},
	{DotnetMethod: "ClosedGenericsWithIsService", DotnetFile: "IsServiceSpec", GodiTest: "spec_isservice: closed generic type IsService=true (IsServiceSpec:60)", Status: covered},
	{DotnetMethod: "IEnumerableWithIsServiceAlwaysReturnsTrue", DotnetFile: "IsServiceSpec", GodiTest: "spec_isservice: Go adaptation — IsService covers GetService face only (IsServiceSpec:82)", Status: adapted, Note: "Go has no implicit IEnumerable→IsService=true mapping; godi checks element type only"},
	{DotnetMethod: "BuiltInServicesWithIsServiceReturnsTrue", DotnetFile: "IsServiceSpec", GodiTest: "spec_isservice: Go adaptation — Provider not auto-registered as service (IsServiceSpec:104)", Status: adapted, Note: "godi does not auto-register Provider/IScopeFactory; only explicitly registered types return true"},

	// ── ServiceProviderIsKeyedServiceSpecificationTests (5 methods) ─────

	{DotnetMethod: "ExplicitServiceRegistrationWithIsKeyedService", DotnetFile: "IsKeyedServiceSpec", GodiTest: "spec_keyed_isservice: registered (type,key) true (ExplicitServiceRegistrationWithIsKeyedService)", Status: covered},
	{DotnetMethod: "OpenGenericsWithIsKeyedService", DotnetFile: "IsKeyedServiceSpec", GodiTest: "spec_opengeneric_isservice: open-generic-iskeyed materialized closed type + correct key", Status: covered},
	{DotnetMethod: "ClosedGenericsWithIsKeyedService", DotnetFile: "IsKeyedServiceSpec", GodiTest: "spec_keyed_isservice: closed-generic keyed reports true (ClosedGenericsWithIsKeyedService)", Status: covered},
	{DotnetMethod: "IEnumerableWithIsKeyedServiceAlwaysReturnsTrue", DotnetFile: "IsKeyedServiceSpec", GodiTest: "", Status: na, Note: "Go has no implicit IEnumerable→IsKeyedService=true mapping (Story 3-4)"},
	{DotnetMethod: "NonKeyedServiceWithIsKeyedService", DotnetFile: "IsKeyedServiceSpec", GodiTest: "spec_keyed_isservice: nil key routes to unkeyed IsService (NonKeyedServiceWithIsKeyedService)", Status: covered},
}

func TestFirstTierThreshold_116_CoverageAssertion(t *testing.T) {
	const expectedTotal = 116

	if len(dotnet116MethodMap) != expectedTotal {
		t.Fatalf("dotnet116MethodMap has %d entries, want exactly %d", len(dotnet116MethodMap), expectedTotal)
	}

	var coveredCount, adaptedCount, naCount, unmappedCount int
	var unmapped []string

	fileGroupCounts := map[string]struct{ total, covered, adapted, na int }{}

	for _, m := range dotnet116MethodMap {
		fg := fileGroupCounts[m.DotnetFile]
		fg.total++

		switch m.Status {
		case covered:
			coveredCount++
			fg.covered++
		case adapted:
			adaptedCount++
			fg.adapted++
		case na:
			naCount++
			fg.na++
		default:
			unmappedCount++
			unmapped = append(unmapped, m.DotnetMethod)
		}

		fileGroupCounts[m.DotnetFile] = fg
	}

	if unmappedCount > 0 {
		t.Errorf("found %d unmapped methods (no status assigned): %v", unmappedCount, unmapped)
	}

	effectiveCoverage := coveredCount + adaptedCount
	effectiveTarget := expectedTotal - naCount

	if effectiveCoverage < effectiveTarget {
		t.Errorf("coverage gap: covered(%d) + adapted(%d) = %d, but need %d (= 116 - %d N/A)",
			coveredCount, adaptedCount, effectiveCoverage, effectiveTarget, naCount)
	}

	t.Logf("First-Tier Threshold 116 Coverage:")
	t.Logf("  Total methods:     %d", expectedTotal)
	t.Logf("  Covered:           %d", coveredCount)
	t.Logf("  Adapted:           %d", adaptedCount)
	t.Logf("  N/A:               %d", naCount)
	t.Logf("  Effective:         %d / %d", effectiveCoverage, effectiveTarget)

	expectedFileCounts := map[string]int{
		"DISpec":            41,
		"ActivatorUtils":    20,
		"KeyedDISpec":       45,
		"IsServiceSpec":     5,
		"IsKeyedServiceSpec": 5,
	}

	for file, want := range expectedFileCounts {
		got := fileGroupCounts[file]
		if got.total != want {
			t.Errorf("file group %s: got %d methods, want %d", file, got.total, want)
		}
		t.Logf("  %-20s total=%-3d covered=%-3d adapted=%-3d na=%-3d", file, got.total, got.covered, got.adapted, got.na)
	}
}
