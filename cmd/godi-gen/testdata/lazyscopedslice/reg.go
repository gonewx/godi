package lazyscopedslice

import "github.com/gonewx/godi"

type IPlugin interface{ Name() string }
type P1 struct{}

func (*P1) Name() string { return "p1" }
func NewP1() *P1         { return &P1{} }

type P2 struct{}

func (*P2) Name() string { return "p2" }
func NewP2() *P2         { return &P2{} }

type IConsumer interface{ Items() []IPlugin }
type Consumer struct{ items []IPlugin }

func (c *Consumer) Items() []IPlugin        { return c.items }
func NewConsumer(items []IPlugin) *Consumer { return &Consumer{items: items} }

func register() {
	var r godi.Registry
	godi.AddScoped[IConsumer, Consumer](&r)
	godi.AddScoped[IPlugin, P1](&r)
	godi.AddScoped[IPlugin, P2](&r)
}
