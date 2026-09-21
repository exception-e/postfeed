package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"postfeed/internal/config"
	"postfeed/internal/graph"
	"postfeed/internal/logger"
	"postfeed/internal/services/auth"
	"postfeed/internal/services/comments"
	"postfeed/internal/services/loader"
	"postfeed/internal/services/posts"
	"postfeed/internal/storage/inmemory"
	"postfeed/internal/storage/psql"
	storageTypes "postfeed/internal/storage/types"
	"syscall"
	"time"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/extension"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/go-chi/chi/v5/middleware"
)

const (
	defaultPort     = "8080"
	shutdownTimeout = 5 * time.Second
)

func main() {
	var logHandler = slog.NewJSONHandler(os.Stdout, nil)
	log := logger.New(logHandler)

	cfg, err := config.Load()
	if err != nil {
		log.Error("Failed to config.Load", "error", err)
		os.Exit(1)
	}

	var postsRepo storageTypes.PostRepo
	var commentsRepo storageTypes.CommentRepo
	switch cfg.StorageType {
	case config.StorageTypePSQL:
		db, err := initDB(cfg.Storage)
		if err != nil {
			log.Error("Failed to initDB", "error", err)
			os.Exit(1)
		}

		storage := psql.NewStorage(db, log.WithGroup("psql"))
		if err = storage.RunMigrations(); err != nil {
			log.Error("Failed to storage.RunMigrations", "error", err)
			os.Exit(1)
		}

		postsRepo, commentsRepo = storage, storage
	case config.StorageTypeInMemory:
		postsStorage := inmemory.NewPostStorage()
		commentsStorage := inmemory.NewCommentStorage(postsStorage)

		postsRepo, commentsRepo = postsStorage, commentsStorage
	default:
		log.Error("Unsupported storage type", "type", cfg.StorageType)
		os.Exit(1)
	}

	postsService := posts.NewService(postsRepo)
	commentsService := comments.NewService(postsService, commentsRepo)

	gqpSrv := handler.New(
		graph.NewExecutableSchema(
			graph.Config{
				Resolvers: &graph.Resolver{
					PostsService:    postsService,
					CommentsService: commentsService,
				},
			},
		),
	)

	gqpSrv.AddTransport(transport.Options{})
	gqpSrv.AddTransport(transport.GET{})
	gqpSrv.AddTransport(transport.POST{})
	gqpSrv.Use(extension.Introspection{})

	srvMux := http.NewServeMux()
	srvMux.Handle(
		"/",
		playground.Handler("GraphQL playground", "/query"),
	)
	srvMux.Handle(
		"/query",
		middleware.Logger(
			middleware.RequestID(
				auth.AuthMiddleware(
					loader.Middleware(commentsRepo, gqpSrv),
				),
			),
		),
	)

	srv := &http.Server{
		Addr:              ":" + defaultPort,
		Handler:           srvMux,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	signalCtx, signalCtxCancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer signalCtxCancel()

	go func() {
		log.Info("HTTP server started", "address", srv.Addr)

		err := srv.ListenAndServe()
		if errors.Is(err, http.ErrServerClosed) {
			return
		}
		log.Error("HTTP server error:", "error", err)
	}()

	<-signalCtx.Done()
	log.Info("Shutting HTTP server down...")

	shutdownCtx, shutdownCtxCancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer shutdownCtxCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error("HTTP server shutdown failed", "error", err)
	}
}

func initDB(cfg psql.Config) (*sql.DB, error) {
	db, err := sql.Open("postgres", cfg.DSN)
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.ConnMaxLifetime)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("database ping failed: %w", err)
	}

	return db, nil
}
