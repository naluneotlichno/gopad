package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/naluneotlichno/FP-GO-API/database"
)

// 🔥 TaskRequest — структура входного JSON-запроса
type TaskRequest struct {
	Date    string `json:"date"`
	Title   string `json:"title"`
	Comment string `json:"comment,omitempty"`
	Repeat  string `json:"repeat,omitempty"`
}

// 🔥 TaskResponse — структура ответа (id или ошибка)
type TaskResponse struct {
	ID    int64  `json:"id,omitempty"`
	Error string `json:"error,omitempty"`
}

// 🔥 AddTaskHandler обрабатывает POST-запросы на /api/task
func AddTaskHandler(w http.ResponseWriter, r *http.Request) {
	// ✅ Устанавливаем заголовок
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if r.Method != http.MethodPost {
		http.Error(w, `{"error": "Метод не поддерживается"}`, http.StatusMethodNotAllowed)
		return
	}

	// ✅ Декодируем JSON-запрос
	var req TaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error": "Ошибка десериализации JSON"}`, http.StatusBadRequest)
		return
	}

	// ✅ Проверяем обязательные поля
	if req.Title == "" {
		http.Error(w, `{"error": "Не указан заголовок задачи"}`, http.StatusBadRequest)
		return
	}

	// ✅ Если дата пустая — подставляем текущую
	if req.Date == "" {
		req.Date = time.Now().Format("20060102")
	}

	// ✅ Парсим дату, если формат кривой — шлём ошибку
	taskDate, err := time.Parse("20060102", req.Date)
	if err != nil {
		taskDate, err = time.Parse("02.01.2006", req.Date)
		if err != nil {
			http.Error(w, `{"error": "Дата указана некорректно"}`, http.StatusBadRequest)
			return
		}
	}

	// ✅ Проверяем дату на корректность
	req.Date = taskDate.Format("20060102")

	// ✅ Если дата в прошлом — применяем правило повторения
	if taskDate.Before(time.Now()) {
		if req.Repeat != "" {
			nextDate, err := NextDate(time.Now(), req.Date, req.Repeat)
			if err != nil {
				http.Error(w, fmt.Sprintf(`{"error": "Неверный формат правила повторения: %s"}`, err.Error()), http.StatusBadRequest)
				return
			}
			req.Date = nextDate
		} else {
			req.Date = time.Now().Format("20060102")
		}
	}

	// ✅ Подключаемся к базе данных
	db, err := database.GetDB()
	if err != nil {
		http.Error(w, `{"error": "Ошибка подключения к БД"}`, http.StatusInternalServerError)
		return
	}

	// ✅ Вставляем новую задачу в базу
	query := `INSERT INTO scheduler (date, title, comment, repeat) VALUES (?, ?, ?, ?)`
	res, err := db.Exec(query, req.Date, req.Title, req.Comment, req.Repeat)
	if err != nil {
		http.Error(w, `{"error": "Ошибка записи в БД"}`, http.StatusInternalServerError)
		return
	}

	// ✅ Получаем ID новой задачи
	taskID, err := res.LastInsertId()
	if err != nil {
		http.Error(w, `{"error": "Ошибка получения ID записи"}`, http.StatusInternalServerError)
		return
	}

	// ✅ Возвращаем JSON-ответ в формате, который ожидает тест
	resp := TaskResponse{ID: taskID}
	json.NewEncoder(w).Encode(resp)
}
