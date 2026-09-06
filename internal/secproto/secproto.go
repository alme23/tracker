// Package secproto provides a lightweight, high-performance binary data transport protocol
// built on top of raw TCP sockets using ephemeral X25519 Diffie-Hellman key exchange (KEX)
// and AES-256-GCM authenticated encryption to guarantee Perfect Forward Secrecy (PFS).
package secproto

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdh"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
)

// maxPacketSize limits data packet size to prevent OOM / Slowloris
const maxPacketSize = 1 * 1024 * 1024

var (
	// ErrConnectionFailed occurs when the client fails to establish a TCP connection to the server.
	ErrConnectionFailed = errors.New("connection failed")

	// ErrGenerateECDHKeyFailed occurs when the ephemeral X25519 key generation fails.
	ErrGenerateECDHKeyFailed = errors.New("failed to generate ecdh key")

	// ErrSendClientKeyFailed occurs when the client fails to write its masked key to the socket.
	ErrSendClientKeyFailed = errors.New("failed to send client key")

	// ErrReadServerKeyFailed occurs when the client fails to read the server's masked key.
	ErrReadServerKeyFailed = errors.New("failed to read server key")

	// ErrInvalidServerPublicKey occurs when the server's unmasked key is mathematically invalid or compromised.
	ErrInvalidServerPublicKey = errors.New("invalid server public key (wrong secret or MitM!)")

	// ErrECDHComputeFailed occurs when the Diffie-Hellman shared secret calculation fails.
	ErrECDHComputeFailed = errors.New("ecdh compute failed")

	// ErrAESInitFailed occurs when the block cipher initialization fails with the derived key.
	ErrAESInitFailed = errors.New("aes init failed")

	// ErrCipherInitFailed occurs when the Galois/Counter Mode (GCM) cipher wrapping fails.
	ErrCipherInitFailed = errors.New("cipher init failed")

	// ErrNonceGenerationFailed occurs when the crypto/rand stream fails to produce a secure token.
	ErrNonceGenerationFailed = errors.New("nonce generation failed")

	// ErrSendPayloadFailed occurs when the encrypted ciphertext fails to transmit over the network.
	ErrSendPayloadFailed = errors.New("failed to send payload")

	// ErrReadClientKeyFailed occurs when the server fails to read the initial handshake packet.
	ErrReadClientKeyFailed = errors.New("failed to read client key")

	// ErrGenerateServerKeyFailed occurs when the server fails to generate its own ephemeral X25519 keys.
	ErrGenerateServerKeyFailed = errors.New("failed to generate server ecdh key")

	// ErrSendServerKeyFailed occurs when the server fails to write its handshake response.
	ErrSendServerKeyFailed = errors.New("failed to send server key")

	// ErrInvalidClientPublicKey occurs when the client's unmasked key is corrupted or mathematically invalid.
	ErrInvalidClientPublicKey = errors.New("invalid client public key (wrong secret or MitM!)")

	// ErrReadPayloadFailed occurs when the encrypted data block transmission is interrupted.
	ErrReadPayloadFailed = errors.New("failed to read payload")

	// ErrDecryptionFailed occurs when the authentication tag mismatch is detected (wrong password or MitM).
	ErrDecryptionFailed = errors.New("decryption failed (wrong secret string or active MitM attack!)")

	// ErrWriteLengthPrefixFailed occurs when writing the 4-byte stream framework header fails.
	ErrWriteLengthPrefixFailed = errors.New("failed to write length prefix")

	// ErrWriteBodyFailed occurs when writing the core byte payload fails.
	ErrWriteBodyFailed = errors.New("failed to write body")

	// ErrReadLengthPrefixFailed occurs when reading the 4-byte stream framework header fails.
	ErrReadLengthPrefixFailed = errors.New("failed to read length prefix")

	// ErrReadFullBodyFailed occurs when the incoming network stream terminates unexpectedly mid-packet.
	ErrReadFullBodyFailed = errors.New("failed to read full body")

	// ErrPackageSizeTooLarge indicates a potential DoS/OOM attack where the length prefix exceeds 1MB.
	ErrPackageSizeTooLarge = errors.New("package size too large (max 1MB)")

	// ErrPayloadTooShort indicates the incoming data block is smaller than the minimum AES-GCM nonce size.
	ErrPayloadTooShort = errors.New("payload too short")
)

// Send performs a lightweight ephemeral X25519 Diffie-Hellman key exchange (KEX)
// over a short-lived TCP connection, derives a unique session key providing Perfect Forward Secrecy (PFS),
// and sends the raw payload bytes encrypted with AES-256-GCM.
func Send(ctx context.Context, serverAddr, sharedSecret string, payload []byte) error {
	dialer := &net.Dialer{}
	conn, err := dialer.DialContext(ctx, "tcp", serverAddr)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrConnectionFailed, err)
	}
	defer func() {
		_ = conn.Close()
	}()

	curve := ecdh.X25519()
	clientPriv, err := curve.GenerateKey(rand.Reader)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrGenerateECDHKeyFailed, err)
	}
	clientPubBytes := clientPriv.PublicKey().Bytes()

	maskedClientPub := xorBytes(clientPubBytes, sharedSecret)

	if err := writeLength(conn, maskedClientPub); err != nil {
		return fmt.Errorf("%w: %w", ErrSendClientKeyFailed, err)
	}

	maskedServerPub, err := readLength(conn)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrReadServerKeyFailed, err)
	}

	serverPubBytes := xorBytes(maskedServerPub, sharedSecret)
	serverPubKey, err := curve.NewPublicKey(serverPubBytes)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrInvalidServerPublicKey, err)
	}

	ecdhSecret, err := clientPriv.ECDH(serverPubKey)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrECDHComputeFailed, err)
	}

	sessionKey := sha256.Sum256(ecdhSecret)

	block, err := aes.NewCipher(sessionKey[:])
	if err != nil {
		return fmt.Errorf("%w: %w", ErrAESInitFailed, err)
	}
	aesCipher, err := cipher.NewGCM(block)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrCipherInitFailed, err)
	}

	nonce := make([]byte, aesCipher.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return fmt.Errorf("%w: %w", ErrNonceGenerationFailed, err)
	}
	encryptedData := aesCipher.Seal(nonce, nonce, payload, nil)

	if err := writeLength(conn, encryptedData); err != nil {
		return fmt.Errorf("%w: %w", ErrSendPayloadFailed, err)
	}

	return nil
}

// HandleConnection processes an incoming TCP connection on the server side,
// executes the obfuscated X25519 key exchange, derives the identical PFS session key,
// and decrypts the network payload via AES-GCM, returning the raw decrypted bytes.
func HandleConnection(conn net.Conn, sharedSecret string) ([]byte, error) {
	curve := ecdh.X25519()

	maskedClientPub, err := readLength(conn)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrReadClientKeyFailed, err)
	}

	serverPriv, err := curve.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrGenerateServerKeyFailed, err)
	}
	serverPubBytes := serverPriv.PublicKey().Bytes()
	maskedServerPub := xorBytes(serverPubBytes, sharedSecret)

	if err := writeLength(conn, maskedServerPub); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrSendServerKeyFailed, err)
	}

	clientPubBytes := xorBytes(maskedClientPub, sharedSecret)
	clientPubKey, err := curve.NewPublicKey(clientPubBytes)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInvalidClientPublicKey, err)
	}

	ecdhSecret, err := serverPriv.ECDH(clientPubKey)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrECDHComputeFailed, err)
	}
	sessionKey := sha256.Sum256(ecdhSecret)

	encryptedPayload, err := readLength(conn)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrReadPayloadFailed, err)
	}

	block, err := aes.NewCipher(sessionKey[:])
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrAESInitFailed, err)
	}
	aesCipher, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrCipherInitFailed, err)
	}

	nonceSize := aesCipher.NonceSize()
	if len(encryptedPayload) < nonceSize {
		return nil, fmt.Errorf("%w: package metrics payload is broken", ErrPayloadTooShort)
	}
	nonce, cipherText := encryptedPayload[:nonceSize], encryptedPayload[nonceSize:]

	decryptedBytes, err := aesCipher.Open(nil, nonce, cipherText, nil)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrDecryptionFailed, err)
	}

	return decryptedBytes, nil
}

// xorBytes masks or unmasks the public key bytes using a SHA-256 hash of the shared secret.
// This obfuscates the ECDH handshake, making the network traffic look like pseudorandom noise.
func xorBytes(data []byte, secret string) []byte {
	salt := sha256.Sum256([]byte(secret))
	result := make([]byte, len(data))
	for i := range data {
		result[i] = data[i] ^ salt[i%len(salt)]
	}
	return result
}

// writeLength writes a 4-byte BigEndian length prefix followed by the actual message body
// into the underlying writer, ensuring safe framing over the TCP stream.
func writeLength(w io.Writer, msg []byte) error {
	if len(msg) > maxPacketSize {
		return fmt.Errorf("%w: payload size %d exceeds limit of %d bytes",
			ErrWriteBodyFailed, len(msg), maxPacketSize)
	}

	// #nosec G115 // ignore overflow check, limited by maxPacketSize
	length := uint32(len(msg))

	if err := binary.Write(w, binary.BigEndian, length); err != nil {
		return fmt.Errorf("%w: %w", ErrWriteLengthPrefixFailed, err)
	}
	if _, err := w.Write(msg); err != nil {
		return fmt.Errorf("%w: %w", ErrWriteBodyFailed, err)
	}
	return nil
}

// readLength reads a 4-byte BigEndian length prefix, validates it against maxPacketSize
// to prevent OOM/Slowloris attacks, and blocks until the full message body is fetched from the stream.
func readLength(r io.Reader) ([]byte, error) {
	var length uint32
	if err := binary.Read(r, binary.BigEndian, &length); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrReadLengthPrefixFailed, err)
	}

	if length > maxPacketSize {
		return nil, fmt.Errorf("%w: current size is %d bytes", ErrPackageSizeTooLarge, length)
	}

	buf := make([]byte, length)
	if _, err := io.ReadFull(r, buf); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrReadFullBodyFailed, err)
	}
	return buf, nil
}
