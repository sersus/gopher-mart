package handlers

import (
	"database/sql"
	"errors"
	"net/http"

	"github.com/sersus/gopher-mart/internal/auth"
	"github.com/sersus/gopher-mart/internal/databases"
)

func RegisterHandler(res http.ResponseWriter, req *http.Request) {
	var unmarshalledBody credentialsBody

	if err := unmarshalBody(req.Body, &unmarshalledBody); err != nil {
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}

	if err := validateRegisterHandlerBody(unmarshalledBody); err != nil {
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}

	_, queryRowError := getUserID(unmarshalledBody.Login)

	if queryRowError != nil && !errors.Is(queryRowError, sql.ErrNoRows) {
		http.Error(res, queryRowError.Error(), http.StatusInternalServerError)
		return
	}

	if queryRowError == nil {
		http.Error(res, "login is already taken", http.StatusConflict)
		return
	}

	newID, insertQueryRowError := insertUser(unmarshalledBody.Login, unmarshalledBody.Password)

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

func validateRegisterHandlerBody(body credentialsBody) error {
	if len(body.Login) == 0 {
		return errors.New("login not specified")
	}

	if len(body.Password) == 0 {
		return errors.New("password not specified")
	}

	return nil
}

func getUserID(login string) (int64, error) {
	queryRow := databases.DB.QueryRow(`
		SELECT id FROM users where login = $1
	`, login)

	var userID int64

	err := queryRow.Scan(&userID)
	return userID, err
}

func insertUser(login, password string) (int64, error) {
	insertQueryRow := databases.DB.QueryRow(`
		INSERT INTO users (login, password) VALUES ($1, $2) RETURNING id
	`, login, password)

	var newID int64

	err := insertQueryRow.Scan(&newID)
	return newID, err
}
