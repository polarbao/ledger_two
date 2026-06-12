package router

import (
	"database/sql"
	"net/http"
	"os"
	"path/filepath"

	"github.com/go-chi/chi/v5"
	mid "github.com/go-chi/chi/v5/middleware"

	"ledger_two/internal/config"
	"ledger_two/internal/dashboard"
	"ledger_two/internal/db/repo"
	"ledger_two/internal/http/handler"
	"ledger_two/internal/http/middleware"
	"ledger_two/internal/http/response"
	"ledger_two/internal/reports"
	"ledger_two/internal/safety"
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

	safetySvc := safety.NewService(dbConn, cfg)
	safetyHandler := safety.NewHandler(safetySvc)

	reportsSvc := reports.NewService(dbConn, dashboardRepo, settlementSvc)
	reportsHandler := reports.NewHandler(reportsSvc)

	r.Route("/api", func(r chi.Router) {
		r.Get("/healthz", func(w http.ResponseWriter, req *http.Request) {
			dbStatus := "ok"
			if dbConn != nil {
				if err := dbConn.PingContext(req.Context()); err != nil {
					dbStatus = "error"
				}
			} else {
				dbStatus = "none"
			}
			response.JSON(w, http.StatusOK, map[string]string{
				"status":  "ok",
				"db":      dbStatus,
				"version": "0.2.0",
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
			r.Get("/accounts", transactionHandler.HandleListAccounts)
			r.Route("/transactions", func(r chi.Router) {
				r.Get("/", transactionHandler.HandleList)
				r.Post("/", transactionHandler.HandleCreate)
				r.Post("/batch-tag", transactionHandler.HandleBatchTag)
				r.Post("/import/parse", transactionHandler.HandleParseCSV)
				r.Post("/import/analyze", transactionHandler.HandleAnalyzeImport)
				r.Post("/import/commit", transactionHandler.HandleCommitImport)
				r.Get("/{id}", transactionHandler.HandleGetByID)
				r.Patch("/{id}", transactionHandler.HandleUpdate)
				r.Delete("/{id}", transactionHandler.HandleDelete)
			})

			r.Route("/import-rules", func(r chi.Router) {
				r.Post("/", transactionHandler.HandleCreateImportRule)
				r.Get("/", transactionHandler.HandleListImportRules)
				r.Delete("/{id}", transactionHandler.HandleDeleteImportRule)
			})

			r.Route("/transaction-templates", func(r chi.Router) {
				r.Post("/", transactionHandler.HandleCreateTemplate)
				r.Get("/", transactionHandler.HandleListTemplates)
				r.Get("/{id}", transactionHandler.HandleGetTemplate)
				r.Put("/{id}", transactionHandler.HandleUpdateTemplate)
				r.Delete("/{id}", transactionHandler.HandleDeleteTemplate)
			})

			r.Route("/recurring-rules", func(r chi.Router) {
				r.Post("/", transactionHandler.HandleCreateRecurringRule)
				r.Get("/", transactionHandler.HandleListRecurringRules)
				r.Delete("/{id}", transactionHandler.HandleDeleteRecurringRule)
			})

			r.Route("/recurring-reminders", func(r chi.Router) {
				r.Get("/", transactionHandler.HandleListRecurringReminders)
				r.Post("/{id}/confirm", transactionHandler.HandleConfirmReminder)
				r.Post("/{id}/ignore", transactionHandler.HandleIgnoreReminder)
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

			// 备份与数据安全管理
			r.Route("/admin", func(r chi.Router) {
				r.Post("/backup", safetyHandler.HandleManualBackup)
				r.Get("/backups", safetyHandler.HandleGetBackups)
				r.Get("/backups/{filename}", safetyHandler.HandleDownloadBackup)
				r.Get("/backups/*", safetyHandler.HandleDownloadBackup)
			})

			// 导出管理
			r.Route("/export", func(r chi.Router) {
				r.Get("/transactions.csv", safetyHandler.HandleExportCSV)
				r.Get("/full.json", safetyHandler.HandleExportJSON)
			})

			// 统计报表管理
			r.Route("/reports", func(r chi.Router) {
				r.Get("/monthly-summary", reportsHandler.HandleGetMonthlySummary)
				r.Get("/category-summary", reportsHandler.HandleGetCategorySummary)
				r.Get("/tag-summary", reportsHandler.HandleGetTagSummary)
				r.Get("/member-summary", reportsHandler.HandleGetMemberSummary)
			})

			r.Get("/dashboard", dashboardHandler.HandleGetDashboard)
		})
	})

	// 生产环境下静态托管前端 SPA 页面，任何非 API 请求若找不到物理文件则 Fallback 重定向回 index.html
	r.NotFound(spaHandler("./web/dist"))

	return r
}

// spaHandler 返回一个用于托管前端单页应用（SPA）静态文件的 HandlerFunc
func spaHandler(staticDir string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 拼接物理磁盘文件路径
		path := filepath.Join(staticDir, r.URL.Path)

		// 检查路径是否存在
		fi, err := os.Stat(path)
		if os.IsNotExist(err) || fi.IsDir() {
			// 如果文件不存在或者是目录，自动 Fallback 托管返回前端 index.html 入口
			http.ServeFile(w, r, filepath.Join(staticDir, "index.html"))
			return
		}

		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// 如果文件在磁盘中存在，提供正常的静态资源服务
		http.FileServer(http.Dir(staticDir)).ServeHTTP(w, r)
	}
}
