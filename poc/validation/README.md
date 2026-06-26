# PoC：IEnumerable[T] + KeyedServices + 构建期校验（B2 架构压实）

补齐 B2（codegen + 直接装配、零反射）下 dotnet DI 的剩余 Tier-1/2 能力，并演示 B2 的
**最大架构红利：把 dotnet 的运行时校验「左移」到生成/编译期**。

## 运行

```bash
cd poc/validation
GOWORK=off go test ./... -v     # 7 个用例全绿
```

## 三块内容

### 1. IEnumerable[T] —— 多实现一并解析（`wire` 包）

`Container.Handlers() []domain.IHandler` 返回全部已注册实现，**按注册顺序**（dotnet 也保证此序）。
零反射，就是一个生成的切片。`TestIEnumerable_AllImplsInRegistrationOrder` 验证顺序。

### 2. KeyedServices —— 按键解析（`wire` 包）

`Container.Cache(key string) (domain.ICache, bool)` 即 dotnet `GetKeyedService<ICache>(key)`，
生成为一个 **key switch**，零反射，未知键返回 `ok=false`（对应 dotnet 返回 null）。
`TestKeyed_ResolveByKey` 验证。

> **架构细节（接口分发）**：键控/枚举解析这里用**类型化访问器**（`Cache(key)` 直接返回 `ICache`），
> 这是接口类型解析的稳健零反射路径。注意 lifecycle PoC 里 `Get[T]()` 的 `any(zero).(type)` 技巧
> **仅对指针/具体类型成立**——对接口类型 T，`var zero T` 是 nil 接口、会匹配 `case nil`。
> 故 godi 对接口的解析应优先走**生成的类型化访问器**，而非泛型零值分发。

### 3. 构建期校验 —— dotnet `ValidateOnBuild`/`ValidateScopes` 的左移（`plan` 包）

`plan.Registry` 是注册图模型（生成器在 emit wiring 前据此构建），`Validate()` 在**生成/编译期**抓出：

| 校验 | dotnet | B2（本 PoC） | 实测报错 |
|---|---|---|---|
| 缺失依赖 | 首次解析时抛异常 | 生成期报错（且本就编译不过） | `missing dependency: transient "UserService" depends on unregistered "DbConnection"` |
| **捕获依赖**（singleton 吃 scoped，含传递） | 仅 `ValidateScopes=true` 运行时检 | 生成期**总是**检，含传递闭包 | `captive dependency: singleton "BadCache" captures scoped "DbConnection"` |
| 循环依赖 | 运行时 `CircularDependencyException` | 生成期 DFS 检出并给出路径 | `dependency cycle: A -> B -> A` |

**红利**：dotnet 这些检查发生在**运行时**（`BuildServiceProvider`/首次解析）；B2 全部前移到 `go generate`，
有问题就生成失败、bug 进不了二进制。

## 三个 PoC 合起来 → godi 的 B2 架构已基本闭合

| 能力 | PoC |
|---|---|
| 开放泛型 → 封闭类型物化 | `poc/opengeneric` |
| 单例/作用域/瞬态 + 释放 + `Get[T]()` | `poc/lifecycle` |
| IEnumerable[T] + KeyedServices | `poc/validation`（wire） |
| 构建期校验（缺失/捕获/循环） | `poc/validation`（plan） |
| 运行时反射 | **无**（混合模式仅可选回退） |

## 仍待后续（实现阶段）

- 真正的**生成器**：把 `plan.Registry` → 校验 → emit `wire`/`Root`/`Scope`/访问器 串成一条 `go generate`。
- 注册 DSL/manifest 的最终形态（注解 vs 显式注册函数 vs 配置文件）。
- 与 ATDD 语料对接：用 `Specification.Tests` 的契约点回归校验 B2 实现。
