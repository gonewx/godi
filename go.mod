module github.com/gonewx/godi

// Language version raised to 1.27, unlocking method-level type parameters on
// concrete structs (AD-5). The toolchain is pinned project-wide via mise.toml.
// The sole external dependency is golang.org/x/tools (go/packages), pinned to
// v0.47.0 per NFR6 for the generator's static analysis; x/mod and x/sync are
// its transitive deps.
go 1.27

require golang.org/x/tools v0.47.0

require (
	golang.org/x/mod v0.37.0 // indirect
	golang.org/x/sync v0.21.0 // indirect
)
