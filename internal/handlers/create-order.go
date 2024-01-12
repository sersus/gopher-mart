package handlers

import (
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

func CreateOrderHandler(dbc *databases.DatabaseClient) http.HandlerFunc {
	return func(res http.ResponseWriter, req *http.Request) {
		tokenClaims := req.Context().Value(auth.TokenClaimsContextFieldName).(*auth.TokenClaims)

		var requestOrderID int64

		if err := unmarshalBody(req.Body, &requestOrderID); err != nil {
			http.Error(res, err.Error(), http.StatusUnprocessableEntity)
			return
		}

		if !luhn.Valid(int(requestOrderID)) {
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

		if accruals[0].OrderAccrual == nil {
			http.Error(res, "invalid request format", http.StatusUnprocessableEntity)
			return
		}

		_ = dbc.OrderProcessing(res, requestOrderID, tokenClaims.UserID)

		fmt.Println(location(), "StatusAccepted")
		res.WriteHeader(http.StatusAccepted)
	}
}
