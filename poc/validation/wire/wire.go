// Package wire is the generated-style, reflection-free runtime for the two
// remaining Tier-1/2 resolution shapes: IEnumerable[T] (multiple implementations)
// and keyed services. Both are plain generated accessors — no reflect, no map
// keyed by reflect.Type, no interface generic method.
package wire

import "godipoc/validation/domain"

type Container struct {
	handlers []domain.IHandler
}

func New() *Container {
	return &Container{
		// IEnumerable[IHandler]: every registered implementation, in
		// registration order (dotnet guarantees this ordering too).
		handlers: []domain.IHandler{
			domain.EmailHandler{},
			domain.SmsHandler{},
			domain.PushHandler{},
		},
	}
}

// Handlers is the IEnumerable[IHandler] accessor (zero reflection).
func (c *Container) Handlers() []domain.IHandler { return c.handlers }

// Cache is the keyed accessor — the dotnet GetKeyedService<ICache>(key)
// equivalent — emitted as a generated key switch. Zero reflection; an unknown
// key returns ok=false (mirrors dotnet returning null for GetKeyedService).
func (c *Container) Cache(key string) (domain.ICache, bool) {
	switch key {
	case "redis":
		return domain.RedisCache{}, true
	case "memory":
		return domain.MemoryCache{}, true
	}
	return nil, false
}
