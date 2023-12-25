package databases

import (
	"database/sql"
	"errors"
	"fmt"

	_ "github.com/jackc/pgx/v5/stdlib"
)

type WithdrawalRow struct {
	ID     int64
	userID int64
	Sum    float64
	Order  string
}

var DB *sql.DB

func Init(databaseURI string) error {
	var err error

	DB, err = sql.Open("pgx", databaseURI)
	if err != nil {
		return err
	}

	migrate()

	return nil
}

func GetWithdrawalsByUserID(userID int64) (*[]WithdrawalRow, error) {
	queryRows, queryRowError := DB.Query(`
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

func migrate() {
	_, err := DB.Exec(`
		CREATE TABLE users
		(
			id BIGSERIAL PRIMARY KEY,
			login TEXT NOT NULL,
			password TEXT NOT NULL
		)`,
	)
	if err != nil {
		fmt.Println(err)
	}

	_, err = DB.Exec(`
		CREATE TABLE orders
		(
			id BIGSERIAL PRIMARY KEY,
			userId BIGSERIAL
		)`,
	)
	if err != nil {
		fmt.Println(err)
	}

	_, err = DB.Exec(`
		CREATE TABLE withdrawals
		(
			id BIGSERIAL PRIMARY KEY,
			userId BIGSERIAL,
			sum FLOAT8,
			orderId TEXT
		)`,
	)
	if err != nil {
		fmt.Println(err)
	}
}
