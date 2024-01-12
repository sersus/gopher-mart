package databases

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"os"
	"runtime"
	"strings"

	_ "github.com/jackc/pgx/v5/stdlib"
)

type WithdrawalRow struct {
	ID     int64
	userID int64
	Sum    float64
	Order  string
}

type DatabaseClient struct {
	DB *sql.DB
}

func NewDatabaseClient(databaseURI string) (*DatabaseClient, error) {
	db, err := sql.Open("pgx", databaseURI)
	if err != nil {
		return nil, err
	}

	client := &DatabaseClient{DB: db}
	err = client.migrate()
	if err != nil {
		return nil, err
	}

	return client, nil
}

func (client *DatabaseClient) GetWithdrawalsByUserID(userID int64) (*[]WithdrawalRow, error) {
	queryRows, queryRowError := client.DB.Query(`
		SELECT id, sum, orderid FROM withdrawals where userId = $1
	`, userID)
	if queryRowError != nil && !errors.Is(queryRowError, sql.ErrNoRows) {
		return nil, queryRowError
	}
	defer queryRows.Close()

	var withdrawalRows = make([]WithdrawalRow, 0)

	if errors.Is(queryRowError, sql.ErrNoRows) {
		return &withdrawalRows, nil
	}

	for queryRows.Next() {
		var r WithdrawalRow

		err := queryRows.Scan(&r.ID, &r.Sum, &r.Order)
		if err != nil {
			return nil, err
		}

		withdrawalRows = append(withdrawalRows, r)
	}

	rowsError := queryRows.Err()
	if rowsError != nil {
		return nil, rowsError
	}

	return &withdrawalRows, nil
}

func (client *DatabaseClient) OrderProcessing(res http.ResponseWriter, requestOrderID int64, userID int64) error {
	var queryUserID int64
	queryRow := client.DB.QueryRow(`
		SELECT userId FROM orders where id = $1
	`, requestOrderID)

	queryRowError := queryRow.Scan(&queryUserID)
	if queryRowError != nil && !errors.Is(queryRowError, sql.ErrNoRows) {
		fmt.Println(location(), "sql.ErrNoRows")

		http.Error(res, queryRowError.Error(), http.StatusInternalServerError)
		return queryRowError
	}
	if queryRowError == nil {
		fmt.Println(location(), "no error")

		if queryUserID == userID {
			res.WriteHeader(http.StatusOK)
			return nil
		}
		fmt.Println(location(), "order has already been uploaded by another user")

		http.Error(res, "order has already been uploaded by another user", http.StatusConflict)
		return nil

	}
	fmt.Println(location(), "insert")

	_, insertError := client.DB.Exec(`
		INSERT INTO orders (id, userId) values ($1, $2)
	`, requestOrderID, userID)
	if insertError != nil {
		fmt.Println(location(), "insert error")
		http.Error(res, insertError.Error(), http.StatusInternalServerError)
		return insertError
	}
	return nil
}

func (client *DatabaseClient) migrate() error {
	_, err := client.DB.Exec(`
		CREATE TABLE users IF NOT EXISTS
		(
			id BIGSERIAL PRIMARY KEY,
			login TEXT NOT NULL,
			password TEXT NOT NULL
		)`,
	)
	if err != nil {
		fmt.Println(err)
		return err
	}

	_, err = client.DB.Exec(`
		CREATE TABLE orders IF NOT EXISTS
		(
			id BIGSERIAL PRIMARY KEY,
			userId BIGSERIAL
		)`,
	)
	if err != nil {
		fmt.Println(err)
		return err
	}

	_, err = client.DB.Exec(`
		CREATE TABLE withdrawals IF NOT EXISTS
		(
			id BIGSERIAL PRIMARY KEY,
			userId BIGSERIAL,
			sum FLOAT8,
			orderId TEXT
		)`,
	)
	if err != nil {
		fmt.Println(err)
		return err
	}
	return nil
}

func (client *DatabaseClient) GetUserID(login string) (int64, error) {
	queryRow := client.DB.QueryRow(`
		SELECT id FROM users where login = $1
	`, login)

	var userID int64

	err := queryRow.Scan(&userID)
	return userID, err
}

func (client *DatabaseClient) InsertUser(login, password string) (int64, error) {
	insertQueryRow := client.DB.QueryRow(`
		INSERT INTO users (login, password) VALUES ($1, $2) RETURNING id
	`, login, password)

	var newID int64

	err := insertQueryRow.Scan(&newID)
	return newID, err
}

func (client *DatabaseClient) GetOrders(res http.ResponseWriter, userID int64) (*[]int64, error) {
	var orderIDs = make([]int64, 0)
	queryRows, queryRowError := client.DB.Query(`
		SELECT id FROM orders where userId = $1
	`, userID)
	if queryRowError != nil && !errors.Is(queryRowError, sql.ErrNoRows) {
		http.Error(res, queryRowError.Error(), http.StatusInternalServerError)
		return &orderIDs, queryRowError
	}
	defer queryRows.Close()

	if errors.Is(queryRowError, sql.ErrNoRows) {
		res.WriteHeader(http.StatusNoContent)
		return &orderIDs, queryRowError
	}

	for queryRows.Next() {
		var orderID int64

		err := queryRows.Scan(&orderID)
		if err != nil {
			http.Error(res, queryRowError.Error(), http.StatusInternalServerError)
			return &orderIDs, err
		}

		orderIDs = append(orderIDs, orderID)
	}

	rowsError := queryRows.Err()
	if rowsError != nil {
		http.Error(res, queryRowError.Error(), http.StatusInternalServerError)
		return &orderIDs, rowsError
	}
	return &orderIDs, queryRowError
}

func (client *DatabaseClient) InsertWithdrawal(userID int64, sum float64, orderID string) error {
	_, insertError := client.DB.Exec(`
		INSERT INTO withdrawals (userId, sum, orderid) values ($1, $2, $3)
	`, userID, sum, orderID)
	return insertError
}

func (client *DatabaseClient) GetUser(login string) (int64, string, error) {
	queryRow := client.DB.QueryRow(`
		SELECT id, password FROM users where login = $1
	`, login)

	var userID int64
	var password string

	err := queryRow.Scan(&userID, &password)
	return userID, password, err
}

func location() string {
	_, file, line, _ := runtime.Caller(1)
	p, _ := os.Getwd()
	return fmt.Sprintf("%s:%d", strings.TrimPrefix(file, p), line)
}
