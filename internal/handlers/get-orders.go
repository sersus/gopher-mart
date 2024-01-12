package handlers

import (
	"encoding/json"
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

func GetOrdersHandler(dbc *databases.DatabaseClient) http.HandlerFunc {
	return func(res http.ResponseWriter, req *http.Request) {
		tokenClaims := req.Context().Value(auth.TokenClaimsContextFieldName).(*auth.TokenClaims)

		orderIDs, _ := dbc.GetOrders(res, tokenClaims.UserID)

		var wg sync.WaitGroup

		var accruals = make([]clients.OrderAccrualResponse, 0)

		wg.Add(len(*orderIDs))

		go clients.GetOrdersAccruals(*orderIDs, &accruals, &wg)

		wg.Wait()

		responseOrders := make([]OrderJSON, 0)

		for _, accrual := range accruals {
			if accrual.Error != nil {
				http.Error(res, accrual.Error.Error(), http.StatusInternalServerError)
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
}
