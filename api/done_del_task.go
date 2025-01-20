package api

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/naluneotlichno/FP-GO-API/database"
)

// DoneTaskHandler обрабатывает POST-запрос /api/task/done?id=<ID>
// Если repeat пустой - удаляем задачу; если repeat есть - меняем дату на NextDate().
func DoneTaskHandler(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		http.Error(w, `{"error":"Не указан идентификатор"}`, http.StatusBadRequest)
		return
	}

	// Считаем задачу из БД
	task, err := getTaskByID(id)
	if err != nil {
		http.Error(w, `{"error":"Задача не найдена"}`, http.StatusNotFound)
		return
	}

	// Если у задачи нет repeat - просто удаляем
	if strings.TrimSpace(task.Repeat) == "" {
		if err := deleteTaskByID(id); err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusBadRequest)
			return
		}
		// Возвращаем пустой JSON
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("{}"))
		return
	}

	// Если repeat есть, нужно изменить дату
	oldDate, err := time.Parse("20060102", task.Date)
	if err != nil {
		http.Error(w, `{"error":"Некорректная дата в базе"}`, http.StatusBadRequest)
		return
	}

	nextDate, err := NextDate(oldDate, task.Repeat)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusBadRequest)
		return
	}

	// Обновим поле date в БД
	if err := updateTaskDate(id, nextDate.Format("20060102")); err != nil {
		http.Error(w, `{"error":"Не удалось обновить дату"}`, http.StatusBadRequest)
		return
	}

	// Возвращаем пустой JSON
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte("{}"))
}

// DeleteTaskHandler обрабатывает DELETE-запрос /api/task?id=<ID>
func DeleteTaskHandler(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		http.Error(w, `{"error":"Не указан идентификатор"}`, http.StatusBadRequest)
		return
	}

	err := deleteTaskByID(id)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte("{}"))
}

// NextDate рассчитывает следующую дату для задачи, исходя из repeat.
// Пример: "d 3" -> +3 дня, "m 1" -> +1 месяц, "y 2" -> +2 года.
// Можно адаптировать под более сложную логику.
func NextDate(current time.Time, repeat string) (time.Time, error) {
	// Пример: "d 3"
	re := regexp.MustCompile(`^([dmy])\s+(\d+)$`)
	matches := re.FindStringSubmatch(strings.TrimSpace(repeat))
	if len(matches) != 3 {
		return time.Time{}, fmt.Errorf("Некорректный формат repeat")
	}
	unit := matches[1]
	n, _ := strconv.Atoi(matches[2])

	switch unit {
	case "d":
		return current.AddDate(0, 0, n), nil
	case "m":
		return current.AddDate(0, n, 0), nil
	case "y":
		return current.AddDate(n, 0, 0), nil
	}
	return time.Time{}, fmt.Errorf("Некорректный повтор: %s", repeat)
}

// ----------------------------------------------------------------------------
// Вспомогательные функции работы с БД. У тебя могут быть свои аналоги.
// ----------------------------------------------------------------------------

// getTaskByID читает задачу по ID.
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

// deleteTaskByID удаляет запись из таблицы scheduler по id.
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

// updateTaskDate обновляет только дату задачи (например, при повторении).
func updateTaskDate(id string, newDate string) error {
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
