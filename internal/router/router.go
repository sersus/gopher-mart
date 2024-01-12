package router

import (
	"github.com/go-chi/chi/v5"

	"github.com/sersus/gopher-mart/internal/auth"
	"github.com/sersus/gopher-mart/internal/databases"
	"github.com/sersus/gopher-mart/internal/handlers"
)

func New(dbc *databases.DatabaseClient) *chi.Mux {
	mux := chi.NewRouter()

	mux.Post("/api/user/register", handlers.RegisterHandler(dbc))
	mux.Post("/api/user/login", handlers.LoginHandler(dbc))

	mux.Post("/api/user/orders", auth.WithAuth(handlers.CreateOrderHandler(dbc)))
	mux.Get("/api/user/orders", auth.WithAuth(handlers.GetOrdersHandler(dbc)))
	mux.Get("/api/user/withdrawals", auth.WithAuth(handlers.GetWithdrawalsHandler(dbc)))

	mux.Get("/api/user/balance", auth.WithAuth(handlers.GetUserBalanceHandler(dbc)))
	mux.Post("/api/user/balance/withdraw", auth.WithAuth(handlers.WithdrawUserBalanceHandler(dbc)))

	return mux
}
