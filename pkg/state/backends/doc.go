// Package statebackends implements persistence for Ork environment state.
//
// Backends run in the Ork control process, not on component runners. Their
// authentication context is therefore intentionally independent from runner
// provider authentication.
package statebackends
