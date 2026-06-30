# 开放泛型 Go 差异归档

godi 与 dotnet `Microsoft.Extensions.DependencyInjection` 在开放泛型（open generics）上存在根本架构差异。本文档集中归档 9 个接触点的差异，供一级门槛审计和未来维护参考。

## 核心架构论点

godi 在 generator 层**无开放泛型概念**（AD-4/AD-10）：

- dotnet 支持运行时 `MakeGenericType` 将开放泛型注册动态封闭为任意类型实参
- Go 无 `MakeGenericType` 等价物——泛型在编译期实例化
- godi 的解法：per-service helper（`Add{Svc}[T]`）显式枚举需要的封闭类型，generator 只见封闭注册
- 约束满足由 Go 编译器保证（编译期 vs dotnet 运行时检查）
- 封闭泛型物化后就是普通注册，generator 透明

## 9 个接触点差异归档表

| # | dotnet 方法 | dotnet 文件:行 | dotnet 依赖的运行时能力 | godi 适配 | godi 测试编号 | 差异原因 |
|---|---|---|---|---|---|---|
| 1 | `OpenGenericServicesCanBeResolved` | DISpec:648 | 开放→封闭实例化（`MakeGenericType`） | per-service helper 显式枚举封闭类型 | 4.1-SPEC-001 / 4.1-CGPATH-001 | AD-4：Go 无 `MakeGenericType` |
| 2 | `ConstrainedOpenGenericServicesCanBeResolved` | DISpec:665 | 运行时约束满足检查 | Go 编译期类型约束保证满足 | 4.2-SPEC-001 / 4.2-CGPATH-001 | AD-10：编译期 vs 运行时约束检查 |
| 3 | `ConstrainedOpenGenericServicesReturnsEmptyWithNoMatches` | DISpec:688 | 运行时约束过滤（不满足→跳过） | 不满足约束 → 不编译/不枚举（编译期阻断） | 4.2-SPEC-002/003 / 4.2-CGPATH-002 | AD-10：不满足约束在 Go 中是编译错误 |
| 4 | `InterfaceConstrainedOpenGenericServicesCanBeResolved` | DISpec:702 | 运行时接口约束检查 | Go `IterableLike` 密封接口约束 | 4.2-SPEC-004 / 4.2-CGPATH-003 | Go 接口约束替代 C# `where T : IEnumerable` |
| 5 | `AbstractClassConstrainedOpenGenericServicesCanBeResolved` | DISpec:725 | 运行时抽象类约束检查 | Go `AbstractClassLike` 密封接口约束 | 4.2-SPEC-005 / 4.2-CGPATH-004 | Go 无抽象类；密封接口替代 |
| 6 | `ClosedServicesPreferredOverOpenGenericServices` | DISpec:748 | 开放/封闭优先级（运行时规则） | 注册序 last-wins（无运行时优先级规则） | 4.3-SPEC-001 / 4.3-CGPATH-001 | AD-4：godi 无开放/封闭区分，优先级由注册序决定 |
| 7 | `ResolvingEnumerableContainingOpenGenericServiceUsesCorrectSlot` | DISpec:766 | 枚举内开放泛型槽位 | 物化后保留槽序（封闭注册 = 普通注册） | 4.3-SPEC-002 / 4.3-CGPATH-002 | 物化后 = 普通注册，槽序 = 注册序 |
| 8 | `ResolvesMixedOpenClosedGenericsAsEnumerable` | DISpec:908 | 测试体内 `MakeGenericType` 创建实例 | 独立 impl 类型替代（`InstanceFakeOpenGenericService`） | 4.3-SPEC-003 / 4.3-CGPATH-003 | AD-4 + R4（distinct impl type 替代 same-type instance identity） |
| 9 | `OpenGenericsWithIsService` | IsServiceSpec:39 | 开放泛型可解析性查询（开放 type token） | 已物化 = true，未物化 = false（无开放 `reflect.Type`） | 4.4-SPEC-001/002 / 4.4-CGPATH-001/002 | Go 无开放泛型 `reflect.Type`；只查已物化的封闭类型 |

## 差异分类

### 类别 A：机制墙（无等价运行时能力）

**接触点 1, 6, 7, 8** — dotnet 的 `MakeGenericType` 在运行时将开放泛型注册实例化为任意封闭类型。Go 泛型在编译期实例化，无运行时等价物。godi 的适配是 per-service helper 显式枚举，使所有封闭类型在编译期已知。

### 类别 B：编译期替代运行时（更强保证）

**接触点 2, 3, 4, 5** — dotnet 在运行时检查约束满足；Go 在编译期检查。Go 的保证更强——不满足约束的实例化是编译错误，不会产生运行时空集合。godi 用密封接口替代 C# 的抽象类/接口约束。

### 类别 C：查询语义差异

**接触点 9** — dotnet 的 `IsService` 可以查询一个开放泛型的 type token（如 `typeof(IFakeOpenGenericService<>)`），返回 true 表示"这个开放泛型有注册"。Go 无开放泛型 `reflect.Type`，godi 只查已物化的封闭类型：`IsService(reflect.TypeFor[IFakeOpenGenericService[IFakeService]]())` 返回 true 当且仅当该封闭类型已被 per-service helper 物化。

## 引用

- ARCHITECTURE-SPINE.md AD-4：开放泛型物化 Model 1（Mechanism A）
- ARCHITECTURE-SPINE.md AD-10：编译期约束满足
- 各 Story deferred-work.md 详细差异标注：
  - Story 4.1 (接触点 1)：`deferred-work.md` Story 4-1 section
  - Story 4.2 (接触点 2-5)：`deferred-work.md` Story 4-2 section
  - Story 4.3 (接触点 6-8)：`deferred-work.md` Story 4-3 section
  - Story 4.4 (接触点 9)：`deferred-work.md` Story 4-4 section
