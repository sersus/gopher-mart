package main

import (
	"fmt"
	"net/http"

	"github.com/sersus/gopher-mart/internal/config"
	"github.com/sersus/gopher-mart/internal/databases"
	"github.com/sersus/gopher-mart/internal/router"
)

func main() {
	if err := config.Configure(); err != nil {
		fmt.Println(err)
		panic(err)
	}

	if err := databases.Init(config.DatabaseURI); err != nil {
		fmt.Println(err)
	}

	server := &http.Server{
		Addr:    config.RunAddress,
		Handler: router.SetRoutes(),
	}

	err := server.ListenAndServe()
	if err != nil {
		fmt.Println(err)
		panic(err)
	}
}
