package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	"github.com/yusufziyrek/bank-app/common/app"
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

// getCORSConfig environment'a göre CORS ayarlarını döner
func getCORSConfig(cfg *app.ConfigurationManager) middleware.CORSConfig {
	if cfg.AppEnv == "production" {
		// Prod ortamında sadece belirli domain'ler
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

	// Development ortamında tüm origin'lere izin ver
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
	// Çalışma dizinini kontrol et
	wd, _ := os.Getwd()
	log.Printf("Çalışma dizini: %s", wd)

	// Ana dizine git (cmd/ klasöründen bir üst dizine)
	envPath := "../.env"
	if _, err := os.Stat(envPath); os.IsNotExist(err) {
		// Eğer ../.env yoksa, mevcut dizinde dene
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

	// Debug: Environment değişkenlerini kontrol et
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
	e.Debug = cfg.AppEnv != "production" // Prod'da debug kapalı
	e.Validator = &CustomValidator{validator: validator.New()}

	// Middleware setup
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.RequestID())
	e.Use(middleware.Secure())
	e.Use(middleware.RateLimiter(middleware.NewRateLimiterMemoryStore(20)))
	e.Use(middleware.CORSWithConfig(getCORSConfig(cfg)))

	repo := repository.NewUserRepository(pool)
	svc := service.NewUserService(repo)

	// Setup routes
	routes.SetupRoutes(e, svc, cfg.JwtSecret, time.Duration(cfg.JwtTTL)*time.Minute)

	go func() {
		addr := "127.0.0.1:" + cfg.AppPort
		log.Printf("⇨ http server started on %s", addr)
		log.Printf("⇨ Environment: %s", cfg.AppEnv)
		if err := e.Start(addr); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Sunucu hatası: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, os.Kill)
	<-quit

	log.Println("Sunucu kapatılıyor…")
	ctxShut, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := e.Shutdown(ctxShut); err != nil {
		log.Printf("Sunucu kapatma hatası: %v", err)
	}
	log.Println("Sunucu başarıyla kapatıldı")
}
