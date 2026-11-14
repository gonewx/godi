[English](./CHANGELOG.md) | 中文

# 变更日志

godi 所有值得注意的变更都记录在这里。格式遵循
[Keep a Changelog](https://keepachangelog.com/zh-CN/1.1.0/)，版本号规则见 README
的[版本策略](./README.zh-CN.md#版本策略)一节。

生成器（`cmd/godi-gen`）与库同属一个模块，共用一个版本号。

## [Unreleased]

## [0.2.0] - 2026-11-13

这个版本改变了生成容器的形态和若干运行期行为。已有的容器都必须重新生成，见下面的“升级”。

### 升级

1. 升级库，然后把生成器作为 tool 记入 `go.mod`，让两者保持同一版本：
   `go get -tool github.com/gonewx/godi/cmd/godi-gen`。
2. 把每条指令改成
   `//go:generate go tool godi-gen -pkg <name> -out container_gen.go .`，
   再运行 `go generate ./...`。
3. 对 `NewRoot()` 返回的 error，把 `==` 比较换成 `errors.Is` / `errors.As`。
4. 检查返回容器外部实例（包级变量里的连接、`os.Stdout`）的形态 B 工厂。容器现在会关闭它；
   要保持它打开，就用一个没有 `Close` / `CloseAsync` 方法的类型包一层。
5. 检查以多种生命周期注册的服务类型：现在是最后注册的那个生效。

### Changed

- **破坏性变更：** `NewRoot()` 把构造失败作为 error 返回，不再 panic，并把原因包在
  `godi.BuildError` 里。这包括 singleton 构造函数或工厂里的 panic、构造过程中遇到的循环依赖，
  以及形态 B 工厂请求未注册的服务。
- **破坏性变更：** scoped 服务改为首次解析时构造。`NewRoot()` 和 `CreateScope()` 不再构造任何
  scoped 服务，所以 scoped 构造函数返回 error 不再让它们失败；这个 error 会在该服务首次被解析时
  以 panic 抛出。scoped 可释放对象按创建的逆序释放。
- **破坏性变更：** 形态 B 工厂返回的值，只要动态类型实现了 `io.Closer` 或
  `godi.AsyncDisposable`，就会被追踪并释放，包括容器外部持有的实例。含形态 B 服务的容器多出一个
  `CloseAsync` 方法。
- **破坏性变更：** 同一服务类型以多种生命周期注册时，`GetService` 返回最后注册的那个，
  `GetServices` 按注册顺序返回全部，两个 provider 一致。此前由固定的
  singleton → scoped → transient 优先级决定结果。
- **破坏性变更：** 生成器读不出的 godi 标记调用（例如位于不受支持的泛型 helper 里）现在是错误。
  此前只给警告，该注册被静默丢弃。
- **破坏性变更：** 从 `plan.Registry` 删除了 `SingletonsTopo`、`ScopedTopo`、
  `KeyedSingletonsTopo` 和 `KeyedScopedTopo`。生成的容器不再依赖静态接线顺序。
- 文档推荐的生成器运行方式改为 `go.mod` 的 `tool` 指令（`go get -tool …`，然后
  `go tool godi-gen`），不再是 `go install …@latest`。

### Added

- 生成器在写出容器之前先对它做类型检查，检查不通过时不写出任何内容。此前会产出不可编译文件的七种
  写法，现在要么能正确生成，要么在生成期被拒绝。
- 可以识别省略了类型实参的标记调用，例如
  `godi.AddSingletonFactory(&r, func(p godi.Provider) Logger { … })`。
- `godi.BuildError`、`godi.CircularDependencyError` 和 `godi.ErrCircularDependency`，
  用于描述 `NewRoot()` 的失败。
- `godi.CloseGuard`：可重入、只执行一次的关闭辅助类型，供生成的容器使用。
- `runtime.Disposer.TrackAny`：按动态类型追踪一个值以便释放。
- 两种语言的 README 都增加了“已知限制”一节。
- `godi-gen -version` 打印生成器版本。每个生成的容器都在第二行记录这个版本
  （`// godi-gen version: v0.2.0`），所以容器若不是由 `go.mod` 里那个库版本对应的生成器生成，
  会表现为产物漂移。

### Fixed

- 生成的容器不再向依赖 `[]T` 的 scoped 服务、或经由 transient 依赖另一个 scoped 服务的 scoped
  服务注入 nil。
- 以具名函数形式给出的形态 B 工厂，不再拿到尚未构造的依赖。
- 不带 key 注册的形态 B 工厂可以解析 keyed singleton。
- 多个 goroutine 同时解析可释放的 transient 不再丢失释放追踪。`Disposer` 现在可以并发使用。
- 在可释放对象的 `Close()` 里调用容器的 `Close()`，runtime provider 不再死锁。
- 构造函数签名变更、已注册类型被重命名或删除之后，重新生成不再被上一次的 `container_gen.go`
  卡住。
- keyed 形态 A 构造函数返回 `(T, error)`、两个服务共用一个构造函数返回 error 的 transient、
  容器里唯一的可释放服务是形态 A transient，这三种情况现在生成的代码可以编译。

## [0.1.1] - 2026-08-22

工具链与依赖收尾，没有 API 变更。

### Changed

- 通过 `mise.toml` 把工具链钉在 Go 1.27.0（2026-08-19 GA）。
- `golang.org/x/tools` 从 v0.47.0 升级到 v0.49.0；生成器产物逐字节不变。
- 重写了两种语言的 README 与 Quick Start，并把各份指南同步到 GA 工具链。

## [0.1.0] - 2026-07-31

首个公开发布版本。

### Added

- 编译期依赖注入：用静态标记调用声明依赖图，`godi-gen` 经 `go/packages` 静态分析产出装配代码，
  主路径零反射。
- 三种生命周期（singleton、scoped、transient）、键控服务、开放泛型的显式封闭注册、
  作用域内 LIFO 释放、图非法则不产码。
- 要求 Go 1.27 或更高版本。

[Unreleased]: https://github.com/gonewx/godi/compare/v0.2.0...HEAD
[0.2.0]: https://github.com/gonewx/godi/compare/v0.1.1...v0.2.0
[0.1.1]: https://github.com/gonewx/godi/compare/v0.1.0...v0.1.1
[0.1.0]: https://github.com/gonewx/godi/releases/tag/v0.1.0

---

[English version →](./CHANGELOG.md)
