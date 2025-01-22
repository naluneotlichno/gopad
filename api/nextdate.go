package api

import (
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
	nextDate, err := NextDate(now, dateStr, repeat)
	if err != nil {
		http.Error(w, fmt.Sprintf("Ошибка расчета следующей даты: %s", err.Error()), http.StatusBadRequest)
		return
	}

	// ✅ Возвращаем результат клиенту
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	fmt.Fprint(w, nextDate)
}

// NextDate вычисляет следующую дату задачи на основе правила повторения.
// Возвращает дату в формате `20060102` (YYYYMMDD) или ошибку, если правило некорректно.
func NextDate(now time.Time, dateStr string, repeat string) (string, error) {
	// Парсим входную дату
	parsedDate, err := time.Parse("20060102", dateStr)
	if err != nil {
		// Если парсинг не удался, тесты ждут "пустой" (ошибку → пустой ответ)
		return "", fmt.Errorf("ошибка парсинга")
	}

	// Если repeat пустой, тест тоже ждёт "", значит ошибка
	if repeat == "" {
		return "", fmt.Errorf("правило повторения не задано")
	}

	// 1) Если repeat = "y" (ежегодное повторение)
	if repeat == "y" {
		// Будем прибавлять год, пока дата не станет строго больше now
		nextDate := parsedDate

		// Если parsedDate <= now, крутим год, пока не уйдём за now
		for !nextDate.After(now) {
			nextDate = addYearFixLeap(parsedDate, nextDate)
		}
		// Если parsedDate > now, делаем одну итерацию
		if parsedDate.After(now) && nextDate == parsedDate {
			nextDate = addYearFixLeap(parsedDate, nextDate)
		}

		return nextDate.Format("20060102"), nil
	}

	// 2) Если repeat = "d N" (повтор через N дней)
	if strings.HasPrefix(repeat, "d ") {
		parts := strings.Split(repeat, " ")
		if len(parts) != 2 {
			return "", fmt.Errorf("формат d N некорректен")
		}
		days, err := strconv.Atoi(parts[1])
		if err != nil || days < 1 || days > 400 {
			// ✅ ВАЖНО: тест ожидает ошибку, если days вне допустимого диапазона
			return "", fmt.Errorf("некорректное число дней '%s'", parts[1])
		}

		nextDate := parsedDate
		// Если parsedDate <= now, прибавляем +days, пока не вылезем за now
		for !nextDate.After(now) {
			nextDate = nextDate.AddDate(0, 0, days)
		}
		// Если parsedDate > now и мы всё ещё на исходной дате, делаем 1 итерацию
		if parsedDate.After(now) && nextDate == parsedDate {
			nextDate = nextDate.AddDate(0, 0, days)
		}

		return nextDate.Format("20060102"), nil
	}

	// 3) Если какие-то другие форматы (w ..., m ...), это уже другая логика
	return "", fmt.Errorf("неподдерживаемый repeat: %s", repeat)
}

// addYearFixLeap прибавляет ровно 1 год к currentDate, учитывая "скачок" с 29.02 на 01.03:
func addYearFixLeap(originalDate, currentDate time.Time) time.Time {
	next := currentDate.AddDate(1, 0, 0)
	// Если исходная дата была 29.02, а Go сдвинул на 28.02, то правим на 01.03
	if originalDate.Month() == time.February && originalDate.Day() == 29 &&
		next.Month() == time.February && next.Day() == 28 {
		return time.Date(next.Year(), time.March, 1, 0, 0, 0, 0, next.Location())
	}
	return next
}
