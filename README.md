# godi

编译期依赖注入容器（Go）。用静态标记声明依赖图，由代码生成器产出装配代码——**主路径零反射**。

## 特点

- **编译期装配** — `godi-gen` 经 `go/packages` + `go/types` 静态读取 DSL 调用，从不执行它们；产出的容器是普通 Go 代码
- **图非法则不产码** — 循环依赖、captive dependency、缺失注册在 `go generate` 阶段即失败，坏图生不出二进制
- **三种生命周期** — singleton / scoped / transient，作用域按 LIFO 释放
- **键控服务与开放泛型** — `AddKeyed*` 系列；开放泛型走显式封闭注册
- **仅一个外部依赖** — `golang.org/x/tools`，且只被生成器使用

## 要求

Go 1.27+（依赖具体类型上的方法级类型参数）。

## 安装

```bash
go get github.com/gonewx/godi
```

## 使用

声明依赖图（这是静态标记，不在运行时执行）：

```go
//go:generate godi-gen -pkg di -out container_gen.go .

package di

import "github.com/gonewx/godi"

func register() {
    var r godi.Registry

    godi.AddSingleton[Config, *AppConfig](&r)
    godi.AddScoped[Repository, *PgRepository](&r)
    godi.AddTransient[Handler, *UserHandler](&r)

    godi.AddSingletonFactory[Logger](&r, func(p godi.Provider) Logger {
        return NewLogger(godi.GetRequired[Config](p))
    })
}
```

生成容器：

```bash
go install github.com/gonewx/godi/cmd/godi-gen@latest
go generate ./...
```

解析服务：

```go
root, err := di.NewRoot()
if err != nil {
    return err
}
defer root.Close()

cfg := godi.GetRequired[Config](root)      // 缺失即 panic
log, ok := godi.Get[Logger](root)          // 缺失返回 false
all := godi.GetServices[Handler](root)     // 全部实现
```

作用域（`CreateScope` 是生成容器上的方法）：

```go
scope := root.CreateScope()
defer scope.Close()   // 按 LIFO 释放该作用域内的 Disposable

repo := godi.GetRequired[Repository](scope)
```

## 生成物

`container_gen.go` 是**构建输入**，应当提交入库——消费方无需安装生成器即可构建。修改 DSL 后须重跑 `go generate ./...`，并确保 `git status --porcelain` 为空（防装配漂移）。

## 许可证

MIT，见 [LICENSE](./LICENSE)。
