// Package secproto provides a lightweight, high-performance binary data
// transport protocol with strong security guarantees.
//
// The protocol is built on top of raw TCP sockets and provides:
//
//   - Ephemeral X25519 Diffie-Hellman key exchange (KEX)
//   - AES-256-GCM authenticated encryption
//   - Perfect Forward Secrecy (PFS)
//   - Obfuscated handshake to resist traffic analysis
//
// Main functions:
//   - Send — client-side: encrypts and sends payload to server
//   - HandleConnection — server-side: receives and decrypts payload
//
// The protocol uses a shared secret to mask public keys during the
// handshake, making the traffic look like pseudorandom noise.
//
// Security features:
//   - Each connection generates unique ephemeral keys
//   - AES-GCM provides both confidentiality and integrity
//   - Maximum packet size limited to 1MB to prevent DoS attacks
//
// Example client:
//
//	err := secproto.Send(ctx, "server:8443", "shared-secret", payload)
//
// Example server:
//
//	data, err := secproto.HandleConnection(conn, "shared-secret")
package secproto
