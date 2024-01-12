package handlers

import (
	"fmt"
	"net/http"
	"sync"

	"github.com/sersus/gopher-mart/internal/auth"
	"github.com/sersus/gopher-mart/internal/clients"
	"github.com/sersus/gopher-mart/internal/databases"
)

type WithdrawUserBalanceHandlerRequest struct {
	Order string  `json:"order"`
	Sum   float64 `json:"sum"`
}

func WithdrawUserBalanceHandler(dbc *databases.DatabaseClient) http.HandlerFunc {
	return func(res http.ResponseWriter, req *http.Request) {
		tokenClaims := req.Context().Value(auth.TokenClaimsContextFieldName).(*auth.TokenClaims)

		var unmarshalledBody WithdrawUserBalanceHandlerRequest

		if err := unmarshalBody(req.Body, &unmarshalledBody); err != nil {
			http.Error(res, err.Error(), http.StatusUnprocessableEntity)
			return
		}

		orderIDs, _ := dbc.GetOrders(res, tokenClaims.UserID)

		var wg sync.WaitGroup

		var accruals = make([]clients.OrderAccrualResponse, 0)

		wg.Add(len(*orderIDs))

		go clients.GetOrdersAccruals(*orderIDs, &accruals, &wg)

		wg.Wait()

		var current float64 = 0

		for _, accrual := range accruals {
			if accrual.Error != nil {
				fmt.Println(accrual.Error.Error())
				http.Error(res, accrual.Error.Error(), http.StatusInternalServerError)
				return
			}

			if accrual.OrderAccrual != nil && accrual.OrderAccrual.Status == clients.PROCESSED {
				current = current + accrual.OrderAccrual.AccrualValue
			}
		}

		withdrawals, withdrawalsErrors := dbc.GetWithdrawalsByUserID(tokenClaims.UserID)
		if withdrawalsErrors != nil {
			fmt.Println(withdrawalsErrors.Error())
			http.Error(res, withdrawalsErrors.Error(), http.StatusInternalServerError)
			return
		}

		var withdrawalsSum float64

		for _, v := range *withdrawals {
			withdrawalsSum = withdrawalsSum + v.Sum
		}

		current = current - withdrawalsSum

		if unmarshalledBody.Sum > current {
			http.Error(res, withdrawalsErrors.Error(), http.StatusPaymentRequired)
			return
		}

		insertError := dbc.InsertWithdrawal(tokenClaims.UserID, unmarshalledBody.Sum, unmarshalledBody.Order)
		if insertError != nil {
			fmt.Println(insertError.Error())
			http.Error(res, insertError.Error(), http.StatusInternalServerError)
			return
		}

		res.WriteHeader(http.StatusOK)
	}
}
