package main

import (
	"log"
	"os"
	"net/http"
	"runtime"
	"path/filepath"
	"github.com/naluneotlicno/FP-GO-API/database"
)

func main() {
	log.Println("✅ 🔥 [main()] Запускаем нашего монстра!")

	dbPath := getDBPath()
	if err := database.InitDB(dbPath); err != nil {
		log.Fatalf("❌ Ошибка повторной инициализации БД (в main): %v", err)
	}

	startServer()
}

// getDBPath вычисляет путь к базе данных
func getDBPath() string {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		log.Println("❌ runtime.Caller(0) не сработал, dbPath будет 'scheduler.db'")
		return "scheduler.db"
	}

	baseDir := filepath.Dir(filename)
	dbPath := filepath.Join(baseDir, "scheduler.db")

	if envDB := os.Getenv("TODO_DBFILE"); envDB != "" {
		return envDB
	}

	return dbPath
}

// startServer запускает HTTP-сервер
func startServer() {
	webDir := "./web"
	http.Handle("/", http.FileServer(http.Dir(webDir)))

	port := os.Getenv("TODO_PORT")
	if port == "" {
		port = "7540"
	}

	log.Printf("✅ 🚀 Сервер выезжает на порт %s. Подрубаемся!", port)

	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalf("❌ Ой-ой, сервер упал: %v", err)
	}
}
