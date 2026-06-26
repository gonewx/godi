// Package spec is the single source of truth for the sample app's DI graph —
// the Go-expressed "manifest/DSL" the generator consumes. Service GoType/Ctor
// and group constructors are source-text fragments referencing package domain.
package spec

import "godipoc/gen/plan"

// Registry returns the registration graph the generator validates and emits.
func Registry() *plan.Registry {
	return plan.NewRegistry().
		Service(plan.Service{
			Name: "Config", GoType: "*domain.Config", Ctor: "domain.NewConfig",
			Lifetime: plan.Singleton,
		}).
		Service(plan.Service{
			Name: "Logger", GoType: "*domain.Logger", Ctor: "domain.NewLogger",
			Lifetime: plan.Singleton, Deps: []string{"Config"},
		}).
		Service(plan.Service{
			Name: "DbConnection", GoType: "*domain.DbConnection", Ctor: "domain.NewDbConnection",
			Lifetime: plan.Scoped, Deps: []string{"Logger"}, Disposable: true,
		}).
		Service(plan.Service{
			Name: "AuditSink", GoType: "*domain.AuditSink", Ctor: "domain.NewAuditSink",
			Lifetime: plan.Scoped, Disposable: true,
		}).
		Service(plan.Service{
			Name: "UserService", GoType: "*domain.UserService", Ctor: "domain.NewUserService",
			Lifetime: plan.Transient, Deps: []string{"DbConnection", "Logger"},
		}).
		Enum(plan.EnumGroup{
			Accessor: "Handlers", IfaceType: "domain.IHandler",
			Ctors: []string{"domain.EmailHandler{}", "domain.SmsHandler{}", "domain.PushHandler{}"},
		}).
		Keyed(plan.KeyedGroup{
			Accessor: "Cache", IfaceType: "domain.ICache",
			Items: []plan.KeyedItem{
				{Key: "redis", Ctor: "domain.RedisCache{}"},
				{Key: "memory", Ctor: "domain.MemoryCache{}"},
			},
		})
}
