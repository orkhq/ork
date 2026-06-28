// Package runners defines execution targets for component operations.
//
// A runner is both a location and an identity boundary. Adapters must execute
// through the selected runner instead of performing equivalent work from the
// local Ork process.
package runners
