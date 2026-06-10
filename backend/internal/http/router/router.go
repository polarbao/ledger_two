package router

import (
	"database/sql"
	"net/http"

	"github.com/go-chi/chi/v5"
	mid "github.com/go-chi/chi/v5/middleware"

	"ledger_two/internal/config"
	"ledger_two/internal/dashboard"
	"ledger_two/internal/db/repo"
	"ledger_two/internal/http/handler"
	"ledger_two/internal/http/middleware"
	"ledger_two/internal/http/response"
	"ledger_two/internal/service"
	"ledger_two/internal/settlement"
	"ledger_two/internal/transaction"
)

// New 接收数据库与环境配置进行依赖链式组装
func New(dbConn *sql.DB, cfg *config.Config) http.Handler {
	r := chi.NewRouter()

	r.Use(mid.Logger)
	r.Use(mid.Recoverer)

	var jwtSecret string
	if cfg != nil {
		jwtSecret = cfg.JWTSecret
	}

	// Dependency Injection
	initRepo := repo.NewInitRepo(dbConn)
	initSvc := service.NewInitService(initRepo)
	initHandler := handler.NewInitHandler(initSvc)

	authRepo := repo.NewAuthRepo(dbConn)
	authSvc := service.NewAuthService(authRepo, jwtSecret)
	authHandler := handler.NewAuthHandler(authSvc)

	transactionRepo := transaction.NewRepository(dbConn)
	transactionSvc := transaction.NewService(transactionRepo)
	transactionHandler := transaction.NewHandler(transactionSvc)

	settlementRepo := settlement.NewRepository(dbConn)
	settlementSvc := settlement.NewService(settlementRepo)
	settlementHandler := settlement.NewHandler(settlementSvc)

	dashboardRepo := dashboard.NewRepository(dbConn)
	dashboardSvc := dashboard.NewService(dashboardRepo, settlementSvc)
	dashboardHandler := dashboard.NewHandler(dashboardSvc)

	r.Route("/api", func(r chi.Router) {
		r.Get("/healthz", func(w http.ResponseWriter, req *http.Request) {
			response.JSON(w, http.StatusOK, map[string]string{
				"status": "ok",
			})
		})

		r.Route("/init", func(r chi.Router) {
			r.Get("/status", initHandler.HandleStatus)
			r.Post("/setup", initHandler.HandleSetup)
		})

		r.Route("/auth", func(r chi.Router) {
			r.Post("/login", authHandler.HandleLogin)
			r.Post("/logout", authHandler.HandleLogout)
			r.With(middleware.RequireAuth(jwtSecret)).Get("/me", authHandler.HandleMe)
		})

		// 重点：加入受保护组，为了未来事务及设置等模块保留
		r.Group(func(r chi.Router) {
			r.Use(middleware.RequireAuth(jwtSecret))
			r.Get("/categories", transactionHandler.HandleListCategories)
			r.Route("/transactions", func(r chi.Router) {
				r.Get("/", transactionHandler.HandleList)
				r.Post("/", transactionHandler.HandleCreate)
				r.Get("/{id}", transactionHandler.HandleGetByID)
				r.Patch("/{id}", transactionHandler.HandleUpdate)
				r.Delete("/{id}", transactionHandler.HandleDelete)
			})

			r.Route("/shared-expenses", func(r chi.Router) {
				r.Post("/", transactionHandler.HandleCreateSharedExpense)
				r.Get("/{id}", transactionHandler.HandleGetSharedExpenseByID)
				r.Patch("/{id}", transactionHandler.HandleUpdateSharedExpense)
			})

			r.Route("/settlements", func(r chi.Router) {
				r.Get("/balance", settlementHandler.HandleGetBalance)
				r.Get("/", settlementHandler.HandleList)
				r.Post("/", settlementHandler.HandleCreate)
			})

			r.Get("/dashboard", dashboardHandler.HandleGetDashboard)
		})
	})

	return r
}
