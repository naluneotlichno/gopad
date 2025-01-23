package api

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/naluneotlichno/FP-GO-API/nextdate"
	"github.com/naluneotlichno/FP-GO-API/database"
)

// Те же имена структур, что в "КОД 1"
type AddTaskRequest struct {
	Date    string `json:"date"`
	Title   string `json:"title"`
	Comment string `json:"comment"`
	Repeat  string `json:"repeat"`
}

type AddTaskResponse struct {
	ID    string `json:"id,omitempty"`
	Error string `json:"error,omitempty"`
}

// Константа с нужным форматом даты
const layout = "20060102"

// AddTaskHandler обрабатывает POST-запросы на /api/task (аналог «КОД 1»).
func AddTaskHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("🚀 [AddTaskHandler] Начинаем обработку запроса")

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if r.Method != http.MethodPost {
		log.Printf("❌ [MethodCheck] Метод %s не поддерживается", r.Method)
		http.Error(w, `{"error":"Метод не поддерживается"}`, http.StatusMethodNotAllowed)
		return
	}

	// 1) Считываем тело запроса
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("❌ [BodyRead] Не удалось прочитать тело: %v", err)
		respondWithJSON(w, http.StatusBadRequest, AddTaskResponse{Error: "не удалось прочитать тело запроса"})
		return
	}
	defer r.Body.Close()

	// 2) Декодируем JSON в нашу структуру
	var req AddTaskRequest
	if err := json.Unmarshal(body, &req); err != nil {
		log.Printf("❌ [JSONDecode] Ошибка декодирования JSON: %v", err)
		respondWithJSON(w, http.StatusBadRequest, AddTaskResponse{Error: "неверный формат JSON"})
		return
	}

	// 3) Проверяем обязательное поле title
	req.Title = strings.TrimSpace(req.Title)
	if req.Title == "" {
		respondWithJSON(w, http.StatusBadRequest, AddTaskResponse{Error: "не указан заголовок задачи"})
		return
	}

	// 4) Если дата не указана - подставляем сегодняшнюю
	now := time.Now()
	if strings.TrimSpace(req.Date) == "" {
		req.Date = now.Format(layout)
	}

	// 5) Пытаемся распарсить дату в формате YYYYMMDD
	taskDate, err := time.Parse(layout, req.Date)
	if err != nil {
		log.Printf("❌ [DateParse] Дата указана неверно: %v", err)
		respondWithJSON(w, http.StatusBadRequest, AddTaskResponse{Error: "дата указана в неверном формате"})
		return
	}

	// 6) Если дата в прошлом — проверяем repeat
	if taskDate.Before(now) {
		if strings.TrimSpace(req.Repeat) == "" {
			// Повторения нет => ставим дату на сегодня
			taskDate = now
		} else {
			// Повторение есть => вызываем некую NextDate.
			// Предположим, она у вас реализована аналогично «КОД 1»
			nextDateStr, err := nextdate.NextDate(now, req.Date, req.Repeat)
			if err != nil {
				respondWithJSON(w, http.StatusBadRequest, AddTaskResponse{Error: "неверное правило повторения"})
				return
			}
			// Парсим то, что вернул NextDate
			taskDate, _ = time.Parse(layout, nextDateStr)
		}
	}

	// 7) Собираем объект для вставки в БД
	newDate := taskDate.Format(layout)
	// Подключаемся к базе
	dbConn, err := database.GetDB()
	if err != nil {
		log.Printf("❌ [DBConnection] Ошибка подключения к базе: %v", err)
		respondWithJSON(w, http.StatusInternalServerError, AddTaskResponse{Error: "ошибка подключения к БД"})
		return
	}

	// 8) Вставляем в таблицу
	query := `INSERT INTO scheduler (date, title, comment, repeat) VALUES (?, ?, ?, ?)`
	res, err := dbConn.Exec(query, newDate, req.Title, req.Comment, req.Repeat)
	if err != nil {
		log.Printf("❌ [DBInsert] Ошибка вставки в базу: %v", err)
		respondWithJSON(w, http.StatusInternalServerError, AddTaskResponse{Error: "Ошибка записи в БД"})
		return
	}

	// 9) Получаем ID новой записи
	taskID, err := res.LastInsertId()
	if err != nil {
		log.Printf("❌ [DBInsert] Ошибка получения ID: %v", err)
		respondWithJSON(w, http.StatusInternalServerError, AddTaskResponse{Error: "Ошибка получения ID записи"})
		return
	}

	// 10) Возвращаем результат с кодом 201 (Created)
	resp := AddTaskResponse{ID: fmt.Sprintf("%d", taskID)}
	log.Printf("✅ [DBInsert] Новая задача добавлена: ID=%d", taskID)
	respondWithJSON(w, http.StatusCreated, resp)
}

// respondWithJSON — аналогичная функция «utils.RespondWithJSON»,
// отправляет JSON и устанавливает код статуса.
func respondWithJSON(w http.ResponseWriter, status int, data interface{}) {
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}
