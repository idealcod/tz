package main

import (
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"

	"effective-mobile.test/internal/config"
	"effective-mobile.test/internal/handler"
	"effective-mobile.test/internal/repository"
	"effective-mobile.test/internal/service"

	_ "effective-mobile.test/docs"
	"github.com/swaggo/files"
	"github.com/swaggo/gin-swagger"
)

func main() {
	logger := logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{})
	logger.SetLevel(logrus.DebugLevel) // Включаем debug-логи

	logger.Debug("Loading .env file")
	if err := godotenv.Load(); err != nil {
		logger.Warn("No .env file found")
	}

	logger.Debug("Loading configuration")
	cfg, err := config.Load()
	if err != nil {
		logger.Fatal("Failed to load config: ", err)
	}

	gin.SetMode(gin.ReleaseMode)

	logger.Debug("Connecting to PostgreSQL database")
	db, err := repository.NewPostgresDB(cfg)
	if err != nil {
		logger.Fatal("Failed to connect to database: ", err)
	}
	defer db.Close()

	logger.Debug("Running database migrations")
	if err := repository.RunMigrations(cfg); err != nil {
		logger.Fatal("Failed to run migrations: ", err)
	}

	repo := repository.NewRepository(db, logger)
	svc := service.NewService(repo, logger)
	h := handler.NewHandler(svc, logger)

	router := h.InitRoutes()

	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	logger.Info("Starting server on port ", cfg.Port)
	if err := router.Run(":" + cfg.Port); err != nil {
		logger.Fatal("Failed to start server: ", err)
	}
}
