# godi 开发指南

## 环境准备

godi 的 Go 工具链由 [mise](https://mise.jdx.dev) 在项目级锁定（`mise.toml`），当前
锁定 `go = "1.27.0"`（Go 1.27 已于 2026-08-19 GA，锁定当前正式补丁版）。Go 1.27 是
必需的：它解锁具体结构体上的方法级类型参数（AD-5），支撑 `provider.Get[T]()` 泛型
方法 API。

```bash
mise install                       # 安装锁定的 Go 工具链（含 go 1.27.0）
eval "$(mise activate bash)"       # 在当前 shell 激活 mise（或用 mise exec -- ...）
go version                         # 应显示 go1.27.0
```

激活 mise 后，`go` 自动解析到锁定版本。唯一外部依赖是 `golang.org/x/tools`
（v0.49.0，供 generator 静态分析），无需 testify 等测试依赖——套件仅用标准库。

> Go 1.27 解锁方法级类型参数（AD-5）。升级到更新 **minor** 时同时升 `mise.toml`
> 与 `go.mod` 的 `go` 指令；1.27.x 补丁版通常只需调整 `mise.toml`。

## 常用命令

所有质量目标封装在 `Makefile`：

| 命令 | 作用 |
| --- | --- |
| `make test` | 全部测试 + race 检测器 + 覆盖率 profile（`coverage.out`） |
| `make test-short` | 快速通过，无 race，跳过慢用例 |
| `make conformance` | 双 Provider 一致性套件（verbose，`-run 'Spec\|Conformance'`） |
| `make golden` | 重生成 golden 基准（`-run Golden -update-golden`） |
| `make cover` | 生成 HTML 覆盖率报告（`coverage.html`） |
| `make vet` | 运行 `go vet` |
| `make fmt` | 格式化全部 Go 源码 |
| `make tidy` | 整理模块图 |
| `make check` | CI 对齐：`vet` + race/coverage 全量测试 |
| `make help` | 列出全部目标 |

等价原始命令：

```bash
go test ./...                 # 普通运行
go test -race ./...           # 带 race 检测器
go test -cover ./...          # 带覆盖率摘要
go test ./... -update-golden  # 重写 golden 基准（审 diff 后再提交！）
```

## 测试约定

- **纯标准库断言（无 testify）**：`internal/testutil` 提供基于 `TB` seam 的轻量
  断言（`Equal`/`Nil`/`Panics` 等），失败行为本身用假 `TB` 做单元测试，保持依赖面
  最小。
- **红灯文件名**：`*_redphase_test.go` 这些文件名是 ATDD 红灯阶段留下的；构建标签
  都已移除，它们现在是默认测试套件的一部分，正常编译运行。
- **golden 文件**：generator 产码测试对比 `testdata/*.golden` 字节基准；变更后用
  `make golden` 重生成并审 diff。
- **双 Provider conformance**：一致性套件写一次、跑两个 Provider——`ProviderFactory`
  ×2 由 `RunSuite` 驱动，`AssertEquivalent` 做跨 Provider 差分等价（AD-14/AD-15）。
  runtime 侧注册手写的 `runtime.Add*` 闭包，codegen 侧使用由 `conformance/codegen`
  中 `godi.Add*` 标记生成的容器，两边的注册须人工保持等价。

## CI

`.github/workflows/test.yml` 在 push / PR 上运行：

1. **`jdx/mise-action`** 按 `mise.toml` 供给工具链，CI 与本地用同一锁定 Go 版本。
2. **格式门禁**：对主模块包目录跑 `gofmt -l`，非空即失败。
3. **`go vet ./...`**。
4. **`go build ./...`**。
5. **Generate gate（AD-3 / NFR2）**：`cmd/godi-gen` 存在时跑 `go generate ./...`，
   校验生成代码无漂移（`git diff` 干净）且无未跟踪产物——坏图（缺失/捕获/循环）必须
   被拒并产零字节。greenfield 上此步自动跳过。
6. **`go test -race -covermode=atomic -coverprofile=coverage.out ./...`**：覆盖
   conformance 双 Provider 套件与 golden 测试，race 检测器为质量门禁。
7. 覆盖率作为构建产物上传。

godi 是库，没有部署步骤：发布 = 打 semver tag，消费者把 `go generate` 作为构建步骤
运行。

## 代码贡献准则

- **等价注册，双 Provider**：runtime 与 codegen 两侧的 conformance 注册须人工保持
  等价（AD-14）。
- **按 `reflect.Type` 解析**：类型键用 `reflect.TypeFor[T]()`，与 Provider 的 Tier B
  派发一致（AD-13）。
- **主路径零反射构造**：不得让 `reflect.New` / `reflect.Value.Call` 进入主路径，
  Tier C 反射仅限 `activator`（AD-13）。
- **警惕 nil 接口陷阱**：nil 接口值不得被读作已注册服务（AD-5）；`testutil.Nil` 专门
  把 typed-nil 当作 nil 处理。
- **断言行为而非实现**：Spec 用例只触碰 `Provider` 契约，令两 Provider 被同一可观察
  行为约束。
- **表驱动子测试**：Spec 用例表达为 `[]SpecCase`，由 `RunSuite` 驱动为嵌套
  `t.Run`；资源用 `t.Cleanup` 释放。
- **提交前保持 gofmt-clean**：CI 有格式门禁；提交前跑 `make fmt`。

## 排障

| 症状 | 原因 / 修复 |
| --- | --- |
| `go: command not found` 或 Go 版本不对 | 当前 shell 未激活 mise。运行 `mise install` 后 `eval "$(mise activate bash)"`（或用 `mise exec -- go ...`） |
| `go.work lists go 1.23.1, but ... requires go >= 1.27` | 上级目录的 `go.work` 早于 godi 的 `go` 指令。命令前加 `GOWORK=off` 运行 |
| `cannot find toolchain ... go1.27` / 下载失败 | mise 无法为当前平台拉取锁定的工具链。确认 mise 可拉取该 patch（`mise ls-remote go \| grep 1.27`）后重跑 `mise install`；如需换 patch，调整 `mise.toml` 的锁定项 |
| `golden: read testdata/...: no such file` | golden 基准缺失。用 `go test ./... -update-golden` 创建，提交前审 diff |
| golden mismatch | 产码字节变了。若属预期，跑 `make golden` 并提交新基准；否则是真实回归 |
