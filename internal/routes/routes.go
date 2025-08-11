package routes

import (
	"time"

	echojwt "github.com/labstack/echo-jwt/v4"
	"github.com/labstack/echo/v4"

	"github.com/yusufziyrek/bank-app/internal/controller"
	"github.com/yusufziyrek/bank-app/internal/service"
)

func SetupRoutes(
	e *echo.Echo,
	userService service.UserService,
	accountService service.AccountService,
	transactionService service.TransactionService,
	cardService service.CardService,
	jwtSecret string,
	jwtTTL time.Duration,
) {
	// Public
	authCtrl := controller.NewAuthController(userService, jwtSecret, jwtTTL)
	e.POST("/api/v1/register", authCtrl.Register)
	e.POST("/api/v1/login", authCtrl.Login)
	e.POST("/api/v1/refresh", authCtrl.Refresh)

	// Protected
	jwtGroup := e.Group("/api/v1")
	jwtGroup.Use(echojwt.WithConfig(echojwt.Config{
		SigningKey: []byte(jwtSecret),
		ErrorHandler: func(c echo.Context, err error) error {
			return controller.SendError(c, 401, controller.ErrUnauthorized, "Missing or malformed JWT", err.Error())
		},
	}))

	// User
	userCtrl := controller.NewUserController(userService)
	jwtGroup.GET("/users", userCtrl.GetAll)
	jwtGroup.GET("/users/:id", userCtrl.GetByID)
	jwtGroup.GET("/users/me", userCtrl.MyProfile)
	jwtGroup.PUT("/users/:id/email", userCtrl.UpdateEmail)
	jwtGroup.PUT("/users/:id/password", userCtrl.UpdatePassword)
	jwtGroup.PUT("/users/:id/status", userCtrl.UpdateStatus)
	jwtGroup.DELETE("/users/:id", userCtrl.DeleteByID)

	// Account
	accountCtrl := controller.NewAccountController(accountService, cardService)
	jwtGroup.GET("/accounts", accountCtrl.GetAll)
	jwtGroup.GET("/accounts/:id", accountCtrl.GetByID)
	jwtGroup.GET("/accounts/me", accountCtrl.MyAccounts)
	jwtGroup.POST("/accounts", accountCtrl.Create)
	jwtGroup.PUT("/accounts/:id", accountCtrl.Update)
	jwtGroup.DELETE("/accounts/:id", accountCtrl.Delete)

	// Transaction
	transactionCtrl := controller.NewTransactionController(transactionService, accountService)
	jwtGroup.GET("/transactions", transactionCtrl.GetAll)
	jwtGroup.GET("/transactions/:id", transactionCtrl.GetByID)
	jwtGroup.POST("/transactions", transactionCtrl.Create)
	jwtGroup.PUT("/transactions/:id", transactionCtrl.Update)
	jwtGroup.DELETE("/transactions/:id", transactionCtrl.Delete)

	// Card
	cardCtrl := controller.NewCardController(cardService, accountService)
	jwtGroup.GET("/cards", cardCtrl.GetAll)
	jwtGroup.GET("/cards/:id", cardCtrl.GetByID)
	jwtGroup.GET("/cards/me", cardCtrl.GetMyCards)
	jwtGroup.POST("/cards", cardCtrl.Create)
	jwtGroup.PUT("/cards/:id", cardCtrl.Update)
	jwtGroup.PATCH("/cards/:id", cardCtrl.Update)
	jwtGroup.PATCH("/cards/:id/status", cardCtrl.UpdateStatus)
	jwtGroup.DELETE("/cards/:id", cardCtrl.Delete)
}
