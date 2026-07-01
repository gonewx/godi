# 异步作用域与异步释放 Go 差异归档（Story 6.3）

godi 落地 CAP-3 的异步作用域 intent（AR12，spine Deferred 行 250 的 **M5/C-2 revisit 点**）时，移植 `AsyncServiceScopeTests`（6 Fact）+ `ServiceProviderContainerTests` 的 12 个 async 方法（SPCT #17,24,25,27,28,29,31,32,33,35,38 + C1）。因为 Go 无 `ValueTask`/`async`/`await`，且方法名唯一性约束，这些语料在 godi 有一组**故意的惯用改写**——本文档集中归档，供二级门槛审计与未来维护参考。

**诚实门槛（NFR7）红线**：所有 adapted 改写都在 **runtime + codegen 双 provider** 上跑绿（`disposer_async_redphase_test.go`、`provider_async_redphase_test.go`、`spec_async_dispose_redphase_test.go`、`conformance/codegen/gen_async_dispose_redphase_test.go`、`async_scope_redphase_test.go`），并升 **Path γ**（预生成 B2 容器严格跑异步 LIFO）+ cross-provider 差分等价（AD-15）。adapted ≠ 假通过。

## 核心差异

### D-1 Go 无 ValueTask/async → `CloseAsync(ctx)` 语义（可观测 adapted）

dotnet `IAsyncDisposable.DisposeAsync()` 返回 `ValueTask`，其价值是「不阻塞线程地 await」。Go 无 async/await，等价价值 = **接受 `context` 的、可取消 / 可超时的优雅释放**。故 `godi.AsyncDisposable = interface { CloseAsync(ctx context.Context) error }`——`ctx` 参数正是这个 Go 版本的意义所在。这是**故意的 Go 惯用改写**（可观测的异步释放语义忠实：LIFO、prefer-async、首错继续、dispose-once），非缺陷。

> 注：绑定语料的 fake `DisposeAsync` 立即完成，**未触发** `ctx` 取消 / 超时路径。真实 I/O 取消语义随真实异步资源出现时 revisit（见 deferred-work.md「Deferred from: dev of 6-3」）。

### D-2 方法名墙：`Close(ctx)` 候选被否

spine Deferred 行 250 提的候选是 `AsyncDisposable interface { Close(ctx) error }`。**不可行**——Go 机制墙：一个 Go 类型**不能**同时有 `Close() error`（`io.Closer`）与 `Close(ctx) error`（候选），方法名必须唯一。而 dotnet `SyncAsyncDisposable`（同时实现 `IDisposable`+`IAsyncDisposable`）与 prefer-async 语料（#25/29/33）**要求**一个类型同时具备同步与异步释放；若异步方法也叫 `Close`，该类型无法书写 → 整段 prefer-async 语料无法忠实移植。**拍定**：异步方法取独立名 `CloseAsync`，与 `io.Closer` 对称（both 服务两者都实现，方法名不冲突）。

### D-3 `InvalidOperationException` → `error`

dotnet 同步 `Dispose` 遇仅实现 `IAsyncDisposable` 的服务抛 `InvalidOperationException("'{0}' type only implements IAsyncDisposable. Use DisposeAsync to dispose the container.")`。Go 无异常，返 `error`（含服务类型名 `%T` + "use CloseAsync" 指引，`disposer.go` `CloseAll`）。在「首错保留继续」策略下，该错误作为首错返回但**不阻断**其余项释放（AC3）。两 provider（runtime `Disposer.CloseAll`、codegen 生成 `Close` 调同一 `Disposer`）口径一致。

### D-4 结构性 opt-in 接口——不改 AD-16 七方法面

dotnet：`IServiceScope : IDisposable`（同步 `Dispose` 在接口内），`IAsyncDisposable`（`DisposeAsync`）是**独立接口**，scope 对象同时实现两者。godi 对称：`Scope`（含 `Close()`）不变；`godi.AsyncDisposable`（`CloseAsync`）是**独立结构性接口**；root/scope/生成容器同时实现两者（加 `CloseAsync` 方法即可，接口声明不变）。异步能力经**类型断言** `x.(godi.AsyncDisposable)` 发现（Tier B，AD-13 合规——判类型不构造）。真容器结构性实现它、假 scope 可不实现（`AsyncServiceScopeTests` test 4 的回退分支据此可测）。**AD-16 钉死的七方法面零改动**（见 spine AD-16 修订记录 Story 6-3）。

### D-5 dispose-once 跨 `Close`/`CloseAsync` 共享守卫

dotnet `BeginDispose` 单一 `_disposed` 让 `Dispose`/`DisposeAsync` **共享**一次性语义（#38）。godi 让 root/scope 的 `Close` 与 `CloseAsync` 共享同一 `closed` 守卫（**first-caller-wins**：先到者释放并缓存 err，另一个 no-op 返缓存 err）。runtime 从 `sync.Once` 收敛为共享 `closed` + `closeMu`（NFR8 许可的调整——`sync.Once` 绑定单一 func，无法让 `Close`/`CloseAsync` 共享；Story 2.3 的 `TestRT_CloseIdempotent_WithDisposables`/`TestRT_ScopeClose_SyncOnceGuard` 断言的是行为而非机制，仍绿）；codegen 用纯 `closed`（NFR8 单 goroutine），两 provider 趋同。

## 处置的 dotnet 方法（永久 N/A）

### SPCT-C1 `GetAsyncService_DisposeAsyncOnSameThread_...` — threading N/A（NFR8）

`ConditionalFact`，测同线程 `DisposeAsync` 不 hang + `DisposeAsync` 被调。其**异步释放语义**已由 SPCT-24..38 的 6-3 移植覆盖（`threshold_spct_test.go` na→adapted）；剩余的**同线程 / 并发 threading 保证**属 NFR8（本期不保证并发安全），**永久 N/A（C-2）**。

### `SafelyDisposeNestedProviderReferences` / `SelfResolveThenDispose` — Go 差异永久 N/A

两者涉及「服务注入 Provider 引用的安全释放」与「解析 Provider 自身后释放」。godi **无「Provider 作为可解析服务」**（`Provider` 接口方法非自注册服务，Story 2-5 D-2；provider 自注入是 2.2 deferred 同源）。故两者标 **N/A（Go 差异，永久）**——本 Story **不实现** provider 自注入（消化 deferred-work.md 行 187 的「go-divergence → Epic 6 视需要」为最终决策）。

## codegen 侧架构记录：异步 fakes 为何 Transient

conformance 共享容器（`codegen.NewRoot()`）的异步 fakes 注册为 **Transient** 而非 Scoped：codegen **eager 物化**每个 scoped 服务（`CreateScope` 内），一个仅异步的 scoped 服务会让共享容器的**每次 sync `scope.Close()` 报错**（AC3「仅异步→报错」），从而破坏 Story 2.3 的 sync dispose 套件（AC6 零回退红线）。Transient 保持 **lazy**（仅显式解析时 `track`），既守 2.3 零回退，又完整验证 scope 级异步释放（从 scope 解析 + `scope.CloseAsync` 释放该 scope 追踪的服务）。runtime spec oracle 用 scoped；受测轴（异步释放 / prefer-async / LIFO / sync-errors / dispose-once）在两 provider observable 一致（AD-15 差分等价 + Path γ）。

## 引用

- ARCHITECTURE-SPINE.md AD-16（Provider 七方法面 + **修订记录 Story 6-3**：异步释放 = 结构性 opt-in，非七方法面变更）
- ARCHITECTURE-SPINE.md Deferred 行 250（异步释放接口候选 `Close(ctx)`——本 Story revisit 落地，方法名墙否决候选、改 `CloseAsync`）
- ARCHITECTURE-SPINE.md AD-11/12/13/15（Disposable `types.Implements` 静态判 + Tier B 断言 + 覆盖义务 / Path γ）
- Story 6.3 Dev Notes D-1..D-9（`_bmad-output/implementation-artifacts/6-3-async-scope-dispose.md`）
- dotnet `.../Abstractions/src/AsyncServiceScope.cs`、`.../src/ServiceLookup/ServiceProviderEngineScope.cs`、`Strings.resx` 行 162-163
- `conformance/threshold_async_test.go`（AsyncServiceScopeTests 6 覆盖 gate）、`conformance/threshold_spct_test.go`（12 async 项 na→adapted 重分类）
- deferred-work.md 行 32（12 async 项）+ 行 187（nested/self-resolve）——本 Story 消化
