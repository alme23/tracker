package main

import (
	"bytes"
	"compress/gzip"
	"encoding/gob"
	"fmt"
	"io"
	"log"
	"net"

	// Обновленные пути импорта
	"github.com/alme23/tracker/internal/crypto"
	"github.com/alme23/tracker/internal/protocol"
)

func main() {
	listener, err := net.Listen("tcp", ":9000")
	if err != nil {
		log.Fatalf("Ошибка старта сервера: %v", err)
	}
	defer listener.Close()
	fmt.Println("Сервер tracker запущен на порту :9000...")

	for {
		conn, err := listener.Accept()
		if err != nil {
			continue
		}
		go handle(conn)
	}
}

func handle(conn net.Conn) {
	defer conn.Close()

	encrypted, err := io.ReadAll(conn)
	if err != nil {
		return
	}

	decryptedGzip, err := crypto.Decrypt(encrypted)
	if err != nil {
		log.Printf("Ошибка дешифрования: %v", err)
		return
	}

	r, err := gzip.NewReader(bytes.NewReader(decryptedGzip))
	if err != nil {
		return
	}
	defer r.Close()

	var packet protocol.DataPacket
	if err := gob.NewDecoder(r).Decode(&packet); err != nil {
		log.Printf("Ошибка декодирования GOB: %v", err)
		return
	}

	fmt.Printf("[Сервер] Получены метрики от хоста: %s\n", packet.Hostname)
}
