package main

import (
	"context"
	"log"
	"net/http"

	"github.com/go-playground/validator/v10"
	_ "github.com/labstack/echo-jwt/v4"
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
	cfg := app.NewConfigurationManager()
	ctx := context.Background()

	pool, err := postgresql.GetConnectionPool(ctx, cfg.PostgreSqlConfig)
	if err != nil {
		log.Fatalf("DB connection failed: %v", err)
	}
	defer pool.Close()

	e := echo.New()
	e.Validator = &CustomValidator{validator: validator.New()}

	// Global middleware
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.RequestID())
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{echo.GET, echo.POST, echo.PUT, echo.DELETE},
		AllowHeaders: []string{echo.HeaderAuthorization, echo.HeaderContentType},
	}))

	// DI
	repo := repository.NewUserRepository(pool)
	svc := service.NewUserService(repo)

	// Auth routes (public)
	authCtrl := controller.NewAuthController(svc, cfg)
	e.POST("/api/v1/register", authCtrl.Register)
	e.POST("/api/v1/login", authCtrl.Login)

	// User routes (protected)
	userCtrl := controller.NewUserController(svc)
	g := e.Group("/api/v1")
	g.Use(middleware.CORSWithConfig(middleware.CSRFConfig{
		SigningKey:  []byte(cfg.JwtSecret),
		TokenLookup: "header:Authorization",
		AuthScheme:  "Bearer",
	}))
	g.GET("/users", userCtrl.GetAll)
	g.GET("/users/:id", userCtrl.GetByID)
	g.PUT("/users/:id/email", userCtrl.UpdateEmail)
	g.PUT("/users/:id/password", userCtrl.UpdatePassword)
	g.PUT("/users/:id/status", userCtrl.UpdateStatus)
	g.DELETE("/users/:id", userCtrl.DeleteByID)

	log.Printf("Server running on port %s…", cfg.AppPort)
	if err := e.Start(":" + cfg.AppPort); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Server error: %v", err)
	}
}
