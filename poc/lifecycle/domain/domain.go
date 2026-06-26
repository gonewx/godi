// Package domain holds sample services exercising the three dotnet DI lifetimes
// (singleton / scoped / transient) plus disposal, for the B2 architecture slice.
//
// Every instance gets a unique id so tests can assert identity: "same instance
// reused" vs "fresh instance".
package domain

import (
	"fmt"
	"sync/atomic"
)

var idSeq atomic.Int64

func nextID() int64 { return idSeq.Add(1) }

// Logger is a SINGLETON dependency (no deps).
type Logger struct{ ID int64 }

func NewLogger() *Logger { return &Logger{ID: nextID()} }

func (l *Logger) Log(m string) string { return fmt.Sprintf("L%d:%s", l.ID, m) }

// DbConnection is a SCOPED, disposable dependency that needs a Logger.
type DbConnection struct {
	ID     int64
	log    *Logger
	closed bool
}

func NewDbConnection(log *Logger) *DbConnection {
	return &DbConnection{ID: nextID(), log: log}
}

func (d *DbConnection) Query() string { return d.log.Log(fmt.Sprintf("db%d.query", d.ID)) }

// Close satisfies io.Closer; the container disposes scoped instances on scope close.
func (d *DbConnection) Close() error { d.closed = true; return nil }

func (d *DbConnection) Closed() bool { return d.closed }

// AuditSink is a second SCOPED disposable, used to prove LIFO disposal order.
type AuditSink struct {
	ID     int64
	closed bool
}

func NewAuditSink() *AuditSink { return &AuditSink{ID: nextID()} }

func (a *AuditSink) Write(string) {}

func (a *AuditSink) Close() error { a.closed = true; return nil }

func (a *AuditSink) Closed() bool { return a.closed }

// UserService is TRANSIENT: a fresh instance per resolve, wired from the scoped
// DbConnection and the singleton Logger.
type UserService struct {
	ID  int64
	db  *DbConnection
	log *Logger
}

func NewUserService(db *DbConnection, log *Logger) *UserService {
	return &UserService{ID: nextID(), db: db, log: log}
}

func (u *UserService) Do() string { return fmt.Sprintf("U%d(%s)", u.ID, u.db.Query()) }
