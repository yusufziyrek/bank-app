package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/joho/godotenv"
	echojwt "github.com/labstack/echo-jwt/v4"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	"github.com/yusufziyrek/bank-app/common/app"
	"github.com/yusufziyrek/bank-app/common/postgresql"
	"github.com/yusufziyrek/bank-app/internal/controller"
	"github.com/yusufziyrek/bank-app/internal/repository"
	"github.com/yusufziyrek/bank-app/internal/service"
)

type CustomValidator struct {
	validator *validator.Validate
}

func (cv *CustomValidator) Validate(i interface{}) error {
	return cv.validator.Struct(i)
}

func main() {
	_ = godotenv.Load()

	cfg := app.NewConfigurationManager()

	ctx := context.Background()
	pool, err := postgresql.GetConnectionPool(ctx, cfg.PostgreSqlConfig)
	if err != nil {
		log.Fatalf("DB bağlantı hatası: %v", err)
	}
	defer pool.Close()

	e := echo.New()
	e.Debug = true
	e.Validator = &CustomValidator{validator: validator.New()}
	e.Use(middleware.Logger(), middleware.Recover(), middleware.RequestID())
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{
			http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete,
		},
		AllowHeaders: []string{
			echo.HeaderOrigin, echo.HeaderContentType,
			echo.HeaderAccept, echo.HeaderAuthorization,
		},
	}))

	repo := repository.NewUserRepository(pool)
	svc := service.NewUserService(repo)

	authCtrl := controller.NewAuthController(svc, cfg)
	e.POST("/api/v1/register", authCtrl.Register)
	e.POST("/api/v1/login", authCtrl.Login)

	jwtGroup := e.Group("/api/v1")
	jwtGroup.Use(echojwt.WithConfig(echojwt.Config{
		SigningKey: []byte(cfg.JwtSecret),
	}))
	userCtrl := controller.NewUserController(svc)
	jwtGroup.GET("/users", userCtrl.GetAll)
	jwtGroup.GET("/users/:id", userCtrl.GetByID)
	jwtGroup.PUT("/users/:id/email", userCtrl.UpdateEmail)
	jwtGroup.PUT("/users/:id/password", userCtrl.UpdatePassword)
	jwtGroup.PUT("/users/:id/status", userCtrl.UpdateStatus)
	jwtGroup.DELETE("/users/:id", userCtrl.DeleteByID)

	go func() {
		addr := "127.0.0.1:" + cfg.AppPort
		log.Printf("⇨ http server started on %s", addr)
		if err := e.Start(addr); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Sunucu hatası: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	<-quit

	log.Println("Sunucu kapatılıyor…")
	ctxShut, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := e.Shutdown(ctxShut); err != nil {
		log.Fatalf("Sunucu kapatma hatası: %v", err)
	}
}
