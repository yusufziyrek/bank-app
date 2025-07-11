package routes

import (
	"time"

	echojwt "github.com/labstack/echo-jwt/v4"
	"github.com/labstack/echo/v4"
	"github.com/yusufziyrek/bank-app/internal/controller"
	"github.com/yusufziyrek/bank-app/internal/service"
)

func SetupRoutes(e *echo.Echo, userService service.UserService, jwtSecret string, jwtTTL time.Duration) {
	// Auth routes (public)
	authCtrl := controller.NewAuthController(userService, jwtSecret, jwtTTL)
	e.POST("/api/v1/register", authCtrl.Register)
	e.POST("/api/v1/login", authCtrl.Login)
	e.POST("/api/v1/refresh", authCtrl.Refresh)

	// Protected routes
	jwtGroup := e.Group("/api/v1")
	jwtGroup.Use(echojwt.WithConfig(echojwt.Config{
		SigningKey: []byte(jwtSecret),
	}))

	userCtrl := controller.NewUserController(userService)
	jwtGroup.GET("/users", userCtrl.GetAll)
	jwtGroup.GET("/users/:id", userCtrl.GetByID)
	jwtGroup.PUT("/users/:id/email", userCtrl.UpdateEmail)
	jwtGroup.PUT("/users/:id/password", userCtrl.UpdatePassword)
	jwtGroup.PUT("/users/:id/status", userCtrl.UpdateStatus)
	jwtGroup.DELETE("/users/:id", userCtrl.DeleteByID)
}
