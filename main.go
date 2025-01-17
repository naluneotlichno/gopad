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

// 🔥 registerHandlers регистрирует все хендлеры
func registerHandlers() {
	http.HandleFunc("/api/task", api.AddTaskHandler) 
	http.HandleFunc("/api/nextdate", api.HandleNextDate) 
	http.Handle("/", http.FileServer(http.Dir("./web"))) 
}

// 🔥 startServer запускает сервер
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
