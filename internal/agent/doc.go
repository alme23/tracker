// Package agent provides the client-side logic for the tracking system.
//
// The agent is responsible for:
//   - Collecting system information via the collector package
//   - Serializing data to binary format via the binproto package
//   - Sending encrypted data to the server via the secproto package
//
// The agent is designed to run once when a user logs in:
//  1. Collect system snapshot
//  2. Serialize to binary format
//  3. Encrypt and send to server
//  4. Exit
//
// Example usage:
//
//	cfg := agent.Config{
//		ServerAddr:   "server.example.com:8443",
//		SharedSecret: "your-secret-key",
//		Timeout:      10 * time.Second,
//	}
//
//	a := agent.New(cfg)
//	if err := a.RunOnce(context.Background()); err != nil {
//		log.Fatal(err)
//	}
package agent
