package api

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/naluneotlichno/FP-GO-API/database"
	"github.com/naluneotlichno/FP-GO-API/nextdate"
)

// Те же имена структур, что в "КОД 1"
type AddTaskRequest struct {
	Date    string `json:"date"`
	Title   string `json:"title"`
	Comment string `json:"comment"`
	Repeat  string `json:"repeat"`
}

type AddTaskResponse struct {
	ID    string `json:"id,omitempty"`
	Error string `json:"error,omitempty"`
}

// Константа с нужным форматом даты
const layout = "20060102"

// AddTaskHandler обрабатывает POST-запросы на /api/task (аналог «КОД 1»).
func AddTaskHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("🚀 [AddTaskHandler] Начинаем обработку запроса")
	switch r.Method {
	case http.MethodPost:
		AddTask(w, r)
	default:
		http.Error(w, "Метод не поддерживается", http.StatusMethodNotAllowed)
	}

}

// func AddTask(w http.ResponseWriter, r *http.Request) {
// 	body, err := io.ReadAll(r.Body)
// 	if err != nil {
// 		JsonResponse(w, http.StatusBadRequest, AddTaskResponse{Error: "не удалось прочитать тело запроса"})
// 		return
// 	}
// 	defer r.Body.Close()

// 	var req AddTaskRequest
// 	err = json.Unmarshal(body, &req)
// 	if err != nil {
// 		JsonResponse(w, http.StatusBadRequest, AddTaskResponse{Error: "неверный формат JSON"})
// 		return
// 	}

// 	req.Title = strings.TrimSpace(req.Title)
// 	if req.Title == "" {
// 		JsonResponse(w, http.StatusBadRequest, AddTaskResponse{Error: "не указан заголовок задачи"})
// 		return
// 	}

// 	var taskDate time.Time
// 	now := time.Now()

// 	if strings.TrimSpace(req.Date) == "" {
// 		req.Date = now.Format(layout)
// 	}

// 	taskDate, err = time.Parse(layout, req.Date)
// 	if err != nil {
// 		JsonResponse(w, http.StatusBadRequest, AddTaskResponse{Error: "дата указана в неверном формате"})
// 		return
// 	}

// 	if taskDate.Before(now) {
// 		if strings.TrimSpace(req.Repeat) == "" {
// 			taskDate = now
// 		} else {
// 			nextDateStr, err := nextdate.NextDate(now, req.Date, req.Repeat, "add")
// 			if err != nil {
// 				JsonResponse(w, http.StatusBadRequest, AddTaskResponse{Error: "неверное правило повторения"})
// 				return
// 			}
// 			taskDate, _ = time.Parse(layout, nextDateStr)
// 		}
// 	}

// 	newTask := database.Task{
// 		Date:    taskDate.Format(layout),
// 		Title:   req.Title,
// 		Comment: req.Comment,
// 		Repeat:  req.Repeat,
// 	}

// 	id, err := database.AddTask(newTask)
// 	if err != nil {
// 		JsonResponse(w, http.StatusInternalServerError, AddTaskResponse{Error: err.Error()})
// 		return
// 	}

// 	JsonResponse(w, http.StatusCreated, AddTaskResponse{ID: fmt.Sprintf("%d", id)})
// }

func AddTask(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		JsonResponse(w, http.StatusBadRequest, AddTaskResponse{Error: "не удалось прочитать тело запроса"})
		return
	}
	defer r.Body.Close()

	var req AddTaskRequest
	err = json.Unmarshal(body, &req)
	if err != nil {
		JsonResponse(w, http.StatusBadRequest, AddTaskResponse{Error: "неверный формат JSON"})
		return
	}

	log.Printf("Получен запрос на добавление задачи: %+v", req) // Добавленное логирование

	req.Title = strings.TrimSpace(req.Title)
	if req.Title == "" {
		JsonResponse(w, http.StatusBadRequest, AddTaskResponse{Error: "не указан заголовок задачи"})
		return
	}

	var taskDate time.Time
	now := time.Now()

	if strings.TrimSpace(req.Date) == "" {
		req.Date = now.Format(layout)
	}

	taskDate, err = time.Parse(layout, req.Date)
	if err != nil {
		JsonResponse(w, http.StatusBadRequest, AddTaskResponse{Error: "дата указана в неверном формате"})
		return
	}

	if taskDate.Before(now) {
		if strings.TrimSpace(req.Repeat) == "" {
			taskDate = now
		} else {
			nextDateStr, err := nextdate.NextDate(now, req.Date, req.Repeat, "add")
			if err != nil {
				JsonResponse(w, http.StatusBadRequest, AddTaskResponse{Error: "неверное правило повторения"})
				return
			}
			taskDate, err = time.Parse(layout, nextDateStr)
			if err != nil {
				JsonResponse(w, http.StatusInternalServerError, AddTaskResponse{Error: "не удалось распарсить следующую дату"})
				return
			}
		}
	}

	log.Printf("Добавление задачи с датой: %s", taskDate.Format(layout)) // Добавленное логирование

	newTask := database.Task{
		Date:    taskDate.Format(layout),
		Title:   req.Title,
		Comment: req.Comment,
		Repeat:  req.Repeat,
	}

	log.Printf("Сохранение задачи в базе данных: %+v", newTask) // Добавленное логирование

	id, err := database.AddTask(newTask)
	if err != nil {
		JsonResponse(w, http.StatusInternalServerError, AddTaskResponse{Error: err.Error()})
		return
	}

	JsonResponse(w, http.StatusCreated, AddTaskResponse{ID: fmt.Sprintf("%d", id)})
}

type TaskResponseItem struct {
	ID      string `json:"id"`
	Date    string `json:"date"`
	Title   string `json:"title"`
	Comment string `json:"comment"`
	Repeat  string `json:"repeat"`
}

type TasksR struct {
	List []TaskResponseItem `json:"list"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

func Tasks(w http.ResponseWriter, r *http.Request) {
	tasks, err := database.GetUpcomingTasks()
	if err != nil {
		log.Printf("Ошибка получения задач: %v", err)
		JsonResponse(w, http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	response := TasksR{List: []TaskResponseItem{}}

	for _, t := range tasks {
		taskItem := TaskResponseItem{
			ID:      fmt.Sprintf("%d", t.ID),
			Date:    t.Date,
			Title:   t.Title,
			Comment: t.Comment,
			Repeat:  t.Repeat,
		}
		response.List = append(response.List, taskItem)
	}

	JsonResponse(w, http.StatusOK, response)
}
