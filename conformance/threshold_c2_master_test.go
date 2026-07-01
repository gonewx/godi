//go:build !atdd_redphase

// Story 6.5 (AC6) — Second-tier (C-2) threshold grand-total ledger.
//
// Before this file, each second-tier gate kept its own accounting in isolation
// (threshold_m2 / csf / spct / async / ext / actutil) with NO master total.
// This gate is the milestone ledger for the C-2 threshold: it sums the six
// gates' declared method maps and asserts the running total, then documents,
// in-code, how that total relates to the acceptance-contract's "~220" estimate.
//
// NOT tagged atdd_redphase — compiles in the default build as a live ledger.
//
// ── D-8 (honest accounting, do NOT pad to 220) ──────────────────────────────
// acceptance-contract.md lines 57-67 record "~220" — an order-of-magnitude
// estimate written during SPEC drafting, with line 66 folding unlisted files
// (CallSiteTests / ServiceCollection*Test / ...) under an ellipsis. The precise
// count is decided by the actual port, file-by-file. The six gates below sum to
// a concrete number; this test asserts THAT number and records the delta vs the
// "~220" estimate as a t.Logf note. The C-2 pass criterion is "every
// high-value file fully dispositioned (covered/adapted/na, no MISSING)", NOT
// "the number equals 220 exactly".
package conformance

import "testing"

// c2Gate names a second-tier gate and the live length of its method map.
type c2Gate struct {
	Name       string
	DotnetFile string
	Count      int
}

// c2SecondTierGates references the ACTUAL slices declared by the sibling gate
// files (same package), so the grand total tracks them automatically: editing
// any gate's map changes its len() and this ledger re-computes on the next run.
func c2SecondTierGates() []c2Gate {
	return []c2Gate{
		{"threshold_m2", "ServiceProviderValidationTests + CircularDependencyTests",
			len(dotnetServiceProviderValidationMap) + len(dotnetCircularDependencyMap)},
		{"threshold_csf", "CallSiteFactoryTest + CallSiteTests", len(dotnetCSFMethodMap)},
		{"threshold_spct", "ServiceProviderContainerTests", len(dotnetSPCTMethodMap)},
		{"threshold_async", "AsyncServiceScopeTests", len(dotnetAsyncScopeMethodMap)},
		{"threshold_ext", "ServiceProviderServiceExtensionsTest + ServiceCollection*Test", len(dotnetExtMethodMap)},
		{"threshold_actutil", "DI.Tests/ActivatorUtilitiesTests (impl, 17+3)", len(dotnetActUtilMethodMap)},
	}
}

// TestC2_SecondTier_220_GrandTotal sums the six second-tier gates and asserts
// the concrete running total. The "~220" from the acceptance-contract is
// recorded as an informational delta — NOT enforced as an equality (D-8).
func TestC2_SecondTier_220_GrandTotal(t *testing.T) {
	// Concrete per-gate expectations (guards against a silent gate-map edit
	// changing the grand total without anyone noticing).
	wantPerGate := map[string]int{
		"threshold_m2":      36, // 26 + 10
		"threshold_csf":     47,
		"threshold_spct":    49,
		"threshold_async":   6,
		"threshold_ext":     55,
		"threshold_actutil": 20, // 17 Fact + 3 Theory (impl layer)
	}

	// grandTotal is the actual, honest sum of the six gates.
	const grandTotal = 213

	// contractEstimate is the acceptance-contract's "~220" order-of-magnitude
	// figure. Recorded for the delta note only; NOT asserted as equality.
	const contractEstimate = 220

	gates := c2SecondTierGates()
	sum := 0
	for _, g := range gates {
		if want, ok := wantPerGate[g.Name]; ok {
			if g.Count != want {
				t.Errorf("gate %q (%s) declares %d methods, want %d — a gate map changed; "+
					"update wantPerGate + grandTotal and re-verify the C-2 ledger",
					g.Name, g.DotnetFile, g.Count, want)
			}
		} else {
			t.Errorf("gate %q is not in wantPerGate — a new second-tier gate was added "+
				"without updating the C-2 master ledger", g.Name)
		}
		sum += g.Count
		t.Logf("C-2 gate %-20s = %2d  (%s)", g.Name, g.Count, g.DotnetFile)
	}

	if sum != grandTotal {
		t.Fatalf("C-2 second-tier grand total = %d, want %d — sum of the six gates changed; "+
			"reconcile file-by-file against acceptance-contract lines 57-67 before updating grandTotal (D-8)",
			sum, grandTotal)
	}

	// Honest delta note (D-8): 213 vs "~220". The 7-method gap is the difference
	// between the SPEC-drafting order-of-magnitude estimate and the file-by-file
	// port count (e.g. ConditionalTheory methods excluded, and unlisted files
	// folded under the acceptance-contract's ellipsis). NOT padded to 220.
	delta := contractEstimate - sum
	t.Logf("C-2 second-tier grand total: %d methods across %d gates", sum, len(gates))
	t.Logf("acceptance-contract estimate: ~%d  →  delta = %d (order-of-magnitude estimate, NOT padded — D-8)",
		contractEstimate, delta)
	t.Logf("C-2 pass criterion = every gate fully dispositioned (no MISSING), NOT a literal 220 match")
}
