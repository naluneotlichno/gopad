package api

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/naluneotlichno/FP-GO-API/database"
)

// DoneTaskHandler обрабатывает POST /api/task/done?id=...
// DoneTaskHandler обрабатывает POST /api/task/done?id=...
func DoneTaskHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("🔥 [DoneTaskHandler] Запрос на /api/task/done получен...")

	id := r.URL.Query().Get("id")
	log.Printf("🔍 [DoneTaskHandler] ID из запроса: %s\n", id)
	if id == "" {
		log.Println("🚨 [DoneTaskHandler] ID не указан")
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "Не указан идентификатор"})
		return
	}

	task, err := getTaskByID(id)
	if err != nil {
		log.Printf("🚨 [DoneTaskHandler] Ошибка получения задачи: %v\n", err)
		jsonResponse(w, http.StatusNotFound, map[string]string{"error": "Задача не найдена"})
		return
	}
	log.Printf("✅ [DoneTaskHandler] Найдена задача: %#v\n", task)

	if strings.TrimSpace(task.Repeat) == "" {
		log.Printf("🔍 [DoneTaskHandler] repeat пустой. Удаляем задачу ID=%s\n", id)
		if err := deleteTaskByID(id); err != nil {
			log.Printf("🚨 [DoneTaskHandler] Ошибка при удалении задачи ID=%s: %v\n", id, err)
			jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		log.Printf("✅ [DoneTaskHandler] Задача ID=%s успешно удалена\n", id)
		jsonResponse(w, http.StatusOK, map[string]any{})
		return
	}

	oldDate, err := time.Parse("20060102", task.Date)
	if err != nil {
		log.Printf("🚨 [DoneTaskHandler] Ошибка парсинга даты задачи (%s): %v\n", task.Date, err)
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "Некорректная дата задачи"})
		return
	}
	log.Printf("✅ [DoneTaskHandler] Текущая дата задачи: %s\n", oldDate.Format("20060102"))

	newDate, err := NextDateAdapter(oldDate, task.Repeat)
	if err != nil {
		log.Printf("🚨 [DoneTaskHandler] Ошибка при расчёте новой даты: %v\n", err)
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "Некорректное правило повторения"})
		return
	}
	log.Printf("✅ [DoneTaskHandler] Новая дата задачи: %s\n", newDate.Format("20060102"))

	if err := updateTaskDate(id, newDate.Format("20060102")); err != nil {
		log.Printf("🚨 [DoneTaskHandler] Ошибка при обновлении даты задачи ID=%s: %v\n", id, err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	log.Printf("✅ [DoneTaskHandler] Дата задачи ID=%s успешно обновлена\n", id)

	jsonResponse(w, http.StatusOK, map[string]any{})
}

func jsonResponse(w http.ResponseWriter, status int, payload interface{}) {
	log.Printf("📤 [jsonResponse] Отправляем ответ: статус=%d, payload=%#v\n", status, payload)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		log.Printf("🚨 [jsonResponse] Ошибка кодирования JSON: %v\n", err)
		http.Error(w, `{"error":"Ошибка генерации ответа"}`, http.StatusInternalServerError)
	}
}


// DeleteTaskHandler обрабатывает DELETE /api/task?id=...
func DeleteTaskHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("🔥 [DeleteTaskHandler] Запрос на DELETE /api/task получен...")

	id := r.URL.Query().Get("id")
	log.Printf("🔍 [DeleteTaskHandler] ID из запроса: %s\n", id)
	if id == "" {
		log.Println("🚨 [DeleteTaskHandler] ID не указан")
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "Не указан идентификатор"})
		return
	}

	log.Printf("🔍 [DeleteTaskHandler] Пытаемся удалить задачу с ID=%s\n", id)
	if err := deleteTaskByID(id); err != nil {
		log.Printf("🚨 [DeleteTaskHandler] Ошибка удаления задачи ID=%s: %v\n", id, err)
		jsonResponse(w, http.StatusNotFound, map[string]string{"error": err.Error()})
		return
	}
	log.Printf("✅ [DeleteTaskHandler] Задача ID=%s успешно удалена\n", id)

	jsonResponse(w, http.StatusOK, map[string]any{})
}

// NextDateAdapter — переходник между твоей NextDate(...) и тем, что ожидают тесты.
func NextDateAdapter(oldDate time.Time, repeat string) (time.Time, error) {
	log.Printf("🔍 [NextDateAdapter] Параметры: oldDate=%s, repeat=%s\n", oldDate.Format("20060102"), repeat)

	repeatParts := strings.Split(repeat, " ")
	if len(repeatParts) != 2 || repeatParts[0] != "d" {
		err := fmt.Errorf("некорректный формат repeat: %s", repeat)
		log.Printf("🚨 [NextDateAdapter] %v\n", err)
		return time.Time{}, err
	}

	days, err := strconv.Atoi(repeatParts[1])
	if err != nil {
		log.Printf("🚨 [NextDateAdapter] Ошибка парсинга дней: %v\n", err)
		return time.Time{}, err
	}

	newDate := oldDate.AddDate(0, 0, days)
	log.Printf("✅ [NextDateAdapter] Новая дата: %s\n", newDate.Format("20060102"))
	return newDate, nil
}

func getTaskByID(id string) (Task, error) {
	log.Printf("🔍 [getTaskByID] Получаем задачу ID=%s из базы данных\n", id)
	db, err := database.GetDB()
	if err != nil {
		log.Printf("🚨 [getTaskByID] Ошибка подключения к базе: %v\n", err)
		return Task{}, errors.New("ошибка подключения к БД")
	}

	idInt, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		log.Printf("🚨 [getTaskByID] Невалидный ID=%s: %v\n", id, err)
		return Task{}, errors.New("задача не найдена")
	}

	var t Task
	log.Println("🔍 [getTaskByID] Выполняем SELECT...")
	row := db.QueryRow(`SELECT id, date, title, comment, repeat FROM scheduler WHERE id=?`, idInt)
	err = row.Scan(&t.ID, &t.Date, &t.Title, &t.Comment, &t.Repeat)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Printf("🚨 [getTaskByID] Задача ID=%s не найдена\n", id)
			return Task{}, errors.New("задача не найдена")
		}
		log.Printf("🚨 [getTaskByID] Ошибка выполнения запроса: %v\n", err)
		return Task{}, err
	}
	log.Printf("✅ [getTaskByID] Найдена задача: %#v\n", t)
	return t, nil
}

func deleteTaskByID(id string) error {
	log.Printf("🔍 [deleteTaskByID] Удаляем задачу ID=%s из базы данных\n", id)
	db, err := database.GetDB()
	if err != nil {
		log.Printf("🚨 [deleteTaskByID] Ошибка подключения к базе: %v\n", err)
		return errors.New("ошибка подключения к БД")
	}

	idInt, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		log.Printf("🚨 [deleteTaskByID] Невалидный ID=%s: %v\n", id, err)
		return errors.New("задача не найдена")
	}

	res, err := db.Exec(`DELETE FROM scheduler WHERE id=?`, idInt)
	if err != nil {
		log.Printf("🚨 [deleteTaskByID] Ошибка выполнения DELETE: %v\n", err)
		return err
	}

	n, _ := res.RowsAffected()
	if n == 0 {
		log.Printf("🚨 [deleteTaskByID] Задача ID=%s не найдена\n", id)
		return errors.New("задача не найдена")
	}
	log.Printf("✅ [deleteTaskByID] Задача ID=%s успешно удалена\n", id)
	return nil
}

func updateTaskDate(id, newDate string) error {
	log.Printf("🔍 [updateTaskDate] Обновляем дату задачи ID=%s на %s\n", id, newDate)
	db, err := database.GetDB()
	if err != nil {
		log.Printf("🚨 [updateTaskDate] Ошибка подключения к базе: %v\n", err)
		return errors.New("ошибка подключения к БД")
	}

	idInt, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		log.Printf("🚨 [updateTaskDate] Невалидный ID=%s: %v\n", id, err)
		return errors.New("задача не найдена")
	}

	res, err := db.Exec(`UPDATE scheduler SET date=? WHERE id=?`, newDate, idInt)
	if err != nil {
		log.Printf("🚨 [updateTaskDate] Ошибка выполнения UPDATE: %v\n", err)
		return err
	}

	n, _ := res.RowsAffected()
	if n == 0 {
		log.Printf("🚨 [updateTaskDate] Задача ID=%s не найдена для обновления\n", id)
		return errors.New("задача не найдена")
	}
	log.Printf("✅ [updateTaskDate] Дата задачи ID=%s успешно обновлена\n", id)
	return nil
}