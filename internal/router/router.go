package router

import (
	"github.com/go-chi/chi/v5"

	"github.com/sersus/gopher-mart/internal/auth"
	"github.com/sersus/gopher-mart/internal/handlers"
)

func SetRoutes() *chi.Mux {
	mux := chi.NewRouter()

	mux.Post("/api/user/register", handlers.RegisterHandler)
	mux.Post("/api/user/login", handlers.LoginHandler)

	mux.Post("/api/user/orders", auth.WithAuth(handlers.CreateOrderHandler))
	mux.Get("/api/user/orders", auth.WithAuth(handlers.GetOrdersHandler))
	mux.Get("/api/user/withdrawals", auth.WithAuth(handlers.GetWithdrawalsHandler))

	mux.Get("/api/user/balance", auth.WithAuth(handlers.GetUserBalanceHandler))
	mux.Post("/api/user/balance/withdraw", auth.WithAuth(handlers.WithdrawUserBalanceHandler))

	return mux
}
