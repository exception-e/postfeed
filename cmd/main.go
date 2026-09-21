package main

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"net/http"
	"os"
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
	"time"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/extension"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/go-chi/chi/v5/middleware"
)

const defaultPort = "8080"

func main() {
	var logHandler = slog.NewJSONHandler(os.Stdout, nil)
	log := logger.New(logHandler)

	cfg, err := config.Load()
	if err != nil {
		log.Error("Failed to config.Load: %v", err)
		os.Exit(1)
	}

	var postsRepo storageTypes.PostRepo
	var commentsRepo storageTypes.CommentRepo
	switch cfg.StorageType {
	case config.StorageTypePSQL:
		db, err := initDB(cfg.Storage)
		if err != nil {
			log.Error("Failed to initDB: %v", err)
			os.Exit(1)
		}

		storage := psql.NewStorage(db, log.WithGroup("psql"))
		if err = storage.RunMigrations(); err != nil {
			log.Error("Failed to storage.RunMigrations: %v", err)
			os.Exit(1)
		}

		postsRepo, commentsRepo = storage, storage
	case config.StorageTypeInMemory:
		postsStorage := inmemory.NewPostStorage()
		commentsStorage := inmemory.NewCommentStorage(postsStorage)

		postsRepo, commentsRepo = postsStorage, commentsStorage
	default:
		log.Error("Unsupported storage type: %s", cfg.StorageType)
		os.Exit(1)
	}

	postsService := posts.NewService(postsRepo)
	commentsService := comments.NewService(postsService, commentsRepo)

	srv := handler.New(
		graph.NewExecutableSchema(
			graph.Config{
				Resolvers: &graph.Resolver{
					PostsService:    postsService,
					CommentsService: commentsService,
				},
			},
		),
	)

	srv.AddTransport(transport.Options{})
	srv.AddTransport(transport.GET{})
	srv.AddTransport(transport.POST{})

	//srv.SetQueryCache(lru.New[*ast.QueryDocument](1000))

	srv.Use(extension.Introspection{})
	//srv.Use(extension.AutomaticPersistedQuery{
	//	Cache: lru.New[string](100),
	//})

	http.Handle(
		"/",
		playground.Handler("GraphQL playground", "/query"),
	)
	http.Handle(
		"/query",
		middleware.Logger(
			middleware.RequestID(
				auth.AuthMiddleware(
					loader.Middleware(commentsRepo, srv),
				),
			),
		),
	)

	srvErr := http.ListenAndServe(":"+defaultPort, nil)
	if srvErr != nil {
		log.Error("Running server:", srvErr.Error())
		os.Exit(1)
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
