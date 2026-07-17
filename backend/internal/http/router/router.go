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
	appErrors "ledger_two/internal/errors"
	"ledger_two/internal/http/handler"
	"ledger_two/internal/http/middleware"
	"ledger_two/internal/http/response"
	"ledger_two/internal/importer"
	"ledger_two/internal/ledger"
	"ledger_two/internal/metadata"
	"ledger_two/internal/reports"
	"ledger_two/internal/safety"
	"ledger_two/internal/service"
	"ledger_two/internal/settlement"
	"ledger_two/internal/transaction"
)

const appVersion = "1.3.0-rc"

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
	uploadDir := "./uploads"
	if cfg != nil && cfg.UploadDir != "" {
		uploadDir = cfg.UploadDir
	}
	transactionHandler := transaction.NewHandler(transactionSvc, uploadDir)

	importRepo := importer.NewRepository(dbConn)
	importXLSXEnabled := true
	importClassificationMode := "off"
	if cfg != nil {
		importXLSXEnabled = cfg.ImportXLSXEnabled
		if cfg.ImportClassificationMode != "" {
			importClassificationMode = cfg.ImportClassificationMode
		}
	}
	importSvc := importer.NewService(
		importRepo,
		importer.WithXLSXEnabled(importXLSXEnabled),
		importer.WithClassificationMode(importClassificationMode),
	)
	importHandler := importer.NewHandler(importSvc)

	settlementRepo := settlement.NewRepository(dbConn)
	settlementSvc := settlement.NewService(settlementRepo)
	settlementHandler := settlement.NewHandler(settlementSvc)

	dashboardRepo := dashboard.NewRepository(dbConn)
	dashboardSvc := dashboard.NewService(dashboardRepo, settlementSvc)
	dashboardHandler := dashboard.NewHandler(dashboardSvc)

	ledgerRepo := ledger.NewRepository(dbConn)
	ledgerSvc := ledger.NewService(ledgerRepo, settlementSvc)
	ledgerHandler := ledger.NewHandler(ledgerSvc)
	rolePolicy := ledger.NewRolePolicy()
	instancePolicy := ledger.NewInstancePolicy(ledgerRepo)

	metadataRepo := metadata.NewRepository(dbConn)
	metadataSvc := metadata.NewService(metadataRepo)
	metadataHandler := metadata.NewHandler(metadataSvc)

	safetySvc := safety.NewService(dbConn, cfg)
	safetyHandler := safety.NewHandler(safetySvc)

	reportsSvc := reports.NewService(dbConn, dashboardRepo, settlementSvc)
	reportsHandler := reports.NewHandler(reportsSvc)

	r.Route("/api", func(r chi.Router) {
		r.Get("/healthz", func(w http.ResponseWriter, req *http.Request) {
			dbStatus := "ok"
			var schemaVersion int64 = 0
			deploymentChannel := "unknown"
			importXLSXEnabled := true
			importClassificationMode := "off"
			if cfg != nil {
				importXLSXEnabled = cfg.ImportXLSXEnabled
				if cfg.ImportClassificationMode != "" {
					importClassificationMode = cfg.ImportClassificationMode
				}
				if cfg.DeploymentChannel != "" {
					deploymentChannel = cfg.DeploymentChannel
				}
			}
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
				"status":                     "ok",
				"db":                         dbStatus,
				"version":                    appVersion,
				"schema_version":             schemaVersion,
				"deployment_channel":         deploymentChannel,
				"import_xlsx_enabled":        importXLSXEnabled,
				"import_classification_mode": importClassificationMode,
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

			r.Route("/ledgers", func(r chi.Router) {
				r.Post("/", ledgerHandler.CreateLedger)
				r.Get("/", ledgerHandler.ListUserLedgers)
				r.Route("/{id}", func(r chi.Router) {
					r.Use(ledger.WithRequiredLedgerContext(ledgerSvc, "id"))
					r.Use(ledger.RequireLedgerContext)
					r.With(ledger.RequireOperation(rolePolicy, ledger.OperationViewLedger)).Get("/", ledgerHandler.GetLedger)
					r.With(ledger.RequireWritableLedger, ledger.RequireOperation(rolePolicy, ledger.OperationRenameLedger)).Patch("/", ledgerHandler.RenameLedger)
					r.With(ledger.RequireWritableLedger, ledger.RequireOperation(rolePolicy, ledger.OperationArchiveLedger)).Get("/archive-preflight", ledgerHandler.GetArchivePreflight)
					r.With(ledger.RequireWritableLedger, ledger.RequireOperation(rolePolicy, ledger.OperationArchiveLedger)).Post("/archive", ledgerHandler.ArchiveLedger)
					r.With(ledger.RequireOperation(rolePolicy, ledger.OperationRestoreLedger)).Post("/restore", ledgerHandler.RestoreLedger)
					r.With(ledger.RequireOperation(rolePolicy, ledger.OperationViewMembers)).Get("/members", ledgerHandler.GetLedgerMembers)
					r.With(ledger.RequireWritableLedger, ledger.RequireOperation(rolePolicy, ledger.OperationManageMembers)).Post("/members", ledgerHandler.AddMember)
					r.With(ledger.RequireWritableLedger, ledger.RequireOperation(rolePolicy, ledger.OperationManageMembers)).Patch("/members/{userId}", ledgerHandler.UpdateMemberRole)
					r.With(ledger.RequireWritableLedger, ledger.RequireOperation(rolePolicy, ledger.OperationManageMembers)).Put("/members/{userId}", ledgerHandler.UpdateMemberRole)
					r.With(ledger.RequireWritableLedger, ledger.RequireOperation(rolePolicy, ledger.OperationManageMembers)).Delete("/members/{userId}", ledgerHandler.RemoveMember)
					r.With(ledger.RequireWritableLedger, ledger.RequireOperation(rolePolicy, ledger.OperationTransferLedgerOwner)).Post("/members/{userId}/transfer-owner", ledgerHandler.TransferOwner)
					r.With(ledger.RequireWritableLedger, ledger.RequireOperation(rolePolicy, ledger.OperationLeaveLedger)).Post("/leave", ledgerHandler.LeaveLedger)
				})
			})

			// 实例运维是全局能力，不读取或伪造账本上下文。
			r.Route("/admin", func(r chi.Router) {
				r.Use(ledger.RequireInstanceAdmin(instancePolicy))
				r.Get("/diagnostics", safetyHandler.HandleDiagnostics)
				r.Post("/backup", safetyHandler.HandleManualBackup)
				r.Post("/restore", safetyHandler.HandleRestoreBackup)
				r.Get("/backups", safetyHandler.HandleGetBackups)
				r.Get("/backups/{filename}", safetyHandler.HandleDownloadBackup)
				r.Get("/backups/*", safetyHandler.HandleDownloadBackup)
			})

			r.Group(func(r chi.Router) {
				r.Use(ledger.WithRequiredLedgerContext(ledgerSvc, ""))
				r.Use(ledger.RequireLedgerContext)

				r.Get("/categories", transactionHandler.HandleListCategories)
				r.Get("/accounts", transactionHandler.HandleListAccounts)
				r.Get("/transaction-defaults", transactionHandler.HandleGetTransactionDefault)
				r.Route("/metadata/default-profile", func(r chi.Router) {
					r.Get("/", metadataHandler.GetDefaultProfile)
					r.Post("/preview", metadataHandler.PreviewDefaultProfile)
					r.With(ledger.RequireWritableLedger, ledger.RequireOperation(rolePolicy, ledger.OperationManageMetadata)).Post("/apply", metadataHandler.ApplyDefaultProfile)
				})
				r.Route("/metadata/{kind}", func(r chi.Router) {
					r.Get("/", metadataHandler.List)
					r.With(ledger.RequireWritableLedger, ledger.RequireOperation(rolePolicy, ledger.OperationManageMetadata)).Post("/", metadataHandler.Create)
					r.With(ledger.RequireWritableLedger, ledger.RequireOperation(rolePolicy, ledger.OperationManageMetadata)).Post("/reorder", metadataHandler.Reorder)
					r.With(ledger.RequireWritableLedger, ledger.RequireOperation(rolePolicy, ledger.OperationManageMetadata)).Patch("/{id}", metadataHandler.Update)
					r.With(ledger.RequireWritableLedger, ledger.RequireOperation(rolePolicy, ledger.OperationManageMetadata)).Post("/{id}/archive", metadataHandler.Archive)
					r.With(ledger.RequireWritableLedger, ledger.RequireOperation(rolePolicy, ledger.OperationManageMetadata)).Post("/{id}/restore", metadataHandler.Restore)
				})
				r.Route("/transactions", func(r chi.Router) {
					r.Get("/", transactionHandler.HandleList)
					r.With(ledger.RequireWritableLedger, ledger.RequireOperation(rolePolicy, ledger.OperationCreateTransaction)).Post("/", transactionHandler.HandleCreate)
					r.With(ledger.RequireWritableLedger, ledger.RequireOperation(rolePolicy, ledger.OperationCreateTransaction)).Post("/batch-tag", transactionHandler.HandleBatchTag)
					r.With(ledger.RequireWritableLedger, ledger.RequireOperation(rolePolicy, ledger.OperationManageImports)).Post("/import/parse", transactionHandler.HandleParseCSV)
					r.With(ledger.RequireWritableLedger, ledger.RequireOperation(rolePolicy, ledger.OperationManageImports)).Post("/import/analyze", transactionHandler.HandleAnalyzeImport)
					r.With(ledger.RequireWritableLedger, ledger.RequireOperation(rolePolicy, ledger.OperationManageImports)).Post("/import/commit", transactionHandler.HandleCommitImport)
					r.Get("/{id}", transactionHandler.HandleGetByID)
					r.With(ledger.RequireWritableLedger, ledger.RequireOperation(rolePolicy, ledger.OperationEditOwnTransaction)).Patch("/{id}", transactionHandler.HandleUpdate)
					r.With(ledger.RequireWritableLedger, ledger.RequireOperation(rolePolicy, ledger.OperationDeleteOwnTransaction)).Delete("/{id}", transactionHandler.HandleDelete)
				})

				r.Route("/imports", func(r chi.Router) {
					r.With(ledger.RequireWritableLedger, ledger.RequireOperation(rolePolicy, ledger.OperationManageImports)).Post("/preview", importHandler.HandlePreview)
					r.With(ledger.RequireOperation(rolePolicy, ledger.OperationManageImports)).Get("/{batchID}", importHandler.HandleGetBatch)
					r.With(ledger.RequireWritableLedger, ledger.RequireOperation(rolePolicy, ledger.OperationManageImports)).Patch("/{batchID}/rows/{rowID}", importHandler.HandleUpdateRow)
					r.With(ledger.RequireWritableLedger, ledger.RequireOperation(rolePolicy, ledger.OperationManageImports)).Post("/{batchID}/reclassify", importHandler.HandleReclassify)
					r.With(ledger.RequireWritableLedger, ledger.RequireOperation(rolePolicy, ledger.OperationManageImports)).Post("/{batchID}/commit", importHandler.HandleCommit)
					r.With(ledger.RequireWritableLedger, ledger.RequireOperation(rolePolicy, ledger.OperationDiscardImportBatch)).Post("/{batchID}/discard", importHandler.HandleDiscardBatch)
				})

				r.Route("/import-rules", func(r chi.Router) {
					r.With(ledger.RequireWritableLedger, ledger.RequireOperation(rolePolicy, ledger.OperationManageImports)).Post("/", importHandler.HandleCreateRule)
					r.With(ledger.RequireOperation(rolePolicy, ledger.OperationManageImports)).Get("/", importHandler.HandleListRules)
					r.With(ledger.RequireWritableLedger, ledger.RequireOperation(rolePolicy, ledger.OperationManageImports)).Patch("/{ruleID}", importHandler.HandleUpdateRule)
					r.With(ledger.RequireWritableLedger, ledger.RequireOperation(rolePolicy, ledger.OperationManageImports)).Post("/{ruleID}/archive", importHandler.HandleArchiveRule)
					r.With(ledger.RequireWritableLedger, ledger.RequireOperation(rolePolicy, ledger.OperationManageImports)).Post("/{ruleID}/restore", importHandler.HandleRestoreRule)
					r.With(ledger.RequireWritableLedger, ledger.RequireOperation(rolePolicy, ledger.OperationManageImports)).Delete("/{ruleID}", importHandler.HandleArchiveRule)
				})

				r.Route("/transaction-templates", func(r chi.Router) {
					r.With(ledger.RequireWritableLedger, ledger.RequireOperation(rolePolicy, ledger.OperationCreateTransaction)).Post("/", transactionHandler.HandleCreateTemplate)
					r.Get("/", transactionHandler.HandleListTemplates)
					r.Get("/{id}", transactionHandler.HandleGetTemplate)
					r.With(ledger.RequireWritableLedger, ledger.RequireOperation(rolePolicy, ledger.OperationCreateTransaction)).Put("/{id}", transactionHandler.HandleUpdateTemplate)
					r.With(ledger.RequireWritableLedger, ledger.RequireOperation(rolePolicy, ledger.OperationCreateTransaction)).Post("/{id}/archive", transactionHandler.HandleArchiveTemplate)
					r.With(ledger.RequireWritableLedger, ledger.RequireOperation(rolePolicy, ledger.OperationCreateTransaction)).Post("/{id}/restore", transactionHandler.HandleRestoreTemplate)
					r.With(ledger.RequireWritableLedger, ledger.RequireOperation(rolePolicy, ledger.OperationCreateTransaction)).Delete("/{id}", transactionHandler.HandleDeleteTemplate)
				})

				r.Route("/recurring-rules", func(r chi.Router) {
					r.With(ledger.RequireWritableLedger, ledger.RequireOperation(rolePolicy, ledger.OperationCreateTransaction)).Post("/", transactionHandler.HandleCreateRecurringRule)
					r.Get("/", transactionHandler.HandleListRecurringRules)
					r.With(ledger.RequireWritableLedger, ledger.RequireOperation(rolePolicy, ledger.OperationCreateTransaction)).Delete("/{id}", transactionHandler.HandleDeleteRecurringRule)
				})

				r.Route("/recurring-reminders", func(r chi.Router) {
					r.Get("/", transactionHandler.HandleListRecurringReminders)
					r.With(ledger.RequireWritableLedger, ledger.RequireOperation(rolePolicy, ledger.OperationCreateTransaction)).Post("/{id}/confirm", transactionHandler.HandleConfirmReminder)
					r.With(ledger.RequireWritableLedger, ledger.RequireOperation(rolePolicy, ledger.OperationCreateTransaction)).Post("/{id}/skip", transactionHandler.HandleIgnoreReminder)
					r.With(ledger.RequireWritableLedger, ledger.RequireOperation(rolePolicy, ledger.OperationCreateTransaction)).Post("/{id}/ignore", transactionHandler.HandleIgnoreReminder)
				})

				r.Route("/shared-expenses", func(r chi.Router) {
					r.With(ledger.RequireWritableLedger, ledger.RequireOperation(rolePolicy, ledger.OperationCreateSharedExpense)).Post("/", transactionHandler.HandleCreateSharedExpense)
					r.Get("/{id}", transactionHandler.HandleGetSharedExpenseByID)
					r.With(ledger.RequireWritableLedger, ledger.RequireOperation(rolePolicy, ledger.OperationEditOwnTransaction)).Patch("/{id}", transactionHandler.HandleUpdateSharedExpense)
				})

				r.Route("/settlements", func(r chi.Router) {
					r.Get("/balance", settlementHandler.HandleGetBalance)
					r.Get("/", settlementHandler.HandleList)
					r.With(ledger.RequireWritableLedger, ledger.RequireOperation(rolePolicy, ledger.OperationCreateSettlement)).Post("/", settlementHandler.HandleCreate)
				})

				// 导出管理
				r.Route("/export", func(r chi.Router) {
					r.Use(ledger.RequireOperation(rolePolicy, ledger.OperationExportData))
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

				r.With(ledger.RequireWritableLedger, ledger.RequireOperation(rolePolicy, ledger.OperationCreateTransaction)).Post("/attachments", transactionHandler.HandleUploadAttachment)
				r.Get("/attachments/{filename}", transactionHandler.HandleGetAttachment)
				r.Get("/dashboard", dashboardHandler.HandleGetDashboard)
			})
		})
	})

	r.HandleFunc("/uploads/*", func(w http.ResponseWriter, r *http.Request) {
		response.WriteError(w, appErrors.NewAppError(http.StatusNotFound, appErrors.ErrCodeNotFound, "附件必须通过受保护接口访问"))
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
