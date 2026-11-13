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

`mise.toml` 还设置了 `GOWORK=off`（godi 作为独立模块构建，上级目录的 `go.work` 不能接管），
并锁定了 `gh`（只有 `make release-publish` 用到）。未激活 mise 的 shell 里，用
`mise exec -- make <目标>` 运行各目标。

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
| `make build` | 运行 `go build ./...` |
| `make fmt-check` | 有 Go 源码不是 gofmt-clean 时失败 |
| `make generate` | 重新运行生成器（`go generate ./...`） |
| `make generate-check` | 已提交的生成代码过期时失败 |
| `make changelog-check` | 两份变更日志列出的版本不一致时失败 |
| `make ci` | CI 运行的全部检查，顺序与 CI 相同（`make check` 是它的别名） |
| `make version` | 打印当前检出的生成器版本 |
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

CI 运行的每条命令都是 Makefile 目标，工具链和环境变量由 `mise.toml` 提供。仓库里没有独立脚本，
所以本地 `make ci` 运行的内容与 CI 完全相同。

`.github/workflows/test.yml` 在 push 到 `main` 和每个 PR 上运行：

1. **`jdx/mise-action`** 按 `mise.toml` 供给工具链与环境（Go、`gh`、`GOWORK=off`）。
2. **`make fmt-check`**：对主模块包目录跑 `gofmt -l`，非空即失败。
3. **`make vet`**。
4. **`make build`**。
5. **`make generate-check`（AD-3 / NFR2）**：跑 `go generate ./...`，生成代码有漂移即失败；
   坏图（缺失/捕获/循环）必须被拒并产零字节。
6. **`make changelog-check`**：`CHANGELOG.md` 与 `CHANGELOG.zh-CN.md` 列出的版本必须一致。
7. **`make test`**：`go test -race -covermode=atomic -coverprofile=coverage.out ./...`，覆盖
   conformance 双 Provider 套件与 golden 测试，race 检测器为质量门禁。
8. 覆盖率作为构建产物上传。

`.github/workflows/release.yml` 在推送 `v*` tag 时运行，见[发布](#发布)。

## 发布

godi 是库：一次发布 = 一个 semver tag 加一个 GitHub Release。版本策略写在 README 里；
下面的每一步都是 Makefile 目标。

1. 在 `CHANGELOG.md` 和 `CHANGELOG.zh-CN.md` 里，把 `## [Unreleased]` 改名为
   `## [X.Y.Z] - YYYY-MM-DD`，在它上面新增一个空的 `## [Unreleased]`，更新文件末尾的链接引用，
   然后在 `main` 上提交。
2. `make release-tag VERSION=vX.Y.Z`：先运行 `release-check`（版本号格式、两份变更日志里都有带日期
   且非空的小节、已跟踪文件没有未提交的改动，然后是整个 `ci` 目标），再创建带注释的 tag，
   tag 消息就是该变更日志小节。这一步不推送任何东西。
   需要签名时设置 `GIT_TAG_FLAGS=-s`。
3. `make release-push VERSION=vX.Y.Z`：在一次原子推送里推送 `main` 和这个 tag。
4. 发布工作流在打了 tag 的提交上重新运行 `make release-check`，然后运行 `make release-publish`，
   用同一个变更日志小节创建 GitHub Release。带预发布后缀的版本（`v0.3.0-rc.1`）会发布为预发布。

`make release-notes VERSION=vX.Y.Z` 可以打印任意已发布版本的说明。

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
| `go.work lists go 1.23.1, but ... requires go >= 1.27` | 上级目录的 `go.work` 接管了模块解析。`mise.toml` 已设置 `GOWORK=off`，出现这个错误说明当前 shell 未激活 mise：用 `mise exec -- make <目标>` 运行 |
| `cannot find toolchain ... go1.27` / 下载失败 | mise 无法为当前平台拉取锁定的工具链。确认 mise 可拉取该 patch（`mise ls-remote go \| grep 1.27`）后重跑 `mise install`；如需换 patch，调整 `mise.toml` 的锁定项 |
| `golden: read testdata/...: no such file` | golden 基准缺失。用 `go test ./... -update-golden` 创建，提交前审 diff |
| golden mismatch | 产码字节变了。若属预期，跑 `make golden` 并提交新基准；否则是真实回归 |
