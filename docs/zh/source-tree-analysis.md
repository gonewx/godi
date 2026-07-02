# godi 源码树分析

## 目录树（带注释）

```text
godi/                              # 模块根 = 根包 godi（import 路径无 /godi 后缀）
├── doc.go                         # 包文档：DSL/API 总述，AD-7 静态分析路径说明
├── dsl.go                         # DSL 静态标记：Add{Singleton,Scoped,Transient}[Factory] + Keyed 变体
├── dsl_test.go                    # DSL 基础测试
├── dsl_form_b_test.go            # 形态 B（工厂逃生舱）测试
├── dsl_lifetimes_redphase_test.go# 三生命周期红灯脚手架
├── provider.go                    # Provider/Scope 契约（AD-16 7 方法）+ ErrProviderClosed
├── provider_test.go
├── get.go                         # 自由泛型函数：Get/GetRequired/GetServices/GetRequiredPanic
├── get_test.go
├── get_extensions_redphase_test.go
├── get_generic_helper_redphase_test.go
├── keyed.go                       # AnyKey 哨兵 + 键控自由泛型（GetKeyed/GetRequiredKeyed/…）
├── marked.go                      # MarkedConstructor（activator best-match 标记）
├── async.go                       # AsyncDisposable 契约 + AsyncScope 值语义包装
├── async_scope_redphase_test.go
├── seed_test.go
├── gomod_test.go                  # go.mod 约束回归测试
├── gomod_go127_redphase_test.go   # Go 1.27 语言版本红灯脚手架
│
├── plan/                          # 注册计划模型 + 校验门禁
│   ├── doc.go
│   ├── registry.go                # Registry / Service / Lifetime 模型
│   ├── validate.go                # Validate()：重复名/保留名/缺失/捕获/循环，聚合 []error
│   ├── registry_test.go / topo_test.go
│   └── validate_*_test.go         # captive / cycle / missing_transitive 各校验测试
│
├── internal/
│   ├── genlib/                    # 唯一产码路径
│   │   ├── doc.go
│   │   ├── generate.go            # Generate()：Registry → 格式化 Go 源码
│   │   ├── generate_*_test.go     # deps/lifetimes/dispose/async/multiimpl/validate 等
│   │   └── testdata/*.golden      # 产码 golden 基准
│   └── testutil/                  # 纯标准库测试辅助（无 testify）
│       ├── assert.go              # Equal/Nil/Panics 等断言（基于 TB seam）
│       ├── factory.go             # 函数式选项数据工厂 Build/BuildFrom
│       ├── golden.go              # golden 比较器 + -update-golden 标志
│       └── testdata/
│
├── cmd/
│   └── godi-gen/                  # generator CLI
│       ├── doc.go
│       ├── main.go                # go/packages 加载 → go/types 读 → registry → 产码
│       ├── generator_test.go / spike_test.go / opengeneric_test.go / keyed_anykey_test.go
│       └── testdata/              # 注册样例：deps/formb/lifetimes/keyed*/opengeneric/…
│           └── */registration.go
│
├── runtime/                       # 运行时反射 Provider（预言机）
│   ├── doc.go
│   ├── provider.go                # Builder → Build → root/scope，reflect.Type 工厂 map
│   ├── collection.go              # 多实现集合解析
│   ├── disposer.go                # Disposer：Track/TrackAsync/CloseAll/CloseAllAsync LIFO
│   └── provider_*_test.go / disposer_*_test.go
│
├── activator/                     # 唯一反射构造点（Tier C）
│   ├── doc.go
│   ├── activator.go               # CreateInstance[T]：唯一 reflect.Value.Call
│   ├── factory.go                 # best-match 构造选择
│   └── activator_*_redphase_test.go
│
├── conformance/                   # 双 Provider 一致性套件（测试框架核心）
│   ├── harness.go                 # Provider 契约 seam、ProviderFactory、RunSuite、AssertEquivalent
│   ├── spec_*_redphase_test.go    # 按能力分组的 Spec 用例（lifetimes/keyed/opengeneric/…）
│   ├── threshold_*_test.go        # 门槛计数（116 等）
│   ├── *_DIVERGENCE.md            # 各能力的语义差异标注（activator/async/callsite/…）
│   ├── fakes/                     # 移植的 52 个 fakes
│   └── codegen/
│       ├── container_gen.go       # 预生成容器（Path γ 敏感场景）
│       └── doc.go
│
├── Makefile                       # test/conformance/golden/cover/vet/fmt/check 等目标
├── mise.toml                      # 工具链锁定：go = 1.27rc1
├── go.mod / go.sum                # 模块 github.com/gonewx/godi，go 1.27
├── TESTING.md                     # 测试框架结构与运行指南
└── .github/workflows/test.yml     # CI：mise-action + vet + build + generate gate + race/coverage
```

## 关键目录职责

| 目录 | 职责 |
| --- | --- |
| `godi`（根包） | 公开 DSL、`Provider`/`Scope` 契约、自由泛型函数、`AnyKey`、`AsyncDisposable` |
| `plan` | 注册计划模型 + `Validate()` 校验门禁 |
| `internal/genlib` | 唯一产码路径 `Generate()` |
| `internal/testutil` | 纯标准库断言/工厂/golden 辅助 |
| `cmd/godi-gen` | generator CLI，静态分析 → 产码 |
| `runtime` | 运行时反射 Provider（预言机）+ Disposer |
| `activator` | 唯一反射构造点 `CreateInstance[T]` |
| `conformance` | 双 Provider 一致性套件、harness、差分等价、fakes |

## 关键约定

- **红灯脚手架 `//go:build atdd_redphase`**：ATDD 红灯阶段的用例带此构建标签（Go 中
  `test.skip` 的等价物），默认不编译；删标签即激活。fixture 的 `init` panic 作为
  绊线，证明 AD-7 静态分析从不执行用户代码。红灯隔离常配 `GOWORK=off`。
- **golden 文件 `testdata/*.golden`**：generator 产码测试的字节基准。`Golden(t,
  name, got)` 对比 `testdata/<name>.golden`；`-update-golden` 重写（提交前审 diff）。
- **唯一产码路径 `genlib.Generate`**：所有代码发射都经此函数（AD-2），先 `Validate`
  再产码。
- **唯一反射构造点 `activator.CreateInstance[T]`**：全 godi 仅此处使用
  `reflect.Value.Call`（Tier C，AD-13），best-match 构造选择也只在这里（AD-9）。

## 测试文件命名约定

| 后缀 | 含义 |
| --- | --- |
| `*_test.go` | 常规单元/集成测试，默认参与编译运行 |
| `*_redphase_test.go` | 带 `//go:build atdd_redphase` 标签的红灯脚手架，默认不编译，删标签激活 |
| `testdata/*.golden` | 产码字节基准，由 `-update-golden` 生成/更新 |
