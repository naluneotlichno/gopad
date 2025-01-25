package database

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/naluneotlichno/FP-GO-API/nextdate"
)

var db *sql.DB
var ErrTask = errors.New("задача не найдена")

type Task struct {
	ID      int64  `json:"id"`
	Date    string `json:"date"`
	Title   string `json:"title"`
	Comment string `json:"comment"`
	Repeat  string `json:"repeat"`
}

// GetDBPath возвращает путь к файлу базы данных
func GetDBPath() string {
	// Получаем путь к корневой директории проекта
	workingDir, err := os.Getwd()
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
		return fmt.Errorf("❌ Ошибка при открытии базы данных: %w", err)
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
		return fmt.Errorf("❌ Ошибка при создании таблицы: %w", err)
	}

	log.Printf("✅ Таблица scheduler в [%s] создана или уже существует", dbPath)
	return nil
}

// GetDB возвращает экземпляр базы данных
func GetDB() (*sql.DB, error) {
	if db == nil {
		return nil, fmt.Errorf("❌ База данных не инициализирована. Сначала вызовите InitDB()")
	}
	return db, nil
}

// DeleteTask удаляет задачу по её ID
func DeleteTask(id int64) error {
	dbInstance, err := GetDB()
	if err != nil {
		return err
	}

	res, err := dbInstance.Exec("DELETE FROM scheduler WHERE id = ?", id)
	if err != nil {
		log.Printf("🚨 [DeleteTask] Ошибка выполнения DELETE: %v\n", err)
		return fmt.Errorf("🚨 [DeleteTask] Ошибка выполнения DELETE: %w", err)
	}

	n, err := res.RowsAffected()
	if err != nil {
		log.Printf("🚨 [DeleteTask] Ошибка при получении количества затронутых строк: %v\n", err)
		return fmt.Errorf("🚨 [DeleteTask] Ошибка при получении количества затронутых строк: %w", err)
	}

	if n == 0 {
		log.Printf("🚨 [DeleteTask] Задача ID=%d не найдена\n", id)
		return ErrTask
	}

	log.Printf("✅ [DeleteTask] Задача ID=%d успешно удалена\n", id)
	return nil
}

// UpdateTask обновляет существующую задачу
func UpdateTask(task Task) error {
	_, err := nextdate.NextDate(time.Now(), task.Date, task.Repeat, "check")
	if err != nil {
		return fmt.Errorf("ошибка при вычислении следующей даты: %w", err)
	}

	dbInstance, err := GetDB()
	if err != nil {
		return err
	}

	query := `
		UPDATE scheduler
		SET date = ?, title = ?, comment = ?, repeat = ?
		WHERE id = ?
	`

	res, err := dbInstance.Exec(query, task.Date, task.Title, task.Comment, task.Repeat, task.ID)
	if err != nil {
		return fmt.Errorf("ошибка при обновлении задачи: %w", err)
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("ошибка при получении количества затронутых строк: %w", err)
	}

	if rowsAffected == 0 {
		return ErrTask
	}

	log.Printf("✅ [UpdateTask] Задача ID=%d успешно обновлена\n", task.ID)
	return nil
}

// GetTaskByID возвращает задачу по её ID
func GetTaskByID(id int64) (Task, error) {
	var task Task
	log.Println("🔍 [GetTaskByID] Выполняем SELECT...")
	query := "SELECT id, date, title, comment, repeat FROM scheduler WHERE id = ?"
	dbInstance, err := GetDB()
	if err != nil {
		return Task{}, err
	}

	err = dbInstance.QueryRow(query, id).Scan(&task.ID, &task.Date, &task.Title, &task.Comment, &task.Repeat)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			log.Printf("🚨 [GetTaskByID] Задача ID=%d не найдена\n", id)
			return Task{}, ErrTask
		}
		log.Printf("🚨 [GetTaskByID] Ошибка выполнения запроса: %v\n", err)
		return Task{}, fmt.Errorf("🚨 [GetTaskByID] Ошибка выполнения запроса: %w", err)
	}
	log.Printf("✅ [GetTaskByID] Найдена задача: %#v\n", task)
	return task, nil
}

// AddTask добавляет новую задачу и возвращает её ID
func AddTask(t Task) (int64, error) {
	dbInstance, err := GetDB()
	if err != nil {
		return 0, err
	}

	query := "INSERT INTO scheduler (date, title, comment, repeat) VALUES (?, ?, ?, ?)"

	res, err := dbInstance.Exec(query, t.Date, t.Title, t.Comment, t.Repeat)
	if err != nil {
		return 0, fmt.Errorf("ошибка при добавлении задачи: %w", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("ошибка при получении ID последней вставленной записи: %w", err)
	}

	log.Printf("✅ [AddTask] Задача добавлена с ID=%d\n", id)
	return id, nil
}

// GetUpcomingTasks возвращает список предстоящих задач
func GetUpcomingTasks() ([]Task, error) {
	dbInstance, err := GetDB()
	if err != nil {
		return nil, err
	}

	query := "SELECT id, date, title, comment, repeat FROM scheduler"
	rows, err := dbInstance.Query(query)
	if err != nil {
		return nil, fmt.Errorf("ошибка при выполнении запроса: %w", err)
	}
	defer rows.Close()

	tasks := []Task{}
	now := time.Now()

	for rows.Next() {
		var task Task
		err := rows.Scan(&task.ID, &task.Date, &task.Title, &task.Comment, &task.Repeat)
		if err != nil {
			return nil, fmt.Errorf("ошибка при чтении строки из результата: %w", err)
		}

		// Предполагается, что формат даты - "20060102". Измени его, если используется другой формат.
		taskDate, err := time.Parse("20060102", task.Date)
		if err != nil {
			return nil, fmt.Errorf("ошибка при разборе даты задачи ID %d: %w", task.ID, err)
		}

		if taskDate.Before(now) || taskDate.Equal(now) {
			nextDateStr, err := nextdate.NextDate(now, task.Date, task.Repeat, "list")
			if err != nil {
				return nil, fmt.Errorf("ошибка при вычислении следующей даты для задачи ID %d: %w", task.ID, err)
			}
			task.Date = nextDateStr
		}
		tasks = append(tasks, task)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("ошибка при обработке результатов запроса: %w", err)
	}

	// Сортировка задач по дате
	sort.Slice(tasks, func(i, j int) bool {
		return tasks[i].Date < tasks[j].Date
	})

	// Ограничение списка задач до 50
	if len(tasks) > 50 {
		tasks = tasks[:50]
	}

	log.Printf("✅ [GetUpcomingTasks] Получено %d задач\n", len(tasks))
	return tasks, nil
}


// package database

// import (
// 	"database/sql"
// 	"errors"
// 	"fmt"
// 	"log"
// 	"os"
// 	"path/filepath"
// 	"sort"
// 	"time"

// 	_ "github.com/go-sql-driver/mysql"
// 	_ "github.com/mattn/go-sqlite3"
// 	"github.com/naluneotlichno/FP-GO-API/nextdate"
// )

// var db *sql.DB
// var ErrTask = fmt.Errorf("задача не найдена")

// type Task struct {
// 	ID      int64  `json:"id"`
// 	Date    string `json:"date"`
// 	Title   string `json:"title"`
// 	Comment string `json:"comment"`
// 	Repeat  string `json:"repeat"`
// }

// func GetDBPath() string {
// 	// Получаем путь к корневой директории проекта
// 	workingDir, err := os.Getwd() // Это вернёт текущую рабочую директорию
// 	if err != nil {
// 		log.Fatalf("❌ Ошибка определения рабочего каталога: %v", err)
// 	}

// 	// Добавляем имя файла базы данных
// 	dbPath := filepath.Join(workingDir, "scheduler.db")

// 	// Если переменная окружения TODO_DBFILE задана, используем её
// 	if envDB := os.Getenv("TODO_DBFILE"); envDB != "" {
// 		return envDB
// 	}

// 	return dbPath
// }

// // InitDB создаёт таблицу scheduler, если её нет
// func InitDB(dbPath string) error {
// 	var err error
// 	db, err = sql.Open("sqlite3", dbPath)
// 	if err != nil {
// 		return err
// 	}

// 	log.Printf("✅🔌 Подключаемся к базе: %s", dbPath)

// 	if err := db.Ping(); err != nil {
// 		return fmt.Errorf("❌ Не удалось подключиться к базе: %w", err)
// 	}

// 	createTableSQL := `
// 	CREATE TABLE IF NOT EXISTS scheduler (
// 		id INTEGER PRIMARY KEY AUTOINCREMENT, 
// 		date TEXT NOT NULL,
// 		title TEXT NOT NULL, 
// 		comment TEXT, 
// 		repeat TEXT(128)
// 	);
// 	CREATE INDEX IF NOT EXISTS idx_date ON scheduler(date); 
// 	CREATE INDEX IF NOT EXISTS idx_title ON scheduler(title);
// 	`
// 	_, err = db.Exec(createTableSQL)
// 	if err != nil {
// 		log.Printf("❌ Ошибка при создании таблицы: %v", err)
// 		return err
// 	}

// 	log.Printf("✅ Таблица scheduler в [%s] создана или уже есть", dbPath)
// 	return nil
// }

// func GetDB() (*sql.DB, error) {
// 	if db == nil {
// 		return nil, fmt.Errorf("❌ База данных не инициализирована. Сначала вызовите InitDB()")
// 	}
// 	return db, nil
// }

// func DeleteTask(id int64) error {

// 	res, err := db.Exec(`DELETE FROM scheduler WHERE id=?`, id)
// 	if err != nil {
// 		log.Printf("🚨 [deleteTaskByID] Ошибка выполнения DELETE: %v\n", err)
// 		return err
// 	}

// 	n, err := res.RowsAffected()
// 	if err != nil {
// 		log.Printf("🚨 [deleteTaskByID] Ошибка при получении количества затронутых строк: %v\n", err)
// 		return err
// 	}

// 	if n == 0 {
// 		log.Printf("🚨 [deleteTaskByID] Задача ID=%d не найдена\n", id)
// 		return errors.New("задача не найдена")
// 	}

// 	log.Printf("✅ [deleteTaskByID] Задача ID=%d успешно удалена\n", id)
// 	return nil
// }

// func UpdateTask(task Task) error {
// 	_, err := nextdate.NextDate(time.Now(), task.Date, task.Repeat, "check")
// 	if err != nil {
// 		return fmt.Errorf("ошибка при вычислении следующей даты: %w", err)
// 	}

// 	query := `
// 		UPDATE scheduler
// 		SET date = ?, title = ?, comment = ?, repeat = ?
// 		WHERE id = ?
// 	`

// 	res, err := db.Exec(query, task.Date, task.Title, task.Comment, task.Repeat, task.ID)
// 	if err != nil {
// 		return fmt.Errorf("ошибка при обновлении задачи: %w", err)
// 	}

// 	rowsAffected, err := res.RowsAffected()
// 	if err != nil {
// 		return fmt.Errorf("ошибка при получении количества затронутых строк: %w", err)
// 	}

// 	if rowsAffected == 0 {
// 		return fmt.Errorf("задача не найдена")
// 	}

// 	return nil
// }

// func GetTaskByID(id int64) (Task, error) {
// 	var task Task
// 	log.Println("🔍 [getTaskByID] Выполняем SELECT...")
// 	query := `SELECT id, date, title, comment, repeat FROM scheduler WHERE id=?`
// 	err := db.QueryRow(query, id).Scan(&task.ID, &task.Date, &task.Title, &task.Comment, &task.Repeat)
// 	if err != nil {
// 		if errors.Is(err, sql.ErrNoRows) {
// 			log.Printf("🚨 [getTaskByID] Задача ID=%d не найдена\n", id)
// 			return Task{}, errors.New("задача не найдена")
// 		}
// 		log.Printf("🚨 [getTaskByID] Ошибка выполнения запроса: %v\n", err)
// 		return Task{}, err
// 	}
// 	log.Printf("✅ [getTaskByID] Найдена задача: %#v\n", task)
// 	return task, nil
// }

// func AddTask(t Task) (int64, error) {
// 	query := `INSERT INTO scheduler (date, title, comment, repeat) VALUES (?, ?, ?, ?)`

// 	res, err := db.Exec(query, t.Date, t.Title, t.Comment, t.Repeat)
// 	if err != nil {
// 		return 0, fmt.Errorf("ошибка при добавлении задачи: %w", err)
// 	}

// 	id, err := res.LastInsertId()
// 	if err != nil {
// 		return 0, fmt.Errorf("ошибка при получении ID последней вставленной записи: %w", err)
// 	}

// 	return id, nil
// }

// func GetUpcomingTasks() ([]Task, error) {
// 	query := `SELECT id, date, title, comment, repeat FROM scheduler`
// 	rows, err := db.Query(query)
// 	if err != nil {
// 		return nil, fmt.Errorf("ошибка при выполнении запроса: %w", err)
// 	}
// 	defer rows.Close()

// 	tasks := []Task{}
// 	now := time.Now()

// 	for rows.Next() {
// 		var task Task
// 		err := rows.Scan(&task.ID, &task.Date, &task.Title, &task.Comment, &task.Repeat)
// 		if err != nil {
// 			return nil, fmt.Errorf("ошибка при чтении строки из результата: %w", err)
// 		}

// 		taskDate, err := time.Parse("20060102", task.Date)
// 		if err != nil {
// 			return nil, fmt.Errorf("ошибка при разборе даты задачи ID %d: %w", task.ID, err)
// 		}

// 		if taskDate.Before(now) || taskDate.Equal(now) {
// 			nextDateStr, err := nextdate.NextDate(now, task.Date, task.Repeat, "list")
// 			if err != nil {
// 				return nil, fmt.Errorf("ошибка при вычислении следующей даты для задачи ID %d: %w", task.ID, err)
// 			}
// 			task.Date = nextDateStr
// 		}
// 		tasks = append(tasks, task)
// 	}

// 	if err = rows.Err(); err != nil {
// 		return nil, fmt.Errorf("ошибка при обработке результатов запроса: %w", err)
// 	}

// 	// Сортировка задач по дате
// 	sort.Slice(tasks, func(i, j int) bool {
// 		return tasks[i].Date < tasks[j].Date
// 	})

// 	// Ограничение списка задач до 50
// 	if len(tasks) > 50 {
// 		tasks = tasks[:50]
// 	}

// 	return tasks, nil
// }
