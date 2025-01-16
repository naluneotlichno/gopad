package main

import (
	"log"
	"net/http"
	"os"

	"github.com/naluneotlichno/FP-GO-API/api"
	"github.com/naluneotlichno/FP-GO-API/database"
)

func main() {
	log.Println("✅ 🔥 Запускаем нашего монстра!")

	// ✅ Инициализация базы данных
	if err := database.InitDB(database.GetDBPath()); err != nil {
		log.Fatalf("❌ Ошибка инициализации БД: %v", err)
	}

	// ✅ Регистрация хендлеров
	registerHandlers()

	// ✅ Запуск сервера
	startServer()
}

// registerHandlers регистрирует все хендлеры
func registerHandlers() {
	http.HandleFunc("/api/nextdate", api.HandleNextDate)        // Регистрация обработчика NextDate
	http.Handle("/static/", http.FileServer(http.Dir("./web"))) // Статика
}

// startServer запускает сервер
func startServer() {
	port := os.Getenv("TODO_PORT")
	if port == "" {
		port = "7540"
	}

	log.Printf("✅ 🚀 Сервер выезжает на порт %s. Подрубаемся!", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalf("❌ Ой-ой, сервер упал: %v", err)
	}
}
