// package nextdate

// import (
// 	"errors"
// 	"fmt"
// 	"log"
// 	"net/http"
// 	"strconv"
// 	"strings"
// 	"time"
// )

// // 🔥 HandleNextDate обрабатывает запросы на /api/nextdate
// func HandleNextDate(w http.ResponseWriter, r *http.Request) {
// 	log.Println("✅ Запрос на расчет даты получен!")

// 	// ✅ Извлекаем параметры из запроса
// 	nowStr := r.FormValue("now")
// 	dateStr := r.FormValue("date")
// 	repeat := r.FormValue("repeat")

// 	// ✅ Парсим параметр `now` в формате time.Time
// 	now, err := time.Parse("20060102", nowStr)
// 	if err != nil {
// 		http.Error(w, "Некорректная дата now", http.StatusBadRequest)
// 		return
// 	}

// 	// ✅ Вызываем функцию NextDate
// 	nextDate, err := NextDate(now, dateStr, repeat, "nextdate")
// 	if err != nil {
// 		http.Error(w, fmt.Sprintf("Ошибка расчета следующей даты: %s", err.Error()), http.StatusBadRequest)
// 		return
// 	}

// 	// ✅ Возвращаем результат клиенту
// 	w.Header().Set("Content-Type", "text/plain")
// 	w.Write([]byte(nextDate))
// }

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
	status := r.FormValue("status") // Сохраняем параметр `status`

	// ✅ Парсим параметр `now` в формате time.Time
	now, err := time.Parse("20060102", nowStr)
	if err != nil {
		http.Error(w, "Некорректная дата now", http.StatusBadRequest)
		return
	}

	// ✅ Вызываем функцию NextDate
	nextDate, err := NextDate(now, dateStr, repeat, status)
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
		return "", errors.New("не указана дата")
	}

	beginDate, err := time.Parse("20060102", dateStr)
	if err != nil {
		return "", fmt.Errorf("nextDate: некорректный формат даты: <%s>, %w", dateStr, err)
	}

	if repeat == "" {
		if beginDate.After(now) {
			return beginDate.Format("20060102"), nil
		}
		return "", nil
	}

	repeatSlice := strings.Split(repeat, " ")
	if len(repeatSlice) < 1 {
		return "", fmt.Errorf("nextDate: некорректный формат повторения: <%s>, повторение пусто", repeat)
	}

	modif := repeatSlice[0]

	switch modif {
	case "y":
		if len(repeatSlice) != 1 {
			return "", fmt.Errorf("nextDate: некорректный формат повторения: [%s], годовое повторение не должно иметь дополнительных значений", repeat)
		}

		next := beginDate.AddDate(1, 0, 0)
		for next.Before(now) {
			next = next.AddDate(1, 0, 0)
		}
		return next.Format("20060102"), nil

	case "d":
		if len(repeatSlice) != 2 {
			return "", fmt.Errorf("nextDate: некорректный формат повторения: [%s], повторение по дням должно иметь одно дополнительное значение", repeat)
		}

		days, err := strconv.Atoi(repeatSlice[1])
		if err != nil {
			return "", fmt.Errorf("nextDate: некорректное количество дней: [%s], %w", repeat, err)
		}

		if days < 1 || days > 400 {
			return "", fmt.Errorf("nextDate: количество дней должно быть между 1 и 400: [%s]", repeat)
		}

		// Обработка параметра `status`
		if status != "done" {
			if isSameDate(beginDate, now) {
				return beginDate.Format("20060102"), nil
			}
		}

		next := beginDate.AddDate(0, 0, days)
		for !next.After(now) {
			next = next.AddDate(0, 0, days)
		}
		return next.Format("20060102"), nil

	case "w":
		if len(repeatSlice) < 2 {
			return "", fmt.Errorf("nextDate: некорректный формат повторения: [%s], повторение по неделям должно иметь одно или более дополнительных значений", repeat)
		}

		weekDaysStringList := strings.Split(repeatSlice[1], ",")
		minDif := int64(^uint64(0) >> 1) // Максимальное значение int64
		var closestDate time.Time

		for _, ds := range weekDaysStringList {
			weekDay, err := strconv.Atoi(ds)
			if err != nil || weekDay < 1 || weekDay > 7 {
				return "", fmt.Errorf("nextDate: некорректный день недели: [%s], %w", ds, err)
			}

			dt := now
			if beginDate.After(now) {
				dt = beginDate
			}
			d, err := nextWeekDay(dt, weekDay)
			if err != nil {
				return "", fmt.Errorf("nextDate: %w", err)
			}

			dif := d.Sub(now).Milliseconds()
			if dif < minDif {
				minDif = dif
				closestDate = d
			}
		}

		if closestDate.IsZero() {
			return "", errors.New("nextDate: не удалось найти ближайшую дату для повторения по неделям")
		}

		return closestDate.Format("20060102"), nil

	case "m":
		if len(repeatSlice) < 2 {
			return "", fmt.Errorf("nextDate: некорректный формат повторения: [%s], повторение по месяцам должно иметь одно или более дополнительных значений", repeat)
		}

		daysStringList := strings.Split(repeatSlice[1], ",")
		var monthStringList []string
		if len(repeatSlice) == 3 {
			monthStringList = strings.Split(repeatSlice[2], ",")
		}

		minDif := int64(^uint64(0) >> 1)
		var closestDate time.Time

		for _, ds := range daysStringList {
			day, err := strconv.Atoi(ds)
			if err != nil {
				return "", fmt.Errorf("nextDate: некорректный день месяца: [%s], %w", ds, err)
			}

			if day == 0 || day < -31 || day > 31 {
				return "", fmt.Errorf("nextDate: день месяца должен быть между -31 и 31, не равен 0: [%d]", day)
			}

			if len(monthStringList) == 0 {
				// Повторение каждый месяц
				d, err := nextMonthDay(now, day)
				if err != nil {
					continue // Пропускаем некорректные даты
				}
				dif := d.Sub(now).Milliseconds()
				if dif < minDif {
					minDif = dif
					closestDate = d
				}
			} else {
				// Повторение в конкретные месяцы
				for _, ms := range monthStringList {
					month, err := strconv.Atoi(ms)
					if err != nil || month < 1 || month > 12 {
						return "", fmt.Errorf("nextDate: некорректный месяц: [%s], %w", ms, err)
					}

					d, err := nextSpecifiedDay(now, day, month)
					if err != nil {
						continue // Пропускаем некорректные даты
					}

					dif := d.Sub(now).Milliseconds()
					if dif < minDif {
						minDif = dif
						closestDate = d
					}
				}
			}
		}

		if closestDate.IsZero() {
			return "", errors.New("nextDate: не удалось найти ближайшую дату для повторения по месяцам")
		}

		return closestDate.Format("20060102"), nil

	default:
		return "", fmt.Errorf("nextDate: неподдерживаемый модификатор повторения: [%s]", modif)
	}
}

// isSameDate проверяет, совпадают ли две даты по году, месяцу и дню
func isSameDate(a, b time.Time) bool {
	return a.Year() == b.Year() && a.Month() == b.Month() && a.Day() == b.Day()
}

// nextWeekDay возвращает ближайшую дату указанного дня недели `weekDay` после `current`.
func nextWeekDay(current time.Time, weekDay int) (time.Time, error) {
	if weekDay < 1 || weekDay > 7 {
		return time.Time{}, fmt.Errorf("недопустимый день недели: %d", weekDay)
	}

	targetWeekday := time.Weekday(weekDay % 7) // Преобразуем 7 в 0 (Воскресенье)
	dif := (int(targetWeekday) - int(current.Weekday()) + 7) % 7
	if dif == 0 {
		dif = 7
	}
	return current.AddDate(0, 0, dif), nil
}

// nextMonthDay возвращает ближайшую дату указанного дня месяца `monthDay` после `current`.
func nextMonthDay(current time.Time, monthDay int) (time.Time, error) {
	if monthDay == 0 || monthDay < -31 || monthDay > 31 {
		return time.Time{}, fmt.Errorf("недопустимый день месяца: %d", monthDay)
	}

	year, month, _ := current.Date()
	location := current.Location()

	for i := 0; i < 24; i++ { // Ограничиваем поиск 2 годами вперед
		var day int
		currentMonth := time.Month(int(month) + i)
		if currentMonth > 12 {
			currentMonth -= 12
			year += 1
		}

		if monthDay > 0 {
			day = monthDay
		} else {
			// Отрицательные дни считаются от конца месяца
			day = monthLength(currentMonth) + monthDay + 1
		}

		// Проверяем корректность дня
		if day < 1 || day > monthLength(currentMonth) {
			continue
		}

		date := time.Date(year, currentMonth, day, 0, 0, 0, 0, location)
		if date.After(current) {
			return date, nil
		}
	}
	return time.Time{}, errors.New("не удалось найти корректную дату для повторения по месяцам")
}

// nextSpecifiedDay возвращает ближайшую дату указанного дня месяца `monthDay` и месяца `month` после `current`.
func nextSpecifiedDay(current time.Time, monthDay, month int) (time.Time, error) {
	if month < 1 || month > 12 {
		return time.Time{}, fmt.Errorf("недопустимый месяц: %d", month)
	}

	year, _, _ := current.Date()
	location := current.Location()

	// Корректируем год, если месяц уже прошел в текущем году
	if time.Month(month) < current.Month() || (time.Month(month) == current.Month() && monthDay < current.Day()) {
		year++
	}

	var day int
	if monthDay > 0 {
		day = monthDay
	} else {
		day = monthLength(time.Month(month)) + monthDay + 1
	}

	// Проверяем корректность дня
	if day < 1 || day > monthLength(time.Month(month)) {
		return time.Time{}, fmt.Errorf("недопустимый день месяца: %d для месяца: %d", monthDay, month)
	}

	date := time.Date(year, time.Month(month), day, 0, 0, 0, 0, location)
	if date.Before(current) {
		return date.AddDate(1, 0, 0), nil
	}
	return date, nil
}

// monthLength возвращает количество дней в заданном месяце.
func monthLength(m time.Month) int {
	return time.Date(2000, m+1, 0, 0, 0, 0, 0, time.UTC).Day()
}

// NextDate вычисляет следующую дату задачи на основе правила повторения.
// Возвращает дату в формате `20060102` (YYYYMMDD) или ошибку, если правило некорректно.
// func NextDate(now time.Time, dateStr string, repeat string, status string) (string, error) {
// 	log.Printf("🔍 Вызвана функция NextDate с параметрами: now=%s, date=%s, repeat=%s, status=%s\n", now.Format("20060102"), dateStr, repeat, status)
// 	if dateStr == "" {
// 		log.Println("❌ Не указана дата")
// 		return "", nil
// 	}

// 	parsedDate, err := time.Parse("20060102", dateStr)
// 	if err != nil {
// 		log.Printf("❌ Ошибка парсинга даты: %v\n", err)
// 		return "", nil
// 	}

// 	if repeat == "" {
// 		if parsedDate.After(now) {
// 			return parsedDate.Format("20060102"), nil
// 		}
// 		return "", nil
// 	}

// 	if strings.HasPrefix(repeat, "d ") {
// 		daysStr := strings.TrimPrefix(repeat, "d ")
// 		days, err := strconv.Atoi(daysStr)

// 		if err != nil || days < 1 || days > 400 {
// 			log.Printf("❌ Ошибка парсинга дней повторения: %s\n", repeat)
// 			return "", errors.New("неверное правило повторения")
// 		}

// 		if status != "done" {
// 			if isSameDate(parsedDate, now) {
// 				return parsedDate.Format("20060102"), nil
// 			}
// 		}

// 		nextDate := parsedDate.AddDate(0, 0, days)

// 		for !nextDate.After(now) {
// 			nextDate = nextDate.AddDate(0, 0, days)
// 		}

// 		return nextDate.Format("20060102"), nil
// 	}


//     if strings.HasPrefix(repeat, "w ") {
//         weeksStr := strings.TrimPrefix(repeat, "w ")
//         weeks, err := strconv.Atoi(weeksStr)
//         if err != nil || weeks < 1 || weeks > 52 {
//             log.Printf("Invalid repeat format: %s", repeat)
//             return "", errors.New("неверное правило повторения")
//         }

//         nextDate := parsedDate.AddDate(0, 0, weeks*7)
//         for !nextDate.After(now) {
//             nextDate = nextDate.AddDate(0, 0, weeks*7)
//         }

//         return nextDate.Format("20060102"), nil
//     }

// 	if repeat == "y" {
// 		nextDate := parsedDate.AddDate(1, 0, 0)
// 		if parsedDate.Month() == time.February && parsedDate.Day() == 29 {
// 			if nextDate.Month() != time.February || nextDate.Day() != 29 {
// 				nextDate = time.Date(nextDate.Year(), time.March, 1, 0, 0, 0, 0, nextDate.Location())
// 			}
// 		}

// 		if nextDate.Before(now) {
// 			for !nextDate.After(now) {
// 				nextDate = nextDate.AddDate(1, 0, 0)
// 			}
// 		}
// 		return nextDate.Format("20060102"), nil
// 	}

// 	log.Printf("❌ Неподдерживаемый формат повторения: %s\n", repeat)
// 	return "", errors.New("неподдерживаемый формат повторения")
// }

// func isSameDate(a, b time.Time) bool {
// 	return a.Year() == b.Year() && a.Month() == b.Month() && a.Day() == b.Day()
// }
