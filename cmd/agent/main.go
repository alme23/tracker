package main

import (
	"bytes"
	"compress/gzip"
	"encoding/gob"
	"log"
	"net"
	"os"

	"github.com/alme23/tracker/internal/crypto"
	"github.com/alme23/tracker/internal/protocol"
)

func main() {
	// 1. Сбор данных на хосте
	hostname, _ := os.Hostname()
	packet := protocol.DataPacket{
		Hostname: hostname,
		OS:       "Linux/Windows",
		CPUUsage: 14.7,
		RAMTotal: 16777216000,
		RAMFree:  12582912000,
	}

	// 2. Сериализация в GOB
	var gobBuf bytes.Buffer
	if err := gob.NewEncoder(&gobBuf).Encode(packet); err != nil {
		log.Fatalf("[Агент] Ошибка кодирования GOB: %v", err)
	}

	// 3. Сжатие в Gzip
	var gzipBuf bytes.Buffer
	w := gzip.NewWriter(&gzipBuf)
	if _, err := w.Write(gobBuf.Bytes()); err != nil {
		log.Fatalf("[Агент] Ошибка сжатия Gzip: %v", err)
	}
	w.Close() // Закрываем, чтобы зафиксировать буфер сжатия

	// 4. Симметричное шифрование AES-GCM
	encrypted, err := crypto.Encrypt(gzipBuf.Bytes())
	if err != nil {
		log.Fatalf("[Агент] Ошибка шифрования: %v", err)
	}

	// 5. Одноразовое подключение и отправка
	conn, err := net.Dial("tcp", "127.0.0.1:9000")
	if err != nil {
		log.Fatalf("[Агент] Не удалось связаться с сервером: %v", err)
	}
	defer conn.Close()

	if _, err := conn.Write(encrypted); err != nil {
		log.Fatalf("[Агент] Ошибка отправки пакета: %v", err)
	}

	log.Println("[Агент] Метрики успешно собраны, зашифрованы и отправлены!")
}
