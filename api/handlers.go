package api

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/naluneotlichno/FP-GO-API/nextdate"
)

// HandleNextDate обрабатывает запросы на /api/nextdate
func HandleNextDate(w http.ResponseWriter, r *http.Request) {
	log.Println("✅ Запрос на расчет даты получен!")

	// Извлекаем параметры из запроса
	nowStr := r.FormValue("now")
	dateStr := r.FormValue("date")
	repeat := r.FormValue("repeat")

	// Парсим параметр `now` в формате time.Time
	now, err := time.Parse("20060102", nowStr)
	if err != nil {
		http.Error(w, "Некорректная дата now", http.StatusBadRequest)
		return
	}

	// Вызываем функцию NextDate
	nextDate, err := nextdate.NextDate(now, dateStr, repeat)
	if err != nil {
		http.Error(w, fmt.Sprintf("Ошибка расчета следующей даты: %s", err.Error()), http.StatusBadRequest)
		return
	}

	// Возвращаем результат клиенту
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	fmt.Fprint(w, nextDate)
}
