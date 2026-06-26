// Package domain holds sample services for the IEnumerable[T] and keyed-service
// slices of the B2 architecture validation.
package domain

// IHandler has THREE implementations, resolved together as IEnumerable[IHandler].
type IHandler interface{ Handle() string }

type EmailHandler struct{}

func (EmailHandler) Handle() string { return "email" }

type SmsHandler struct{}

func (SmsHandler) Handle() string { return "sms" }

type PushHandler struct{}

func (PushHandler) Handle() string { return "push" }

// ICache has TWO implementations distinguished by a key (KeyedServices).
type ICache interface{ Kind() string }

type RedisCache struct{}

func (RedisCache) Kind() string { return "redis" }

type MemoryCache struct{}

func (MemoryCache) Kind() string { return "memory" }
