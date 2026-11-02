package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"task-api/internal/auth"
	"task-api/internal/config"
	"task-api/internal/database"
	"task-api/internal/handler"
	"task-api/internal/middleware"
	"task-api/internal/repository"
	"task-api/internal/service"
	"task-api/internal/worker"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	db, err := database.Connect(cfg.DatabaseURL)
	if err != nil {
		slog.Error("failed to connect database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(25)
	db.SetConnMaxLifetime(5 * time.Minute)

	userRepo := repository.NewUserRepository(db)
	taskRepo := repository.NewTaskRepository(db)
	projectRepo := repository.NewProjectRepository(db)

	userService := service.NewUserService(userRepo)
	taskService := service.NewTaskService(taskRepo, projectRepo, userRepo)
	projectService := service.NewProjectService(projectRepo, userRepo)

	authHandler := handler.NewAuthHandler(db, cfg.JWTSecret)
	userHandler := handler.NewUserHandler(userService)
	taskHandler := handler.NewTaskHandler(taskService)
	projectHandler := handler.NewProjectHandler(projectService, taskService)

	authMiddleware := auth.Middleware(cfg.JWTSecret)
	protected := func(h http.HandlerFunc) http.Handler { return authMiddleware(h) }

	workerCtx, stopWorker := context.WithCancel(context.Background())
	defer stopWorker()

	if cfg.ReminderEnabled {
		reminder := worker.NewReminderWorker(taskRepo, cfg.ReminderInterval, slog.Default())
		go reminder.Run(workerCtx)
	}

	authLimiter := middleware.RateLimit(1, 5)

	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", handler.HealthHandler)
	mux.Handle("POST /register", authLimiter(http.HandlerFunc(authHandler.Register)))
	mux.Handle("POST /login", authLimiter(http.HandlerFunc(authHandler.Login)))

	mux.Handle("GET /me", protected(userHandler.GetProfile))
	mux.Handle("PATCH /me", protected(userHandler.UpdateProfile))
	mux.Handle("PATCH /me/password", protected(userHandler.ChangePassword))

	mux.Handle("POST /tasks", protected(taskHandler.CreateTask))
	mux.Handle("GET /tasks", protected(taskHandler.ListTasks))
	mux.Handle("GET /tasks/{id}", protected(taskHandler.GetTaskByID))
	mux.Handle("PUT /tasks/{id}", protected(taskHandler.UpdateTaskByID))
	mux.Handle("DELETE /tasks/{id}", protected(taskHandler.DeleteTaskByID))

	mux.Handle("POST /projects", protected(projectHandler.CreateProject))
	mux.Handle("GET /projects", protected(projectHandler.ListProjects))
	mux.Handle("GET /projects/{id}", protected(projectHandler.GetProjectByID))
	mux.Handle("PUT /projects/{id}", protected(projectHandler.UpdateProjectByID))
	mux.Handle("DELETE /projects/{id}", protected(projectHandler.DeleteProjectByID))
	mux.Handle("GET /projects/{id}/members", protected(projectHandler.ListMembers))
	mux.Handle("POST /projects/{id}/members", protected(projectHandler.AddMember))
	mux.Handle("PATCH /projects/{id}/members/{userID}", protected(projectHandler.UpdateMemberRole))
	mux.Handle("DELETE /projects/{id}/members", protected(projectHandler.RemoveMember))
	mux.Handle("GET /projects/{id}/tasks", protected(projectHandler.ListProjectTasks))
	mux.Handle("POST /projects/{id}/tasks", protected(projectHandler.CreateProjectTask))

	mux.Handle("GET /dashboard", protected(taskHandler.Dashboard))

	server := &http.Server{
		ReadTimeout:       5 * time.Second,
		ReadHeaderTimeout: 3 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
		Addr:              ":" + cfg.Port,
		Handler:           middleware.Chain(mux, middleware.Recover, middleware.RequestLogger, middleware.CORS(cfg.AllowedOrigins)),
	}

	serverErr := make(chan error, 1)
	go func() {
		slog.Info("starting server", "port", cfg.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			serverErr <- err
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-serverErr:
		slog.Error("server error", "error", err)
		os.Exit(1)
	case <-quit:
		slog.Info("shutting down server gracefully")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	stopWorker()

	if err := server.Shutdown(ctx); err != nil {
		slog.Error("graceful shutdown failed", "error", err)
		os.Exit(1)
	}

	slog.Info("server exited cleanly")
}
