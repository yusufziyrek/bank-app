package controller

import (
	"context"
	"errors"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
	"github.com/yusufziyrek/bank-app/internal/controller/dto"
	"github.com/yusufziyrek/bank-app/internal/service"
)

// Default timeout for database operations
const defaultTimeout = 5 * time.Second

// withTimeout creates a context with default timeout
func withTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, defaultTimeout)
}

// parseID parses and validates user ID from URL parameter
func parseID(c echo.Context) (int64, *echo.HTTPError) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return 0, echo.NewHTTPError(http.StatusBadRequest, dto.ErrorResponse{
			Message: "Invalid user ID",
			Code:    "INVALID_USER_ID",
			Details: "ID must be a valid number",
		})
	}
	if id <= 0 {
		return 0, echo.NewHTTPError(http.StatusBadRequest, dto.ErrorResponse{
			Message: "Invalid user ID",
			Code:    "INVALID_USER_ID",
			Details: "ID must be greater than 0",
		})
	}
	return id, nil
}

// sendError sends a standardized error response
func sendError(c echo.Context, status int, code, msg, details string) error {
	// In production, don't expose internal error details
	if os.Getenv("APP_ENV") == "production" && status == http.StatusInternalServerError {
		details = ""
	}

	return c.JSON(status, dto.ErrorResponse{
		Message: msg,
		Code:    code,
		Details: details,
	})
}

// sendValidationError sends validation error response
func sendValidationError(c echo.Context, errors []dto.ValidationError) error {
	return c.JSON(http.StatusBadRequest, dto.ErrorResponse{
		Message:          "Validation failed",
		Code:             "VALIDATION_ERROR",
		ValidationErrors: errors,
	})
}

// handleServiceError handles service layer errors and returns appropriate HTTP responses
func handleServiceError(c echo.Context, err error, operation string) error {
	switch {
	case errors.Is(err, service.ErrUserNotFound):
		return sendError(c, http.StatusNotFound, "USER_NOT_FOUND", err.Error(), "")
	case errors.Is(err, service.ErrEmailAlreadyRegistered):
		return sendError(c, http.StatusConflict, "EMAIL_EXISTS", err.Error(), "")
	case errors.Is(err, service.ErrInvalidCredentials), errors.Is(err, service.ErrInactiveAccount):
		return sendError(c, http.StatusUnauthorized, "AUTH_FAILED", err.Error(), "")
	default:
		// In production, use generic error message
		errorMsg := "Could not " + operation
		if os.Getenv("APP_ENV") == "production" {
			return sendError(c, http.StatusInternalServerError, "INTERNAL_ERROR", errorMsg, "")
		}
		return sendError(c, http.StatusInternalServerError, "INTERNAL_ERROR", errorMsg, err.Error())
	}
}

// bindAndValidate binds request body and validates it
func bindAndValidate(c echo.Context, req interface{}) error {
	if err := c.Bind(req); err != nil {
		return sendError(c, http.StatusBadRequest, "INVALID_BODY", "Invalid JSON", err.Error())
	}
	if err := c.Validate(req); err != nil {
		// Convert validation errors to structured format
		if validationErrors, ok := err.(validator.ValidationErrors); ok {
			var errors []dto.ValidationError
			for _, fieldError := range validationErrors {
				errors = append(errors, dto.ValidationError{
					Field: fieldError.Field(),
					Tag:   fieldError.Tag(),
					Value: fieldError.Param(),
				})
			}
			return sendValidationError(c, errors)
		}
		return sendError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Validation failed", err.Error())
	}
	return nil
}
