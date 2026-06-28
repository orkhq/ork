// Package state defines persisted environment lifecycle data, tool-state
// artifacts, and the backend-independent state manager.
//
// State is operational recovery data, not a secret store. Sensitive outputs and
// runner credentials must not be persisted in these models.
package state
