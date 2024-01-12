package main

import (
	"fmt"
	"net/http"

	"github.com/sersus/gopher-mart/internal/config"
	"github.com/sersus/gopher-mart/internal/databases"
	"github.com/sersus/gopher-mart/internal/router"
)

func main() {
	var err error
	if err = config.Configure(); err != nil {
		fmt.Println(err)
		panic(err)
	}

	var dbc *databases.DatabaseClient
	if dbc, err = databases.NewDatabaseClient(config.DatabaseURI); err != nil {
		fmt.Println(err)
	}

	server := &http.Server{
		Addr:    config.RunAddress,
		Handler: router.SetRoutes(dbc),
	}

	err = server.ListenAndServe()
	if err != nil {
		fmt.Println(err)
		panic(err)
	}
}
