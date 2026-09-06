// Command server is the server-side executable for the tracking system.
//
// The server:
//   - Listens for TCP connections from agents
//   - Decrypts and decodes incoming data
//   - Stores data in SQLite
//   - Manages sessions, inventory, metrics, and alerts
//
// Usage:
//
//	server.exe
package main
