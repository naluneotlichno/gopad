package api

import (
    "fmt"
    "log"
    "strconv"
    "strings"
    "time"
)

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
        // По условию теста в таком случае мы должны вернуть ошибку
        return "", fmt.Errorf("❌ Ошибка: Задача не повторяется, можно удалить")
    }

 

    // 1) Ежегодное повторение: repeat = "y"
    if repeat == "y" {
        nextDate := parsedDate
        // Если дата <= now, увеличиваем год, пока не станет > now
        for !nextDate.After(now) {
            nextDate = nextDate.AddDate(1, 0, 0)
        }
        log.Printf("✅ [DEBUG] Ежегодное повторение! Следующая дата: %s", nextDate.Format("20060102"))
        return nextDate.Format("20060102"), nil
    }

    // 2) Повтор через N дней: repeat = "d N"
    if strings.HasPrefix(repeat, "d ") {
        parts := strings.Split(repeat, " ")
        if len(parts) != 2 {
            return "", fmt.Errorf("❌ Ошибка: Неверный формат правила '%s'", repeat)
        }

        days, err := strconv.Atoi(parts[1])
        if err != nil || days < 1 || days > 400 {
            return "", fmt.Errorf("❌ Ошибка: Некорректное количество дней '%s'", parts[1])
        }

        nextDate := parsedDate
        // Если дата <= now, крутим +days, пока не станет > now
        for !nextDate.After(now) {
            nextDate = nextDate.AddDate(0, 0, days)
        }
        log.Printf("✅ [DEBUG] Повтор каждые %d дней. Следующая дата: %s", days, nextDate.Format("20060102"))
        return nextDate.Format("20060102"), nil
    }

    // 3) Повтор по дням недели: repeat = "w 1,3,5" и т.п.
    //    Допустим, Sunday=0, Monday=1, ..., Saturday=6. Нужно найти ближайшую дату, которая > now.
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

        // Ищем ближайшую дату, удовлетворяющую условию dayOfWeek ∈ validDays и > now
        nextDate := parsedDate
        // Сдвигаемся по дням вперёд, пока не найдём день, который больше now и подходит по день недели
        for !nextDate.After(now) || !containsDayOfWeek(nextDate, validDays) {
            nextDate = nextDate.AddDate(0, 0, 1)
        }
        log.Printf("✅ [DEBUG] Повтор по дням недели %v. Следующая дата: %s", validDays, nextDate.Format("20060102"))
        return nextDate.Format("20060102"), nil
    }

    // 4) Если правило не поддерживается
    return "", fmt.Errorf("❌ Ошибка: Неподдерживаемый формат повторения '%s'", repeat)
}

// containsDayOfWeek проверяет, попадает ли день недели даты t в список validDays
func containsDayOfWeek(t time.Time, validDays []int) bool {
    wday := int(t.Weekday()) // Sunday=0, Monday=1, ...
    for _, d := range validDays {
        if d == wday {
            return true
        }
    }
    return false
}
