package handlers

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"

	"github.com/sersus/gopher-mart/internal/auth"
	"github.com/sersus/gopher-mart/internal/databases"
)

func RegisterHandler(dbc *databases.DatabaseClient) http.HandlerFunc {
	return func(res http.ResponseWriter, req *http.Request) {
		/*if true {
			newJwt, jwtError := auth.CreateJwtToken(1)
			if jwtError != nil {
				http.Error(res, jwtError.Error(), http.StatusInternalServerError)
				return
			}

			res.Header().Set(auth.AuthHeader, newJwt)
			res.WriteHeader(http.StatusOK)
			return
		}*/

		var unmarshalledBody credentialsBody

		fmt.Println("register handler")
		if err := unmarshalBody(req.Body, &unmarshalledBody); err != nil {
			http.Error(res, err.Error(), http.StatusBadRequest)
			return
		}

		if err := validateRegisterHandlerBody(unmarshalledBody); err != nil {
			http.Error(res, err.Error(), http.StatusBadRequest)
			return
		}

		_, queryRowError := dbc.GetUserID(unmarshalledBody.Login)

		if queryRowError != nil && !errors.Is(queryRowError, sql.ErrNoRows) {
			http.Error(res, queryRowError.Error(), http.StatusInternalServerError)
			return
		}

		if queryRowError == nil {
			http.Error(res, "login is already taken", http.StatusConflict)
			return
		}

		newID, insertQueryRowError := dbc.InsertUser(unmarshalledBody.Login, unmarshalledBody.Password)

		if insertQueryRowError != nil {
			http.Error(res, insertQueryRowError.Error(), http.StatusInternalServerError)
			return
		}

		newJwt, jwtError := auth.CreateJwtToken(newID)
		if jwtError != nil {
			http.Error(res, jwtError.Error(), http.StatusInternalServerError)
			return
		}

		res.Header().Set(auth.AuthHeader, newJwt)
		res.WriteHeader(http.StatusOK)
	}
}

func validateRegisterHandlerBody(body credentialsBody) error {
	if len(body.Login) == 0 {
		return errors.New("login not specified")
	}

	if len(body.Password) == 0 {
		return errors.New("password not specified")
	}

	return nil
}
