// Package container is the B2 (codegen + direct wiring) architecture slice:
// a reflection-free DI runtime with singleton / scoped / transient lifetimes,
// nested scopes, and LIFO disposal.
//
// This file is hand-written in the SHAPE a generator would emit — the
// opengeneric PoC already proved the codegen mechanism; here we validate the
// runtime SEMANTICS of the lifetimes/scopes/disposal model under direct wiring.
//
// Lifetime is encoded by WHERE an instance lives, not by a runtime descriptor:
//   - singleton -> field on Root, built once in NewRoot
//   - scoped    -> field on Scope, built lazily once per scope
//   - transient -> a New* method that constructs a fresh instance every call
package container

import "godipoc/lifecycle/domain"

// Root owns SINGLETON instances; constructed exactly once.
type Root struct {
	logger *domain.Logger
	disp   *disposer
}

func NewRoot() *Root {
	return &Root{
		logger: domain.NewLogger(), // singleton: one instance for the whole Root
		disp:   newDisposer(),
	}
}

// Logger is the typed accessor for the singleton — the zero-reflection
// replacement for dotnet's provider.GetService<Logger>().
func (r *Root) Logger() *domain.Logger { return r.logger }

// Close disposes Root-level (singleton) disposables, LIFO.
func (r *Root) Close() error { return r.disp.closeAll() }

func (r *Root) DisposedOrder() []string { return r.disp.order() }

// Scope is a child scope; SCOPED instances are cached here, one per scope.
type Scope struct {
	root  *Root
	db    *domain.DbConnection // scoped: built once per scope
	audit *domain.AuditSink    // scoped: built once per scope
	disp  *disposer
}

func (r *Root) NewScope() *Scope {
	return &Scope{root: r, disp: newDisposer()}
}

// Root exposes the owning Root so transitive singleton access stays explicit.
func (s *Scope) Root() *Root { return s.root }

// DbConnection is a SCOPED typed accessor: identical within a scope, distinct
// across scopes; registered for disposal the first time it is built.
func (s *Scope) DbConnection() *domain.DbConnection {
	if s.db == nil {
		s.db = domain.NewDbConnection(s.root.Logger()) // inject the singleton
		s.disp.track("DbConnection", s.db)
	}
	return s.db
}

// AuditSink is a second SCOPED accessor (used to demonstrate LIFO disposal).
func (s *Scope) AuditSink() *domain.AuditSink {
	if s.audit == nil {
		s.audit = domain.NewAuditSink()
		s.disp.track("AuditSink", s.audit)
	}
	return s.audit
}

// NewUserService is the TRANSIENT accessor: a fresh instance every call, wired
// from the scoped DbConnection and the singleton Logger.
func (s *Scope) NewUserService() *domain.UserService {
	return domain.NewUserService(s.DbConnection(), s.root.Logger())
}

// Close disposes SCOPED disposables, LIFO.
func (s *Scope) Close() error { return s.disp.closeAll() }

func (s *Scope) DisposedOrder() []string { return s.disp.order() }
