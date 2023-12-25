package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/sersus/gopher-mart/internal/auth"
	"github.com/sersus/gopher-mart/internal/databases"
)

type WithdrawalJSON struct {
	Order string  `json:"order"`
	Sum   float64 `json:"sum"`
}

func GetWithdrawalsHandler(res http.ResponseWriter, req *http.Request) {
	tokenClaims := req.Context().Value(auth.TokenClaimsContextFieldName).(*auth.TokenClaims)

	withdrawals, withdrawalsErrors := databases.GetWithdrawalsByUserID(tokenClaims.UserID)
	if withdrawalsErrors != nil {
		http.Error(res, withdrawalsErrors.Error(), http.StatusInternalServerError)
		return
	}

	response := make([]WithdrawalJSON, 0)

	for _, v := range *withdrawals {
		response = append(response, WithdrawalJSON{Order: v.Order, Sum: v.Sum})
	}

	marshaledResp, err := json.Marshal(response)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	res.Header().Set("content-type", "application/json")

	if len(response) == 0 {
		res.WriteHeader(http.StatusNoContent)
	} else {
		res.WriteHeader(http.StatusOK)
	}

	res.Write(marshaledResp)
}
