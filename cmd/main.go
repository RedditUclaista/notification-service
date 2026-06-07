package main

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/RedditUclaista/notification-service/internal/cache"
	"github.com/RedditUclaista/notification-service/internal/database"
	deliveryhttp "github.com/RedditUclaista/notification-service/internal/delivery/http"
	"github.com/RedditUclaista/notification-service/internal/lib"
	"github.com/RedditUclaista/notification-service/internal/messaging"
	"github.com/RedditUclaista/notification-service/internal/usecases"
	dotenv "github.com/joho/godotenv"
	"github.com/labstack/echo/v5"
)

func main() {
	// Cargamos las variables de entorno
	if err := dotenv.Load(); err != nil {
		fmt.Println("Environment: No .env file found, using system environment variables.")
	}

	// 1. Configuración y Conexión de PostgreSQL
	dbPort, err := strconv.Atoi(os.Getenv("DB_PORT"))
	if err != nil {
		dbPort = 5433
	}

	maxOpenConn, err := strconv.Atoi(os.Getenv("DB_MAX_OPEN_CONN"))
	if err != nil {
		maxOpenConn = 25
	}

	maxIdleConn, err := strconv.Atoi(os.Getenv("DB_MAX_IDLE_CONN"))
	if err != nil {
		maxIdleConn = 23
	}

	connMaxLifetime, err := strconv.Atoi(os.Getenv("DB_CONN_MAX_LIFETIME"))
	if err != nil {
		connMaxLifetime = 5
	}

	connMaxIdleTime, err := strconv.Atoi(os.Getenv("DB_CONN_MAX_IDLE_TIME"))
	if err != nil {
		connMaxIdleTime = 5
	}

	db, err := database.NewConnection(database.DBConfig{
		Host:            os.Getenv("DB_HOST"),
		Port:            dbPort,
		User:            os.Getenv("DB_USER"),
		Password:        os.Getenv("DB_PASS"),
		DBName:          os.Getenv("DB_NAME"),
		SSLMode:         os.Getenv("DB_SSLMODE"),
		MaxOpenConn:     maxOpenConn,
		MaxIdleConn:     maxIdleConn,
		ConnMaxLifetime: connMaxLifetime,
		ConnMaxIdleTime: connMaxIdleTime,
	})
	if err != nil {
		fmt.Printf("Database: Connection failed: %v\n", err)
		panic(err)
	}
	defer db.Close()
	fmt.Println("Database: Connected to PostgreSQL successfully.")

	// Aplicamos la migración del esquema de manera automática
	schemaSql, err := os.ReadFile("sql/init.sql")
	if err == nil {
		_, err = db.Exec(string(schemaSql))
		if err != nil {
			fmt.Printf("Database: Warning, failed to apply migrations from init.sql: %v\n", err)
		} else {
			fmt.Println("Database: Migrations applied successfully (schema is up to date).")
		}
	} else {
		fmt.Printf("Database: Warning, could not read sql/init.sql: %v\n", err)
	}

	// 2. Configuración y Conexión de Valkey (Caché)
	cachePort, err := strconv.Atoi(os.Getenv("CACHE_PORT"))
	if err != nil {
		cachePort = 6380
	}

	cacheDB, err := strconv.Atoi(os.Getenv("CACHE_DB"))
	if err != nil {
		cacheDB = 0
	}

	cacheTimeout, err := strconv.Atoi(os.Getenv("CACHE_TIMEOUT"))
	if err != nil {
		cacheTimeout = 5
	}

	notificationCache, err := cache.NewNotificationCache(
		os.Getenv("CACHE_HOST"),
		cachePort,
		cacheDB,
		time.Duration(cacheTimeout)*time.Second,
	)
	if err != nil {
		fmt.Printf("Cache: Connection failed: %v\n", err)
		panic(err)
	}
	fmt.Println("Cache: Connected to Valkey successfully.")

	// 3. Inicialización del Cliente Push (Firebase Cloud Messaging)
	credentialsPath := os.Getenv("FIREBASE_CREDENTIALS_PATH")
	if credentialsPath == "" {
		credentialsPath = "service-account.json"
	}
	fcmMockStr := os.Getenv("FCM_MOCK")
	fcmMock := fcmMockStr == "true" || fcmMockStr == ""

	pushClient, err := lib.NewPushServiceClient(credentialsPath, fcmMock)
	if err != nil {
		fmt.Printf("FCM: Failed to initialize push service: %v\n", err)
		panic(err)
	}

	// 4. Inicialización de Casos de Uso y Repositorios
	repo := &database.DBRepo{Cursor: db}
	useCase := usecases.NewNotificationUseCase(repo, notificationCache, pushClient)

	// 5. Conexión e Inicio del Consumidor de LavinMQ
	qHost := os.Getenv("QUEUE_HOST")
	if qHost == "" {
		qHost = "0.0.0.0"
	}
	qPort := os.Getenv("QUEUE_PORT")
	if qPort == "" {
		qPort = "5672"
	}
	qUser := os.Getenv("QUEUE_USER")
	if qUser == "" {
		qUser = "user"
	}
	qPass := os.Getenv("QUEUE_PASSWORD")
	if qPass == "" {
		qPass = "password"
	}
	qVhost := os.Getenv("QUEUE_VHOST")
	if qVhost == "" {
		qVhost = "/"
	}

	amqpUrl := fmt.Sprintf("amqp://%s:%s@%s:%s%s", qUser, qPass, qHost, qPort, qVhost)
	ctx := context.Background()

	consumer := messaging.NewAMQPConsumer(amqpUrl, useCase)
	consumer.Start(ctx)

	// 6. Configuración y Arranque de la API HTTP con Echo v5
	app := echo.New()

	notificationHandler := deliveryhttp.NewNotificationHandler(useCase)
	deliveryhttp.SetupRoutes(app, notificationHandler)

	appPort := os.Getenv("APP_PORT")
	if appPort == "" {
		appPort = "10001"
	}

	fmt.Printf("Starting notification-service REST API on port %s...\n", appPort)
	if err := app.Start(":" + appPort); err != nil {
		fmt.Println("Error starting Echo server:", err)
	}
}
