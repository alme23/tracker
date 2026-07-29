package main

import (
	"bytes"
	"compress/gzip"
	"encoding/gob"
	"fmt"
	"io"
	"log"
	"net"
	"strings"
	"time"

	"github.com/alme23/tracker/internal/crypto"
	"github.com/alme23/tracker/pkg/collector"
	// Импортируем ваш крипто-модуль и пакет сбора данных
)

func main() {
	// 1. Одноразовый сбор метрик низкоуровневыми системными вызовами Win32
	metrics, err := collector.CollectMetrics()
	if err != nil {
		// Так как у агента отключено окно консоли (флаг -H=windowsgui),
		// логи завершения запишутся в системный лог или сработают при ручном вызове из cmd.
		log.Printf("[Агент] Критическая ошибка сбора метрик: %v", err)
		return
	}

	// 2. Динамически определяем текущий домен для поиска SRV-записи
	domainName := strings.ToLower(strings.TrimSpace(metrics.Host.DomainName))

	// Если ПК находится вне домена (в обычной WORKGROUP), сразу прекращаем работу
	if domainName == "" || domainName == "workgroup" || domainName == "n/a" {
		log.Printf("[Агент] Работа завершена: компьютер находится вне домена (Текущий статус: %s)", domainName)
		return // Безопасный и бесшумный выход из приложения
	}

	// 3. Запрашиваем адрес и порт сервера через DNS SRV-запись домена
	// Ищет запись вида: _tracker._tcp.rogsibal.ru (или вашего целевого домена)
	_, addrs, err := net.LookupSRV("tracker", "tcp", domainName)
	if err != nil || len(addrs) == 0 {
		log.Printf("[Агент] Работа завершена: в DNS домена %s не найдена SRV-запись _tracker._tcp", domainName)
		return // Завершаем работу, если инфраструктурная запись в DNS еще не создана
	}

	// 4. Строим валидную строку подключения на основе приоритетного ответа DNS
	// Очищаем потенциальную терминирующую точку на конце хоста, которую любят возвращать Windows DNS
	targetHost := strings.TrimSuffix(addrs[0].Target, ".")
	targetPort := addrs[0].Port
	serverAddress := net.JoinHostPort(targetHost, fmt.Sprintf("%d", targetPort))

	// 5. Криптография: Генерация временной пары ECDH-P256 ключей для текущей сессии
	agentPrivKey, agentPubKeyBytes, err := crypto.GenerateECDHKeypair()
	if err != nil {
		log.Printf("[Агент] Критическая ошибка генерации крипто-ключей: %v", err)
		return
	}

	// 6. Одноразовое TCP-подключение к динамически найденному серверу трекера
	dialer := net.Dialer{Timeout: 7 * time.Second}
	conn, err := dialer.Dial("tcp", serverAddress)
	if err != nil {
		log.Printf("[Агент] Работа завершена: сервер %s найден в DNS, но сейчас недоступен: %v", serverAddress, err)
		return // Выходим, если сервер выключен или закрыт брандмауэром
	}
	defer conn.Close() // Гарантированно закрываем сокет после передачи данных

	// 7. Сетевое рукопожатие: отправляем свой открытый ключ серверу в сырой сокет
	if _, err := conn.Write(agentPubKeyBytes); err != nil {
		log.Printf("[Агент] Ошибка отправки публичного ключа: %v", err)
		return
	}

	// 8. Сетевое рукопожатие: считываем из сокета ответный открытый ключ сервера (65 байт для P-256)
	serverPubKeyBytes := make([]byte, 65)
	if _, err := io.ReadFull(conn, serverPubKeyBytes); err != nil {
		log.Printf("[Агент] Ошибка получения публичного ключа сервера: %v", err)
		return
	}

	// 9. Математическое вычисление общего сессионного ключа на основе ECDH-алгоритма
	sessionKey, err := crypto.DeriveSharedSecret(agentPrivKey, serverPubKeyBytes)
	if err != nil {
		log.Printf("[Агент] Ошибка вычисления общего секрета сессии: %v", err)
		return
	}

	// 10. Бинарная сериализация структуры метрик в поток GOB
	// Ваши кастомные типы MACAddress и IPAddress внутри пакета автоматически закодируются методами GobEncode
	var gobBuf bytes.Buffer
	encoder := gob.NewEncoder(&gobBuf)
	if err := encoder.Encode(metrics); err != nil {
		log.Printf("[Агент] Ошибка сериализации GOB: %v", err)
		return
	}

	// 11. Сжатие бинарного потока GOB через архиватор Gzip для экономии сетевого трафика
	var gzipBuf bytes.Buffer
	gzipWriter := gzip.NewWriter(&gzipBuf)
	if _, err := gzipWriter.Write(gobBuf.Bytes()); err != nil {
		log.Printf("[Агент] Ошибка компрессии Gzip: %v", err)
		return
	}
	// Обязательно закрываем writer, чтобы зафиксировать и сбросить все буферы сжатия в массив gzipBuf
	_ = gzipWriter.Close()

	// 12. Симметричное шифрование сжатого пакета сессионным ключом по стандарту AES-256-GCM
	// Внутрь пакета автоматически запишется уникальный криптографический случайный вектор инициализации (nonce)
	encryptedData, err := crypto.Encrypt(sessionKey, gzipBuf.Bytes())
	if err != nil {
		log.Printf("[Агент] Ошибка аутентифицированного шифрования пакета: %v", err)
		return
	}

	// 13. Отправка зашифрованного тела данных в сетевой канал
	_, err = conn.Write(encryptedData)
	if err != nil {
		log.Printf("[Агент] Ошибка передачи зашифрованного пакета в сокет: %v", err)
		return
	}

	// Задача успешно выполнена, сессия закрыта, память очищена
}
