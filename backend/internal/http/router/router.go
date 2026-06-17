package router

import (
	"database/sql"
	"net/http"
	"os"
	"path/filepath"

	"github.com/go-chi/chi/v5"
	mid "github.com/go-chi/chi/v5/middleware"
	"github.com/pressly/goose/v3"

	"ledger_two/internal/config"
	"ledger_two/internal/dashboard"
	"ledger_two/internal/db/repo"
	"ledger_two/internal/http/handler"
	"ledger_two/internal/http/middleware"
	"ledger_two/internal/http/response"
	"ledger_two/internal/ledger"
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
	if cfg != nil {
		authHandler.SetConfig(cfg)
	}

	transactionRepo := transaction.NewRepository(dbConn)
	transactionSvc := transaction.NewService(transactionRepo)
	transactionHandler := transaction.NewHandler(transactionSvc)
	if cfg != nil {
		transactionHandler.UploadDir = cfg.UploadDir
	}

	settlementRepo := settlement.NewRepository(dbConn)
	settlementSvc := settlement.NewService(settlementRepo)
	settlementHandler := settlement.NewHandler(settlementSvc)

	dashboardRepo := dashboard.NewRepository(dbConn)
	dashboardSvc := dashboard.NewService(dashboardRepo, settlementSvc)
	dashboardHandler := dashboard.NewHandler(dashboardSvc)

	ledgerRepo := ledger.NewRepository(dbConn)
	ledgerSvc := ledger.NewService(ledgerRepo)
	ledgerHandler := ledger.NewHandler(ledgerSvc)

	safetySvc := safety.NewService(dbConn, cfg)
	safetyHandler := safety.NewHandler(safetySvc)

	reportsSvc := reports.NewService(dbConn, dashboardRepo, settlementSvc)
	reportsHandler := reports.NewHandler(reportsSvc)

	r.Route("/api", func(r chi.Router) {
		r.Get("/healthz", func(w http.ResponseWriter, req *http.Request) {
			dbStatus := "ok"
			var schemaVersion int64 = 0
			if dbConn != nil {
				if err := dbConn.PingContext(req.Context()); err != nil {
					dbStatus = "error"
				} else {
					version, err := goose.GetDBVersion(dbConn)
					if err == nil {
						schemaVersion = version
					}
				}
			} else {
				dbStatus = "none"
			}
			response.JSON(w, http.StatusOK, map[string]interface{}{
				"status":         "ok",
				"db":             dbStatus,
				"version":        "0.2.0",
				"schema_version": schemaVersion,
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

		// 重点：加入受保护组，验证登录身份
		r.Group(func(r chi.Router) {
			r.Use(middleware.RequireAuth(jwtSecret))
			
			// 1. 全局用户级操作（不需要具体账本上下文）
			r.Route("/ledgers", func(r chi.Router) {
				r.Post("/", ledgerHandler.CreateLedger)
				r.Get("/", ledgerHandler.ListUserLedgers)
			})

			// 2. 账本级业务操作（校验 LedgerContext）
			r.Group(func(r chi.Router) {
				r.Use(middleware.RequireLedgerContext(dbConn))

				// 针对特定账本的成员管理（只允许 owner 操作）
				r.Route("/ledgers/{id}/members", func(r chi.Router) {
					r.Get("/", ledgerHandler.GetLedgerMembers)
					r.With(middleware.RequireLedgerRole("owner")).Post("/", ledgerHandler.AddMember)
					r.With(middleware.RequireLedgerRole("owner")).Put("/{userId}", ledgerHandler.UpdateMemberRole)
					r.With(middleware.RequireLedgerRole("owner")).Delete("/{userId}", ledgerHandler.RemoveMember)
				})

				// 分类与账户获取
				r.Get("/categories", transactionHandler.HandleListCategories)
				r.Get("/accounts", transactionHandler.HandleListAccounts)

				// 交易流水的读写（屏蔽 viewer 写入）
				r.Route("/transactions", func(r chi.Router) {
					r.Get("/", transactionHandler.HandleList)
					r.With(middleware.RequireLedgerRole("owner", "editor")).Post("/", transactionHandler.HandleCreate)
					r.With(middleware.RequireLedgerRole("owner", "editor")).Post("/batch-tag", transactionHandler.HandleBatchTag)
					r.With(middleware.RequireLedgerRole("owner", "editor")).Post("/import/parse", transactionHandler.HandleParseCSV)
					r.With(middleware.RequireLedgerRole("owner", "editor")).Post("/import/analyze", transactionHandler.HandleAnalyzeImport)
					r.With(middleware.RequireLedgerRole("owner", "editor")).Post("/import/commit", transactionHandler.HandleCommitImport)
					r.Get("/{id}", transactionHandler.HandleGetByID)
					r.With(middleware.RequireLedgerRole("owner", "editor")).Patch("/{id}", transactionHandler.HandleUpdate)
					r.With(middleware.RequireLedgerRole("owner", "editor")).Delete("/{id}", transactionHandler.HandleDelete)
				})

				r.Route("/import-rules", func(r chi.Router) {
					r.Get("/", transactionHandler.HandleListImportRules)
					r.With(middleware.RequireLedgerRole("owner", "editor")).Post("/", transactionHandler.HandleCreateImportRule)
					r.With(middleware.RequireLedgerRole("owner", "editor")).Delete("/{id}", transactionHandler.HandleDeleteImportRule)
				})

				r.Route("/transaction-templates", func(r chi.Router) {
					r.Get("/", transactionHandler.HandleListTemplates)
					r.Get("/{id}", transactionHandler.HandleGetTemplate)
					r.With(middleware.RequireLedgerRole("owner", "editor")).Post("/", transactionHandler.HandleCreateTemplate)
					r.With(middleware.RequireLedgerRole("owner", "editor")).Put("/{id}", transactionHandler.HandleUpdateTemplate)
					r.With(middleware.RequireLedgerRole("owner", "editor")).Delete("/{id}", transactionHandler.HandleDeleteTemplate)
				})

				r.Route("/recurring-rules", func(r chi.Router) {
					r.Get("/", transactionHandler.HandleListRecurringRules)
					r.With(middleware.RequireLedgerRole("owner", "editor")).Post("/", transactionHandler.HandleCreateRecurringRule)
					r.With(middleware.RequireLedgerRole("owner", "editor")).Delete("/{id}", transactionHandler.HandleDeleteRecurringRule)
				})

				r.Route("/recurring-reminders", func(r chi.Router) {
					r.Get("/", transactionHandler.HandleListRecurringReminders)
					r.With(middleware.RequireLedgerRole("owner", "editor")).Post("/{id}/confirm", transactionHandler.HandleConfirmReminder)
					r.With(middleware.RequireLedgerRole("owner", "editor")).Post("/{id}/ignore", transactionHandler.HandleIgnoreReminder)
				})

				r.Route("/shared-expenses", func(r chi.Router) {
					r.Get("/{id}", transactionHandler.HandleGetSharedExpenseByID)
					r.With(middleware.RequireLedgerRole("owner", "editor")).Post("/", transactionHandler.HandleCreateSharedExpense)
					r.With(middleware.RequireLedgerRole("owner", "editor")).Patch("/{id}", transactionHandler.HandleUpdateSharedExpense)
				})

				r.Route("/settlements", func(r chi.Router) {
					r.Get("/balance", settlementHandler.HandleGetBalance)
					r.Get("/", settlementHandler.HandleList)
					r.With(middleware.RequireLedgerRole("owner", "editor")).Post("/", settlementHandler.HandleCreate)
				})

				// 备份与数据安全管理（只允许 owner 操作）
				r.Route("/admin", func(r chi.Router) {
					r.Use(middleware.RequireLedgerRole("owner"))
					r.Post("/backup", safetyHandler.HandleManualBackup)
					r.Post("/restore", safetyHandler.HandleRestoreBackup)
					r.Get("/backups", safetyHandler.HandleGetBackups)
					r.Get("/backups/{filename}", safetyHandler.HandleDownloadBackup)
					r.Get("/backups/*", safetyHandler.HandleDownloadBackup)
				})

				// 导出管理（只允许 owner 操作）
				r.Route("/export", func(r chi.Router) {
					r.Use(middleware.RequireLedgerRole("owner"))
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

				r.With(middleware.RequireLedgerRole("owner", "editor")).Post("/attachments", transactionHandler.HandleUploadAttachment)
				r.Get("/dashboard", dashboardHandler.HandleGetDashboard)
			})
		})
	})

	// 托管上传的物理文件附件
	uploadDir := "uploads"
	if cfg != nil && cfg.UploadDir != "" {
		uploadDir = cfg.UploadDir
	}
	r.Handle("/uploads/*", http.StripPrefix("/uploads/", http.FileServer(http.Dir(uploadDir))))

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
