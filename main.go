package main

import (
	"log"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/naluneotlichno/FP-GO-API/api"
	"github.com/naluneotlichno/FP-GO-API/database"
)

func main() {
	log.Println("✅ 🔥 Запускаем нашего монстра!")

	// ✅ Инициализация базы данных
	if err := database.InitDB(database.GetDBPath()); err != nil {
		log.Fatalf("❌ Ошибка инициализации БД: %v", err)
	}

	// ✅ Создание маршрутизатора
	r := chi.NewRouter()

	// ✅ Регистрация хендлеров
	registerHandlers()

	// ✅ Подключение файлов /web
	webDir := "./web"
	fileServer := http.FileServer(http.Dir(webDir))
	r.Mount("/", fileServer)

	// ✅ Запуск сервера
	startServer()
}

// 🔥 registerHandlers регистрирует все хендлеры
func registerHandlers(r *chi.Mux) {
	// Добавление задачи (POST)
	r.HandleFunc("/api/task", api.AddTaskHandler) // Для добавления новой задачи (по примеру твоего предыдущего кода)

	// Получение задач (GET)
	r.HandleFunc("/api/task", api.GetTaskHandler) // Для получения задачи по ID (GET запрос)

	// Обновление задачи (PUT)
	r.HandleFunc("/api/task", api.UpdateTaskHandler) // Для обновления задачи (PUT запрос)

	// Обработчик для следующих запросов (например, обработка даты)
	r.HandleFunc("/api/nextdate", api.HandleNextDate)

	// Получение всех задач (GET)
	r.HandleFunc("/api/tasks", api.GetTasksHandler)

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
