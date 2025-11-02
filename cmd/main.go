package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	"github.com/yusufziyrek/bank-app/common/app"
	"github.com/yusufziyrek/bank-app/common/cache"
	"github.com/yusufziyrek/bank-app/common/postgresql"
	"github.com/yusufziyrek/bank-app/internal/repository"
	"github.com/yusufziyrek/bank-app/internal/routes"
	"github.com/yusufziyrek/bank-app/internal/service"
)

type CustomValidator struct {
	validator *validator.Validate
}

func (cv *CustomValidator) Validate(i interface{}) error {
	return cv.validator.Struct(i)
}

func getCORSConfig(cfg *app.ConfigurationManager) middleware.CORSConfig {
	if cfg.AppEnv == "production" {
		origins := strings.Split(cfg.AllowedOrigins, ",")
		for i, origin := range origins {
			origins[i] = strings.TrimSpace(origin)
		}

		return middleware.CORSConfig{
			AllowOrigins: origins,
			AllowMethods: []string{
				http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete,
			},
			AllowHeaders: []string{
				echo.HeaderOrigin, echo.HeaderContentType,
				echo.HeaderAccept, echo.HeaderAuthorization,
			},
			MaxAge: 86400,
		}
	}

	return middleware.CORSConfig{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{
			http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete,
		},
		AllowHeaders: []string{
			echo.HeaderOrigin, echo.HeaderContentType,
			echo.HeaderAccept, echo.HeaderAuthorization,
		},
		MaxAge: 86400,
	}
}

func main() {
	wd, _ := os.Getwd()
	log.Printf("Çalışma dizini: %s", wd)

	envPath := "../.env"
	if _, err := os.Stat(envPath); os.IsNotExist(err) {
		envPath = ".env"
		if _, err := os.Stat(envPath); os.IsNotExist(err) {
			log.Printf("Warning: .env dosyası bulunamadı")
		} else {
			log.Printf("✓ .env dosyası mevcut (mevcut dizinde)")
		}
	} else {
		log.Printf("✓ .env dosyası mevcut (ana dizinde)")
	}

	err := godotenv.Load(envPath)
	if err != nil {
		log.Printf("Warning: .env dosyası yüklenemedi: %v", err)
	} else {
		log.Printf("✓ .env dosyası başarıyla yüklendi")
	}

	log.Printf("DEBUG: APP_PORT = '%s'", os.Getenv("APP_PORT"))
	log.Printf("DEBUG: PG_HOST = '%s'", os.Getenv("PG_HOST"))
	log.Printf("DEBUG: PG_PORT = '%s'", os.Getenv("PG_PORT"))

	cfg := app.NewConfigurationManager()

	ctx := context.Background()
	pool, err := postgresql.GetConnectionPool(ctx, cfg.PostgreSqlConfig)
	if err != nil {
		log.Fatalf("DB bağlantı hatası: %v", err)
	}
	defer pool.Close()

	e := echo.New()
	e.Debug = cfg.AppEnv != "production"
	e.Validator = &CustomValidator{validator: validator.New()}

	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.RequestID())
	e.Use(middleware.Secure())
	e.Use(middleware.RateLimiter(middleware.NewRateLimiterMemoryStore(20)))
	e.Use(middleware.CORSWithConfig(getCORSConfig(cfg)))

	repo := repository.NewUserRepository(pool)
	accountRepo := repository.NewAccountRepository(pool)

	svc := service.NewUserService(repo)
	accountSvc := service.NewAccountService(accountRepo)

	if cfg.RedisAddr != "" {
		client, cerr := cache.NewRedisClient(cfg.RedisAddr, cfg.RedisPassword, cfg.RedisDB)
		if cerr != nil {
			log.Printf("Redis bağlanılamadı: %v", cerr)
		} else {
			defer client.Close()
			svc = service.NewUserServiceWithCache(repo, client, 5*time.Minute)
			accountSvc = service.NewAccountServiceWithCache(accountRepo, client, 5*time.Minute)
		}
	}

	transactionRepo := repository.NewTransactionRepository(pool)
	transactionSvc := service.NewTransactionService(transactionRepo, accountRepo, accountSvc)

	cardRepo := repository.NewCardRepository(pool)
	cardSvc := service.NewCardService(cardRepo, cfg.CardEncryptionKey)

	routes.SetupRoutes(e, svc, accountSvc, transactionSvc, cardSvc, cfg.JwtSecret, time.Duration(cfg.JwtTTL)*time.Minute)

	go func() {
		addr := ":" + cfg.AppPort
		log.Printf("⇨ http server started on %s", addr)
		log.Printf("⇨ Environment: %s", cfg.AppEnv)
		if err := e.Start(addr); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Sunucu hatası: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	log.Println("Sunucu kapatılıyor…")
	ctxShut, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := e.Shutdown(ctxShut); err != nil {
		log.Printf("Sunucu kapatma hatası: %v", err)
	}
	log.Println("Sunucu başarıyla kapatıldı")
}
