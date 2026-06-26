# PoC：端到端代码生成器（三 PoC 合一）

把前三个 PoC 的零散结构合并成**一条真正的生成器管线**：

```
spec.Registry()  →  plan.Validate()  →  genlib.Generate()  →  app/container_gen.go  →  go build/test
   (DSL/事实源)      (构建期校验/拒绝)      (emit 零反射 wiring)      (生成物)             (行为测试)
```

## 运行（端到端）

```bash
cd poc/gen
GOWORK=off go generate ./app     # 跑生成器，产出 app/container_gen.go（幂等）
GOWORK=off go test ./...         # app 行为测试 + genlib 生成测试 全绿
GOWORK=off go run ./gen app/container_gen.go   # 等价的手动调用
```

## 各包职责

| 包 | 角色 |
|---|---|
| `spec` | **单一事实源**：用 Go 表达的注册图 DSL（生命周期/依赖/keyed/enumerable） |
| `plan` | 注册模型 + `Validate()`（缺失/捕获/循环）+ **单例拓扑排序** |
| `genlib` | `Generate(reg)`：**先校验，不过即拒绝产码**；过则 emit 零反射 wiring 并 gofmt |
| `gen` | 命令行入口（`go run`/`go:generate` 调用） |
| `app` | 示例应用；`runtime.go`（手写 disposer）+ `container_gen.go`（**生成**）+ 行为测试 |
| `domain` | 被装配的样例服务 |

## 生成器覆盖的能力（全部零反射）

- **单例**：`Root` 字段，`NewRoot` 按**拓扑序**构造（`Config` → `Logger(r.f_Config)`）。
- **作用域**：`Scope` 字段，惰性构造一次/作用域；跨作用域不同。
- **瞬态**：`New*()` 方法，每次新建，复用作用域内的 scoped 依赖。
- **释放**：disposable 服务自动 `track`，`Close()` 按 **LIFO** 释放。
- **`Get[T]()`**：生成的类型分发（仅指针/具体类型；接口走类型化访问器）。
- **IEnumerable[T]**：`Handlers() []IHandler` 按注册序。
- **KeyedServices**：`Cache(key) (ICache, bool)`。

## 构建期校验是硬门槛（B2 的左移红利）

`genlib.Generate` 调 `plan.Validate`，**图非法则返回错误且不产任何码**。测试 `genlib_test.go` 证明：

```
TestGenerate_RejectsCaptiveDependency  → 拒绝：captive dependency: singleton "Cache" captures scoped "Db"
TestGenerate_RejectsMissingDependency  → 拒绝：missing dependency: ... depends on unregistered "Nope"
```

即 dotnet 的运行时 `ValidateOnBuild`/`ValidateScopes` 在这里前移到 `go generate`，**坏图根本生不出二进制**。

## 这对 godi 意味着什么

这套 `spec → plan → genlib → app` 即 **godi B2 架构的可执行骨架**。落地时需扩展：
- `spec` 从手写 Go 升级为**注解扫描 / 配置文件**（真实 manifest 形态）；
- `genlib` 接入**开放泛型封闭类型物化**（见 `poc/opengeneric`），并按服务类型是否实现 `io.Closer` 自动判定 `Disposable`（需 go/packages 类型信息）；
- 与 **ATDD 语料**（`Specification.Tests`）对接，用契约测试回归生成实现。
