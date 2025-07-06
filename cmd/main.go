package main

import (
	"context"
	"log"
	"net/http"

	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	"github.com/yusufziyrek/bank-app/common/app"
	"github.com/yusufziyrek/bank-app/common/postgresql"

	"github.com/yusufziyrek/bank-app/internal/controller"
	"github.com/yusufziyrek/bank-app/internal/controller/dto"
	"github.com/yusufziyrek/bank-app/internal/repository"
	"github.com/yusufziyrek/bank-app/internal/service"
)

type CustomValidator struct {
	validator *validator.Validate
}

func (cv *CustomValidator) Validate(i interface{}) error {
	if err := cv.validator.Struct(i); err != nil {
		return err
	}
	return nil
}

func customHTTPErrorHandler(err error, c echo.Context) {
	code := http.StatusInternalServerError
	message := "An internal server error occurred"
	details := ""
	validationErrors := []dto.ValidationError{}

	if he, ok := err.(*echo.HTTPError); ok {
		code = he.Code
		if he.Message != nil {
			message = he.Message.(string)
		}
		if he.Internal != nil {
			details = he.Internal.Error()
		}

		if valErrs, isValErrs := he.Internal.(validator.ValidationErrors); isValErrs {
			message = "Validation failed"
			for _, fieldErr := range valErrs {
				validationErrors = append(validationErrors, dto.ValidationError{
					Field: fieldErr.Field(),
					Tag:   fieldErr.Tag(),
					Value: fieldErr.Param(),
				})
			}
		}
	} else if valErrs, ok := err.(validator.ValidationErrors); ok {
		code = http.StatusBadRequest
		message = "Validation failed"
		for _, fieldErr := range valErrs {
			validationErrors = append(validationErrors, dto.ValidationError{
				Field: fieldErr.Field(),
				Tag:   fieldErr.Tag(),
				Value: fieldErr.Param(),
			})
		}
		details = err.Error()
	} else {
		details = err.Error()
	}

	resp := dto.ErrorResponse{
		Message:          message,
		Code:             http.StatusText(code),
		Details:          details,
		ValidationErrors: validationErrors,
	}

	if !c.Response().Committed {
		if err := c.JSON(code, resp); err != nil {
			log.Printf("Failed to send error response: %v", err)
		}
	}
}

func main() {
	ctx := context.Background()

	configManager := app.NewConfigurationManager()

	dbPool, err := postgresql.GetConnectionPool(ctx, configManager.PostgreSqlConfig)
	if err != nil {
		log.Fatalf("Failed to establish database connection: %v", err)
	}
	defer dbPool.Close()

	e := echo.New()

	e.Validator = &CustomValidator{validator: validator.New()}

	e.HTTPErrorHandler = customHTTPErrorHandler

	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.RequestID())
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{echo.GET, echo.PUT, echo.POST, echo.DELETE},
		AllowHeaders: []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept, echo.HeaderAuthorization},
	}))

	userRepo := repository.NewUserRepository(dbPool)
	userService := service.NewUserService(userRepo)
	userController := controller.NewUserHandler(userService)

	userController.RegisterRoutes(e)

	log.Printf("Starting server on port %s...", configManager.AppPort)
	if err := e.Start(":" + configManager.AppPort); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Server failed to start: %v", err)
	}

	log.Println("Server gracefully stopped.")

}
