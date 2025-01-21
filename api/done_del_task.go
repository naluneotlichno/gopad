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
// 1) Если repeat пустой — удаляем задачу.
// 2) Если repeat есть — меняем дату на "следующий раз".
func DoneTaskHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("🔥 [DoneTaskHandler] Запрос на /api/task/done получен...")

	id := r.URL.Query().Get("id")
	if id == "" {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "Не указан идентификатор"})
		return
	}

	task, err := getTaskByID(id)
	if err != nil {
		jsonResponse(w, http.StatusNotFound, map[string]string{"error": "Задача не найдена"})
		return
	}

	if strings.TrimSpace(task.Repeat) == "" {
		if err := deleteTaskByID(id); err != nil {
			jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		jsonResponse(w, http.StatusOK, map[string]any{})
		return
	}

	oldDate, err := time.Parse("20060102", task.Date)
	if err != nil {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "Некорректная дата задачи"})
		return
	}

	newDate, err := NextDateAdapter(oldDate, task.Repeat)
	if err != nil {
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	if err := updateTaskDate(id, newDate.Format("20060102")); err != nil {
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	jsonResponse(w, http.StatusOK, map[string]any{})
}

// DeleteTaskHandler обрабатывает DELETE /api/task?id=...
// Удаляет задачу по ID, возвращает {} или {"error":"..."}.
func DeleteTaskHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("🔥 [DeleteTaskHandler] Запрос на DELETE /api/task получен...")

	id := r.URL.Query().Get("id")
	if id == "" {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "Не указан идентификатор"})
		return
	}

	if err := deleteTaskByID(id); err != nil {
		jsonResponse(w, http.StatusNotFound, map[string]string{"error": err.Error()})
		return
	}

	jsonResponse(w, http.StatusOK, map[string]any{})
}

// NextDateAdapter — переходник между твоей NextDate(now, date, repeat) и тем,
// что ожидают тесты (просто time.Time на выход).
func NextDateAdapter(oldDate time.Time, repeat string) (time.Time, error) {
	log.Println("🔍 [NextDateAdapter] Адаптируем вызов твоей NextDate(...)")

	repeatParts := strings.Split(repeat, " ")
	if len(repeatParts) != 2 || repeatParts[0] != "d" {
		return time.Time{}, fmt.Errorf("некорректный формат repeat: %s", repeat)
	}

	days, err := strconv.Atoi(repeatParts[1])
	if err != nil {
		return time.Time{}, fmt.Errorf("ошибка парсинга количества дней: %w", err)
	}

	newDate := oldDate.AddDate(0, 0, days)
	log.Printf("✅ [NextDateAdapter] Новая дата: %s\n", newDate.Format("20060102"))
	return newDate, nil
}

// ----------------------------------------------------------------------
// ВСПОМОГАТЕЛЬНЫЕ ФУНКЦИИ для работы с БД (scheduler)
// ----------------------------------------------------------------------

// getTaskByID читает задачу по ID из таблицы scheduler.
func getTaskByID(id string) (Task, error) {
	log.Println("🔍 [getTaskByID] Подключаемся к БД...")
	db, err := database.GetDB()
	if err != nil {
		log.Printf("🚨 [getTaskByID] Ошибка получения DB: %v\n", err)
		return Task{}, errors.New("ошибка подключения к БД")
	}

	log.Printf("🔍 [getTaskByID] Парсим id='%s' в int...\n", id)
	idInt, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		log.Printf("🚨 [getTaskByID] Невалидный ID='%s': %v\n", id, err)
		return Task{}, errors.New("задача не найдена")
	}

	var t Task
	log.Println("🔍 [getTaskByID] Выполняем SELECT ... FROM scheduler WHERE id=?")
	row := db.QueryRow(`SELECT id, date, title, comment, repeat FROM scheduler WHERE id=?`, idInt)
	err = row.Scan(&t.ID, &t.Date, &t.Title, &t.Comment, &t.Repeat)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Println("🚨 [getTaskByID] Запись не найдена")
			return Task{}, errors.New("задача не найдена")
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
		return errors.New("ошибка подключения к БД")
	}

	log.Printf("🔍 [deleteTaskByID] Парсим id='%s' в int...\n", id)
	idInt, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		log.Printf("🚨 [deleteTaskByID] Невалидный ID='%s': %v\n", id, err)
		return errors.New("задача не найдена")
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
		return errors.New("задача не найдена")
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
		return errors.New("ошибка подключения к БД")
	}

	log.Printf("🔍 [updateTaskDate] Парсим id='%s' в int...\n", id)
	idInt, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		log.Printf("🚨 [updateTaskDate] Невалидный ID='%s': %v\n", id, err)
		return errors.New("неверный ID")
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
		return errors.New("задача не найдена")
	}
	log.Println("✅ [updateTaskDate] Дата задачи успешно обновлена!")
	return nil
}

// jsonResponse отправляет JSON-ответ клиенту.
func jsonResponse(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		log.Printf("🚨 Ошибка кодирования JSON: %v\n", err)
		http.Error(w, `{"error":"Ошибка генерации ответа"}`, http.StatusInternalServerError)
	}
}
