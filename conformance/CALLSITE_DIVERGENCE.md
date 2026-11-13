# CallSite 构建细节 Go 差异归档（Story 6.2）

godi 与 dotnet `Microsoft.Extensions.DependencyInjection` 在 **CallSite 构建**上存在根本架构差异。`CallSiteFactoryTest`（34 方法）+ `CallSiteTests`（13 方法）**不是黑盒行为测试，而是 dotnet 内部 CallSite IR 的白盒测试**——它们直接断言一棵 `ServiceCallSite` 树的形状、缓存位置、槽索引、以及表达式编译产物。godi 按 **AD-1 根本没有这层 IR**，因此本文档集中归档这些白盒断言为何 N/A，供二级门槛审计（CAP-7 靶）和未来维护参考。

**诚实门槛（NFR7）红线**：godi **绝不**为让白盒测试"通过"而引入运行时 CallSite/slot/cache 表征。N/A = 有架构理由的 gate 条目（`threshold_csf_test.go`），不是伪造的假通过测试。

## 核心架构论点

dotnet 的 `CallSiteFactory` 在解析前构建一棵 `ServiceCallSite` 树作为运行时 IR：

- `ConstructorCallSite` / `IEnumerableCallSite` / `ConstantCallSite` / `FactoryCallSite` 是 IR 节点类型
- `CallSiteResultCacheLocation`（Dispose/Scope/Root/None）编码生命周期缓存位置
- `GetSlot()` 提供反向槽索引
- `ExpressionResolverBuilder` 把 CallSite 树编译成委托（`CompileCallSite`）

godi 无这一整层（**AD-1**）：

- **B2 codegen** 直接把依赖图 emit 成 Go 代码（字段/方法），无中间 IR
- **runtime provider** 是 reflect-free 闭包解释器，无 CallSite 树
- **生命周期由「实例存放位置」编码**（单例=Root 字段、作用域=Scope 字段、瞬态=`New*()` 方法），而非缓存标志
- 只有 **Runtime 引擎**（AD-6）——无表达式编译引擎

## CallSite 层差异归档表

| 差异 | dotnet 机制 | godi 等价 | 影响的方法 | N/A 类别（`threshold_csf_test.go`） |
|---|---|---|---|---|
| **CallSite 树 IR 缺失** | `ServiceCallSite` 树（节点形状可断言） | 无——B2 直接 emit / runtime 闭包（AD-1） | 所有白盒「断言 CallSite 树形状」的部分 | 白盒断言 N/A，可观测行为 adapted |
| **缓存位置标志缺失** | `CallSiteResultCacheLocation`（None/Scope/Root） | 生命周期 = 字段位置（AD-1），无缓存标志 | CSF-30（`EnumberableCachedAtLowestLevel`） | CallSite/slot/cache IR |
| **槽索引缺失** | `GetSlot()` 反向索引 | 无 slot 表征；注册序由集合解析可观测 | CSF-26（`GetSlotTests`） | CallSite/slot/cache IR |
| **表达式编译引擎缺失** | `ExpressionResolverBuilder` / `CompileCallSite` | 无；只有 Runtime 引擎（AD-6） | CallSiteTests 全部 `Compile*` 部分 | 编译引擎 N/A，可观测解析结果 adapted |
| **约束矩阵差异** | 运行时约束反射过滤 | per-helper 编译期物化（Go 约束种类 ≠ dotnet） | CSF-22 | adapted（可观测）+ 精确矩阵 N/A（本文档 §约束矩阵） |
| **best-match 多构造** | `CallSiteFactory` 多构造选择/歧义检测 | activator 域（AD-9），容器路径单 `New{Impl}` | CSF-06/23/27/28/29 | best-match activator 域 |
| **trimming/NativeAOT 回退** | `DynamicallyAccessedMembers` 注解 + 动态代码回退 | godi 全程无反射构造（AD-13），trim-safe/AOT-safe 是常态 | CSF-C3/C4 | trimming/NativeAOT 常态化 |
| **并发 CallSite 唯一性** | 并发构建 CallSite 唯一性（内部锁） | 只有有限的并发契约（见 README「已知限制」并发条目）+ 无 CallSite IR | CSF-C1/C2 | NFR8 concurrency + CallSite IR |
| **可选/默认 ctor 参数** | 可选参数默认值 CallSite | Go 无可选/默认 ctor 参数 | CSF-24, CST-09 | 无可选/默认参数 |
| **dotnet 已知 bug（禁用）** | `[ActiveIssue #57333]` 禁用测试 | godi slot-order 确定性（Story 4.3），无此 bug | CST-10 | dotnet known-bug disabled |

## 差异分类

### 类别 A：CallSite IR 缺失（白盒断言 N/A 的根因）

**CSF-26（GetSlot）、CSF-30（Cache.Location）、CallSiteTests 的表达式编译部分** — 这些断言 dotnet 内部 IR 的形状/缓存/槽。godi 无 IR（AD-1）：生命周期由字段位置编码，注册序由集合解析可观测，解析结果由 provider 直接产出。**可观测行为**（解析结果、生命周期 identity、集合序）已由 Epic 2/3/4 交付并在 `spec_callsite_redphase_test.go` 确认；**白盒 IR 断言**是 N/A。

### 类别 B：约束矩阵差异（CSF-22，必须显式归档）

dotnet `CreateCallSite_ReturnsMatchingTypesThatMatchCorrectConstraints` 注册全部约束 impl，解析 `IEnumerable<IFakeOpenGenericService<int>>`，运行时过滤出「int 满足其约束」的子集。**godi 无法 1:1 复现此矩阵**，因为 Go 约束种类与 dotnet 不对应：

| dotnet 约束 | int | string | godi 约束 | int | string |
|---|---|---|---|---|---|
| `struct` | ✓ | ✗（引用类型） | `comparable` | ✓ | ✓ |
| `class` | ✗ | ✓ | `any` | ✓ | ✓ |
| `new()` | ✓ | ✗ | `any` | ✓ | ✓ |
| `IComparable<T>` | ✓ | ✓ | `interface{ CompareTo(T) int }` | ✗（int 无 CompareTo） | ✗ |

**结论**：godi 的「过滤」是**开发者编译期选择哪些 `Add{Svc}[T]` 调用能编译**，不是运行时约束反射。

- **可观测层面**（集合只含已物化封闭类型、按注册序）→ **adapted**（`spec_callsite: CSF-21/22`，`IFakeOpenGenericService[int]` 集合含 struct/class/new/no-constraint 四个 impl，self-ref 因 int 无 `CompareTo(int) int` 被编译期排除）
- **dotnet 精确约束矩阵**（哪个约束接受哪个类型）→ **N/A**（Go 无对应运行时过滤，AD-10）

### 类别 C：best-match 多构造 = activator 域（AD-9）

**CSF-06/23/27/28/29** 测多构造 best-match / 歧义检测。按 **AD-9**：因 `New{Impl}` 约定=每类型唯一构造器，dotnet 的 best-match 多构造选择在容器/装配路径上消失，只存在于 `ActivatorUtilities`。godi 容器路径（form-A）每类型恰好一个 `New{Impl}`，根本不存在多构造选择。这些语义归 `activator`（Story 2.6 已覆盖 Spec 层 20 方法，Story 6.5 覆盖 impl 层）。

### 类别 D：trimming/NativeAOT 常态化（AD-13）

**CSF-C3（`VerifyOpenGenericTrimmabilityChecks`）/ C4（`VerifyDynamicCodeNotSupportedChecks`）** 测 dotnet 的 trimming 注解与 NativeAOT 无动态代码回退。godi codegen 本质 reflection-free/trim-safe（AD-13）——**dotnet 需要特殊回退的场景正是 godi 的常态**（全程无动态代码构造）。故这些机制在 godi 不存在=常态，标 N/A。

## CAP-7 二级靶落地（Story 6.2 新增消费）

`CallSiteFactoryTest` 是 acceptance-contract 点名的 **M4 二级靶**（CAP-7）。真正的新增工作是消费 Story 4.2 已移植但「无 SpecCase」的 5 个结构性约束 fakes（`conformance/fakes/fakes_opengeneric_constrained.go`）：

- **约束满足场景**（CSF-08/10/12/19）→ 封闭物化解析 SpecCase（`spec_callsite: CSF-08/10/12/19`）
- **约束不满足场景**（CSF-07/09/11/13/15/17/18）→ **在 Go 中无法书写**（编译错误）= N/A 的活证据，不写测试
- **codegen 覆盖（AD-15）**：`ClassWithSelfReferencingConstraint`（唯一有 form-A 构造器的结构性 fake）在 codegen 路径物化（`registration.go`）+ cross-provider 差分等价（`TestCallSite_SelfRef_DifferentialEquivalence`），闭合 6.1 runtime-only 教训；其余 4 个无构造器的结构性 fake 保持 runtime-only（架构决定，非覆盖缺口）

## 引用

- ARCHITECTURE-SPINE.md AD-1：B2 codegen 直接装配，生命周期由字段位置编码（无 CallSite IR 根据）
- ARCHITECTURE-SPINE.md AD-6：只取 Runtime 引擎对应项（表达式编译引擎 N/A）
- ARCHITECTURE-SPINE.md AD-9：best-match 多构造选择只存在于 activator，容器路径单 `New{Impl}`
- ARCHITECTURE-SPINE.md AD-10：编译期约束满足（约束矩阵差异根据）
- ARCHITECTURE-SPINE.md AD-11/AD-13：`types.Implements(io.Closer)` 门 + 主路径零反射式构造（trim-safe）
- ARCHITECTURE-SPINE.md AD-15：开放泛型封闭物化覆盖义务（codegen 场景 + 差分等价）
- `conformance/threshold_csf_test.go`：47 方法覆盖率 gate + 7 类 N/A 分类
- `conformance/spec_callsite_redphase_test.go`：结构性约束消费 + CallSiteTests 可观测 SpecCase
- `conformance/OPEN_GENERIC_DIVERGENCE.md`：9 接触点约束语义差异（本文档追加 CallSite-specific 段落）
- deferred-work.md 行 77：5 结构性约束 fakes 无 SpecCase，consumed by Epic 6 CallSiteFactoryTest（本 Story 消化）
