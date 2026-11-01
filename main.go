package main

import (
	"context"
	"log"
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
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	db, err := database.Connect(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	defer db.Close()

	mux := http.NewServeMux()

	taskRepo := repository.NewTaskRepository(db)
	taskService := service.NewTaskService(taskRepo)
	taskHandler := handler.NewTaskHandler(db, taskService)

	authHandler := handler.NewAuthHandler(db, cfg.JWTSecret)
	authMiddleware := auth.Middleware(cfg.JWTSecret)

	mux.HandleFunc("GET /health", handler.HealthHandler)
	mux.Handle("POST /tasks", authMiddleware(http.HandlerFunc(taskHandler.CreateTask)))
	mux.Handle("GET /tasks", authMiddleware(http.HandlerFunc(taskHandler.GetTask)))
	mux.Handle("GET /tasks/{id}", authMiddleware(http.HandlerFunc(taskHandler.GetTaskByID)))
	mux.Handle("PUT /tasks/{id}", authMiddleware(http.HandlerFunc(taskHandler.UpdateTaskByID)))
	mux.Handle("DELETE /tasks/{id}", authMiddleware(http.HandlerFunc(taskHandler.DeleteTaskByID)))
	mux.HandleFunc("POST /register", authHandler.Register)
	mux.HandleFunc("POST /login", authHandler.Login)

	server := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: mux,
	}

	go func() {
		log.Printf("Starting server on :%s", cfg.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server gracefully...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited cleanly")
}
