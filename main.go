package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"task-api/internal/auth"
	"task-api/internal/config"
	"task-api/internal/database"
	"task-api/internal/handler"
	"task-api/internal/repository"
	"task-api/internal/service"
	"time"
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

	mux := http.NewServeMux()

	taskRepo := repository.NewTaskRepository(db)
	taskService := service.NewTaskService(taskRepo)
	taskHandler := handler.NewTaskHandler(db, taskService)
	projectRepo := repository.NewProjectRepository(db)
	projectService := service.NewProjectService(projectRepo)
	projectHandler := handler.NewProjectHandler(projectService)

	authHandler := handler.NewAuthHandler(db, cfg.JWTSecret)
	authMiddleware := auth.Middleware(cfg.JWTSecret)

	mux.HandleFunc("GET /health", handler.HealthHandler)
	mux.Handle("POST /tasks", authMiddleware(http.HandlerFunc(taskHandler.CreateTask)))
	mux.Handle("GET /tasks", authMiddleware(http.HandlerFunc(taskHandler.GetTask)))
	mux.Handle("GET /tasks/{id}", authMiddleware(http.HandlerFunc(taskHandler.GetTaskByID)))
	mux.Handle("PUT /tasks/{id}", authMiddleware(http.HandlerFunc(taskHandler.UpdateTaskByID)))
	mux.Handle("DELETE /tasks/{id}", authMiddleware(http.HandlerFunc(taskHandler.DeleteTaskByID)))
	mux.Handle("POST /projects", authMiddleware(http.HandlerFunc(projectHandler.CreateProject)))
	mux.Handle("GET /projects", authMiddleware(http.HandlerFunc(projectHandler.GetProjects)))
	mux.Handle("GET /projects/{id}", authMiddleware(http.HandlerFunc(projectHandler.GetProjectByID)))
	mux.Handle("POST /projects/{id}/members", authMiddleware(http.HandlerFunc(projectHandler.AddMember)))
	mux.HandleFunc("POST /register", authHandler.Register)
	mux.HandleFunc("POST /login", authHandler.Login)

	server := &http.Server{
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  15 * time.Second,
		Addr:         ":" + cfg.Port,
		Handler:      mux,
	}

	go func() {
		slog.Info("starting server", "port", cfg.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	slog.Info("Shutting down server gracefully")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		slog.Error("server error", "error", err)
		os.Exit(1)
	}

	slog.Info("Server exited cleanly")
}
