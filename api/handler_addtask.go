package api

import (
    "database/sql"
    "encoding/json"
    "fmt"
    "log"
    "net/http"
    "time"

    "github.com/naluneotlichno/FP-GO-API/database"
)

// 🔥 TasksResponse — структура ответа со списком задач
type TasksResponse struct {
    Tasks []TaskItem `json:"tasks"`
}

// 🔥 TaskItem — структура для отдельной задачи в списке
// Обратите внимание, все поля строковые (требование теста)
type TaskItem struct {
    ID      string `json:"id"`
    Date    string `json:"date"`
    Title   string `json:"title"`
    Comment string `json:"comment"`
    Repeat  string `json:"repeat"`
}

// 🔥 GetTasksHandler обрабатывает GET-запросы на /api/tasks
func GetTasksHandler(w http.ResponseWriter, r *http.Request) {
    log.Println("🔥 [GetTasksHandler] Запрос на получение списка задач")

    // ✅ Устанавливаем заголовок
    w.Header().Set("Content-Type", "application/json; charset=utf-8")

    // ✅ Проверяем метод (GET)
    if r.Method != http.MethodGet {
        log.Printf("❌ [MethodCheck] Метод %s не поддерживается", r.Method)
        http.Error(w, `{"error":"Метод не поддерживается"}`, http.StatusMethodNotAllowed)
        return
    }

    // ✅ Подключаемся к базе данных
    log.Println("✅ [DBConnection] Получаем соединение с базой")
    db, err := database.GetDB()
    if err != nil {
        log.Printf("❌ [DBConnection] Ошибка подключения к базе данных: %v", err)
        http.Error(w, `{"error":"Ошибка подключения к БД"}`, http.StatusInternalServerError)
        return
    }

    // ✅ Получаем параметр search=? из URL
    searchParam := r.URL.Query().Get("search")

    // Можно ограничить кол-во записей, чтобы не возращать "тонну" данных.
    limit := 50

    // Запросим нужные поля
    // Т.к. тест требует сортировку "по дате в сторону увеличения",
    // добавим "ORDER BY date".
    // Если searchParam пустой → выбираем все.
    var rows *sql.Rows

    if searchParam == "" {
        // ➜ Нет параметра search → выдать все (до limit)
        query := `SELECT id, date, title, comment, repeat
                  FROM scheduler
                  ORDER BY date
                  LIMIT ?`
        rows, err = db.Query(query, limit)
        if err != nil {
            log.Printf("❌ [DBQuery] Ошибка запроса без поиска: %v", err)
            http.Error(w, `{"error":"Ошибка чтения из БД"}`, http.StatusInternalServerError)
            return
        }
    } else {
        // ➜ Есть параметр search
        log.Printf("✅ [Search] Параметр search=%s", searchParam)

        // Пробуем распарсить search как дату в формате dd.mm.yyyy (02.01.2006)
        parsedDate, parseErr := time.Parse("02.01.2006", searchParam)
        if parseErr == nil {
            // Удалось распарсить → значит ищем задачи на эту дату
            dateStr := parsedDate.Format("20060102")
            log.Printf("✅ [Search] Распознали дату %s (YYYYMMDD)", dateStr)

            query := `SELECT id, date, title, comment, repeat
                      FROM scheduler
                      WHERE date = ?
                      ORDER BY date
                      LIMIT ?`
            rows, err = db.Query(query, dateStr, limit)
            if err != nil {
                log.Printf("❌ [DBQuery] Ошибка запроса по дате: %v", err)
                http.Error(w, `{"error":"Ошибка чтения из БД"}`, http.StatusInternalServerError)
                return
            }
        } else {
            // Иначе ищем подстроку в title или comment
            likePattern := "%" + searchParam + "%"
            log.Printf("✅ [Search] Строковый поиск LIKE '%s'", likePattern)

            query := `SELECT id, date, title, comment, repeat
                      FROM scheduler
                      WHERE title LIKE ? OR comment LIKE ?
                      ORDER BY date
                      LIMIT ?`
            rows, err = db.Query(query, likePattern, likePattern, limit)
            if err != nil {
                log.Printf("❌ [DBQuery] Ошибка запроса по LIKE: %v", err)
                http.Error(w, `{"error":"Ошибка чтения из БД"}`, http.StatusInternalServerError)
                return
            }
        }
    }

    defer rows.Close()

    // ✅ Сканируем результат в срез структур TaskItem
    var tasks []TaskItem
    for rows.Next() {
        var (
            id      int64
            date    string
            title   string
            comment string
            repeat  string
        )
        if err := rows.Scan(&id, &date, &title, &comment, &repeat); err != nil {
            log.Printf("❌ [DBScan] Ошибка чтения строки: %v", err)
            http.Error(w, `{"error":"Ошибка чтения из БД"}`, http.StatusInternalServerError)
            return
        }
        tasks = append(tasks, TaskItem{
            ID:      fmt.Sprint(id),
            Date:    date,
            Title:   title,
            Comment: comment,
            Repeat:  repeat,
        })
    }

    // Проверка на ошибки при итерации
    if err := rows.Err(); err != nil {
        log.Printf("❌ [RowsErr] Ошибка при итерировании строк: %v", err)
        http.Error(w, `{"error":"Ошибка чтения из БД"}`, http.StatusInternalServerError)
        return
    }

    // ✅ Если задач нет, tasks=nil → сделаем tasks = []TaskItem{}
    if tasks == nil {
        tasks = []TaskItem{}
    }

    // Формируем ответ
    response := TasksResponse{
        Tasks: tasks,
    }

    // Отправляем JSON
    if err := json.NewEncoder(w).Encode(response); err != nil {
        log.Printf("❌ [Response] Ошибка кодирования JSON: %v", err)
    }
}
