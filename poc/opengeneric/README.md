# PoC：开放泛型注册（Tier 3）—— 显式封闭注册 vs Codegen

针对研究报告中唯一**无法运行时 1:1 复刻**的 dotnet DI 机制 —— 开放泛型注册
`services.AddSingleton(typeof(IRepository<>), typeof(Repository<>))` —— 用最小可运行代码
对比两条 Go 落地路线，并以测试**证明** Go 不存在运行时等价物。

> 结论先行：**功能可达、机制不可达**。Go 能解析任何在源码中"出现过"的封闭类型，但**无法**解析编译期未知的封闭类型（dotnet 靠运行时 `MakeGenericType` 可以）。Go 1.27 **不改变**这一点（它没有动态类型创建 / 运行时泛型实例化）。

## 运行

```bash
cd poc/opengeneric
GOWORK=off go test ./... -v        # 4 个用例：路线A / B1 / B2 / 限制验证
GOWORK=off go run ./wiring/gen      # 重新生成 registrations_gen.go（codegen 路线的"编译器"）
GOWORK=off go test -bench . -benchmem -run '^$' ./wiring
```

> 用 `GOWORK=off` 是因为父级 `go.work` 引用的 `./godi` 尚未初始化 `go.mod`，本 PoC 为自包含独立模块，刻意与工作区隔离。

## 场景

`domain` 包定义开放泛型契约 `IRepository[T]` 与实现 `Repository[T]`（构造依赖一个非泛型
`Logger`，用于演示 Tier-1 构造注入与开放泛型组合）。三个元素类型：`User`、`Order`（注册/生成），
`Product`（**故意不注册**，用于限制验证）。

## 三条路线

| 文件 | 路线 | 谁来枚举闭合类型 | 运行时反射 |
|---|---|---|---|
| `wiring/routea_test.go` | **A 显式封闭注册** | 人工手写 `AddRepository[User](c)` | 有（容器解析） |
| `wiring/routeb_test.go`（B1） | **B1 codegen + 反射容器** | `manifest.txt` → 生成 `RegisterGeneratedRepositories` | 有（容器解析） |
| `wiring/routeb_test.go`（B2） | **B2 codegen + 直接装配**（wire 风格） | `manifest.txt` → 生成 `NewAppRepositories` | **无** |

`wiring/gen/main.go` 是 codegen 路线的生成器：读 `manifest.txt`（Go 版"单一事实源"，对应
dotnet 的一处开放泛型注册），`go fmt` 后写出 `registrations_gen.go`。已验证生成产物**可完全复现**。

## 对比结论（含实测）

| 维度 | A 显式 | B1 codegen+容器 | B2 codegen+直接 | dotnet 开放泛型（参照） |
|---|---|---|---|---|
| 单一事实源 | ❌ 分散在注册代码 | ✅ `manifest.txt` | ✅ `manifest.txt` | ✅ `typeof(Repo<>)` 一处 |
| 样板代码 | 每类型一行（人工） | 每类型一行（自动） | 每类型一行（自动） | 零 |
| 运行时反射 | 有 | 有 | **无** | 有 |
| 解析性能（本机实测） | 15.5 ns/op · 0 allocs | 15.5 ns/op · 0 allocs | **0.10 ns/op** | （JIT/IL，未测） |
| 类型安全 | 编译期 | 编译期 | 编译期 | 运行时（可能抛 `IL2055`/`InvalidOperation`） |
| AOT / 裁剪友好 | 好 | 好 | **最好** | 需 trim 注解 |
| 构建步骤 | 无 | `go generate` | `go generate` | 无 |
| **解析编译期未知的闭合类型** | ❌ | ❌ | ❌ | ✅（唯一真差距） |

## 关键证据：Tier-3 硬墙

`wiring/limitation_test.go` 中 `IRepository[Product]` 解析**失败**（`Product` 既没手写注册也不在
manifest）。原因不是 API 设计，而是语言层面：Go **没有**开放的 `Repository[T]` 运行时类型句柄可供实例化，
`reflect` 无 `MakeGenericType` 等价物，`go/types.Instantiate` 仅编译期。**封闭类型必须在源码中出现**
（手写或生成）。

## 对 godi 的建议

- **默认 API 取路线 A**：最 Go 化、零构建步骤、编译期安全；把"开放泛型注册"重定义为
  `godi.AddRepository[User](c)` 这类**泛型注册辅助函数**。
- **路线 B（codegen）作为可选增强**：闭合类型多时用 `manifest.txt` + `go generate` 消除样板；
  其中 **B2 直接装配**面向 AOT / 性能极致场景（零反射、~150× 更快，但 15.5ns 对绝大多数 DI 已可忽略）。
- **文档须明确**：godi 不提供"注册一次开放泛型、解析任意未知闭合类型"的 dotnet 式能力，这是 Go 理念下的
  自觉取舍，而非缺陷。
