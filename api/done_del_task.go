package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/naluneotlichno/FP-GO-API/database"
	"github.com/naluneotlichno/FP-GO-API/nextdate"
)

// DoneTaskHandler обрабатывает POST /api/task/done?id=...
func DoneTaskHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("🔥 [DoneTaskHandler] Запрос на /api/task/done получен...")

	idStr := r.URL.Query().Get("id")
	log.Printf("🔍 [DoneTaskHandler] ID из запроса: %s\n", idStr)
	if idStr == "" {
		log.Println("🚨 [DoneTaskHandler] ID не указан")
		JsonResponse(w, http.StatusBadRequest, map[string]string{"error": "Не указан идентификатор"})
		return
	}

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		log.Printf("🚨 [DoneTaskHandler] Ошибка парсинга ID=%d: %v\n", id, err)
		JsonResponse(w, http.StatusBadRequest, map[string]string{"error": "Некорректный идентификатор"})
		return
	}

	task, err := database.GetTaskByID(id)
	if err != nil {
		if errors.Is(err, database.ErrTask) {
			JsonResponse(w, http.StatusNotFound, map[string]string{"error": "Задача не найдена"})
			return
		}
		log.Printf("🚨 [DoneTaskHandler] Ошибка получения задачи ID=%d: %v\n", id, err)
		JsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "Ошибка при получении задачи"})
		return
	}

	log.Printf("✅ [DoneTaskHandler] Найдена задача: %#v\n", task)

	if task.Repeat == "" {
		log.Printf("🔍 [DoneTaskHandler] repeat пустой. Удаляем задачу ID=%d\n", id)
		if err := database.DeleteTask(id); err != nil {
			log.Printf("🚨 [DoneTaskHandler] Ошибка при удалении задачи ID=%d: %v\n", id, err)
			JsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "Ошибка при удалении задачи"})
			return
		}
	} else {
		now := time.Now()
		nextDate, err := nextdate.NextDate(now, task.Date, task.Repeat, "done")
		if err != nil {
			log.Printf("🚨 [DoneTaskHandler] Ошибка вычисления следующей даты: %v\n", err)
			JsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "Ошибка при вычислении следующей даты"})
			return
		}

		task.Date = nextDate
		err = database.UpdateTask(task)
		if err != nil {
			JsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "Ошибка при обновлении задачи"})
			return
		}
	}

	JsonResponse(w, http.StatusOK, map[string]any{})
}

func JsonResponse(w http.ResponseWriter, status int, payload interface{}) {
	log.Printf("📤 [jsonResponse] Отправляем ответ: статус=%d, payload=%#v\n", status, payload)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(payload)
}

// DeleteTaskHandler обрабатывает DELETE /api/task?id=...
func DeleteTaskHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("🔥 [DeleteTaskHandler] Запрос на DELETE /api/task получен...")

	idStr := r.URL.Query().Get("id")
	log.Printf("🔍 [DeleteTaskHandler] ID из запроса: %s\n", idStr)
	if idStr == "" {
		log.Println("🚨 [DeleteTaskHandler] ID не указан")
		JsonResponse(w, http.StatusBadRequest, map[string]string{"error": "Не указан идентификатор"})
		return
	}

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		log.Printf("🚨 [DeleteTaskHandler] Ошибка парсинга ID=%d: %v\n", id, err)
		JsonResponse(w, http.StatusBadRequest, map[string]string{"error": "Некорректный идентификатор"})
		return
	}

	log.Printf("🔍 [DeleteTaskHandler] Пытаемся удалить задачу с ID=%d\n", id)
	if err := database.DeleteTask(id); err != nil {
		if errors.Is(err, fmt.Errorf("задача не найдена")) {
			JsonResponse(w, http.StatusNotFound, map[string]string{"error": err.Error()})
			return
		}
		log.Printf("🚨 [DeleteTaskHandler] Ошибка удаления задачи ID=%d: %v\n", id, err)
		JsonResponse(w, http.StatusNotFound, map[string]string{"error": err.Error()})
		return
	}

	log.Printf("✅ [DeleteTaskHandler] Задача ID=%d успешно удалена\n", id)
	JsonResponse(w, http.StatusOK, map[string]interface{}{})
}
