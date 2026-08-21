package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"task_manager/internal/handler"
	"task_manager/internal/middleware"
	"task_manager/internal/repository"
	"task_manager/internal/service"

	_ "github.com/go-sql-driver/mysql"
	"github.com/redis/go-redis/v9"

	"task_manager/internal/config"
)

func main() {
	cfg := config.Load()

	db, err := sql.Open("mysql", cfg.MySQLDSN)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatal(err)
	}

	rdb := redis.NewClient(&redis.Options{
		Addr: cfg.RedisAddr,
	})
	defer rdb.Close()

	if err := rdb.Ping(context.Background()).Err(); err != nil {
		log.Fatal(err)
	}

	log.Println("mysql connected")
	log.Println("redis connected")

	jwtManager, err := service.NewJWTManager(cfg.JWTSecret)
	if err != nil {
		log.Fatal(err)
	}

	userRepo := repository.NewUserRepository(db)

	authMiddleware := middleware.NewAuthMiddleware(jwtManager)
	authService := service.NewAuthService(userRepo, jwtManager)
	authHandler := handler.NewAuthHandler(authService)

	teamRepo := repository.NewTeamRepository(db)
	teamService := service.NewTeamService(teamRepo)
	teamHandler := handler.NewTeamHandler(teamService)

	taskCache := repository.NewTaskCache(rdb)

	taskRepo := repository.NewTaskRepository(db)
	taskService := service.NewTaskService(taskRepo, teamRepo, taskCache)
	taskHandler := handler.NewTaskHandler(taskService)

	commentRepo := repository.NewCommentRepository(db)
	commentService := service.NewCommentService(commentRepo, taskRepo, teamRepo)
	commentHandler := handler.NewCommentHandler(commentService)

	statsRepo := repository.NewStatsRepository(db)
	statsService := service.NewStatsService(statsRepo, teamRepo)
	statsHandler := handler.NewStatsHandler(statsService)

	mux := http.NewServeMux()

	mux.HandleFunc("/api/v1/register", authHandler.Register)
	mux.HandleFunc("/api/v1/login", authHandler.Login)
	mux.Handle(
		"POST /api/v1/teams",
		authMiddleware.Authenticate(
			http.HandlerFunc(teamHandler.Create),
		),
	)
	mux.Handle(
		"GET /api/v1/teams",
		authMiddleware.Authenticate(
			http.HandlerFunc(teamHandler.GetAll),
		),
	)
	mux.Handle(
		"POST /api/v1/teams/{id}/invite",
		authMiddleware.Authenticate(
			http.HandlerFunc(teamHandler.Invite),
		),
	)
	mux.Handle(
		"PUT /api/v1/teams/{id}/members/{user_id}/role",
		authMiddleware.Authenticate(
			http.HandlerFunc(teamHandler.UpdateMemberRole),
		),
	)
	mux.Handle(
		"POST /api/v1/tasks",
		authMiddleware.Authenticate(
			http.HandlerFunc(taskHandler.Create),
		),
	)
	mux.Handle(
		"GET /api/v1/tasks",
		authMiddleware.Authenticate(
			http.HandlerFunc(taskHandler.GetAll),
		),
	)
	mux.Handle(
		"PUT /api/v1/tasks/{id}",
		authMiddleware.Authenticate(
			http.HandlerFunc(taskHandler.Update),
		),
	)
	mux.Handle(
		"GET /api/v1/tasks/{id}/history",
		authMiddleware.Authenticate(
			http.HandlerFunc(taskHandler.GetHistory),
		),
	)
	mux.Handle(
		"POST /api/v1/tasks/{id}/comments",
		authMiddleware.Authenticate(
			http.HandlerFunc(commentHandler.Create),
		),
	)
	mux.Handle(
		"GET /api/v1/tasks/{id}/comments",
		authMiddleware.Authenticate(
			http.HandlerFunc(commentHandler.GetAll),
		),
	)
	mux.Handle(
		"GET /api/v1/teams/{team_id}/stats",
		authMiddleware.Authenticate(
			http.HandlerFunc(statsHandler.GetTeamStats),
		),
	)

	server := &http.Server{
		Addr:    ":" + cfg.HTTPPort,
		Handler: mux,
	}

	log.Printf("server started on port %s", cfg.HTTPPort)

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}
