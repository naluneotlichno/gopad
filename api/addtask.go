package api

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/naluneotlichno/FP-GO-API/database"
)

// 🔥 GetSingleTaskHandler обрабатывает GET-запросы на /api/task?id=<ID>
func GetSingleTaskHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("🔥 [GetSingleTaskHandler] Начинаем обработку GET /api/task")

	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	// Проверяем метод
	if r.Method != http.MethodGet {
		log.Printf("❌ [MethodCheck] Метод %s не поддерживается", r.Method)
		http.Error(w, `{"error":"Метод не поддерживается"}`, http.StatusMethodNotAllowed)
		return
	}

	// Считываем ?id=...
	idStr := r.URL.Query().Get("id")
	if idStr == "" {
		// Тест ожидает, что без ID будет ошибка с полем "error"
		log.Println("⚠️ [GetSingleTaskHandler] Параметр id отсутствует. Возвращаем ошибку.")
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, `{"error":"Не указан параметр id"}`)
		return
	}

	// Пробуем распарсить в число
	taskID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		// Если id не число — тоже ошибка
		log.Printf("❌ [GetSingleTaskHandler] Некорректный id: %s", idStr)
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, `{"error":"Некорректный id: %s"}`, idStr)
		return
	}

	// Подключаемся к базе через твой пакет database
	db, err := database.GetDB()
	if err != nil {
		log.Printf("❌ [DBConnection] Ошибка подключения к базе данных: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, `{"error":"Ошибка подключения к БД"}`)
		return
	}

	// Достаём задачу из таблицы scheduler
	var (
		tID      int64
		tDate    string
		tTitle   string
		tComment string
		tRepeat  string
	)
	row := db.QueryRow(`SELECT id, date, title, comment, repeat 
                        FROM scheduler 
                        WHERE id = ?`, taskID)
	err = row.Scan(&tID, &tDate, &tTitle, &tComment, &tRepeat)
	if err != nil {
		if err == sql.ErrNoRows {
			// Нет такой задачи
			log.Printf("⚠️ [GetSingleTaskHandler] Задача с id=%d не найдена", taskID)
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, `{"error":"Задача с ID %d не найдена"}`, taskID)
			return
		}
		// Любая другая ошибка
		log.Printf("❌ [GetSingleTaskHandler] Ошибка чтения из БД: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, `{"error":"Ошибка чтения из БД"}`)
		return
	}

	// Формируем JSON-ответ — тест ждёт именно такие поля:
	resp := map[string]string{
		"id":      strconv.FormatInt(tID, 10),
		"date":    tDate,
		"title":   tTitle,
		"comment": tComment,
		"repeat":  tRepeat,
	}

	// Отправляем результат
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		log.Printf("❌ [GetSingleTaskHandler] Ошибка кодирования JSON: %v", err)
	}
}
