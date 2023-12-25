package config

import (
	"flag"

	"github.com/caarlos0/env/v9"
)

var RunAddress string
var DatabaseURI string
var AccrualSystemAddress string

var (
	runAddressFlag           = flag.String("a", "localhost:8080", "Service start address and port")
	databaseURIFlag          = flag.String("d", "host=localhost user=yandex password=yandex dbname=go-diploma-1 sslmode=disable", "Database connection address")
	accrualSystemAddressFlag = flag.String("r", "", "Address of the accrual calculation system")
)

type EnvValues struct {
	RunAddress           *string `env:"RUN_ADDRESS"`
	DatabaseURI          *string `env:"DATABASE_URI"`
	AccrualSystemAddress *string `env:"ACCRUAL_SYSTEM_ADDRESS"`
}

func Configure() error {
	flag.Parse()

	envValues := EnvValues{}

	err := env.Parse(&envValues)
	if err != nil {
		return err
	}

	if envValues.RunAddress != nil {
		RunAddress = *envValues.RunAddress
	} else {
		RunAddress = *runAddressFlag
	}

	if envValues.DatabaseURI != nil {
		DatabaseURI = *envValues.DatabaseURI
	} else {
		DatabaseURI = *databaseURIFlag
	}

	if envValues.AccrualSystemAddress != nil {
		AccrualSystemAddress = *envValues.AccrualSystemAddress
	} else {
		AccrualSystemAddress = *accrualSystemAddressFlag
	}

	return nil
}
