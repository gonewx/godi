# 扩展方法与集合操作 Go 差异归档（Story 6.4）

godi 落地 M5 二级门槛中 `ServiceProviderServiceExtensionsTest`(17) + `ServiceCollectionServiceExtensionsTest`(14) + `ServiceCollectionKeyedServiceExtensionsTest`(19) + `ServiceCollectionDescriptorExtensionsTests`(4) + `ServiceCollectionTests`(1) = 55 方法时，因 Go 无 `ISupportRequiredService`、无独立 `IServiceScopeFactory` 可注入服务、无可反射的 `ServiceDescriptor` 属性对象、无 C# 类型擦除 `object` 分支，且 codegen 路径的集合突变是 AD-7 静态标记硬墙——以下差异集中归档。

**诚实门槛（NFR7）红线**：所有 covered/adapted 项在 runtime（语义 oracle）绿 + codegen 差分等价（AD-15，解析扩展）或最终集等价（集合操作结果）。N/A 项有架构锚点，非伪装通过。

## 核心差异

### EXT-D1 ISupportRequiredService — Go 无可选扩展点接口（N/A，3 项）

dotnet `ISupportRequiredService` 是可选接口：provider 实现它时 `GetRequiredService` 直接调其方法（可用于异常透传），否则走 `GetService` + null 检查。godi runtime provider 无此可选扩展点；`GetRequired[T]` 直接实现「命中返实例 / 缺失 panic(error)」两段逻辑。

影响：SPSE-02 / SPSE-04 / SPSE-06（3 项 N/A）。SPSE-04 附带的 `RankException` 透传语义通过 godi 的 panic/error 值透传 adapted 覆盖。

### EXT-D2 IServiceScopeFactory — godi 无独立可注入服务（N/A，1 项）

dotnet `CreateAsyncScope(IServiceScopeFactory)` 接受一个可注入的 `IServiceScopeFactory` 服务。godi 用 `Provider.CreateScope()` 接口方法替代（AD-16 七方法面含 `CreateScope`），无独立 `IServiceScopeFactory` 可注入服务。延续 Story 2-5 D-2 + deferred-work.md 行 220 终裁。

影响：SPSE-17（1 项 N/A）。`CreateAsyncScope(Provider)` 入口已由 Story 6-3 落地（`async.go:75`）。

### EXT-D3 ServiceDescriptor 属性反射 → 可观测解析（adapted）

dotnet 测试断言 `descriptor.ImplementationType` / `descriptor.Lifetime` / `descriptor.ServiceKey` 等属性。godi runtime 吃 form-B 闭包 `func(Provider) I`，无可反射的 `ServiceDescriptor` 属性对象。

adapted 策略：descriptor 属性断言改为**可观测解析结果**断言：
- `ImplementationType` → `GetService` 返回值的 `reflect.TypeOf` 动态类型
- `Lifetime` → 单例复用 / 瞬态新建 / 作用域行为可观测
- `ServiceKey` → `IsKeyedService` / `GetKeyedService` 可观测

符合 R4「禁指针 identity 跨 provider」与既有 conformance probe 哲学。

### EXT-D4 ServiceDescriptor.ToString — godi 无可格式化对象（N/A，1 项）

dotnet `ServiceDescriptor.ToString()` 格式化输出注册信息。godi 无公开的 `ServiceDescriptor` 对象（方案 γ：descriptor 是 runtime 内部类型）。

影响：SCDE-04（1 项 N/A）。

### EXT-D5 C# 类型擦除 object 分支 — Go form-B 返回类型静态已知（adapted）

dotnet `TryAddEnumerable` 在 factory-based descriptor 上 `ImplementationType == typeof(object)` 时抛 `ArgumentException`（工厂退化为 `Func<IServiceProvider, object>`，impl 不可区分）。Go form-B 工厂 `func(Provider) T` 的返回类型 `T` 在编译期静态已知（==ServiceType），无 C# 的 `object` 类型擦除分支。

Go 的 `TryAddEnumerable` dedup 退化为 `(ServiceType, Key)` — 与 `TryAdd` 等价，因为 ImplType 总等于 ServiceType。不可区分校验的**触发条件**不同但**语义等价**（都是"无法区分 impl 时拒绝"，只是 Go 的 form-B 永远"可区分"因为 T 是静态的）。

影响：SCSE-10 / SCKSE-10（2 项 adapted）。

### EXT-D6 集合突变 codegen 硬墙（N/A — 操作本身；结果义务轴）

codegen 路径的 DSL 是 AD-7 纯静态标记（`cmd/godi-gen` 一次性静态扫描所有 `godi.Add*` 调用点），无运行时可变列表。`TryAdd` / `TryAddEnumerable` / `Replace` / `RemoveAll` 是运行时列表突变操作，与静态标记模型根本冲突 — 同性质于开放泛型 `MakeGenericType`（AD-4）/ AnyKey catch-all（AD-17）机制墙。

**codegen 用户**经静态 DSL **直接声明「最终想要的注册集」**（无需突变操作）。两 provider 等价性体现在：runtime 用突变操作得到的**最终注册集** == codegen 静态 DSL 声明的**同一最终集** → 解析结果差分等价（D-8 义务轴判定，测「最终集解析等价」非「突变操作本身」）。

spine 对账：「集合突变 = runtime-only 注册期能力，codegen 静态 DSL 直接声明最终集」。

### EXT-D7 AddSequence — godi 无批量 Add API（adapted，2 项）

dotnet `Add(IEnumerable<ServiceDescriptor>)` 批量添加。godi 无 `ServiceDescriptor` 集合 API；多次 `Add*` 调用顺序追加达到同一效果。

影响：SCSE-11 / SCKSE-11（2 项 adapted）。

### EXT-D8 AnyKey 作注册键 — Go 机制墙（existing N/A，引 AD-4 类比）

`AddKeyed*(AnyKey, ...)` 在 runtime 上 panic（`AnyKeyRegistrationError`，`keyed.go:33` 已存在）。dotnet 的 AnyKey catch-all 注册（一个服务被 ANY 具体键解析时，工厂接收 resolve-time key）在 Go 无机制——form-B `func(Provider) I` / form-A `New{Impl}` 无法接收 resolve-time key。同性质于 AD-4 开放泛型机制墙。

AnyKey 查询三态不变（AD-17 Story 3-4 已固化）：
- `GetKeyedService(AnyKey)` → panic
- `GetKeyedServices(AnyKey)` → catch-all 全部键控
- `IsKeyedService(AnyKey)` → false 不 panic
