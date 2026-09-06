// Package server provides the server-side logic for the tracking system.
//
// The server is responsible for:
//   - Listening for incoming TCP connections from agents
//   - Decrypting data via the secproto package
//   - Decoding binary data via the binproto package
//   - Storing data in SQLite via the Storage type
//   - Managing user sessions and computer inventory
//   - Generating alerts for low disk space and high RAM usage
//
// The server supports graceful shutdown on SIGINT/SIGTERM signals.
//
// Example usage:
//
//	cfg := server.Config{
//		ListenAddr:   ":8443",
//		SharedSecret: "your-secret-key",
//		DBPath:       "tracker.db",
//	}
//
//	srv, err := server.New(cfg)
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	if err := srv.RunWithSignals(); err != nil {
//		log.Fatal(err)
//	}
package server
