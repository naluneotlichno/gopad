package api

import (
    "database/sql"
    "errors"
    "fmt"
    "log"
    "net/http"
    "strconv"
    "strings"
    "time"
    "encoding/json"

    "github.com/naluneotlichno/FP-GO-API/database"
)

// DoneTaskHandler обрабатывает POST /api/task/done?id=...
//  1) Если repeat пустой — удаляем задачу.
//  2) Если repeat есть — меняем дату на "следующий раз".
func DoneTaskHandler(w http.ResponseWriter, r *http.Request) {
    log.Println("🔥 [DoneTaskHandler] Запрос на /api/task/done получен...")

    id := r.URL.Query().Get("id")
    if id == "" {
        log.Println("🚨 [DoneTaskHandler] Нет параметра id")
        http.Error(w, `{"error":"Не указан идентификатор"}`, http.StatusBadRequest)
        return
    }
    log.Printf("🔍 [DoneTaskHandler] Ищем задачу с id=%s\n", id)

    // Получаем задачу
    task, err := getTaskByID(id)
    if err != nil {
        log.Printf("🚨 [DoneTaskHandler] Задача не найдена: %v\n", err)
        http.Error(w, `{"error":"Задача не найдена"}`, http.StatusNotFound)
        return
    }
    log.Printf("✅ [DoneTaskHandler] Задача найдена: ID=%s, Date=%s, Repeat=%s\n", task.ID, task.Date, task.Repeat)

    // Если repeat пустой — удаляем задачу
    if strings.TrimSpace(task.Repeat) == "" {
        log.Printf("⚠️ [DoneTaskHandler] repeat пустой, удаляем задачу id=%s\n", id)
        if err := deleteTaskByID(id); err != nil {
            log.Printf("🚨 [DoneTaskHandler] Ошибка при удалении задачи: %v\n", err)
            response := map[string]string{"error": err.Error()}
            w.Header().Set("Content-Type", "application/json")
            json.NewEncoder(w).Encode(response)
            return
        }
        log.Println("✅ [DoneTaskHandler] Задача успешно удалена, отправляем {}")

        w.Header().Set("Content-Type", "application/json")
        w.Write([]byte("{}"))
        return
    }

    // Если repeat есть, меняем дату задачи (например, +3 дня)
    oldDate, err := time.Parse("20060102", task.Date)
    if err != nil {
        log.Printf("🚨 [DoneTaskHandler] Некорректная дата в БД (%s): %v\n", task.Date, err)
        http.Error(w, `{"error":"Некорректная дата задачи в базе"}`, http.StatusBadRequest)
        return
    }
    log.Printf("🔍 [DoneTaskHandler] Текущая дата задачи: %s\n", oldDate.Format("20060102"))

    // Вызываем адаптер, который внутри использует твою NextDate(...)
    newDate, err := NextDateAdapter(oldDate, task.Repeat)
    if err != nil {
        log.Printf("🚨 [DoneTaskHandler] Ошибка NextDateAdapter: %v\n", err)
        http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusBadRequest)
        return
    }
    log.Printf("✅ [DoneTaskHandler] Новая дата задачи: %s\n", newDate.Format("20060102"))

    // Обновляем задачу в БД
    if err := updateTaskDate(id, newDate.Format("20060102")); err != nil {
        log.Printf("🚨 [DoneTaskHandler] Ошибка обновления даты задачи: %v\n", err)
        http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusBadRequest)
        return
    }
    log.Println("✅ [DoneTaskHandler] Дата задачи успешно обновлена!")

    // Успех
    w.Header().Set("Content-Type", "application/json")
    w.Write([]byte("{}"))
}

// DeleteTaskHandler обрабатывает DELETE /api/task?id=...
// Удаляет задачу по ID, возвращает {} или {"error":"..."}.
func DeleteTaskHandler(w http.ResponseWriter, r *http.Request) {
    log.Println("🔥 [DeleteTaskHandler] Запрос на DELETE /api/task получен...")

    id := r.URL.Query().Get("id")
    if id == "" {
        log.Println("🚨 [DeleteTaskHandler] Нет параметра id")
        http.Error(w, `{"error":"Не указан идентификатор"}`, http.StatusBadRequest)
        return
    }
    log.Printf("🔍 [DeleteTaskHandler] Удаляем задачу с id=%s\n", id)

    if err := deleteTaskByID(id); err != nil {
        log.Printf("🚨 [DeleteTaskHandler] Ошибка удаления задачи: %v\n", err)
        response := map[string]string{"error": err.Error()}
        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(response)
        return
    }
    log.Println("✅ [DeleteTaskHandler] Задача успешно удалена, отправляем {}")

    w.Header().Set("Content-Type", "application/json")
    w.Write([]byte("{}"))
}

// NextDateAdapter — переходник между твоей NextDate(now, date, repeat) и тем,
// что ожидают тесты (просто time.Time на выход).
func NextDateAdapter(oldDate time.Time, repeat string) (time.Time, error) {
    log.Println("🔍 [NextDateAdapter] Адаптируем вызов твоей NextDate(...).")

    // Превращаем oldDate в строку, как требует твоя функция
    oldDateStr := oldDate.Format("20060102")

    // Вызываем твою старую функцию
    newDateStr, err := NextDate(oldDate, oldDateStr, repeat)
    if err != nil {
        log.Printf("🚨 [NextDateAdapter] Ошибка в твоей NextDate: %v\n", err)
        return time.Time{}, err
    }

    parsed, err := time.Parse("20060102", newDateStr)
    if err != nil {
        log.Printf("🚨 [NextDateAdapter] Ошибка парсинга '%s': %v\n", newDateStr, err)
        return time.Time{}, fmt.Errorf("Ошибка парсинга даты '%s': %w", newDateStr, err)
    }
    log.Printf("✅ [NextDateAdapter] Итоговая дата: %s\n", parsed.Format("20060102"))
    return parsed, nil
}

// ----------------------------------------------------------------------
// ВСПОМОГАТЕЛЬНЫЕ ФУНКЦИИ для работы с БД (scheduler)
// ----------------------------------------------------------------------

// getTaskByID читает задачу по ID из таблицы scheduler.
func getTaskByID(id string) (Task, error) {
    log.Println("🔍 [getTaskByID] Подключаемся к БД...")
    db, err := database.GetDB() // Если возвращает (db *sql.DB, err error)
    if err != nil {
        log.Printf("🚨 [getTaskByID] Ошибка получения DB: %v\n", err)
        return Task{}, errors.New("Ошибка подключения к БД")
    }

    log.Printf("🔍 [getTaskByID] Парсим id='%s' в int...\n", id)
    idInt, err := strconv.ParseInt(id, 10, 64)
    if err != nil {
        log.Printf("🚨 [getTaskByID] Невалидный ID='%s': %v\n", id, err)
        return Task{}, errors.New("Задача не найдена")
    }

    var t Task
    log.Println("🔍 [getTaskByID] Выполняем SELECT ... FROM scheduler WHERE id=?")
    row := db.QueryRow(`SELECT id, date, title, comment, repeat FROM scheduler WHERE id=?`, idInt)
    err = row.Scan(&t.ID, &t.Date, &t.Title, &t.Comment, &t.Repeat)
    if err != nil {
        if err == sql.ErrNoRows {
            log.Println("🚨 [getTaskByID] Запись не найдена")
            return Task{}, errors.New("Задача не найдена")
        }
        log.Printf("🚨 [getTaskByID] Ошибка запроса: %v\n", err)
        return Task{}, err
    }

    log.Printf("✅ [getTaskByID] Успешно получена задача: %#v\n", t)
    return t, nil
}

// deleteTaskByID удаляет задачу из таблицы scheduler.
func deleteTaskByID(id string) error {
    log.Println("🔍 [deleteTaskByID] Подключаемся к БД...")
    db, err := database.GetDB()
    if err != nil {
        log.Printf("🚨 [deleteTaskByID] Ошибка получения DB: %v\n", err)
        return errors.New("Ошибка подключения к БД")
    }

    log.Printf("🔍 [deleteTaskByID] Парсим id='%s' в int...\n", id)
    idInt, err := strconv.ParseInt(id, 10, 64)
    if err != nil {
        log.Printf("🚨 [deleteTaskByID] Невалидный ID='%s': %v\n", id, err)
        return errors.New("Задача не найдена")
    }

    log.Println("🔍 [deleteTaskByID] Выполняем DELETE FROM scheduler WHERE id=?")
    res, err := db.Exec(`DELETE FROM scheduler WHERE id=?`, idInt)
    if err != nil {
        log.Printf("🚨 [deleteTaskByID] Ошибка DELETE: %v\n", err)
        return err
    }

    n, _ := res.RowsAffected()
    if n == 0 {
        log.Println("🚨 [deleteTaskByID] Строка не найдена, нечего удалять")
        return errors.New("Задача не найдена")
    }
    log.Println("✅ [deleteTaskByID] Задача успешно удалена!")
    return nil
}

// updateTaskDate меняет поле date в таблице scheduler.
func updateTaskDate(id, newDate string) error {
    log.Println("🔍 [updateTaskDate] Подключаемся к БД...")
    db, err := database.GetDB()
    if err != nil {
        log.Printf("🚨 [updateTaskDate] Ошибка получения DB: %v\n", err)
        return errors.New("Ошибка подключения к БД")
    }

    log.Printf("🔍 [updateTaskDate] Парсим id='%s' в int...\n", id)
    idInt, err := strconv.ParseInt(id, 10, 64)
    if err != nil {
        log.Printf("🚨 [updateTaskDate] Невалидный ID='%s': %v\n", id, err)
        return errors.New("Неверный ID")
    }

    log.Printf("🔍 [updateTaskDate] UPDATE scheduler SET date='%s' WHERE id=%d\n", newDate, idInt)
    res, err := db.Exec(`UPDATE scheduler SET date=? WHERE id=?`, newDate, idInt)
    if err != nil {
        log.Printf("🚨 [updateTaskDate] Ошибка UPDATE: %v\n", err)
        return err
    }

    n, _ := res.RowsAffected()
    if n == 0 {
        log.Println("🚨 [updateTaskDate] Не найдена строка с таким id")
        return errors.New("Задача не найдена")
    }
    log.Println("✅ [updateTaskDate] Дата задачи успешно обновлена!")
    return nil
}
