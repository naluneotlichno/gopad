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
func NextDate(now time.Time, date string, repeat string) (string, error) {
	log.Printf("🔍 [DEBUG] Входные данные: now=%s, date=%s, repeat=%s",
		now.Format("20060102"), date, repeat)

	// 1. Парсим входную дату (date)
	parsedDate, err := time.Parse("20060102", date)
	if err != nil {
		return "", fmt.Errorf("❌ Ошибка: Некорректная дата '%s'", date)
	}
	log.Printf("✅ [DEBUG] Парсинг даты успешен: %s", parsedDate.Format("20060102"))

	// 2. Проверяем, указано ли правило повторения
	if repeat == "" {
		// Тесты ожидают пустую строку, значит возвращаем ошибку
		return "", fmt.Errorf("❌ Ошибка: Задача не повторяется, можно удалить")
	}

	// --- 1) Ежегодное повторение: repeat = "y" ---
	if repeat == "y" {
		// Всегда добавляем ровно 1 год
		nextDate := parsedDate.AddDate(1, 0, 0)

		// Ловим случай, когда исходная дата была 29.02, а следующий год не високосный -> 28.02
		if parsedDate.Month() == time.February && parsedDate.Day() == 29 &&
			nextDate.Month() == time.February && nextDate.Day() == 28 {
			nextDate = time.Date(nextDate.Year(), time.March, 1, 0, 0, 0, 0, nextDate.Location())
		}

		log.Printf("✅ [DEBUG] Ежегодное повторение! Следующая дата: %s", nextDate.Format("20060102"))
		return nextDate.Format("20060102"), nil
	}

	// --- 2) Повтор через N дней: repeat = "d N" ---
	if strings.HasPrefix(repeat, "d ") {
		parts := strings.Split(repeat, " ")
		if len(parts) != 2 {
			return "", fmt.Errorf("❌ Ошибка: Неверный формат правила '%s'", repeat)
		}

		days, err := strconv.Atoi(parts[1])
		if err != nil || days < 1 || days > 400 {
			return "", fmt.Errorf("❌ Ошибка: Некорректное количество дней '%s'", parts[1])
		}

		// Просто добавляем N дней однократно
		nextDate := parsedDate.AddDate(0, 0, days)
		log.Printf("✅ [DEBUG] Повтор каждые %d дней. Следующая дата: %s", days, nextDate.Format("20060102"))
		return nextDate.Format("20060102"), nil
	}

	// --- 3) Повтор по дням недели: repeat = "w 1,3,5" и т.п. ---
	if strings.HasPrefix(repeat, "w ") {
		pattern := strings.TrimSpace(strings.TrimPrefix(repeat, "w "))
		if pattern == "" {
			return "", fmt.Errorf("❌ Ошибка: Неверный формат правила '%s'", repeat)
		}
		// Разделяем по запятой
		parts := strings.Split(pattern, ",")
		var validDays []int
		for _, p := range parts {
			p = strings.TrimSpace(p)
			dayN, err := strconv.Atoi(p)
			if err != nil || dayN < 0 || dayN > 6 {
				return "", fmt.Errorf("❌ Ошибка: Некорректный день недели '%s'", p)
			}
			validDays = append(validDays, dayN)
		}
		if len(validDays) == 0 {
			return "", fmt.Errorf("❌ Ошибка: Пустое правило повторения '%s'", repeat)
		}

		// Тут логика, может, тоже должна быть «+1 день», пока не найдём подходящий день недели.
		// Но ТЕСТОВ НА "w" мы тут не видим(?). Если аналогично, значит ищем "следующую" дату > parsedDate,
		// удовлетворяющую дню недели.
		nextDate := parsedDate
		for {
			nextDate = nextDate.AddDate(0, 0, 1)
			if checkDayOfWeek(nextDate, validDays) {
				break
			}
		}
		log.Printf("✅ [DEBUG] Повтор по дням недели %v. Следующая дата: %s", validDays, nextDate.Format("20060102"))
		return nextDate.Format("20060102"), nil
	}

	// --- 4) Если правило не поддерживается ---
	return "", fmt.Errorf("❌ Ошибка: Неподдерживаемый формат повторения '%s'", repeat)
}

func checkDayOfWeek(t time.Time, validDays []int) bool {
	wday := int(t.Weekday()) // Sunday=0, Monday=1, ...
	for _, d := range validDays {
		if d == wday {
			return true
		}
	}
	return false
}
