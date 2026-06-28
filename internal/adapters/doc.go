// Package adapters defines the component adapter contract and implementations.
//
// Adapters translate Ork lifecycle operations into native tool operations, but
// they do not choose where those operations run. All execution and file access
// must go through the component's runner so remote execution preserves its
// network location and provider identity.
package adapters
