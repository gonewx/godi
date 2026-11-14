# godi 设计与实现评估报告

- 评估日期：2026-11-13
- 评估对象：`main` 分支，提交 `cd4ffcd`
- 性质：内部评审产物，中文单语，不属于 `docs/en` / `docs/zh` 双语文档集，未加入索引

## 1. 结论

基线是健康的：`go build`、`go vet`、`go test -race ./...` 全部通过（10 个包）。form-A（`New{Impl}` 构造器发现）这条主路径的校验是可靠的，缺失依赖、captive、循环都能在生成期拦住，输出也是确定性的。

但项目目前还不适合直接用于生产，原因有三类：

1. **生成的容器会静默接错线。** 在几种很普通的注册写法下，生成器退出码为 0、产物能编译，运行时却注入 nil、漏掉释放，或者返回与注册意图不符的实现。共 7 条，全部已复现。
2. **README 的核心承诺对 form-B 不成立。** “非法图无法构建”只对 form-A 和“内联闭包 + `godi.Get*`”成立；具名函数工厂、keyed 工厂会绕过缺失依赖和 captive 检查。
3. **生成器会产出不可编译的代码，并且会被自己的旧产物卡死。** 前者有 7 种形态；后者意味着给任何已注册的构造函数加一个参数后，`go generate` 就拒绝运行，必须手工删除 `container_gen.go`。

这些问题有共同的设计根因，见第 5 节。最主要的一条是：容器采用“字段急切接线”，它要求生成器掌握完整的静态依赖顺序，而 form-B 工厂对生成器是不透明的。

## 2. 方法与范围

- 通读了全部非测试源码：根包、`runtime`、`plan`、`internal/genlib`、`cmd/godi-gen`、`activator`。
- 对照了 README、`docs/en/architecture.md`、`skill/SKILL.md`、`conformance/*_DIVERGENCE.md`，以及本机的 `_bmad-output/implementation-artifacts/deferred-work.md`。
- 每条疑点都在仓库外的独立 module 里做了复现（`replace` 指向本仓库，`GOWORK=off`），流程是“`godi-gen` 生成 → `go build` → 运行测试”。仓库本身没有被改动。
- `activator` 包读过，没有发现值得列出的问题。

每条发现标注两个属性：

- **状态**：「已复现」表示有实际运行结果；「代码推断」表示只读了代码。
- **已知性**：「内部已知」表示 `deferred-work.md` 里有记录。注意 `_bmad-output/` 在 `.gitignore` 里，仓库读者看不到这份清单。「新」表示在上述所有文档里都没有找到。

## 3. 发现总表

| 编号 | 发现 | 路径 | 状态 | 已知性 |
| --- | --- | --- | --- | --- |
| A1 | scoped 的拓扑排序不处理 `[]T` 和经 transient 的间接依赖，注入 nil | codegen | 已复现 | 新 |
| A2 | 具名函数工厂依赖后注册的单例，得到 typed-nil | codegen | 已复现 | 新 |
| A3 | form-B 工厂产出的 `io.Closer` 从不被释放 | codegen | 已复现 | 新，且与 `skill/SKILL.md:274` 矛盾 |
| A4 | 同一服务类型混合生命周期：不是 last-wins，`GetServices` 丢注册或乱序 | 两者 | 已复现 | 内部已知（106、228 行） |
| A5 | 工厂闭包捕获局部变量：通常编译失败，有同名包级标识符时静默改绑 | codegen | 已复现 | 新 |
| A6 | 注册顺序 = 文件名 + 偏移，不是调用顺序，影响 last-wins | codegen | 已复现 | 新 |
| A7 | 多 goroutine 共享 Root 解析可释放 transient，约一半实例丢失追踪 | 两者 | 已复现 | 内部已知（NFR8、209 行），用户文档未披露 |
| B1 | form-B 请求未注册的服务，生成期不报错，`NewRoot` panic | codegen | 已复现 | 新，与 README 承诺矛盾 |
| B2 | unkeyed 工厂解析 keyed 单例，`NewRoot` panic | codegen | 已复现 | 新 |
| B3 | 具名函数工厂、keyed 工厂绕过 captive 检查 | codegen | 已复现 | 新，与 README 承诺矛盾 |
| B4 | 每次 `CreateScope` 急切构造全部 scoped；无关服务的构造错误直接 panic | codegen | 已复现 | 仅在 `ASYNC_DIVERGENCE.md:43` 顺带提到 |
| C1 | 原地生成布局下，旧产物类型检查失败会阻止重新生成 | 生成器 | 已复现 | 新 |
| C2 | 类型实参被推断掉的 form-B 标记被跳过，退出码仍为 0 | 生成器 | 已复现 | 内部已知（304 行） |
| D | 生成器产出不可编译代码，共 7 种形态 | 生成器 | 已复现 | 新（6 种）；`NewRoot` 泄漏另见 201 行 |
| E1 | `IsKeyedService` 传不可哈希 key 会 panic，违反“永不 panic”契约 | runtime | 已复现 | 新 |
| E2 | `Build()` 按 map 迭代序构造单例，释放顺序每次运行不同 | runtime | 已复现 | 内部已知（185、262 行） |
| E3 | scope 路径和 transient 的循环依赖是不可恢复的栈溢出 | runtime | 已复现 | 内部已知（315 行），`architecture.md:60` 未加限定 |
| E4 | 可释放对象的 `Close` 里回调 `provider.Close()` 会自锁 | runtime | 已复现 | 新 |
| F | 文档和注释过期、互相矛盾 | 文档 | 已核对 | — |

## 4. 详细发现

### 4.1 A 组：生成的容器静默出错

这一组最严重，因为没有任何报错：生成成功、编译通过、测试如果不专门断言就发现不了。

#### A1. scoped 拓扑排序比 singleton 的弱

`SingletonsTopo`（`plan/registry.go:175`）会展开 `[]T` 依赖，也会穿透非单例依赖继续找。`ScopedTopo`（`:239`）、`KeyedSingletonsTopo`（`:310`）、`KeyedScopedTopo`（`:346`）都没有这两个处理。结果是 scoped 消费者可能排在它依赖的 scoped 之前接线。

复现一，`[]T` 依赖：

```go
godi.AddScoped[IConsumer, Consumer](&r) // NewConsumer(items []IPlugin)
godi.AddScoped[IPlugin, P1](&r)
godi.AddScoped[IPlugin, P2](&r)
```

`Consumer` 拿到的切片长度是 2，两个元素都是 nil。scope 和 root 上都一样。

复现二，经 transient 间接依赖：

```go
godi.AddScoped[IA, A](&r)     // NewA(t IT)
godi.AddTransient[IT, T](&r)  // NewT(b IB)
godi.AddScoped[IB, B](&r)
```

`A` 持有的那个 `T`，其 `IB` 是 nil；事后单独解析的 `T` 则是正常的。

把被依赖的服务挪到前面注册就能规避，但没有任何提示告诉用户需要这样做。

#### A2. 具名函数工厂得到 typed-nil

`factoryDeps`（`cmd/godi-gen/main.go:1471`）只扫描工厂**表达式文本**里的 `godi.Get` / `GetRequired` / `GetServices` 调用。工厂如果是一个函数名，表达式里什么都扫不到，也就没有排序约束。

```go
func newFoo(p godi.Provider) IFoo { return &Foo{cfg: godi.GetRequired[*Cfg](p)} }

godi.AddSingletonFactory[IFoo](&r, newFoo)
godi.AddSingleton[*Cfg, Cfg](&r)
```

`NewRoot` 成功返回，`Foo.cfg` 是 nil 指针，而直接从 root 取 `*Cfg` 是正常的。原因是 `newFoo` 运行时 `r.f_Cfg` 还没赋值；服务类型是指针时，nil 指针装进 `any` 不等于 nil，于是 `GetRequired` 认为命中。

工厂里直接调 `p.GetService(...)` 属于同一类情况（代码推断）。

#### A3. form-B 产出的可释放对象从不被释放

`buildRegistry` 只在 form-A 分支设置 `Disposable` / `AsyncDisposable`（`main.go:430-431`）；form-B 分支在 `main.go:329-330` 就 `continue` 了。

```go
godi.AddSingletonFactory[IBar](&r, func(p godi.Provider) IBar { return &Bar{} }) // Bar 实现 io.Closer
godi.AddScopedFactory[IFoo](&r, func(p godi.Provider) IFoo { return NewFoo() })   // Foo 实现 io.Closer
```

| provider | scope.Close + root.Close 之后 `Close` 被调用的情况 |
| --- | --- |
| 生成容器 | 一次都没有 |
| runtime | 两个都被调用 |

`skill/SKILL.md:274` 写的是“实现 `io.Closer` 的服务会被自动追踪和释放”，没有任何例外说明。README 的 Quick Start 也恰好用 form-B 注册了 `Logger`。

#### A4. 同一服务类型混合生命周期

两个 provider 都按“singleton → scoped → transient”的固定优先级查找（`runtime/provider.go:120-152`、`generate.go:512`），而不是按注册顺序。

```go
godi.AddSingleton[I, S](&r)   // 先
godi.AddTransient[I, T](&r)   // 后
godi.AddTransient[J, JT](&r)  // 先
godi.AddSingleton[J, JS](&r)  // 后
```

| | `GetService[I]` | `GetServices[I]` | `GetServices[J]` |
| --- | --- | --- | --- |
| dotnet 语义 | T（后注册者） | [S, T] | [JT, JS] |
| 生成容器 | S | [S, T] | [JS, JT]（顺序反了） |
| runtime | S | [S]（T 丢失） | [JS]（JT 丢失） |

runtime 的 `getUnkeyedServices`（`provider.go:272`）命中 singleton 组就直接返回。`conformance/codegen/registration.go` 的注释自己承认这是“未处理的轴”，并用全 transient 规避。问题在于它既没实现也没被拒绝，而是静默返回错误结果，而且作为 oracle 的 runtime 比生成容器错得更多。

#### A5. 工厂闭包捕获局部变量

form-B 工厂是按源码文本内联进 `NewRoot` 的，闭包捕获的局部变量在那里不存在。

```go
func register(dsn string) {
    var r godi.Registry
    godi.AddSingletonFactory[IFoo](&r, func(p godi.Provider) IFoo { return &Foo{name: dsn} })
}
```

- 通常情况：生成器退出码 0，产物报 `undefined: dsn`。
- 包里恰好有同名的包级变量 `var dsn = "..."` 时：产物能编译，工厂静默用了包级变量的值。

这背后是一个更大的设计缺口：没有任何途径把运行时的值送进容器，见 5.3。

#### A6. 注册顺序由文件名决定

`collectRegistrations` 按“文件名、偏移”排序（`main.go:508-520`），并且遍历整个包的 AST，不看调用关系和控制流。

```go
// z_main.go
func register() {
    var r godi.Registry
    godi.AddSingleton[I, Default](&r)
    registerOverrides(&r) // 意图：覆盖默认实现
}

// a_extra.go
func registerOverrides(r *godi.Registry) { godi.AddSingleton[I, Override](r) }
```

`GetService[I]` 返回的是 `Default`，因为 `a_extra.go` 排在前面。同一文件里，定义在 `register()` 上方的辅助函数也有同样的问题。`if` 分支里的标记一律生效，`Registry` 实参也被忽略：一个包里所有标记都进同一个容器。

#### A7. 并发下丢失释放追踪

`Disposer.Track` 是无锁的 `append`（`runtime/disposer.go:35`），生成容器和 runtime 都用它。

8 个 goroutine 共享一个生成的 `Root`，各解析 500 次可释放的 transient：

- 一次运行里创建了 4000 个实例，`Disposer` 只追踪到 1950 个，其余在 `Close` 时不会被释放。
- `-race` 报告 DATA RACE。

需要准确描述范围：生成容器的 singleton 和 scoped 是急切构造的，并发读是安全的；有问题的是可释放 transient 的追踪，以及 `closed` 标志的无同步读写（后者是代码推断）。runtime 另外还有惰性 scoped 的 map 写竞态（内部已知，239 行）。

规划文档把它定为 NFR8“本期不保证并发安全”，`CALLSITE_DIVERGENCE.md` 和 `ASYNC_DIVERGENCE.md` 也提到了 NFR8。但 README、`docs/en`、`skill/` 里没有一处告诉用户容器不能跨 goroutine 共享。对一个 DI 容器来说，这是使用者默认会假设成立的性质。

### 4.2 B 组：运行期 panic 与校验缺口

README 的 Features 表写着：循环、captive、缺失注册都会在 `go generate` 失败。实测结果如下。

| 写法 | 缺失依赖 | captive |
| --- | --- | --- |
| form-A | 生成期拦住 | 生成期拦住 |
| form-B，内联闭包 + `godi.Get*` | **放行**（B1） | 生成期拦住 |
| form-B，具名函数 | 放行 | **放行**（B3） |
| keyed form-B，内联闭包 | 放行 | **放行**（B3） |

#### B1. form-B 的缺失依赖不被检查

```go
godi.AddSingletonFactory[IFoo](&r, func(p godi.Provider) IFoo {
    return &Foo{d: godi.GetRequired[IDep](p)} // IDep 从未注册
})
```

生成器退出码 0。`NewRoot()` panic：`no service registered for type IDep`。注意是 panic，不是 `NewRoot` 返回的 `error`。`factoryDeps` 对解析不到的类型直接忽略（`main.go:1508`），这本来是为 `GetServices` 的空集合设计的，但 `Get` / `GetRequired` 也走了同一分支。

#### B2. unkeyed 工厂解析 keyed 单例

```go
godi.AddKeyedSingleton[IDep, Dep](&r, "k")
godi.AddSingletonFactory[IFoo](&r, func(p godi.Provider) IFoo {
    return &Foo{d: godi.GetRequiredKeyed[IDep](p, "k")}
})
```

生成成功，`NewRoot()` panic：`no keyed service registered for type IDep with key k`。服务明明注册了。原因是 `emitNewRoot` 固定按“unkeyed singleton → unkeyed scoped → keyed singleton → keyed scoped”接线（`generate.go:280-283`），keyed 的 `Get*` 又不参与排序。runtime 是惰性物化，这个写法在 runtime 上是正常的。

#### B3. captive 检查可以被绕过

```go
godi.AddScoped[IReq, Req](&r)
godi.AddSingletonFactory[ICache](&r, newCache) // newCache 内部 GetRequired[IReq]
```

- 具名函数：生成器退出码 0；运行时 `NewRoot` panic，报的是“IReq 未注册”，信息有误导性。
- 改成内联闭包：正确报出 captive 错误。
- 改成 `AddKeyedSingletonFactory` + 内联闭包：生成器退出码 0。keyed form-B 完全不计算 `Deps`（`main.go:326`）。

#### B4. scoped 急切构造

生成容器在 `NewRoot` 里构造一遍全部 scoped（根级），之后每次 `CreateScope` 再构造一遍全部 scoped，不管用不用（`generate.go:738-780`）。runtime 是按需构造。

注册 A、B、C 三个 scoped，其中 `NewC` 可能返回 error：

| 观察项 | 生成容器 | runtime |
| --- | --- | --- |
| `NewRoot` / `Build` 之后已构造 | A、B、C | 无 |
| `CreateScope` 后、未解析任何服务时已构造 | A、B、C | 无 |
| 先解析 B 再解析 A，`Close` 的释放顺序 | B、A | A、B（逆创建序，与 dotnet 一致） |
| `NewC` 返回 error，只想用 A | `CreateScope` panic | 不受影响 |

这是 AD-1（生命周期 = 实例存放位置）的设计决定，不是编码失误。但它的后果没有在面向用户的文档里说明：每个请求级 scope 都要付出全部 scoped 服务的构造成本和副作用；`CreateScope()` 没有 error 返回值，任何一个 scoped 构造失败都只能 panic；两个 provider 的构造时机和释放顺序可观测地不同，而 AD-15 声称两者差分等价。

### 4.3 C 组：生成器工作流

#### C1. 被自己的旧产物卡死

README 推荐的布局是原地生成（`-pkg di -out container_gen.go .`）。`loadPackage` 遇到任何类型错误就中止（`main.go:108-115`），而它加载的包里包含上一次生成的 `container_gen.go`。

复现步骤：

1. 注册 `ILog → Log`、`IFoo → Foo`，生成成功。
2. 把 `NewFoo()` 改成 `NewFoo(l ILog)`，这是 DI 里最常见的改动。
3. 再次生成，失败：`container_gen.go:23:19: not enough arguments in call to NewFoo`。
4. 手工删掉 `container_gen.go` 后再生成，成功。

删除或重命名已注册的实现类型、构造函数时同理。CI 里的 “Generate gate” 在这种情况下也会失败。

#### C2. 推断掉类型实参的标记被跳过

```go
godi.AddSingleton[IFoo, Foo](&r)
godi.AddSingletonFactory(&r, func(p godi.Provider) ILog { return &Log{} }) // 省略了 [ILog]
```

生成器在 stderr 打一行 warning，退出码 0，生成的容器里没有 `ILog`。`genericCallee` 只认带 `[...]` 的调用（`main.go:614`）。

这不是罕见写法：对显式写出 `[ILog]` 的版本运行 `gopls check`，gopls 会报 `unnecessary type arguments` 并提供删除它的快速修复。也就是说编辑器会主动引导用户写出被跳过的形式。`TypesInfo.Instances` 对推断出的实例同样有记录，所以这在技术上是可以支持的。

NFR2 的表述是“硬门槛、不静默丢弃”，但这里的实际行为是警告加继续，容器缺服务而构建照常通过。

### 4.4 D 组：生成器产出不可编译的代码

以下每种写法，生成器都以退出码 0 写出文件，随后 `go build` 失败。

| 写法 | 编译错误 | 原因位置 |
| --- | --- | --- |
| keyed form-A，构造函数返回 `(T, error)` | `assignment mismatch: 1 variable but NewFoo returns 2 values` | `emitKeyedWiringBlock` 没有 `CtorErr` 分支（`generate.go:361`） |
| 构造函数参数是具体类型 `*Foo`，而 `Foo` 以接口 `IFoo` 注册 | `cannot use r.f_Foo (IFoo) as *Foo value` | `depIndex` 把 `Impl` / `*Impl` 也映射到服务（`main.go:220-223`），字段类型却是接口 |
| `AddSingleton[IFoo, Baz]`，`Baz` 没实现 `IFoo` | `*Baz does not implement IFoo` | 只校验返回值个数（`main.go:1177`），不校验可赋值性 |
| 两个服务依赖同一个“构造函数返回 error 的 transient” | `no new variables on left side of :=` | `emitErrTransientDeps` 在同一函数体里重复声明 `dep_X`（`generate.go:439`） |
| key 是另一个包的常量，例如 `keys.Primary` | `undefined: keys` | `keyText` 原样作为 `case`，不记录 import（`main.go:265`） |
| 实现类型名为 `IsKeyedService`、`Get` 等 | `method Root.Get already declared` | `reservedMethodNames` 缺 `IsKeyedService`、`Get`、`GetRequired`、`CloseAsync`（`plan/validate.go:12`） |
| 容器里唯一的可释放服务是 form-A transient | `"io" imported and not used` | 只要有可释放服务就导入 `io`（`generate.go:166`），但这条路径不使用它 |

最后一种是很常见的形态。

共同的根因是生成器不对自己的产物做类型检查：`genlib.Generate` 末尾只有 `format.Source`（`generate.go:114`），只查语法。测试方面，`cmd/godi-gen` 里只有两个测试对产物跑了 `go vet`（`concrete_factory_test.go:42`、`optionshelper_test.go:180`），其余 fixture 都是文本或 golden 比对，`testdata/` 也不参与 `go build ./...`。

这些都是构建期的响亮失败，危害低于 A 组。但它们直接违反 AD-3 / NFR2 “坏图产出零字节”的承诺，报错位置还在用户不该去读的生成文件里。

相关的一条内部已知项：`NewRoot` 中途 `return nil, err` 时不释放已经追踪的可释放对象（`deferred-work.md` 201 行），也不 recover 工厂的 panic；runtime 的 `Build` 两者都做了。

### 4.5 E 组：runtime provider

`architecture.md` 把 runtime 定位为语义 oracle，把生成容器定位为生产路径，所以这一组排在后面。但它们仍然重要：`runtime` 是公开包；`EXTENSIONS_DIVERGENCE.md` 的 EXT-D6 规定 `TryAdd` / `Replace` / `Remove` 只能在 runtime 上用；而且 oracle 自身有错会削弱差分测试的意义。

- **E1. 不可哈希的 key。** `IsKeyedService(t, []string{"k"})` panic：`hash of unhashable type []string`。契约（`provider.go:36-38`）和 `architecture.md` 都写明这是永不 panic 的纯查询。用这种 key 注册时，panic 会直接逃出 `Build()`，因为 map 写入发生在 `defer recover` 安装之前（`runtime/provider.go:697-746`）。
- **E2. 释放顺序不确定。** 5 个互不依赖的可释放单例，200 次 `Build` + `Close` 出现了 5 种释放顺序。`Build` 用 `for t := range sf` 急切构造（`provider.go:752`），释放是创建序的逆序。生成容器的顺序是确定的。
- **E3. 循环依赖变成栈溢出。** 循环检测只在 `root.GetService` 里。两个 scoped 互相依赖并从 scope 解析，或者两个 transient 互相依赖，结果都是 `fatal error: stack overflow`，进程直接退出，`recover` 接不住。`architecture.md:60` 写的是“惰性物化并带循环检测”，没有加限定。
- **E4. `Close` 重入自锁。** 可释放对象的 `Close()` 里再调 `provider.Close()`，`root.Close()` 永不返回。`closeMu` 在调用 `CloseAll` 期间一直被持有（`provider.go:347-356`）。生成容器没有这把锁，不会卡住，两者行为不同。
- **`TryAddEnumerable` 等同于 `TryAdd`**，无法为同一服务类型累积多个实现。这一条 `EXTENSIONS_DIVERGENCE.md` 的 EXT-D9 已经如实记录，这里只作提醒。
- **form-A 的 `[]T` 参数在没有任何 `T` 注册时被生成器拒绝**（报缺失依赖），而 dotnet 和 runtime 的 `GetServices` 都返回空集合。属于未记录的行为差异。

### 4.6 F 组：文档与注释

- `TESTING.md` 整体过期：第 7 行仍称“framework scaffold、当前跑的是测试内桩 provider”；第 62 行引用不存在的 `spec_sample_test.go`；第 119 行把本机绝对路径 `/mnt/disk0/project/auth/go.work` 写进了公开文档，并且说的是 “go >= 1.26”。
- `architecture.md:10` 称 runtime 是 “reflect-based oracle”，`provider.go:10` 的注释称它是 “reflect-free runtime oracle”。
- `architecture.md:157` 的 AD-14 写“两个 provider 派生自同一份 DSL 注册”，实际不是，见 5.2。
- `generate.go:17`、`:157` 的注释仍写 “six-method”，接口早已是 7 个方法。`plan/validate.go` 的保留名表同样没跟上（见 D 组）。
- 根包 `doc.go` 仍写着 “AddScoped / AddTransient / AddKeyed 将在后续 epic 落地”；`keyed.go:11-13` 写着“本 story 只定义哨兵”。
- 有 56 个文件名带 `_redphase_test.go`，但它们都已经没有 build tag，默认就会运行。文件名会让人以为它们是被隔离的红灯测试。
- `plan/validate.go:43` 的重名报错建议“用显式的不同名字注册”，而 DSL 并没有指定名字的 API。

## 5. 设计层面的根因

### 5.1 字段急切接线需要完整的静态顺序，form-B 给不了

singleton 和 scoped 都在 `NewRoot` / `CreateScope` 里按拓扑序一次性赋值到字段上。这对 form-A 是成立的，因为构造函数签名就是完整的依赖信息。form-B 工厂拿到的是一个 `Provider`，可以在任意位置、以任意方式解析任意服务，生成器只能靠扫描文本去猜。

A1、A2、B1、B2、B3、B4 都是这一个矛盾的不同表现。只要 form-B 还是“逃生舱”，靠增强文本扫描是补不完的。方向上可以考虑把字段读取改成“首次访问时构造”的访问器，让顺序在运行时自然成立，同时用“构造中”标记把循环变成明确的错误。

### 5.2 “同一来源，两个 provider”没有实现

AD-14 / AD-15 是整个测试策略的基础：两个 provider 来自同一份注册，结果应差分等价。实际上：

- `godi.Add*` 标记是空函数，runtime 无法消费它。
- runtime 有另一套 API（`runtime.AddSingleton[T](b, factory)`），只支持工厂形式。
- conformance 里两边的注册是各写一遍的（`conformance/spec_test.go:30` 起是手写的 `runtime.Add*` 闭包，`conformance/codegen/registration.go` 是 DSL）。

因此差分测试比较的是两份人工对齐的注册，只覆盖被专门写出来的场景。B4 的构造时机和释放顺序、A3 的 form-B 释放、A4 的混合生命周期，都是两边确实不同却没有被测到的地方。

### 5.3 没有把运行时的值送进容器的途径

`NewRoot()` 没有参数；DSL 从不执行；`runtime.AddSingletonInstance` 在 codegen 侧没有对应物。配置、已有的连接、`context` 这类只有运行时才知道的东西，只能通过包级全局变量传入。A5 是用户按直觉去写时会撞上的结果。README 没有说明这个限制，也没有给出推荐做法。

同一类限制还有：一个包只能生成一个容器，`Registry` 实参没有意义，条件注册不被支持（A6）。

### 5.4 并发契约没有对用户说明

见 A7。内部把它定为本期不做（NFR8），这个取舍本身可以接受，但必须写进 README 和 `Provider` 的文档注释。目前的状态是用户会默认它安全。

### 5.5 “硬门槛”实际上依赖警告和语法检查

AD-3 / NFR2 的承诺是坏输入产出零字节并以非零码退出。现状是：识别不了的标记只打 warning（C2）；产物只过 `format.Source`（D 组）；form-B 的图基本不校验（B 组）。门槛对 form-A 的图结构是硬的，对其余部分不是。

### 5.6 相对项目目标的差距

`.meta/p.md` 的目标是“完整实现 dotnet 依赖注入的所有功能，所有测试完全通过”。当前的“全绿”有一部分来自把用例归为 N/A 或 adapted：并发与压力测试、AnyKey 兜底注册、`TryAddEnumerable` 的多实现累积、按引用 `Remove`、`IServiceScopeFactory` 等。分歧文档对此记录得很诚实，这是优点；但对外描述项目状态时，应以“在 Go 约束下的子集”为准，而不是“等价能力”。

## 6. 做得好的部分

- 分层清晰：`cmd/godi-gen → genlib → plan` 单向依赖，生成的容器不依赖生成器。
- form-A 的校验是实打实的：缺失、captive、循环一次性聚合报出，错误信息带完整依赖链。
- 输出确定：按源码位置排序，有 golden 和重新生成比对的测试，CI 有漂移门槛。
- 工程卫生好：工具链用 mise 固定，CI 有 gofmt、vet、race、generate 门槛；依赖只有 `x/tools`。
- 分歧文档诚实：`conformance/*_DIVERGENCE.md` 没有用假通过凑数，EXT-D9 还主动推翻了自己早先的错误结论。
- 细节考虑周到的地方不少：导入别名冲突处理、nil key 归一到 unkeyed、关闭后解析用统一的哨兵错误。

## 7. 建议的处理顺序

1. **先止血文档。** 在 README 写明：容器不能跨 goroutine 共享；form-B 不享受图校验；form-B 产物不会被自动释放；标记必须显式写类型实参。这一步不需要改代码。
2. **让生成器对产物做类型检查。** 生成后、写文件前用 `go/types` 检查一遍，失败则按 NFR2 处理。这一项能一次性把 D 组全部变成生成期错误。
3. **修 C1。** 加载包时排除输出文件（overlay 空文件或 build tag）。
4. **修 A3。** form-B 的追踪改成运行时断言 `io.Closer` / `AsyncDisposable`。
5. **决定 5.1 的方向。** 要么改成惰性访问器，要么明确限制 form-B 的写法并在生成期拒绝扫描不到的形式（具名函数、`p.GetService`、keyed `Get*`）。A1 的拓扑缺口无论哪种方向都应补上。
6. **把 C2 和未识别标记从 warning 改成 error**，或者直接支持推断形式。
7. **混合生命周期（A4）**：实现注册序，或者在 `Build` 和生成期明确拒绝。
8. **给 `Disposer` 和 `closed` 加同步**，并补并发测试；现在的 `-race` 全绿是因为没有任何并发用例。
9. 清理 F 组的过期文档。

## 附录：复现方法

所有实验都在仓库之外的独立 module 中进行，结构如下：

```text
exp/
  go.mod          module exp; go 1.27; require + replace github.com/gonewx/godi => <本仓库路径>
  <用例名>/
    reg.go        正文中给出的注册片段，加上最简的类型和 New{Impl} 定义
    x_test.go     调用 NewRoot / CreateScope 并打印观察结果
```

每个用例的命令：

```bash
GOWORK=off go build -o /tmp/godi-gen ./cmd/godi-gen      # 在本仓库根目录执行
cd exp/<用例名>
GOWORK=off /tmp/godi-gen -pkg <用例名> -out container_gen.go .
GOWORK=off go build .                                    # D 组看这一步
GOWORK=off go test -count=1 -v .                         # A、B 组看这一步
GOWORK=off go test -count=1 -race .                      # A7
```

E 组直接用 `runtime.NewBuilder()` 写测试即可。E3 会让进程崩溃，需要放在子进程里运行。
