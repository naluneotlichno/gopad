package main

import (
	"log"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/naluneotlichno/FP-GO-API/api"
	"github.com/naluneotlichno/FP-GO-API/database"
	"github.com/naluneotlichno/FP-GO-API/nextdate"
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
	registerHandlers(r)

	// ✅ Подключение файлов /web
	webDir := "./web"
	fileServer := http.FileServer(http.Dir(webDir))
	r.Handle("/*", fileServer)

	// ✅ Запуск сервера
	startServer(r)
}

// 🔥 registerHandlers регистрирует все хендлеры
func registerHandlers(r *chi.Mux) {
	r.Get("/api/nextdate", nextdate.HandleNextDate) // +
	r.Post("/api/task", api.AddTaskHandler)	// +
	r.Get("/api/tasks", api.GetTaskHandler)
	r.Put("/api/task", api.UpdateTaskHandler)
	r.Post("/api/task/done", api.DoneTaskHandler)
	r.Delete("/api/task", api.DeleteTaskHandler) // +
}

// 🔥 startServer запускает сервер
func startServer(r *chi.Mux) {
	port := os.Getenv("TODO_PORT")
	if port == "" {
		port = "7540"
	}

	log.Printf("✅ 🚀 Сервер выезжает на порт %s. Подрубаемся!", port)
	if err := http.ListenAndServe(":"+port, r); err != nil {
		log.Fatalf("❌ Ой-ой, сервер упал: %v", err)
	}
}
