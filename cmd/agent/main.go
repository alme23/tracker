package main

import (
	"bytes"
	"compress/gzip"
	"encoding/gob"
	"log"
	"net"
	"time"

	"github.com/alme23/tracker/internal/crypto"
	"github.com/alme23/tracker/pkg/collector"
	// Импортируем ваш модуль шифрования и пакет сбора данных
)

func main() {
	// 1. Вызов вашей комплексной функции сбора данных из пакета collector
	metrics, err := collector.CollectMetrics()
	if err != nil {
		// Так как у агента нет окна консоли (флаг -H=windowsgui),
		// лог запишется в Windows Event Log или файл (если настроено).
		log.Fatalf("[Агент] Критическая ошибка сбора метрик: %v", err)
	}

	// 2. Сериализация структуры в бинарный поток GOB
	// Ваши типы MACAddress и IPAddress автоматически сериализуются через свои методы GobEncode
	var gobBuf bytes.Buffer
	encoder := gob.NewEncoder(&gobBuf)
	if err := encoder.Encode(metrics); err != nil {
		log.Fatalf("[Агент] Ошибка сериализации GOB: %v", err)
	}

	// 3. Сжатие бинарного потока через Gzip для минимизации сетевого трафика
	var gzipBuf bytes.Buffer
	gzipWriter := gzip.NewWriter(&gzipBuf)
	if _, err := gzipWriter.Write(gobBuf.Bytes()); err != nil {
		log.Fatalf("[Агент] Ошибка компрессии Gzip: %v", err)
	}
	// Обязательно закрываем writer, чтобы сбросить все буферы сжатия в gzipBuf
	gzipWriter.Close()

	// 4. Симметричное шифрование AES-256-GCM
	// Внутрь автоматически упакуется случайный одноразовый вектор инициализации (nonce)
	encryptedData, err := crypto.Encrypt(gzipBuf.Bytes())
	if err != nil {
		log.Fatalf("[Агент] Ошибка шифрования пакета: %v", err)
	}

	// 5. Одноразовое подключение к серверу с таймаутом на случай проблем с сетью
	// Укажите IP-адрес и порт вашего центрального сервера
	serverAddress := "127.0.0.1:9000"
	dialer := net.Dialer{Timeout: 7 * time.Second}

	conn, err := dialer.Dial("tcp", serverAddress)
	if err != nil {
		log.Fatalf("[Агент] Сервер %s недоступен: %v", serverAddress, err)
	}
	defer conn.Close() // Гарантированно закрываем сокет после отправки данных

	// 6. Пересылаем весь зашифрованный массив байт целиком
	_, err = conn.Write(encryptedData)
	if err != nil {
		log.Fatalf("[Агент] Ошибка отправки пакета в сокет: %v", err)
	}

	// Сессия завершена, агент успешно выполнил свою одноразовую задачу
}
