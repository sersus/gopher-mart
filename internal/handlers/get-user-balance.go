package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"sync"

	"github.com/sersus/gopher-mart/internal/auth"
	"github.com/sersus/gopher-mart/internal/clients"
	"github.com/sersus/gopher-mart/internal/databases"
)

type GetUserBalanceHandlerResponse struct {
	Current   float64 `json:"current"`
	Withdrawn float64 `json:"withdrawn"`
}

func GetUserBalanceHandler(res http.ResponseWriter, req *http.Request) {
	tokenClaims := req.Context().Value(auth.TokenClaimsContextFieldName).(*auth.TokenClaims)

	queryRows, queryRowError := databases.DB.Query(`
		SELECT id FROM orders where userId = $1
	`, tokenClaims.UserID)
	if queryRowError != nil && !errors.Is(queryRowError, sql.ErrNoRows) {
		http.Error(res, queryRowError.Error(), http.StatusInternalServerError)
		return
	}
	defer queryRows.Close()

	if errors.Is(queryRowError, sql.ErrNoRows) {
		res.WriteHeader(http.StatusNoContent)
		return
	}

	var orderIDs = make([]int64, 0)

	for queryRows.Next() {
		var orderID int64

		err := queryRows.Scan(&orderID)
		if err != nil {
			http.Error(res, queryRowError.Error(), http.StatusInternalServerError)
			return
		}

		orderIDs = append(orderIDs, orderID)
	}

	rowsError := queryRows.Err()
	if rowsError != nil {
		http.Error(res, queryRowError.Error(), http.StatusInternalServerError)
		return
	}

	var wg sync.WaitGroup

	var accruals = make([]clients.OrderAccrualResponse, 0)

	wg.Add(len(orderIDs))

	go clients.GetOrdersAccruals(orderIDs, &accruals, &wg)

	wg.Wait()

	withdrawals, withdrawalsErrors := databases.GetWithdrawalsByUserID(tokenClaims.UserID)
	if withdrawalsErrors != nil {
		http.Error(res, withdrawalsErrors.Error(), http.StatusInternalServerError)
		return
	}

	response := GetUserBalanceHandlerResponse{}

	var withdrawalsSum float64

	for _, v := range *withdrawals {
		withdrawalsSum = withdrawalsSum + v.Sum
	}

	var currentFromRemote float64

	for _, accrual := range accruals {
		if accrual.Error != nil {
			http.Error(res, queryRowError.Error(), http.StatusInternalServerError)
			return
		}

		if accrual.OrderAccrual != nil && accrual.OrderAccrual.Status == clients.PROCESSED {
			currentFromRemote = currentFromRemote + accrual.OrderAccrual.AccrualValue
		}
	}

	response.Current = currentFromRemote - withdrawalsSum
	response.Withdrawn = withdrawalsSum

	marshaledResp, err := json.Marshal(response)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	res.Header().Set("content-type", "application/json")
	res.WriteHeader(http.StatusOK)
	res.Write(marshaledResp)
}
