# ActivatorUtilities 移植分歧归档（Story 6.5）

本文件集中归档 godi 移植 dotnet `DI.Tests/ActivatorUtilitiesTests.cs`（impl 层，v10.0.9）
时的**语义分歧**：既包括无法在 Go 中表达而标记为 **N/A** 的 12 个 `ConditionalTheory`，
也包括为适配 Go 机制而重写、但保留核心语义意图的 4 个 **adapted** 方法。

> **诚实门槛（NFR7）**：N/A 与 adapted 条目不得静默丢弃——每一条都必须携带架构锚点
> （AD-n / CAP-n / NFR-n）说明「为何不能字面移植」以及「godi 保留了什么」。
>
> 记账口径见 `conformance/threshold_actutil_test.go`：
> - **20 非条件方法**（17 Fact + 3 Theory）计入二级门槛 220 总账（covered 16 + adapted 4）。
> - **12 ConditionalTheory** 是 Go 机制墙，**不计入** 220，仅在此归档 + gate 计数审计。

---

## 一、adapted 方法（4，计入 220，语义等价但机制重写）

这 4 个方法**保留了核心断言**，仅将无 Go 对应物的机制细节标为 N/A。

### #3 `CreateInstance_BadlyConfiguredIServiceProviderIsService_ProperlyCreatesUnambiguousInstance`

- **dotnet 前提**：注入一个「配置错误」的 `IServiceProviderIsService`，验证即便如此，
  无歧义的单构造仍成功，且 `IsService` 确实被调用（`IsServiceGotCalled` 探针）。
- **Go 分歧（锚点 AD-16）**：godi 的 `IsService` / `IsKeyedService` 是 **Provider 接口的固有方法**，
  **不是**可注入 / 可替换的 `IServiceProviderIsService` 服务。因此「配置错误的 IsService」这一前提
  在 godi 中**不成立**——没有可被外部替换的 IsService 实现。
- **godi 保留**：核心断言「无歧义单构造成功」（`TestActivatorImpl_UnambiguousConstructionSucceeds`）。
- **N/A 部分**：`IsServiceGotCalled` 探针——IsService 不可注入，无从探测其调用。

### #5 `CreateInstance_NullInstance_HandlesBadInputWithInvalidOperationException`

- **dotnet 前提**：传入一个**类型为 B 的 null**（typed null）作为显式实例，期望
  `InvalidOperationException`——运行时仍能从 null 的静态类型 `B` 恢复出参数类型。
- **Go 分歧**：Go 的 `nil` interface 值**丢失静态类型**——`activator.CreateInstance(p, ctors, nil, ...)`
  中的 `nil` 无法参与按类型的参数匹配（`reflect.ValueOf(nil)` 无类型）。没有编译期/运行时机制
  从一个无类型 nil 恢复出「它本应是 B」。
- **godi 保留**：核心意图「坏输入 → 干净的 error，而非深层 NPE」
  （`TestActivatorImpl_BadInputSurfacesAsError`：无类型 nil 使得无 ctor 可满足 → error）。
- **N/A 部分**：null-typed-as-B 的精确恢复机制。

### #20 `CreateFactory_CreatesFactoryMethod_4Types_3Injected`

- **dotnet 前提**：`CreateFactory(Type, ...)` 返回 `ObjectFactory`（非泛型命名委托），
  `CreateFactory<T>` 返回 `ObjectFactory<T>`；测试用 `Assert.IsType<ObjectFactory>` 断言返回类型形态。
- **Go 分歧**：godi 只有单一泛型 `CreateFactory[T](ctors, argTypes...) (func(Provider, ...any) (T, error), error)`，
  返回一个 **func 值**——没有非泛型 `ObjectFactory` 与泛型 `ObjectFactory<T>` 的命名委托分野。
- **godi 保留**：核心语义「显式 slot `[*B]` + provider 解析 A/C/S，call 时传 B，四字段全填充」
  （`TestActivatorImpl_CreateFactory_4Types_3Injected`）。
- **N/A 部分**：`Assert.IsType<ObjectFactory>` 的返回类型形态断言。

### #21 `CreateFactory_CreatesFactoryMethod_5Types_5Injected`

- **dotnet 前提**：5 个参数——.NET 中 5 是「固定参数数组 vs `Span<>` invoker 参数」的
  反射实现阈值分野，测试藉此覆盖两条 invoker 代码路径。
- **Go 分歧（锚点 AD-13）**：godi 的 Tier C 反射用 `reflect.Value.Call` 统一构造，
  **没有** fixed-vs-Span 的阈值分裂——参数数量不改变 invoker 路径。
- **godi 保留**：语义意图「全部 5 个依赖 A/B/C/S/Z 均由 provider 解析」
  （`TestActivatorImpl_CreateFactory_5Types_5Injected`）。
- **N/A 部分**：5-type 阈值断言（纯 .NET 反射实现细节）。

---

## 二、ConditionalTheory 方法（12，Go-N/A，不计入 220）

这 12 个方法在 dotnet 中以 `[ConditionalTheory(typeof(RemoteExecutor))]` 标注，
分三类，均为 Go 无法表达的机制墙。

### KeyedParams ×4 — 键控工厂参数注入

| dotnet 方法 | N/A 锚点 |
|---|---|
| `CreateFactory_CreatesFactoryMethod_KeyedParams` | godi `CreateFactory` 无键控参数注入 slot；键控工厂参数 = 未来能力（deferred），AD-17 键控语义 |
| `CreateFactory_CreatesFactoryMethod_KeyedParams_5Types` | 同上：无键控 factory-param slot（deferred） |
| `CreateFactory_CreatesFactoryMethod_KeyedParams_ValueTypes` | 同上：值类型键的 factory 参数暂不支持（deferred） |
| `CreateFactory_CreatesFactoryMethod_KeyedParams_1Injected` | 同上：混合「显式 + 键控 provider」参数（deferred） |

**架构说明**：dotnet 的 `[FromKeyedServices(key)]` 参数特性让 `CreateFactory` 能在工厂参数层面
声明「此参数从键控服务解析」。godi 的 `CreateFactory[T]` 当前只区分 `fromArg` / `fromProvider`
（见 `activator/factory.go` 的 `paramPlan`），无键控 slot。键控工厂参数属未来能力，本期 defer。

### RemoteExecutor ×6 — 跨进程 + DisableDynamicCode 验证

| dotnet 方法 | N/A 锚点 |
|---|---|
| `CreateFactory_RemoteExecutor_CreatesFactoryMethod` | Go 无 xunit `RemoteExecutor`（跨进程测试框架），亦无动态代码开关；单一 reflect-free codegen + Tier C 反射后端（AD-6 仅 Runtime 引擎） |
| `CreateFactory_RemoteExecutor_NullArguments_Throws` | 跨进程 + DisableDynamicCode，无 Go 对应物（AD-6） |
| `CreateFactory_RemoteExecutor_NoArguments_UseNullDefaultValue` | 跨进程 + DisableDynamicCode；C# 可选参数默认值无 Go ctor 对应（AD-6） |
| `CreateFactory_RemoteExecutor_NoArguments_ThrowRequiredValue` | 跨进程 + DisableDynamicCode；required-value 错误路径（AD-6） |
| `CreateFactory_RemoteExecutor_NullArgument_UseDefaultValue` | 跨进程 + DisableDynamicCode；C# 默认值参数（AD-6） |
| `CreateFactory_RemoteExecutor_NoParameters_Success` | 跨进程 + DisableDynamicCode；无参工厂（AD-6） |

**架构说明**：`RemoteExecutor` 在独立子进程中运行测试体，用于验证 `DisableDynamicCode`
（AOT/裁剪场景禁用运行时代码生成）下的行为。godi 无「动态代码开关」——它只有两条固定后端：
reflect-free codegen（B2 主路径）与 Tier C 反射（activator，opt-in）。不存在需要跨进程隔离
验证的动态/静态代码切换，且 Go 测试无 `RemoteExecutor` 等价物（AD-6：只取 Runtime 引擎对应项）。
C# 的可选参数默认值（`arg = null` / required-value）也无 Go 构造器等价语法。

### CollectibleAssembly ×2 — 可卸载程序集

| dotnet 方法 | N/A 锚点 |
|---|---|
| `CreateInstance_CollectibleAssembly` | Go 无可卸载 / collectible 程序集概念（`AssemblyLoadContext`）；且在 `RemoteExecutor` 下运行（AD-6） |
| `CreateInstance_NormalAssembly` | CollectibleAssembly 的对照组；同样的 RemoteExecutor + 无可卸载程序集墙（AD-6） |

**架构说明**：.NET 的 `AssemblyLoadContext` 支持「可卸载程序集」（collectible assemblies），
测试验证 activator 缓存的委托不会阻止程序集卸载（内存泄漏防护）。Go **没有可卸载编译单元**
的概念——包在编译期静态链接，运行时无程序集加载/卸载生命周期。此二方法（含 `NormalAssembly`
对照组）连同其 RemoteExecutor 宿主一并 N/A。

---

## 三、计数核对

| 类别 | 数量 | 计入 220？ |
|---|---|---|
| covered（直接映射绿测） | 16 | ✓ |
| adapted（机制重写，语义等价） | 4 | ✓ |
| **小计（非条件方法）** | **20** | **✓（17 Fact + 3 Theory）** |
| ConditionalTheory: KeyedParams | 4 | ✗ |
| ConditionalTheory: RemoteExecutor | 6 | ✗ |
| ConditionalTheory: CollectibleAssembly | 2 | ✗ |
| **小计（ConditionalTheory）** | **12** | **✗（Go 机制墙）** |

- 20 计入二级门槛 220 总账（`threshold_actutil` gate；当前 C-2 总账 = 213，见 `threshold_c2_master_test.go`）。
- 12 由 `TestActUtil_ConditionalTheoryNASummary` 按类审计，**不**加入 20 或 213。

---

_dotnet 源：`DI.Tests/ActivatorUtilitiesTests.cs`（v10.0.9，行 745-1051）。_
_相关：`conformance/threshold_actutil_test.go`（gate 记账）· `activator/activator_impl_redphase_test.go`（红灯脚手架）· Story 6.5 D-1（与 2.6 Spec 层同名文件区分）。_
