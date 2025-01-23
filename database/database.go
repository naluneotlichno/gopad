package database

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/mattn/go-sqlite3"
	"github.com/naluneotlichno/FP-GO-API/api"
)

var db *sql.DB
var ErrTask = fmt.Errorf("задача не найдена")

type Task struct {
	ID      int64  `json:"id"`
	Date    string `json:"date"`
	Title   string `json:"title"`
	Comment string `json:"comment"`
	Repeat  string `json:"repeat"`
}

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

func DeleteTask(id int64) error {

	res, err := db.Exec(`DELETE FROM scheduler WHERE id=?`, id)
	if err != nil {
		log.Printf("🚨 [deleteTaskByID] Ошибка выполнения DELETE: %v\n", err)
		return err
	}

	n, err := res.RowsAffected()
	if err != nil {
		log.Printf("🚨 [deleteTaskByID] Ошибка при получении количества затронутых строк: %v\n", err)
		return err
	}

	if n == 0 {
		log.Printf("🚨 [deleteTaskByID] Задача ID=%d не найдена\n", id)
		return errors.New("задача не найдена")
	}

	log.Printf("✅ [deleteTaskByID] Задача ID=%d успешно удалена\n", id)
	return nil
}

func UpdateTask(task Task) error {
	_, err := api.NextDate(time.Now(), task.Date, task.Repeat, "check")
	if err != nil {
		return fmt.Errorf("ошибка при вычислении следующей даты: %w", err)
	}

	query := `
		UPDATE scheduler
		SET date = ?, title = ?, comment = ?, repeat = ?
		WHERE id = ?
	`

	res, err := db.Exec(query, task.Date, task.Title, task.Comment, task.Repeat, task.ID)
	if err != nil {
		return fmt.Errorf("ошибка при обновлении задачи: %w", err)
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("ошибка при получении количества затронутых строк: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("задача не найдена")
	}

	return nil
}

func GetTaskByID(id int64) (Task, error) {
	var task Task
	log.Println("🔍 [getTaskByID] Выполняем SELECT...")
	query := `SELECT id, date, title, comment, repeat FROM scheduler WHERE id=?`
	err := db.QueryRow(query, id).Scan(&task.ID, &task.Date, &task.Title, &task.Comment, &task.Repeat)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			log.Printf("🚨 [getTaskByID] Задача ID=%s не найдена\n", id)
			return Task{}, errors.New("задача не найдена")
		}
		log.Printf("🚨 [getTaskByID] Ошибка выполнения запроса: %v\n", err)
		return Task{}, err
	}
	log.Printf("✅ [getTaskByID] Найдена задача: %#v\n", task)
	return task, nil
}
