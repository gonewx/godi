// Package lazyctorerr exercises the error paths of the lazy singleton emit: a
// singleton whose constructor takes error-returning transients (scalar and
// collection) and itself returns an error, plus a root-level scoped service
// with an error-returning constructor. Test INPUT only (testdata/); it is
// generated, compiled and run by lazy_singleton_test.go.
package lazyctorerr

import (
	"errors"

	"github.com/gonewx/godi"
)

// ErrRepo is returned by NewRepo while FailRepo is set.
var ErrRepo = errors.New("repo failed")

// FailRepo makes NewRepo fail; ConnClosed records that the Conn was released.
var (
	FailRepo   bool
	ConnClosed bool
)

type IConn interface{ Ping() }
type Conn struct{}

func (*Conn) Ping()        {}
func (*Conn) Close() error { ConnClosed = true; return nil }
func NewConn() *Conn       { return &Conn{} }

type IPart interface{ Part() string }
type PartA struct{}
type PartB struct{}

func (*PartA) Part() string     { return "a" }
func (*PartB) Part() string     { return "b" }
func NewPartA() (*PartA, error) { return &PartA{}, nil }
func NewPartB() (*PartB, error) { return &PartB{}, nil }

type ITool interface{ Tool() }
type Tool struct{}

func (*Tool) Tool()           {}
func NewTool() (*Tool, error) { return &Tool{}, nil }

type IRepo interface{ Parts() []IPart }
type Repo struct{ parts []IPart }

func (r *Repo) Parts() []IPart { return r.parts }
func NewRepo(c IConn, t ITool, parts []IPart) (*Repo, error) {
	if FailRepo {
		return nil, ErrRepo
	}
	return &Repo{parts: parts}, nil
}

type IReq interface{ Repo() IRepo }
type Req struct{ repo IRepo }

func (q *Req) Repo() IRepo                  { return q.repo }
func NewReq(r IRepo, t ITool) (*Req, error) { return &Req{repo: r}, nil }

func register() {
	var r godi.Registry
	godi.AddSingleton[IConn, Conn](&r)
	godi.AddTransient[IPart, PartA](&r)
	godi.AddTransient[IPart, PartB](&r)
	godi.AddTransient[ITool, Tool](&r)
	godi.AddSingleton[IRepo, Repo](&r)
	godi.AddScoped[IReq, Req](&r)
}
