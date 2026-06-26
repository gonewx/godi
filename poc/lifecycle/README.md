# PoC：Tier 1/2 生命周期切片（B2 codegen + 直接装配架构预演）

承接开放泛型 PoC 的 **B2（codegen + 直接装配、零反射）**方向，验证 dotnet DI 的
**Tier 1/2** 能否在纯 B2 模型下完整表达：三类生命周期 + 作用域 + 释放 + `Get[T]()` 人体工学。

> 结论先行：**纯 B2 可完整覆盖 Tier 1/2，零反射**。代价只有一个 —— 放弃"用运行时 `reflect.Type`
> 动态解析任意类型"，改为**生成的类型化访问器** + **类型分发（type switch）的 `Get[T]()`**，
> 这在 Go 下反而更类型安全。

## 运行

```bash
cd poc/lifecycle
GOWORK=off go test ./... -v     # 6 个用例全绿
```

## 核心思想：生命周期由「实例存放位置」编码，而非运行时描述符

dotnet 用 `ServiceDescriptor{Lifetime}` + 引擎在运行时决定缓存行为。B2 把它**结构化**：

| 生命周期 | dotnet | **B2 直接装配** | 测试 |
|---|---|---|---|
| **Singleton** | `AddSingleton` | `Root` 的字段，`NewRoot` 中构造一次 | `TestSingleton_SharedAcrossScopes` |
| **Scoped** | `AddScoped` | `Scope` 的字段，每作用域惰性构造一次 | `TestScoped_SameWithinScope_DifferAcrossScopes` |
| **Transient** | `AddTransient` | `New*()` 方法，每次调用新建 | `TestTransient_FreshEachResolve_SharesScopedDep` |
| **释放（IDisposable）** | scope 关闭按 LIFO Dispose | `disposer` 跟踪 `io.Closer`，`Scope.Close()` 按 LIFO 释放 | `TestDisposal_LIFO_OnScopeClose` |
| **单例不随作用域释放** | 单例随 root 释放 | 单例在 Root，作用域关闭不动它 | `TestSingleton_NotDisposedOnScopeClose` |
| **`GetService<T>()` 人体工学** | 接口扩展方法（运行时反射/字典） | `Get[T]()` 生成的**类型分发**，零反射 | `TestGet_GenericFacade_ZeroReflection` |

## 关键发现

1. **零反射覆盖 Tier 1/2。** 全程无 `reflect`、无 `map[reflect.Type]`、无接口泛型方法（Go 禁止）。
   依赖注入靠构造函数直接传参，生命周期靠实例存放位置。
2. **`Get[T]()` 可以保留。** 用生成的 `switch any(zero).(type)` 实现 `Get[*Logger](scope)` 这类调用，
   零反射、未注册类型返回 `ok=false`。即"既要 `Get[T]()` 顺手、又要零反射"两者可兼得。
3. **唯一让渡**：无法用运行时 `reflect.Type` 解析编译期未知的类型（与 Tier 3 同源的根本限制）。
   B2 用**类型化访问器**替代，类型安全更强；若确需运行时动态解析，再叠一层小型 reflect map 走混合即可。
4. **意外收益**：B2/wire 风格下，**循环依赖在生成/编译期即可发现**，而非 dotnet 的运行时
   `CircularDependencyException` —— 是严格的增强。

## 两个 PoC 合起来 → godi 的 B2 架构雏形

| 关注点 | 由谁负责 | 来源 PoC |
|---|---|---|
| 开放泛型 → 封闭类型物化 | codegen 从 `manifest` 生成封闭注册/装配 | `poc/opengeneric`（B2） |
| 单例/作用域/瞬态 + 释放 | 生成 `Root`/`Scope` + 类型化访问器；薄 `disposer` 运行时 | `poc/lifecycle`（本 PoC） |
| `Get[T]()` 人体工学 | 生成类型分发，零反射 | `poc/lifecycle` |
| 运行时反射 | **无**（混合模式下仅作可选回退） | — |

## 尚未覆盖（下一步候选）

`IEnumerable[T]` 多实现解析、KeyedServices 键控、构建期校验（`ValidateOnBuild`/`ValidateScopes`）。
这些在 B2 下多为"生成更多访问器/分支"，预期可纳入同一模型。
