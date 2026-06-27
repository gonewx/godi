// Package runtime is the reflect-free provider (the compliance oracle) plus a
// thin LIFO disposer. It builds services with closures and zero reflective
// construction (AD-1 / AD-13 / AD-14); the generated container depends on its
// disposer but never on the generator (AD-2).
//
// Story 1.1 placeholder: package declaration only. The provider lands in Story
// 1.3 / M1.
package runtime
