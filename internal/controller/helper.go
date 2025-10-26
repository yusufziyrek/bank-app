package controller

import (
	"context"
	"errors"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/yusufziyrek/bank-app/internal/service"

	"github.com/go-playground/validator/v10"
	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
	"github.com/yusufziyrek/bank-app/internal/controller/dto"
)

const defaultTimeout = 5 * time.Second

func withTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, defaultTimeout)
}

func parseID(c echo.Context) (int64, *echo.HTTPError) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return 0, echo.NewHTTPError(http.StatusBadRequest, map[string]interface{}{
			"error": dto.ErrorResponse{
				Code:    ErrInvalidUserID,
				Message: "Invalid user ID",
				Details: "ID must be a valid number",
			},
		})
	}
	if id <= 0 {
		return 0, echo.NewHTTPError(http.StatusBadRequest, map[string]interface{}{
			"error": dto.ErrorResponse{
				Code:    ErrInvalidUserID,
				Message: "Invalid user ID",
				Details: "ID must be greater than 0",
			},
		})
	}
	return id, nil
}

func sendError(c echo.Context, status int, code, msg, details string) error {
	if os.Getenv("APP_ENV") == "production" && status == http.StatusInternalServerError {
		details = ""
	}
	return c.JSON(status, map[string]interface{}{
		"error": dto.ErrorResponse{
			Code:    code,
			Message: msg,
			Details: details,
		},
	})
}

func sendValidationError(c echo.Context, errors []dto.ValidationError) error {
	return c.JSON(http.StatusBadRequest, map[string]interface{}{
		"error": dto.ErrorResponse{
			Code:             "VALIDATION_ERROR",
			Message:          "Validation failed",
			ValidationErrors: errors,
		},
	})
}

func handleServiceError(c echo.Context, err error, operation string) error {
	switch {
	case errors.Is(err, service.ErrUserNotFound):
		return sendError(c, http.StatusNotFound, ErrUserNotFound, err.Error(), "")
	case errors.Is(err, service.ErrAccountNotFound):
		return sendError(c, http.StatusNotFound, ErrAccountNotFound, err.Error(), "")
	case errors.Is(err, service.ErrCardNotFound):
		return sendError(c, http.StatusNotFound, ErrInvalidCardID, err.Error(), "")
	case errors.Is(err, service.ErrTransactionNotFound):
		return sendError(c, http.StatusNotFound, ErrTransactionNotFound, err.Error(), "")
	case errors.Is(err, service.ErrEmailAlreadyRegistered):
		return sendError(c, http.StatusConflict, ErrEmailExists, err.Error(), "")
	case errors.Is(err, service.ErrInvalidCredentials), errors.Is(err, service.ErrInactiveAccount):
		return sendError(c, http.StatusUnauthorized, ErrAuthFailed, err.Error(), "")
	case errors.Is(err, service.ErrMaxAccountsExceeded):
		return sendError(c, http.StatusBadRequest, ErrMaxAccountsReached, err.Error(), "")
	case errors.Is(err, service.ErrCardAlreadyExists):
		return sendError(c, http.StatusConflict, ErrCardAlreadyExists, err.Error(), "")
	case errors.Is(err, service.ErrInsufficientFunds):
		return sendError(c, http.StatusBadRequest, ErrInsufficientFunds, err.Error(), "")
	case errors.Is(err, service.ErrInvalidTransactionType):
		return sendError(c, http.StatusBadRequest, ErrInvalidTransactionType, err.Error(), "")
	default:
		errorMsg := "Could not " + operation
		if os.Getenv("APP_ENV") == "production" {
			return sendError(c, http.StatusInternalServerError, ErrInternalServerError, errorMsg, "")
		}
		return sendError(c, http.StatusInternalServerError, ErrInternalServerError, errorMsg, err.Error())
	}
}

func bindAndValidate(c echo.Context, req interface{}) bool {
	if err := c.Bind(req); err != nil {
		sendError(c, http.StatusBadRequest, "INVALID_BODY", "Invalid JSON", err.Error())
		return false
	}
	if err := c.Validate(req); err != nil {
		if validationErrors, ok := err.(validator.ValidationErrors); ok {
			var errors []dto.ValidationError
			for _, fieldError := range validationErrors {
				errors = append(errors, dto.ValidationError{
					Field: fieldError.Field(),
					Tag:   fieldError.Tag(),
					Value: fieldError.Param(),
				})
			}
			sendValidationError(c, errors)
			return false
		}
		sendError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Validation failed", err.Error())
		return false
	}
	return true
}

func getUserIDFromToken(c echo.Context) (int64, error) {
	user := c.Get("user")
	if user == nil {
		return 0, errors.New("user not found in context")
	}
	claims, ok := user.(*jwt.Token)
	if !ok {
		return 0, errors.New("invalid jwt token")
	}
	mapClaims, ok := claims.Claims.(jwt.MapClaims)
	if !ok {
		return 0, errors.New("invalid jwt claims")
	}
	sub, ok := mapClaims["sub"].(float64)
	if !ok {
		return 0, errors.New("user id (sub) not found in token")
	}
	return int64(sub), nil
}
