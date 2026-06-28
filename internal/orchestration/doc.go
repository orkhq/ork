// Package orchestration coordinates manifest components through apply, state
// capture, recovery, and reverse teardown.
//
// It owns lifecycle policy while delegating tool-specific work to adapters,
// execution to runners, and persistence to state backends.
package orchestration
