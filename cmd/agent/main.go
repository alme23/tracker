package main

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/alme23/tracker/internal/collector"
)

func main() {
	startTime := time.Now()
	log.Println("Запуск агента сбора данных...")

	// Инициализируем единый оркестратор
	sysCollector := collector.NewSystemCollector(1 * time.Second)

	// Вызываем один метод, который делает всю тяжелую работу параллельно
	snapshot, err := sysCollector.CollectAll()
	if err != nil {
		log.Fatalf("Критическая ошибка при сборе метрик: %v", err)
	}

	// Форматируем результат в JSON для вывода в консоль
	jsonData, err := json.MarshalIndent(snapshot.RAM, "", "  ")
	if err != nil {
		log.Fatalf("Ошибка маршалинга JSON: %v", err)
	}

	fmt.Println("\n=== СЛЕПОК СИСТЕМЫ (JSON) ===")
	fmt.Println(string(jsonData))

	log.Printf("Сбор успешно завершен за %v\n", time.Since(startTime))
}
