# godi — Source Tree Analysis

## Annotated Directory Tree

```text
godi/                          # Module root (github.com/gonewx/godi)
├── go.mod                     # Go 1.27, golang.org/x/tools v0.47.0 (sole external dep)
├── mise.toml                  # Toolchain pin: go 1.27rc1
├── Makefile                   # test / conformance / golden / cover / vet / fmt / tidy / check targets
├── TESTING.md                 # Test framework documentation
├── doc.go                     # Root package godoc
├── dsl.go                     # DSL markers: Add{Singleton,Scoped,Transient}[Factory] × unkeyed/keyed
├── provider.go                # Provider / Scope interfaces + ErrProviderClosed sentinel
├── get.go                     # Free generics: Get[T], GetRequired[T], GetServices[T]
├── keyed.go                   # AnyKey sentinel + GetKeyed[T], GetRequiredKeyed[T], GetKeyedServices[T]
├── marked.go                  # MarkedConstructor + Marked() (activator best-match marker)
├── async.go                   # AsyncDisposable interface, AsyncScope, CreateAsyncScope
├── plan/                      # Registration model + validation + topological sort
│   ├── registry.go            # Registry, Service, Lifetime, *Topo()
│   ├── validate.go            # Validate(): duplicate / reserved / missing / captive / circular
│   └── doc.go                 # Package godoc
├── internal/
│   ├── genlib/                # Code-emission (the sole code path)
│   │   ├── generate.go        # Generate() + all emit* functions
│   │   ├── doc.go             # Package godoc
│   │   └── testdata/          # Golden files (*.golden)
│   └── testutil/              # stdlib-only test helpers
│       ├── assert.go          # Equal / Nil / Panics assertions over a TB seam
│       ├── factory.go         # functional-options data factory (Build / BuildFrom)
│       └── golden.go          # golden-file comparator with -update-golden flag
├── cmd/godi-gen/              # Static-analysis CLI
│   ├── main.go                # go/packages load → go/types → genlib.Generate
│   ├── doc.go                 # Package godoc
│   └── testdata/              # Test-case registration packages (deps, formb, keyed*, lifetimes, opengeneric, ...)
├── runtime/                   # Reflect-based provider oracle + disposer
│   ├── provider.go            # Builder, root/scope, Build()
│   ├── disposer.go            # Track, TrackAsync, CloseAll, CloseAllAsync (LIFO)
│   ├── collection.go          # Collection / enumerable resolution operations
│   └── doc.go                 # Package godoc
├── activator/                 # Tier C reflective construction
│   ├── activator.go           # CreateInstance[T] — best-match constructor selection
│   ├── factory.go             # GetServiceOrCreateInstance
│   └── doc.go                 # Package godoc
└── conformance/               # Dual-provider parameterized test suite
    ├── harness.go             # ProviderFactory, SpecCase, RunSuite, AssertEquivalent
    ├── fakes/                 # 52 ported .NET fakes
    ├── codegen/               # Pre-generated container (Path γ) + its tests
    └── *_DIVERGENCE.md        # Semantic-divergence docs (5 files)
```

## Key Directory Responsibilities

| Path | Responsibility |
| --- | --- |
| `godi` (root) | Public surface: DSL markers, `Provider`/`Scope`, free generic functions, `AnyKey`, async contracts |
| `plan` | Canonical registration model + validation gate + topo sort |
| `internal/genlib` | Sole code-emission path — nothing else writes generated Go |
| `internal/testutil` | Standard-library-only test helpers, importable across the module |
| `cmd/godi-gen` | Static-analysis CLI turning a registration package into a container |
| `runtime` | The reflect-based semantic oracle + LIFO disposer |
| `activator` | The single reflective-construction point (`reflect.Value.Call`) |
| `conformance` | Dual-provider spec suite, ported fakes, pre-generated container, divergence notes |

The five `*_DIVERGENCE.md` files (`ACTIVATOR`, `ASYNC`, `CALLSITE`, `EXTENSIONS`,
`OPEN_GENERIC`) document each place where godi's Go semantics deliberately differ
from the .NET reference.

## Key Conventions

| Convention | Detail |
| --- | --- |
| Red-phase scaffold | Files carry the `//go:build atdd_redphase` build tag — inert until the tag is removed to activate them |
| Golden files | Generator emission fixtures live in `internal/genlib/testdata/*.golden`, regenerated with `-update-golden` |
| Sole code-emission path | All generated Go flows through `genlib.Generate` (AD-2) |
| Sole reflective construction point | `reflect.Value.Call` appears only in `activator.CreateInstance[T]` (AD-13) |
| One registration source | Runtime and codegen providers are never hand-forked; both derive from the same DSL source (AD-14) |

## Test File Naming

| Pattern | Meaning |
| --- | --- |
| `*_test.go` | Active tests, compiled and run normally |
| `*_redphase_test.go` | ATDD red-phase scaffolds gated by the `atdd_redphase` build tag; excluded from the default build until activated |

Tests live beside the code they exercise (no dedicated `tests/` tree), which is
why `TESTING.md` sits at the module root rather than in a test directory.
