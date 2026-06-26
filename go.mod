module github.com/gonewx/godi

// Language version held at 1.26 while Go 1.27 (the architecture target) is
// pre-GA. The toolchain is pinned project-wide via mise.toml. No external
// dependencies: tests use the standard library only (see internal/testutil).
go 1.26
