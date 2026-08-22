# godi — 编译期依赖注入容器

> Go 依赖注入库 · `github.com/gonewx/godi`

[English](./README.md) | 中文

---

用静态标记声明依赖图，由代码生成器产出装配代码——**主路径零反射**。

## 特点

| 能力 | 说明 |
| --- | --- |
| **编译期装配** | `godi-gen` 经 `go/packages` + `go/types` 静态读取 DSL 调用，从不执行它们；产出的容器是普通 Go 代码 |
| **图非法则不产码** | 循环依赖、captive dependency、缺失注册在 `go generate` 阶段即失败，坏图生不出二进制 |
| **三种生命周期** | singleton / scoped / transient，作用域按 LIFO 释放 |
| **键控服务与开放泛型** | `AddKeyed*` 系列；开放泛型走显式封闭注册 |
| **仅一个外部依赖** | `golang.org/x/tools`，且只被生成器使用 |

## 要求

Go 1.27+（依赖具体类型上的方法级类型参数）。

## 安装

```bash
go get github.com/gonewx/godi
```

## 快速开始

### 1. 声明依赖图

这是静态标记，不在运行时执行。形态 A 按严格命名约定发现 `New{Impl}` 构造器；
没有合适构造器时，形态 B 工厂是逃生舱：

```go
package di

import "github.com/gonewx/godi"

//go:generate godi-gen -pkg di -out container_gen.go .

func register() {
    var r godi.Registry

    godi.AddSingleton[Config, AppConfig](&r)   // 形态 A：发现 NewAppConfig
    godi.AddScoped[Repo, PgRepo](&r)           // 作用域生命周期
    godi.AddTransientFactory[Logger](&r, func(p godi.Provider) Logger {
        return NewLogger(godi.GetRequired[Config](p)) // 形态 B：工厂逃生舱
    })
}
```

### 2. 生成容器

```bash
go install github.com/gonewx/godi/cmd/godi-gen@latest
go generate ./...
```

### 3. 解析服务

```go
root, err := di.NewRoot()
if err != nil {
    return err
}
defer root.Close()

cfg := godi.GetRequired[Config](root)   // 缺失即 panic
log, ok := godi.Get[Logger](root)       // 缺失返回 false
all := godi.GetServices[Repo](root)     // 全部实现
```

### 作用域

```go
scope := root.CreateScope()
defer scope.Close()   // 按 LIFO 释放该作用域内的 Disposable

repo := godi.GetRequired[Repo](scope)
```

## 生成物

`container_gen.go` 是**构建输入**——应当提交入库，消费方无需安装生成器即可构建。
修改 DSL 声明后须重跑 `go generate ./...`，并确保 `git status --porcelain`
保持为空（防装配漂移）。

## 文档

完整文档为中英双语（`docs/en` / `docs/zh`），从索引进入：

| 语言 | 索引 |
| --- | --- |
| English | [docs/en/index.md](./docs/en/index.md) |
| 中文 | [docs/zh/index.md](./docs/zh/index.md) |

| 主题 | 内容 |
| --- | --- |
| [测试指南](./TESTING.md) | 测试框架、运行命令、最佳实践 |
| [语义差异说明](./conformance/) | 与 .NET DI 的行为差异（Activator / Async / CallSite / Extensions / Open Generic） |

## 许可证

MIT，见 [LICENSE](./LICENSE)。

---

[English version →](./README.md)
