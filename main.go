package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/naluneotlichno/FP-GO-API/database"
	"github.com/naluneotlichno/FP-GO-API/nextdate"
)

func main() {
	log.Println("✅ 🔥 Запускаем нашего монстра!")

	dbPath := getDBPath()
	if err := database.InitDB(dbPath); err != nil {
		log.Fatalf("❌ Ошибка повторной инициализации БД (в main): %v", err)
	}

	log.Println("✅ Регистрируем обработчик для /api/nextdate")
	http.HandleFunc("/api/nextdate", handleNextDate)

	startServer()
}

func handleNextDate(w http.ResponseWriter, r *http.Request) {
	log.Println("✅ Запрос на расчет даты получен!")

	nowStr := r.FormValue("now")    // Получаем "now" из запроса
	dateStr := r.FormValue("date")  // Получаем "date"
	repeat := r.FormValue("repeat") // Получаем "repeat"

	// ✅ Проверяем и парсим `now`
	now, err := time.Parse("20060102", nowStr)
	if err != nil {
		http.Error(w, "Некорректная дата now", http.StatusBadRequest)
		return
	}

	// ✅ Вызываем NextDate(), которая должна рассчитать следующую дату
	nextDate, err := nextdate.NextDate(now, dateStr, repeat)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// ✅ Отправляем пользователю ответ в нужном формате
	fmt.Fprint(w, nextDate)
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
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir(webDir))))

	port := os.Getenv("TODO_PORT")
	if port == "" {
		port = "7540"
	}

	log.Printf("✅ 🚀 Сервер выезжает на порт %s. Подрубаемся!", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalf("❌ Ой-ой, сервер упал: %v", err)
	}
}
