package main

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"attendance/config"
	"attendance/internal/handler"
	"attendance/internal/logger"
	"attendance/internal/mailer"
	appMiddleware "attendance/internal/middleware"
	"attendance/internal/repository/postgres"
	"attendance/internal/service/impl"
	"attendance/internal/system"

	_ "github.com/lib/pq"

	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/jmoiron/sqlx"
)

func main() {
	// ======== config loader ========
	cfg, err := config.Load()
	if err != nil {
		panic(err)
	}

	log, err := logger.Setup(cfg)
	if err != nil {
		panic(err)
	}
	defer log.Sync()

	cfg.LogConfig(log)

	// ======== db connector ========
	db, err := sqlx.Connect("postgres", cfg.DatabaseURL)
	if err != nil {
		log.Errorf("db connection failed: %v", err)
		os.Exit(1)
	}
	defer db.Close()
	log.Info("database connected")
	// ======== repository ========
	usersRepository := postgres.NewUsersRepository(db)
	sessionsRepository := postgres.NewSessionsRepository(db)
	attendanceRepository := postgres.NewAttendanceRepository(db)
	adminRepository := postgres.NewAdminRepository(db)
	systemRepository := postgres.NewSystemRepository(db)
	txManager := postgres.NewTransactionManager(db)

	// ======== service ========
	usersService := impl.NewUsersService(cfg, usersRepository)
	sessionsService := impl.NewSessionsService(cfg, sessionsRepository)
	attendanceService := impl.NewAttendanceService(cfg, txManager, attendanceRepository, systemRepository)
	adminService := impl.NewAdminService(cfg, txManager, adminRepository)
	excelService := impl.NewExcelService(adminService, cfg)

	// ======== handler ========
	usersHandler := handler.NewUsersHandler(usersService)
	authHandler := handler.NewAuthHandler(cfg, usersService, sessionsService, adminService)
	attendanceHandler := handler.NewAttendanceHandler(attendanceService)
	reportMailer := mailer.NewReportMailer(cfg, adminService, excelService, log)
	reportMailer.Start(context.Background())
	heartbeatMonitor := system.NewHeartbeatMonitor(cfg, adminRepository, log)
	heartbeatMonitor.Start(context.Background())
	adminHandler := handler.NewAdminHandler(adminService, reportMailer, excelService)

	// ======== middleware ========
	authMiddleware := appMiddleware.NewAuthMiddleware(sessionsService, adminService)

	// ======== router ========
	r := chi.NewRouter()
	r.Use(chiMiddleware.Logger)
	r.Use(chiMiddleware.Recoverer)
	r.Use(cors.Handler(
		cors.Options{
			AllowedOrigins:   []string{"https://*", "http://*"},
			AllowedHeaders:   []string{"Origin", "Content-Type", "Authorization"},
			AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS"},
			AllowCredentials: true,
		},
	))

	r.Route("/auth", func(r chi.Router) {
		authHandler.RegisterPublicRoutes(r)
	})

	r.Route("/api", func(r chi.Router) {
		r.Group(func(r chi.Router) {
			r.Use(authMiddleware.RequireAuth)
			usersHandler.RegisterRoutes(r)
			authHandler.RegisterProtectedRoutes(r)
			attendanceHandler.RegisterRoutes(r)
		})

		r.Group(func(r chi.Router) {
			r.Use(authMiddleware.RequireAdmin)
			authHandler.RegisterAdminRoutes(r)
			adminHandler.RegisterRoutes(r)
		})
	})
	registerStaticRoutes(r, "./web")

	log.Info("server starting")
	if err := http.ListenAndServe(":"+cfg.Port, r); err != nil {
		log.Error("server stopped.\n>>>ERROR\n%v", err)
		os.Exit(1)
	}
}

func registerStaticRoutes(r chi.Router, staticDir string) {
	if _, err := os.Stat(staticDir); err != nil {
		return
	}

	fileServer := http.FileServer(http.Dir(staticDir))
	r.Get("/*", func(w http.ResponseWriter, req *http.Request) {
		if strings.HasPrefix(req.URL.Path, "/api/") || strings.HasPrefix(req.URL.Path, "/auth/") {
			http.NotFound(w, req)
			return
		}

		path := filepath.Join(staticDir, filepath.Clean(req.URL.Path))
		if info, err := os.Stat(path); err == nil && !info.IsDir() {
			fileServer.ServeHTTP(w, req)
			return
		}

		http.ServeFile(w, req, filepath.Join(staticDir, "index.html"))
	})
}
