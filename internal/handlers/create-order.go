package handlers

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"os"
	"runtime"
	"strings"
	"sync"

	"github.com/sersus/gopher-mart/internal/auth"
	"github.com/sersus/gopher-mart/internal/clients"
	"github.com/sersus/gopher-mart/internal/databases"
	"github.com/theplant/luhn"
)

func location() string {
	_, file, line, _ := runtime.Caller(1)
	p, _ := os.Getwd()
	return fmt.Sprintf("%s:%d", strings.TrimPrefix(file, p), line)
}

func CreateOrderHandler(res http.ResponseWriter, req *http.Request) {
	tokenClaims := req.Context().Value(auth.TokenClaimsContextFieldName).(*auth.TokenClaims)

	var requestOrderID int64

	if err := unmarshalBody(req.Body, &requestOrderID); err != nil {
		http.Error(res, err.Error(), http.StatusUnprocessableEntity)
		return
	}

	if !validateCreateOrderHandlerRequest(int(requestOrderID)) {
		fmt.Println(location(), `"invalid order value"`, "invalid order value")
		http.Error(res, "invalid order value", http.StatusUnprocessableEntity)
		return
	}

	var wg sync.WaitGroup

	var accruals = make([]clients.OrderAccrualResponse, 0)

	wg.Add(1)

	go clients.GetOrdersAccruals([]int64{requestOrderID}, &accruals, &wg)

	wg.Wait()

	fmt.Println(location(), `accruals`, accruals)

	var queryUserID int64

	if accruals[0].OrderAccrual == nil {
		http.Error(res, "invalid request format", http.StatusUnprocessableEntity)
		return
	}

	queryRow := databases.DB.QueryRow(`
		SELECT userId FROM orders where id = $1
	`, requestOrderID)

	queryRowError := queryRow.Scan(&queryUserID)
	if queryRowError != nil && !errors.Is(queryRowError, sql.ErrNoRows) {
		fmt.Println(location(), "sql.ErrNoRows")

		http.Error(res, queryRowError.Error(), http.StatusInternalServerError)
		return
	} else if queryRowError == nil {
		fmt.Println(location(), "no error")

		if queryUserID == tokenClaims.UserID {
			res.WriteHeader(http.StatusOK)
			return
		} else {
			fmt.Println(location(), "order has already been uploaded by another user")

			http.Error(res, "order has already been uploaded by another user", http.StatusConflict)
			return
		}
	} else {
		fmt.Println(location(), "insert")

		_, insertError := databases.DB.Exec(`
			INSERT INTO orders (id, userId) values ($1, $2)
		`, requestOrderID, tokenClaims.UserID)
		if insertError != nil {
			fmt.Println(location(), "insert error")
			http.Error(res, insertError.Error(), http.StatusInternalServerError)
			return
		}
	}

	fmt.Println(location(), "StatusAccepted")
	res.WriteHeader(http.StatusAccepted)
}

func validateCreateOrderHandlerRequest(orderID int) bool {
	if luhn.Valid(orderID) {
		return true
	} else {
		return false
	}
}
