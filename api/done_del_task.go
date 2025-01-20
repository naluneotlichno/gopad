package api

import (
    "database/sql"
    "errors"
    "fmt"
    "net/http"
    "strconv"
    "strings"
    "time"

    "github.com/naluneotlichno/FP-GO-API/database"
)

// DoneTaskHandler обрабатывает POST /api/task/done?id=<ID>:
//  1) Если repeat пустой — удаляет задачу.
//  2) Если repeat есть — сдвигает дату на следующий раз (через NextDateAdapter).
func DoneTaskHandler(w http.ResponseWriter, r *http.Request) {
    id := r.URL.Query().Get("id")
    if id == "" {
        http.Error(w, `{"error":"Не указан идентификатор"}`, http.StatusBadRequest)
        return
    }

    // Получаем задачу из БД
    task, err := getTaskByID(id)
    if err != nil {
        http.Error(w, `{"error":"Задача не найдена"}`, http.StatusNotFound)
        return
    }

    // Если нет repeat — удаляем задачу.
    if strings.TrimSpace(task.Repeat) == "" {
        if err := deleteTaskByID(id); err != nil {
            http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusBadRequest)
            return
        }
        w.Header().Set("Content-Type", "application/json")
        w.Write([]byte("{}"))
        return
    }

    // Если repeat есть, вычисляем следующую дату
    oldDate, err := time.Parse("20060102", task.Date)
    if err != nil {
        http.Error(w, `{"error":"Некорректная дата в базе"}`, http.StatusBadRequest)
        return
    }

    // Вызываем адаптер, который внутри использует твою функцию NextDate(...)
    nextDate, err := NextDateAdapter(oldDate, task.Repeat)
    if err != nil {
        http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusBadRequest)
        return
    }

    // Сохраняем новую дату в БД
    if err := updateTaskDate(id, nextDate.Format("20060102")); err != nil {
        http.Error(w, `{"error":"Ошибка при обновлении даты"}`, http.StatusBadRequest)
        return
    }

    // Возвращаем пустой JSON
    w.Header().Set("Content-Type", "application/json")
    w.Write([]byte("{}"))
}

// DeleteTaskHandler обрабатывает DELETE /api/task?id=<ID>
func DeleteTaskHandler(w http.ResponseWriter, r *http.Request) {
    id := r.URL.Query().Get("id")
    if id == "" {
        http.Error(w, `{"error":"Не указан идентификатор"}`, http.StatusBadRequest)
        return
    }

    if err := deleteTaskByID(id); err != nil {
        http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusBadRequest)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    w.Write([]byte("{}"))
}

// NextDateAdapter адаптирует вызов твоей старой функции NextDate(now, date, repeat),
// чтобы вернуть time.Time вместо строки "YYYYMMDD".
func NextDateAdapter(current time.Time, repeat string) (time.Time, error) {
    // Формируем date как текущую дату в формате YYYYMMDD
    currentDateStr := current.Format("20060102")

    // Вызываем твою функцию NextDate(now, date, repeat) (string, error)
    nextDateStr, err := NextDate(current, currentDateStr, repeat)
    if err != nil {
        return time.Time{}, err
    }

    // Превращаем полученную строку "20240201" обратно в time.Time
    parsed, err := time.Parse("20060102", nextDateStr)
    if err != nil {
        return time.Time{}, fmt.Errorf("Ошибка парсинга '%s': %v", nextDateStr, err)
    }

    return parsed, nil
}

// ---------------------- Вспомогательные функции ----------------------

// getTaskByID читает задачу из таблицы scheduler
func getTaskByID(id string) (Task, error) {
    db, _ := database.GetDB()

    idInt, err := strconv.ParseInt(id, 10, 64)
    if err != nil {
        return Task{}, errors.New("Задача не найдена")
    }

    var t Task
    row := db.QueryRow(`SELECT id, date, title, comment, repeat FROM scheduler WHERE id=?`, idInt)
    err = row.Scan(&t.ID, &t.Date, &t.Title, &t.Comment, &t.Repeat)
    if err != nil {
        if err == sql.ErrNoRows {
            return Task{}, errors.New("Задача не найдена")
        }
        return Task{}, err
    }
    return t, nil
}

// deleteTaskByID удаляет строку из таблицы scheduler
func deleteTaskByID(id string) error {
    db, _ := database.GetDB()

    idInt, err := strconv.ParseInt(id, 10, 64)
    if err != nil {
        return errors.New("Задача не найдена")
    }

    res, err := db.Exec(`DELETE FROM scheduler WHERE id=?`, idInt)
    if err != nil {
        return err
    }
    n, _ := res.RowsAffected()
    if n == 0 {
        return errors.New("Задача не найдена")
    }
    return nil
}

// updateTaskDate обновляет поле date в таблице scheduler
func updateTaskDate(id, newDate string) error {
    db, _ := database.GetDB()

    idInt, err := strconv.ParseInt(id, 10, 64)
    if err != nil {
        return errors.New("Неверный ID")
    }

    res, err := db.Exec(`UPDATE scheduler SET date=? WHERE id=?`, newDate, idInt)
    if err != nil {
        return err
    }

    n, _ := res.RowsAffected()
    if n == 0 {
        return errors.New("Задача не найдена")
    }
    return nil
}
