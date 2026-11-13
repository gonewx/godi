package transientcollectiondeps

import (
	"testing"

	"github.com/gonewx/godi"
)

// IMonitor is registered before ISetup, and its factory resolves a transient
// IFactory that captures GetServices[ISetup]. The collection must hold the built
// ISetup singleton, not a nil left over from construction order.
func TestMonitorFactorySeesBuiltSetupCollection(t *testing.T) {
	root, err := NewRoot()
	if err != nil {
		t.Fatalf("NewRoot: %v", err)
	}
	m, ok := godi.GetRequired[IMonitor](root).(monitor)
	if !ok {
		t.Fatal("IMonitor is not a monitor")
	}
	f, ok := m.factory.(factory)
	if !ok {
		t.Fatal("monitor.factory is not a factory")
	}
	if len(f.setups) != 1 || f.setups[0] == nil {
		t.Fatalf("captured ISetup collection = %v, want one built ISetup", f.setups)
	}
}
