package api

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/naluneotlichno/FP-GO-API/database" // Предполагаем, что тут лежит твоя логика DB
)

// Task описывает поля задачи
type Task struct {
	ID      string `json:"id"`      // Идентификатор задачи в формате строки
	Date    string `json:"date"`    // Дата задачи в формате YYYYMMDD
	Title   string `json:"title"`   // Заголовок задачи
	Comment string `json:"comment"` // Комментарий
	Repeat  string `json:"repeat"`  // Параметры повторения задачи, например "d 5"
}

// GetTaskByID получает задачу из таблицы scheduler по ID.
// Возвращает ошибку, если не найдена.
func GetTaskByID(id string) (Task, error) {
	db, _ := database.GetDB()

	idInt, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		// Если ID не число, возвращаем ту же ошибку, что и "не найдена"
		return Task{}, errors.New("task not found")
	}

	var t Task
	row := db.QueryRow(`SELECT id, date, title, comment, repeat FROM scheduler WHERE id=?`, idInt)
	err = row.Scan(&t.ID, &t.Date, &t.Title, &t.Comment, &t.Repeat)
	if err != nil {
		if err == sql.ErrNoRows {
			return Task{}, errors.New("task not found")
		}
		return Task{}, err
	}

	return t, nil
}

// UpdateTask обновляет запись о задаче в таблице scheduler.
// Возвращает ошибку, если задача не найдена или данные невалидны.
func UpdateTask(task Task) error {
	db, _ := database.GetDB()

	// Проверяем, что ID — целое число
	idInt, err := strconv.ParseInt(task.ID, 10, 64)
	if err != nil {
		return errors.New("task not found")
	}

	// Проверяем формат даты: YYYYMMDD
	parsedDate, err := time.Parse("20060102", task.Date)
	if err != nil {
		return fmt.Errorf("invalid date format") // Перехватим это выше и вернём JSON "error"
	}
	// Дата не должна быть в прошлом
	now := time.Now().Truncate(24 * time.Hour)
	if parsedDate.Before(now) {
		return fmt.Errorf("date is in the past")
	}

	// Проверяем, что title не пустой
	if strings.TrimSpace(task.Title) == "" {
		return fmt.Errorf("title is empty")
	}

	// Валидируем repeat. Тест явно ожидает ошибку, если repeat = "ooops".
	// Допустим, разрешаем пустую строку, или формат "d 5", "d 7", "m 1" и т.д.
	if task.Repeat != "" {
		matched, _ := regexp.MatchString(`^[dmw]\s+\d+$`, task.Repeat)
		if !matched {
			return fmt.Errorf("invalid repeat format")
		}
	}

	// Пытаемся обновить задачу в БД
	res, err := db.Exec(`
        UPDATE scheduler
           SET date    = ?,
               title   = ?,
               comment = ?,
               repeat  = ?
         WHERE id = ?;
    `,
		task.Date,
		task.Title,
		task.Comment,
		task.Repeat,
		idInt,
	)
	if err != nil {
		return err
	}

	// Смотрим, затронута ли хотя бы одна строка
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return errors.New("task not found")
	}

	return nil
}

// GetTaskHandler обрабатывает GET /api/task?id=<ID>
func GetTaskHandler(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		http.Error(w, `{"error": "Не указан идентификатор"}`, http.StatusBadRequest)
		return
	}

	task, err := GetTaskByID(id)
	if err != nil {
		http.Error(w, `{"error": "Задача не найдена"}`, http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(task)
}

// UpdateTaskHandler обрабатывает PUT /api/task (JSON в теле)
func UpdateTaskHandler(w http.ResponseWriter, r *http.Request) {
	var t Task
	if err := json.NewDecoder(r.Body).Decode(&t); err != nil {
		http.Error(w, `{"error": "Ошибка при чтении данных"}`, http.StatusBadRequest)
		return
	}

	if t.ID == "" {
		http.Error(w, `{"error": "Не указан идентификатор"}`, http.StatusBadRequest)
		return
	}

	// Сама логика валидации и обновления — в UpdateTask
	if err := UpdateTask(t); err != nil {
		// Для всех ошибок, которые мы вернули — сообщим их наружу
		http.Error(w, fmt.Sprintf(`{"error": "%s"}`, adaptErrorMsg(err.Error())), http.StatusBadRequest)
		return
	}

	// Если обновление прошло успешно — возвращаем пустой JSON
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte("{}"))
}

// adaptErrorMsg просто помогает вернуть нужную строку ошибки.
// Можно сделать switch, если нужно отличать "task not found" от других.
func adaptErrorMsg(errMsg string) string {
	switch {
	case strings.Contains(errMsg, "task not found"):
		return "Задача не найдена"
	case strings.Contains(errMsg, "invalid date format"):
		return "Некорректный формат даты"
	case strings.Contains(errMsg, "date is in the past"):
		return "Дата не может быть меньше сегодняшней"
	case strings.Contains(errMsg, "empty"):
		return "Пустое поле title"
	case strings.Contains(errMsg, "invalid repeat format"):
		return "Неверный формат repeat"
	default:
		// Пусть будет общий случай
		return errMsg
	}
}
