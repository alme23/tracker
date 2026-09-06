// Package binproto provides binary serialization for SystemSnapshot.
//
// The package implements a compact binary format for efficient data transfer
// between agent and server. The format uses little-endian encoding and
// includes:
//
//   - Magic header "TRCK1" for version identification
//   - 2-byte length-prefixed strings
//   - Single-byte boolean flags
//   - Fixed-size integers for numeric fields
//
// The package provides:
//   - Encoder — serializes SystemSnapshot to binary format
//   - Decoder — deserializes binary data back to SystemSnapshot
//
// Example usage:
//
//	// Encoding
//	encoder := binproto.NewEncoder()
//	data, err := encoder.Encode(snapshot)
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	// Decoding
//	decoder := binproto.NewDecoder(data)
//	snapshot, err := decoder.Decode()
//	if err != nil {
//		log.Fatal(err)
//	}
package binproto
