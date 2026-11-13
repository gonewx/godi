# godi — 编译期依赖注入容器

> Go 依赖注入库 · `github.com/gonewx/godi`

[English](./README.md) | 中文

---

用静态标记声明依赖图，由代码生成器产出装配代码——**主路径零反射**。

## 特点

| 能力 | 说明 |
| --- | --- |
| **编译期装配** | `godi-gen` 经 `go/packages` + `go/types` 静态读取 DSL 调用，从不执行它们；产出的容器是普通 Go 代码 |
| **图非法则不产码** | 对形态 A 构造函数，循环依赖、captive dependency、缺失注册在 `go generate` 阶段即失败；形态 B 见[已知限制](#已知限制) |
| **三种生命周期** | singleton / scoped / transient，作用域按 LIFO 释放 |
| **键控服务与开放泛型** | `AddKeyed*` 系列；开放泛型走显式封闭注册 |
| **仅一个外部依赖** | `golang.org/x/tools`，且只被生成器使用 |

## 要求

Go 1.27+（依赖具体类型上的方法级类型参数）。

## 安装

```bash
go get github.com/gonewx/godi
go get -tool github.com/gonewx/godi/cmd/godi-gen
```

第二条命令把生成器作为 `tool` 记入你的 `go.mod`。此后 `go tool godi-gen`
运行的始终是模块所依赖的那个 `github.com/gonewx/godi` 版本的生成器，
生成器与库不会出现版本不一致。

## 快速开始

### 1. 声明依赖图

这是静态标记，不在运行时执行。形态 A 按严格命名约定发现 `New{Impl}` 构造器；
没有合适构造器时，形态 B 工厂是逃生舱：

```go
package di

import "github.com/gonewx/godi"

//go:generate go tool godi-gen -pkg di -out container_gen.go .

func register() {
    var r godi.Registry

    godi.AddSingleton[Config, AppConfig](&r)   // 形态 A：发现 NewAppConfig
    godi.AddScoped[Repo, PgRepo](&r)           // 作用域生命周期
    godi.AddTransientFactory[Logger](&r, func(p godi.Provider) Logger {
        return NewLogger(godi.GetRequired[Config](p)) // 形态 B：工厂逃生舱
    })
}
```

### 2. 生成容器

```bash
go generate ./...
```

### 3. 解析服务

```go
root, err := di.NewRoot()
if err != nil {
    return err
}
defer root.Close()

cfg := godi.GetRequired[Config](root)   // 缺失即 panic
log, ok := godi.Get[Logger](root)       // 缺失返回 false
all := godi.GetServices[Repo](root)     // 全部实现
```

### 作用域

```go
scope := root.CreateScope()
defer scope.Close()   // 按 LIFO 释放该作用域内的 Disposable

repo := godi.GetRequired[Repo](scope)
```

## 生成物

`container_gen.go` 是**构建输入**——应当提交入库，消费方无需安装生成器即可构建。
修改 DSL 声明后须重跑 `go generate ./...`，并确保 `git status --porcelain`
保持为空（防装配漂移）。

## 已知限制

- **并发。** `NewRoot()` 返回之后，从 `Root` 解析 singleton、解析不（直接或间接）依赖 scoped 服务的 transient、以及调用 `CreateScope`，可以在多个 goroutine 中并发进行。scoped 实例不行：每个 `Scope` 同一时刻只能由一个 goroutine 使用，也不要让跨 goroutine 共享的 `Root` 去解析任何涉及 scoped 的东西——scoped 服务本身、依赖它的 transient，或者对含有 scoped 注册的类型调用 `GetServices`。所有解析结束之后再调用 `Close`。
- **形态 B 工厂在运行时校验，而不是在 `go generate` 时。** 缺失注册、captive 依赖、循环依赖这三项检查只对形态 A（`New{Impl}`）构造函数是静态的。单例的形态 B 工厂如果解析了未注册的服务或构成循环，`NewRoot()` 会返回 error；作用域的形态 B 工厂出现同样问题时，会在所属作用域内首次解析该服务时 panic。单例工厂通过具名函数或键控注册去解析 scoped 服务时，不会有任何诊断。
- **构造错误出现的时机因生命周期而异。** singleton 的构造函数返回 error 时，`NewRoot()` 返回包了一层的 error——用 `errors.Is` / `errors.As` 判断，不要用 `==`。scoped 或 transient 的构造函数返回 error 时，`NewRoot()` 和 `CreateScope()` 都不会失败；error 会在首次解析该服务时以 panic 的形式出现（除非有 singleton 依赖它，那样的话 `NewRoot()` 会按上面的方式返回它）。
- **形态 B 工厂按源码文本内联。** 工厂表达式会被原样复制进生成的容器，所以它不能捕获注册函数里的局部变量或参数。捕获了的话通常会在生成时被拒绝；但如果包里恰好有同名的包级标识符，工厂会静默地绑定到那个包级标识符上。运行时才知道的值（配置、已有的连接）目前只能通过包级变量交给工厂。
- **形态 B 的产物归容器所有。** 形态 B 工厂返回的值只要动态类型满足 `io.Closer` 或 `godi.AsyncDisposable`，就会被追踪并在关闭时释放。如果工厂返回的是容器已经在追踪的实例（例如为了以第二个接口暴露它而从 provider 里解析出来的那个），这个实例会被关闭不止一次，所以它的 `Close` 必须是幂等的。工厂返回容器之外持有的实例时（例如放在包级变量里的数据库连接，或者 `os.Stdout`），容器同样会接管它：只要它的动态类型有 `Close() error` 或 `CloseAsync(ctx) error`，关闭容器时就会关掉它（`root.Close()`；只有 `CloseAsync` 的实例由 `root.CloseAsync(ctx)` 关闭），即使注册的服务类型本身不含 `Close`。想让这样的实例保持打开，就返回一个没有 `Close`、也没有 `CloseAsync` 方法的包装类型，由它把服务的方法转发给该实例。
- **注册顺序是源码位置，不是调用顺序。** 生成器按文件名、再按文件内偏移排列包里所有的标记调用，不看调用关系和控制流：放在辅助函数里的标记、`if` 分支里的标记都会生效，并按它们在源码中的位置参与“后注册者胜出”和集合顺序。想让某条注册后生效，就让它在源码里排在后面——同一个文件里靠后，或放进文件名排序靠后的文件。
- **生成器与库的版本必须一致。** 重新生成的容器会调用库里的辅助符号，所以生成器比 `go.mod` 里 `github.com/gonewx/godi` 的版本新时，产出的容器通不过类型检查，生成会被拒绝。用 `go tool godi-gen` 运行生成器（见[安装](#安装)）可以让两者保持同一版本；用 `go install` 单独安装的生成器不受 `go.mod` 约束，可能与库不一致。升级库之后，要重新生成所有容器。

## 文档

完整文档为中英双语（`docs/en` / `docs/zh`），从索引进入：

| 语言 | 索引 |
| --- | --- |
| English | [docs/en/index.md](./docs/en/index.md) |
| 中文 | [docs/zh/index.md](./docs/zh/index.md) |

| 主题 | 内容 |
| --- | --- |
| [测试指南](./TESTING.md) | 测试框架、运行命令、最佳实践 |
| [变更日志](./CHANGELOG.zh-CN.md) | 各版本的变更与升级步骤 |
| [语义差异说明](./conformance/) | 与 .NET DI 的行为差异（Activator / Async / CallSite / Extensions / Open Generic） |

## 许可证

MIT，见 [LICENSE](./LICENSE)。

---

[English version →](./README.md)
