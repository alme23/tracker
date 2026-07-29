package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdh"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"io"
)

// GenerateECDHKeypair создает пару из закрытого и открытого ключей для участника сессии
func GenerateECDHKeypair() (*ecdh.PrivateKey, []byte, error) {
	priv, err := ecdh.P256().GenerateKey(rand.Reader)
	if err != nil {
		return nil, nil, err
	}
	// Возвращаем объект закрытого ключа и сырые байты открытого ключа для отправки в сеть
	return priv, priv.PublicKey().Bytes(), nil
}

// DeriveSharedSecret вычисляет общий симметричный ключ на основе чужого открытого и своего закрытого ключа
func DeriveSharedSecret(privateKey *ecdh.PrivateKey, remotePublicKeyBytes []byte) ([]byte, error) {
	remotePub, err := ecdh.P256().NewPublicKey(remotePublicKeyBytes)
	if err != nil {
		return nil, fmt.Errorf("invalid remote public key: %w", err)
	}

	// Математическое вычисление общего секрета (Shared Secret)
	secret, err := privateKey.ECDH(remotePub)
	if err != nil {
		return nil, fmt.Errorf("ecdh derivation failed: %w", err)
	}

	// Пропускаем секрет через SHA-256, чтобы получить идеальный 32-байтовый ключ для AES-256
	hashedKey := sha256.Sum256(secret)
	return hashedKey[:], nil
}

// Encrypt выполняет аутентифицированное шифрование AES-256-GCM сессионным ключом
func Encrypt(sessionKey []byte, plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(sessionKey)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	return gcm.Seal(nonce, nonce, plaintext, nil), nil
}

// Decrypt выполняет дешифрование и проверку целостности сессионным ключом
func Decrypt(sessionKey []byte, ciphertext []byte) ([]byte, error) {
	block, err := aes.NewCipher(sessionKey)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}

	nonce, actualCiphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	return gcm.Open(nil, nonce, actualCiphertext, nil)
}
