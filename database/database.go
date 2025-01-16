package database

import (
	"database/sql"
	"log"
	"os"
	"runtime"
	"path/filepath"
	"fmt"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/mattn/go-sqlite3"
)

var db *sql.DB

func GetDBPath() string {
	// Возвращаем путь к базе данных как строку
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		log.Println("❌ runtime.Caller(0) не сработал, dbPath будет 'scheduler.db'")
		return "scheduler.db"
	}

	baseDir := filepath.Dir(filename)
	dbPath := filepath.Join(baseDir, "scheduler.db")

	if envDB := os.Getenv("TODO_DBFILE"); envDB != "" {
		return envDB
	}

	return dbPath
}

// InitDB создаёт таблицу scheduler, если её нет
func InitDB(dbPath string) error {
	var err error
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return err
	}

	log.Printf("✅🔌 Подключаемся к базе: %s", dbPath)

	defer func() {
		if cerr := db.Close(); cerr != nil {
			log.Printf("❌ Ошибка при закрытии соединения с БД: %v", cerr)
		}
	}()

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