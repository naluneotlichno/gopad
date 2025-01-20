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
	// Один маршрут /api/task, но разные методы внутри
	http.HandleFunc("/api/task", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		// Обработка добавления задачи (POST)
		case http.MethodPost:
			api.AddTaskHandler(w, r)
		// Получение задачи по ID (GET)
		case http.MethodGet:
			api.GetTaskHandler(w, r)
		// Обновление задачи (PUT)
		case http.MethodPut:
			api.UpdateTaskHandler(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// Получение всех задач (GET)
	http.HandleFunc("/api/tasks", api.GetTasksHandler)

	// Для отдачи статики (веб-страницы)
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
