// Package domain holds the sample services the generated container wires.
package domain

import (
	"fmt"
	"sync/atomic"
)

var idSeq atomic.Int64

func nextID() int64 { return idSeq.Add(1) }

// Config is a SINGLETON with no deps. It exists so the generated NewRoot must
// topologically order singletons (Config before Logger).
type Config struct{ Env string }

func NewConfig() *Config { return &Config{Env: "dev"} }

// Logger is a SINGLETON that depends on the singleton Config.
type Logger struct {
	ID  int64
	Env string
}

func NewLogger(cfg *Config) *Logger { return &Logger{ID: nextID(), Env: cfg.Env} }

func (l *Logger) Log(m string) string { return fmt.Sprintf("L%d[%s]:%s", l.ID, l.Env, m) }

// DbConnection is SCOPED + disposable, depends on the singleton Logger.
type DbConnection struct {
	ID     int64
	log    *Logger
	closed bool
}

func NewDbConnection(log *Logger) *DbConnection { return &DbConnection{ID: nextID(), log: log} }

func (d *DbConnection) Query() string { return d.log.Log(fmt.Sprintf("db%d", d.ID)) }
func (d *DbConnection) Close() error  { d.closed = true; return nil }
func (d *DbConnection) Closed() bool  { return d.closed }

// AuditSink is a second SCOPED + disposable service (proves LIFO disposal).
type AuditSink struct {
	ID     int64
	closed bool
}

func NewAuditSink() *AuditSink { return &AuditSink{ID: nextID()} }

func (a *AuditSink) Close() error { a.closed = true; return nil }
func (a *AuditSink) Closed() bool { return a.closed }

// UserService is TRANSIENT, depends on scoped DbConnection + singleton Logger.
type UserService struct {
	ID  int64
	db  *DbConnection
	log *Logger
}

func NewUserService(db *DbConnection, log *Logger) *UserService {
	return &UserService{ID: nextID(), db: db, log: log}
}

func (u *UserService) Do() string { return fmt.Sprintf("U%d(%s)", u.ID, u.db.Query()) }

// IHandler has three implementations resolved as IEnumerable[IHandler].
type IHandler interface{ Handle() string }

type EmailHandler struct{}

func (EmailHandler) Handle() string { return "email" }

type SmsHandler struct{}

func (SmsHandler) Handle() string { return "sms" }

type PushHandler struct{}

func (PushHandler) Handle() string { return "push" }

// ICache has two implementations distinguished by key (KeyedServices).
type ICache interface{ Kind() string }

type RedisCache struct{}

func (RedisCache) Kind() string { return "redis" }

type MemoryCache struct{}

func (MemoryCache) Kind() string { return "memory" }
