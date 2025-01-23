package database

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/mattn/go-sqlite3"
)

var db *sql.DB

func GetDBPath() string {
	// Получаем путь к корневой директории проекта
	workingDir, err := os.Getwd() // Это вернёт текущую рабочую директорию
	if err != nil {
		log.Fatalf("❌ Ошибка определения рабочего каталога: %v", err)
	}

	// Добавляем имя файла базы данных
	dbPath := filepath.Join(workingDir, "scheduler.db")

	// Если переменная окружения TODO_DBFILE задана, используем её
	if envDB := os.Getenv("TODO_DBFILE"); envDB != "" {
		return envDB
	}

	return dbPath
}

// InitDB создаёт таблицу scheduler, если её нет
func InitDB(dbPath string) error {
	var err error
	db, err = sql.Open("sqlite3", dbPath)
	if err != nil {
		return err
	}

	log.Printf("✅🔌 Подключаемся к базе: %s", dbPath)

	if err := db.Ping(); err != nil {
		return fmt.Errorf("❌ Не удалось подключиться к базе: %w", err)
	}

	createTableSQL := `
	CREATE TABLE IF NOT EXISTS scheduler (
		id INTEGER PRIMARY KEY AUTOINCREMENT, 
		date TEXT NOT NULL,
		title TEXT NOT NULL, 
		comment TEXT, 
		repeat TEXT(128)
	);
	CREATE INDEX IF NOT EXISTS idx_date ON scheduler(date); 
	CREATE INDEX IF NOT EXISTS idx_title ON scheduler(title);
	`
	_, err = db.Exec(createTableSQL)
	if err != nil {
		log.Printf("❌ Ошибка при создании таблицы: %v", err)
		return err
	}

	log.Printf("✅ Таблица scheduler в [%s] создана или уже есть", dbPath)
	return nil
}

func GetDB() (*sql.DB, error) {
	if db == nil {
		return nil, fmt.Errorf("❌ База данных не инициализирована. Сначала вызовите InitDB()")
	}
	return db, nil
}

func deleteTask(id int64) error {

	res, err := db.Exec(`DELETE FROM scheduler WHERE id=?`, idInt)
	if err != nil {
		log.Printf("🚨 [deleteTaskByID] Ошибка выполнения DELETE: %v\n", err)
		return err
	}

	db, err := GetDB()
	if err != nil {
		log.Printf("🚨 [deleteTaskByID] Ошибка подключения к базе: %v\n", err)
		return errors.New("ошибка подключения к БД")
	}

	idInt, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		log.Printf("🚨 [deleteTaskByID] Невалидный ID=%s: %v\n", id, err)
		return errors.New("задача не найдена")
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
