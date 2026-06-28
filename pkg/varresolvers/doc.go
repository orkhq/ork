// Package varresolvers resolves manifest interpolation references from inputs,
// environment variables, and component outputs.
//
// Callers choose which resolver chain is valid for a lifecycle phase. State
// backend configuration, for example, is resolved before component outputs
// exist and therefore cannot reference them.
package varresolvers
