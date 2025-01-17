package api

import (
    "encoding/json"
    "fmt"
    "log"
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
    log.Println("🔥 [AddTaskHandler] Начинаем обработку запроса")

    // ✅ Устанавливаем заголовок
    w.Header().Set("Content-Type", "application/json; charset=utf-8")

    // ✅ Проверяем метод
    if r.Method != http.MethodPost {
        log.Printf("❌ [MethodCheck] Метод %s не поддерживается", r.Method)
        http.Error(w, `{"error": "Метод не поддерживается"}`, http.StatusMethodNotAllowed)
        return
    }

    // ✅ Декодируем JSON-запрос
    log.Println("🔍 [JSONDecode] Декодируем тело запроса")
    var req TaskRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        log.Printf("❌ [JSONDecode] Ошибка декодирования JSON: %v", err)
        http.Error(w, `{"error": "Ошибка десериализации JSON"}`, http.StatusBadRequest)
        return
    }

    // ✅ Проверяем обязательные поля
    if req.Title == "" {
        log.Println("⚠️ [FieldCheck] Отсутствует заголовок задачи")
        http.Error(w, `{"error": "Не указан заголовок задачи"}`, http.StatusBadRequest)
        return
    }

    // ✅ Если дата пустая — подставляем текущую
    if req.Date == "" {
        log.Println("📅 [DefaultDate] Дата не указана. Подставляем текущую.")
        req.Date = time.Now().Format("20060102")
    }

    // ✅ Парсим дату ТОЛЬКО в формате YYYYMMDD
    log.Println("🔍 [DateParse] Проверяем дату на корректность (только формат YYYYMMDD)")
    taskDate, err := time.Parse("20060102", req.Date)
    if err != nil {
        log.Printf("❌ [DateParse] Дата некорректна: %v", err)
        http.Error(w, `{"error": "Дата указана некорректно"}`, http.StatusBadRequest)
        return
    }

    req.Date = taskDate.Format("20060102") // Снова приводим к формату
    log.Printf("✅ [DateParse] Дата корректна: %s", req.Date)

    // ✅ Определяем «сегодня» (без учёта часов/минут), сравниваем с taskDate
    todayStr := time.Now().Format("20060102")
    today, _ := time.Parse("20060102", todayStr)

    switch {
    case taskDate.Before(today):
        // Если дата строго меньше "сегодня"
        log.Println("⏲️ [PastDate] Дата в прошлом. Применяем правило повторения")
        if req.Repeat != "" {
            nextDate, err := NextDate(today, req.Date, req.Repeat)
            if err != nil {
                log.Printf("❌ [RepeatRule] Ошибка правила повторения: %v", err)
                http.Error(w,
                    fmt.Sprintf(`{"error": "Неверный формат правила повторения: %s"}`, err.Error()),
                    http.StatusBadRequest,
                )
                return
            }
            req.Date = nextDate
            log.Printf("✅ [RepeatRule] Новая дата после повторения: %s", req.Date)
        } else {
            log.Println("📅 [PastDate] Дата в прошлом, повторение не указано. Ставим сегодняшнюю дату.")
            req.Date = todayStr
        }
    case taskDate.Equal(today):
        // Дата ровно "сегодня" - не трогаем
        log.Println("✅ [SameDate] Дата совпадает с сегодняшней. Оставляем как есть.")
    default:
        // taskDate.After(today) → будущее, ничего не делаем
        log.Println("🚀 [FutureDate] Дата в будущем. Оставляем без изменений.")
    }

    // ✅ Подключаемся к базе данных
    log.Println("🔗 [DBConnection] Подключаемся к базе данных")
    db, err := database.GetDB()
    if err != nil {
        log.Printf("❌ [DBConnection] Ошибка подключения к базе данных: %v", err)
        http.Error(w, `{"error": "Ошибка подключения к БД"}`, http.StatusInternalServerError)
        return
    }
    log.Println("✅ [DBConnection] Соединение с базой данных успешно")

    // ✅ Вставляем новую задачу
    log.Println("📝 [DBInsert] Вставляем задачу в базу данных")
    query := `INSERT INTO scheduler (date, title, comment, repeat) VALUES (?, ?, ?, ?)`
    res, err := db.Exec(query, req.Date, req.Title, req.Comment, req.Repeat)
    if err != nil {
        log.Printf("❌ [DBInsert] Ошибка вставки в базу: %v", err)
        http.Error(w, `{"error": "Ошибка записи в БД"}`, http.StatusInternalServerError)
        return
    }

    // ✅ Получаем ID новой задачи
    log.Println("🆔 [DBInsert] Получаем ID новой записи")
    taskID, err := res.LastInsertId()
    if err != nil {
        log.Printf("❌ [DBInsert] Ошибка получения ID: %v", err)
        http.Error(w, `{"error": "Ошибка получения ID записи"}`, http.StatusInternalServerError)
        return
    }
    log.Printf("✅ [DBInsert] Новая задача добавлена с ID: %d", taskID)

    // ✅ Формируем ответ
    resp := TaskResponse{ID: taskID}
    log.Printf("📤 [Response] Отправляем ответ клиенту: %+v", resp)
    if err := json.NewEncoder(w).Encode(resp); err != nil {
        log.Printf("❌ [Response] Ошибка кодирования ответа: %v", err)
    }
}
