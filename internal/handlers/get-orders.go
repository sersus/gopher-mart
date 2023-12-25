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

type OrderJSON struct {
	Number  string  `json:"number"`
	Status  string  `json:"status"`
	Accrual float64 `json:"accrual"`
}

func GetOrdersHandler(res http.ResponseWriter, req *http.Request) {
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

	go clients.GetOrdersAccruals(orderIDs, accruals, &wg)

	wg.Wait()

	responseOrders := make([]OrderJSON, 0)

	for _, accrual := range accruals {
		if accrual.Error != nil {
			http.Error(res, queryRowError.Error(), http.StatusInternalServerError)
			return
		}

		if accrual.OrderAccrual != nil {
			responseOrders = append(responseOrders, OrderJSON{
				Number:  accrual.OrderAccrual.OrderID,
				Accrual: accrual.OrderAccrual.AccrualValue,
				Status:  accrual.OrderAccrual.Status,
			})
		}
	}

	marshaledResp, err := json.Marshal(responseOrders)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	res.Header().Set("content-type", "application/json")

	if len(responseOrders) == 0 {
		res.WriteHeader(http.StatusNoContent)
	} else {
		res.WriteHeader(http.StatusOK)
	}

	res.Write(marshaledResp)
}
