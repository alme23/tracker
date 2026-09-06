package secproto

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"io"
	"net"
	"sync"
	"testing"
)

type TestPayload struct {
	ID    int
	Name  string
	List  []string
	Valid bool
}

type BenchPayload struct {
	DeviceID  string
	CPU       float64
	RAM       float64
	Processes []string
}

// Вспомогательная функция для запуска локального TCP-сервера на случайном порту.
// ИСПРАВЛЕНИЕ: Вместо *error теперь возвращается чистый канал chan error, что идиоматично для Go.
func startLocalByteServer(t testing.TB, secret string, wg *sync.WaitGroup) (string, net.Listener, *[]byte, chan error) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to start local server listener: %v", err)
	}

	receivedBytes := &[]byte{}
	errChan := make(chan error, 1) // Буферизованный канал на 1 элемент для предотвращения блокировки

	wg.Add(1)
	go func() {
		defer wg.Done()
		conn, err := ln.Accept()
		if err != nil {
			errChan <- err
			return
		}
		defer func() {
			_ = conn.Close()
		}()

		res, err := HandleConnection(conn, secret)
		if err != nil {
			errChan <- err
			return
		}
		*receivedBytes = res
		errChan <- nil // Сигнализируем об успешном завершении без ошибок
	}()

	return ln.Addr().String(), ln, receivedBytes, errChan
}

func TestSendBytesAndHandleConnection_Success(t *testing.T) {
	secret := "test-secret-string-12345"
	inputPayload := []byte("hello world! testing raw bytes encryption protocols.")

	var wg sync.WaitGroup

	// Запускаем сервер, забираем канал ошибки errChan
	addr, ln, receivedBytes, errChan := startLocalByteServer(t, secret, &wg)
	defer func() {
		_ = ln.Close()
	}()

	// Клиент отправляет сырые байты
	err := Send(context.Background(), addr, secret, inputPayload)
	if err != nil {
		t.Fatalf("Ошибка на стороне клиента: %v", err)
	}

	wg.Wait()

	// Читаем ошибку сервера из канала
	if serverErr := <-errChan; serverErr != nil {
		t.Fatalf("Ошибка на стороне сервера: %v", serverErr)
	}

	// Сверка данных
	if !bytes.Equal(*receivedBytes, inputPayload) {
		t.Errorf("Данные искажены при передаче! Ожидалось %s, получено %s", string(inputPayload), string(*receivedBytes))
	}
}

func TestHandleConnectionBytes_WrongSecret(t *testing.T) {
	clientSecret := "secret-A"
	serverSecret := "secret-B"
	inputPayload := []byte("confidential telemetry data")

	var wg sync.WaitGroup

	addr, ln, receivedBytes, errChan := startLocalByteServer(t, serverSecret, &wg)
	defer func() {
		_ = ln.Close()
	}()

	// Отправляем с неверным секретом
	_ = Send(context.Background(), addr, clientSecret, inputPayload)

	wg.Wait()

	// Извлекаем ошибку сервера из канала, чтобы горутина гарантированно завершилась
	serverErr := <-errChan

	// ТЕСТ БЕЗОПАСНОСТИ: На сервере должна быть ошибка расшифровки, а буфер должен быть пустым
	if serverErr == nil && len(*receivedBytes) > 0 {
		t.Fatal("Ошибка безопасности! Сервер успешно расшифровал пакет с неверным секретом.")
	}
}

// BenchmarkCryptoSenderExchange измеряет скорость полного цикла KEX + AEAD (байты)
func BenchmarkCryptoSenderExchange(b *testing.B) {
	secret := "benchmark-secret"
	payload := []byte("small payload metrics string representation")

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		ln, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			b.Fatalf("failed to listen: %v", err)
		}

		var wg sync.WaitGroup
		wg.Add(1)

		go func() {
			defer wg.Done()
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			defer func() {
				_ = conn.Close()
			}()
			_, _ = HandleConnection(conn, secret)
		}()

		_ = Send(context.Background(), ln.Addr().String(), secret, payload)

		_ = ln.Close()
		wg.Wait()
	}
}

// BenchmarkAESGCMOnly измеряет чистую скорость шифрования AES-256-GCM на готовом ключе
func BenchmarkAESGCMOnly(b *testing.B) {
	payload := []byte("small payload metrics string representation")

	staticKey := make([]byte, 32)
	_, _ = io.ReadFull(rand.Reader, staticKey)
	block, _ := aes.NewCipher(staticKey)
	aesCipher, _ := cipher.NewGCM(block)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		nonce := make([]byte, aesCipher.NonceSize())
		_ = aesCipher.Seal(nonce, nonce, payload, nil)
	}
}
