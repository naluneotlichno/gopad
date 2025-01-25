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
	idStr := r.URL.Query().Get("id")
	if idStr == "" {
		JsonResponse(w, http.StatusBadRequest, map[string]string{"error": "отсутствует id"})
		return
	}

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		JsonResponse(w, http.StatusBadRequest, map[string]string{"error": "недопустимый параметр id"})
		return
	}

	foundTask, err := database.GetTaskByID(id)
	if err != nil {
		JsonResponse(w, http.StatusNotFound, map[string]string{"error": "задача не найдена"})
		return
	}

	response := map[string]string{
		"id":      strconv.FormatInt(foundTask.ID, 10),
		"date":    foundTask.Date,
		"title":   foundTask.Title,
		"comment": foundTask.Comment,
		"repeat":  foundTask.Repeat,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

var task struct {
	ID      string `json:"id"`
	Date    string `json:"date"`
	Title   string `json:"title"`
	Comment string `json:"comment"`
	Repeat  string `json:"repeat"`
}


func UpdateTaskHandler(w http.ResponseWriter, r *http.Request) {

	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&task)

	if err != nil {
		JsonResponse(w, http.StatusBadRequest, map[string]string{"error": "Неверный формат JSON"})
		return
	}

	if task.ID == "" {
		JsonResponse(w, http.StatusBadRequest, map[string]string{"error": "Не указан идентификатор"})
		return
	}

	id, err := strconv.ParseInt(task.ID, 10, 64)
	if err != nil {
		JsonResponse(w, http.StatusBadRequest, map[string]string{"error": "Недопустимый формат идентификатора"})
		return
	}

	if task.Date == "" {
		JsonResponse(w, http.StatusBadRequest, map[string]string{"error": "Дата не может быть пустой"})
		return
	}

	_, err = time.Parse(layout, task.Date)
	if err != nil {
		JsonResponse(w, http.StatusBadRequest, map[string]string{"error": "Неверный формат даты"})
		return
	}

	if task.Title == "" {
		JsonResponse(w, http.StatusBadRequest, map[string]string{"error": "Заголовок не может быть пустым"})
		return
	}

	updatedTask := database.Task{
		ID:      id,
		Date:    task.Date,
		Title:   task.Title,
		Comment: task.Comment,
		Repeat:  task.Repeat,
	}

	taskErr := database.UpdateTask(updatedTask)
	if taskErr != nil {
		if errors.Is(taskErr, database.ErrTask) {
			JsonResponse(w, http.StatusNotFound, map[string]string{"error": "Задача не найдена"})
		} else {
			JsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "Ошибка обновления задачи"})
		}
		return
	}

	JsonResponse(w, http.StatusOK, map[string]interface{}{})

}

