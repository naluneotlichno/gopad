package nextdate

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// 🔥 HandleNextDate обрабатывает запросы на /api/nextdate
func HandleNextDate(w http.ResponseWriter, r *http.Request) {
	log.Println("✅ Запрос на расчет даты получен!")

	// ✅ Извлекаем параметры из запроса
	nowStr := r.FormValue("now")
	dateStr := r.FormValue("date")
	repeat := r.FormValue("repeat")

	// ✅ Парсим параметр `now` в формате time.Time
	now, err := time.Parse("20060102", nowStr)
	if err != nil {
		http.Error(w, "Некорректная дата now", http.StatusBadRequest)
		return
	}

	// ✅ Вызываем функцию NextDate
	nextDate, err := NextDate(now, dateStr, repeat, "nextdate")
	if err != nil {
		http.Error(w, fmt.Sprintf("Ошибка расчета следующей даты: %s", err.Error()), http.StatusBadRequest)
		return
	}

	// ✅ Возвращаем результат клиенту
	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte(nextDate))
}

// NextDate вычисляет следующую дату задачи на основе правила повторения.
// Возвращает дату в формате `20060102` (YYYYMMDD) или ошибку, если правило некорректно.
func NextDate(now time.Time, dateStr string, repeat string, status string) (string, error) {
	log.Printf("🔍 Вызвана функция NextDate с параметрами: now=%s, date=%s, repeat=%s, status=%s\n", now.Format("20060102"), dateStr, repeat, status)
	if dateStr == "" {
		log.Println("❌ Не указана дата")
		return "", nil
	}

	parsedDate, err := time.Parse("20060102", dateStr)
	if err != nil {
		log.Printf("❌ Ошибка парсинга даты: %v\n", err)
		return "", nil
	}

	if repeat == "" {
		if parsedDate.After(now) {
			return parsedDate.Format("20060102"), nil
		}
		return "", nil
	}

	if strings.HasPrefix(repeat, "d ") {
		daysStr := strings.TrimPrefix(repeat, "d ")
		days, err := strconv.Atoi(daysStr)

		if err != nil || days < 1 || days > 400 {
			log.Printf("❌ Ошибка парсинга дней повторения: %s\n", repeat)
			return "", errors.New("неверное правило повторения")
		}

		if status != "done" {
			if isSameDate(parsedDate, now) {
				return parsedDate.Format("20060102"), nil
			}
		}

		nextDate := parsedDate.AddDate(0, 0, days)

		for !nextDate.After(now) {
			nextDate = nextDate.AddDate(0, 0, days)
		}

		return nextDate.Format("20060102"), nil
	}

	if repeat == "y" {
		nextDate := parsedDate.AddDate(1, 0, 0)
		if parsedDate.Month() == time.February && parsedDate.Day() == 29 {
			if nextDate.Month() != time.February || nextDate.Day() != 29 {
				nextDate = time.Date(nextDate.Year(), time.March, 1, 0, 0, 0, 0, nextDate.Location())
			}
		}

		if nextDate.Before(now) {
			for !nextDate.After(now) {
				nextDate = nextDate.AddDate(1, 0, 0)
			}
		}
		return nextDate.Format("20060102"), nil
	}

	log.Printf("❌ Неподдерживаемый формат повторения: %s\n", repeat)
	return "", errors.New("неподдерживаемый формат повторения")
}

func isSameDate(a, b time.Time) bool {
	return a.Year() == b.Year() && a.Month() == b.Month() && a.Day() == b.Day()
}
