package database

import (
	"database/sql"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

type DBConnection struct {
	Database  *sql.DB
	Connected bool
}

// Connect initializes the database connection and waits for it to become available or times out after a specified duration
func (dbConn *DBConnection) Connect(username, password, host, port, dbName string, timeout time.Duration) error {
	if dbName == "" {
		log.Println("No Database specified, skipping..")
		dbConn.Connected = false
		return nil
	}

	connectionString := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true", username, password, host, port, dbName)
	var err error
	dbConn.Database, err = sql.Open("mysql", connectionString)
	if err != nil {
		return err
	}

	startTime := time.Now()
	for {
		err = dbConn.Database.Ping()
		if err == nil {
			log.Printf("Connected to the database: [%s]", dbName)
			dbConn.Connected = true
			return nil
		}

		if time.Since(startTime) >= timeout {
			return fmt.Errorf("connection to the database: [%s] timed out after %v", dbName, timeout)
		}

		time.Sleep(1 * time.Second)
	}
}

// ExecuteQuery executes a single SQL query
func (dbConn *DBConnection) ExecuteQuery(query string, args ...interface{}) (sql.Result, error) {
	result, err := dbConn.Database.Exec(query, args...)
	if err != nil {
		return nil, err
	}

	return result, nil
}

// QueryRow executes a query that is expected to return a single row
func (dbConn *DBConnection) QueryRow(query string, args ...interface{}) *sql.Row {
	row := dbConn.Database.QueryRow(query, args...)
	return row
}

// QueryRows executes a query that is expected to return multiple rows
func (dbConn *DBConnection) QueryRows(query string, args ...interface{}) (*sql.Rows, error) {
	rows, err := dbConn.Database.Query(query, args...)
	if err != nil {
		return nil, err
	}
	return rows, nil
}

// ExecuteSQLFile executes the SQL statements from a .sql file
func (dbConn *DBConnection) ExecuteSQLFile(filename string) error {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}

	// Split the contents of the file into individual SQL statements
	sqlStatements := strings.Split(string(data), ";\n")

	tx, err := dbConn.Database.Begin()
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	for _, stmt := range sqlStatements {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}

		_, err := tx.Exec(stmt)
		if err != nil {
			return err
		}
	}

	err = tx.Commit()
	if err != nil {
		return err
	}

	return nil
}

// Load a mysqldump file into a database
func (dbConn *DBConnection) LoadDB(dbName string, directory string) {

	sqlFile := fmt.Sprintf("%s/data/%s.sql", directory, dbName)

	// create the db
	log.Printf("Creating new DB [%s]...\n", dbName)
	createQuery := fmt.Sprintf("CREATE DATABASE %s", dbName)
	_, err := dbConn.ExecuteQuery(createQuery)
	if err != nil {
		log.Fatalf("Failed to create DB: %v", err)
	}

	// bail if there is no file to import
	if !fileExists(sqlFile) {
		return
	}

	useQuery := fmt.Sprintf("USE %s", dbName)
	_, err = dbConn.ExecuteQuery(useQuery)
	if err != nil {
		log.Fatalf("Failed to switch DB: %v", err)
	}

	// load the data
	log.Printf("Loading SQL file for %s...\n", dbName)
	err = dbConn.ExecuteSQLFile(sqlFile)
	if err != nil {
		log.Fatalf("Failed to load SQL file: %v", err)
	}

	log.Printf("SQL file for %s loaded successfully!\n", dbName)
}

func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}
