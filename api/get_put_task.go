package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"
)

type Task struct {
	ID      string `json:"id"`
	Date    string `json:"date"`
	Title   string `json:"title"`
	Comment string `json:"comment"`
	Repeat  string `json:"repeat"`
}

// Функция для получения задачи по ID
func GetTaskByID(id string) (Task, error) {
	// Здесь должна быть логика получения задачи из базы данных
	// Для примера возвращаем пустую задачу или ошибку, если не найдена
	if id == "unknown" {
		return Task{}, errors.New("task not found")
	}
	return Task{
		ID:      id,
		Date:    "20240201",
		Title:   "Подвести итог",
		Comment: "",
		Repeat:  "d 5",
	}, nil
}

// Функция для обновления задачи
func UpdateTask(task Task) error {
	// Здесь должна быть логика обновления задачи в базе данных
	// Для примера просто возвращаем nil, как будто обновили задачу успешно
	if task.ID == "" {
		return errors.New("task not found")
	}

	// Если задача была найдена и обновлена
	return nil
}

// Обработчик GET-запроса для получения задачи по ID
func GetTaskHandler(w http.ResponseWriter, r *http.Request) {
	// Получаем id из параметров
	id := r.URL.Query().Get("id")
	if id == "" {
		http.Error(w, `{"error": "Не указан идентификатор"}`, http.StatusBadRequest)
		return
	}

	// Получаем задачу из базы данных
	task, err := GetTaskByID(id)
	if err != nil {
		http.Error(w, `{"error": "Задача не найдена"}`, http.StatusNotFound)
		return
	}

	// Возвращаем задачу в формате JSON
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(task)
}

// Обработчик PUT-запроса для обновления задачи
func UpdateTaskHandler(w http.ResponseWriter, r *http.Request) {
	var task Task
	// Декодируем JSON тело запроса
	if err := json.NewDecoder(r.Body).Decode(&task); err != nil {
		http.Error(w, `{"error": "Ошибка при чтении данных"}`, http.StatusBadRequest)
		return
	}

	// Проверяем, что ID передан
	if task.ID == "" {
		http.Error(w, `{"error": "Не указан идентификатор"}`, http.StatusBadRequest)
		return
	}

	// Проверка на корректность даты
	now := time.Now().Format(`20060102`)
	if task.Date < now {
		http.Error(w, `{"error": "Дата не может быть меньше сегодняшней"}`, http.StatusBadRequest)
		return
	}

	// Пытаемся обновить задачу в базе данных
	err := UpdateTask(task)
	if err != nil {
		http.Error(w, `{"error": "Задача не найдена"}`, http.StatusNotFound)
		return
	}

	// Если задача обновлена, возвращаем пустой JSON
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte("{}"))
}
