package controller

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
	"github.com/yusufziyrek/bank-app/internal/controller/dto"
	"github.com/yusufziyrek/bank-app/internal/model"
	"github.com/yusufziyrek/bank-app/internal/service"
)

type CustomValidator struct {
	validator *validator.Validate
}

func (cv *CustomValidator) Validate(i interface{}) error {
	if err := cv.validator.Struct(i); err != nil {
		var validationErrors []dto.ValidationError
		for _, err := range err.(validator.ValidationErrors) {
			validationErrors = append(validationErrors, dto.ValidationError{
				Field: err.Field(),
				Tag:   err.Tag(),
				Value: err.Param(),
			})
		}
		return echo.NewHTTPError(http.StatusBadRequest, dto.ErrorResponse{
			Message:          "Validation failed",
			Code:             "VALIDATION_ERROR",
			ValidationErrors: validationErrors,
		})
	}
	return nil
}

type UserHandler struct {
	userService service.UserService
}

func NewUserHandler(userService service.UserService) *UserHandler {
	return &UserHandler{userService: userService}
}

func (h *UserHandler) RegisterRoutes(e *echo.Echo) {
	apiV1 := e.Group("/api/v1")

	apiV1.GET("/users", h.GetAllUsers)
	apiV1.GET("/users/:id", h.GetUserByID)
	apiV1.POST("/users", h.CreateUser)
	apiV1.PUT("/users/:id/email", h.UpdateUserEmail)
	apiV1.PUT("/users/:id/password", h.UpdateUserPassword)
	apiV1.PUT("/users/:id/status", h.UpdateUserActiveStatus)
	apiV1.DELETE("/users/:id", h.DeleteUserByID)

	apiV1.POST("/login", h.LoginUser)
}

func createUserRequestToModel(req dto.CreateUserRequest) *model.User {
	return &model.User{
		FullName:     req.FullName,
		Email:        req.Email,
		PasswordHash: req.Password,
	}
}

func userModelToResponse(user model.User) dto.UserResponse {
	return dto.UserResponse{
		ID:        user.ID,
		FullName:  user.FullName,
		Email:     user.Email,
		Role:      user.Role,
		IsActive:  user.IsActive,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	}
}

func (h *UserHandler) GetAllUsers(c echo.Context) error {
	ctx := c.Request().Context()
	users, err := h.userService.GetAllUsers(ctx)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Message: "An internal server error occurred",
			Code:    "INTERNAL_SERVER_ERROR",
			Details: err.Error(),
		})
	}
	userResponses := make([]dto.UserResponse, 0, len(users))
	for _, user := range users {
		userResponses = append(userResponses, userModelToResponse(user))
	}
	return c.JSON(http.StatusOK, dto.UsersResponse{
		Users: userResponses,
		Count: len(userResponses),
	})
}

func (h *UserHandler) GetUserByID(c echo.Context) error {
	ctx := c.Request().Context()
	userID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Message: "Invalid request",
			Code:    "INVALID_USER_ID",
			Details: "User ID must be a valid number.",
		})
	}
	user, err := h.userService.GetUserByID(ctx, userID)
	if err != nil {
		if errors.Is(err, service.ErrUserNotFound) {
			return c.JSON(http.StatusNotFound, dto.ErrorResponse{
				Message: "User not found",
				Code:    "USER_NOT_FOUND",
				Details: err.Error(),
			})
		}
		return c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Message: "An internal server error occurred",
			Code:    "INTERNAL_SERVER_ERROR",
			Details: err.Error(),
		})
	}
	return c.JSON(http.StatusOK, userModelToResponse(user))
}

func (h *UserHandler) CreateUser(c echo.Context) error {
	ctx := c.Request().Context()
	req := new(dto.CreateUserRequest)

	if err := c.Bind(req); err != nil {
		return c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Message: "Invalid request body",
			Code:    "INVALID_REQUEST_BODY",
			Details: err.Error(),
		})
	}
	if err := c.Validate(req); err != nil {
		return err
	}

	userModel := createUserRequestToModel(*req)

	err := h.userService.CreateUser(ctx, userModel)
	if err != nil {
		if errors.Is(err, service.ErrEmailAlreadyRegistered) {
			return c.JSON(http.StatusConflict, dto.ErrorResponse{
				Message: "Email is already registered",
				Code:    "EMAIL_ALREADY_REGISTERED",
				Details: err.Error(),
			})
		}
		return c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Message: "An internal server error occurred",
			Code:    "INTERNAL_SERVER_ERROR",
			Details: err.Error(),
		})
	}

	return c.JSON(http.StatusCreated, userModelToResponse(*userModel))
}

func (h *UserHandler) UpdateUserEmail(c echo.Context) error {
	ctx := c.Request().Context()
	userID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Message: "Invalid request",
			Code:    "INVALID_USER_ID",
			Details: "User ID must be a valid number.",
		})
	}
	req := new(dto.UpdateUserEmailRequest)
	if err := c.Bind(req); err != nil {
		return c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Message: "Invalid request body",
			Code:    "INVALID_REQUEST_BODY",
			Details: err.Error(),
		})
	}
	if err := c.Validate(req); err != nil {
		return err
	}
	err = h.userService.UpdateUserEmail(ctx, userID, req.NewEmail)
	if err != nil {
		if errors.Is(err, service.ErrUserNotFound) {
			return c.JSON(http.StatusNotFound, dto.ErrorResponse{
				Message: "User not found",
				Code:    "USER_NOT_FOUND",
				Details: err.Error(),
			})
		}
		if errors.Is(err, service.ErrEmailAlreadyRegistered) {
			return c.JSON(http.StatusConflict, dto.ErrorResponse{
				Message: "New email is already in use",
				Code:    "EMAIL_ALREADY_REGISTERED",
				Details: err.Error(),
			})
		}
		return c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Message: "An internal server error occurred",
			Code:    "INTERNAL_SERVER_ERROR",
			Details: err.Error(),
		})
	}
	return c.NoContent(http.StatusOK)
}

func (h *UserHandler) UpdateUserPassword(c echo.Context) error {
	ctx := c.Request().Context()
	userID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Message: "Invalid request",
			Code:    "INVALID_USER_ID",
			Details: "User ID must be a valid number.",
		})
	}
	req := new(dto.UpdateUserPasswordRequest)
	if err := c.Bind(req); err != nil {
		return c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Message: "Invalid request body",
			Code:    "INVALID_REQUEST_BODY",
			Details: err.Error(),
		})
	}
	if err := c.Validate(req); err != nil {
		return err
	}
	err = h.userService.UpdateUserPassword(ctx, userID, req.NewPassword)
	if err != nil {
		if errors.Is(err, service.ErrUserNotFound) {
			return c.JSON(http.StatusNotFound, dto.ErrorResponse{
				Message: "User not found",
				Code:    "USER_NOT_FOUND",
				Details: err.Error(),
			})
		}
		return c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Message: "An internal server error occurred",
			Code:    "INTERNAL_SERVER_ERROR",
			Details: err.Error(),
		})
	}
	return c.NoContent(http.StatusOK)
}

func (h *UserHandler) UpdateUserActiveStatus(c echo.Context) error {
	ctx := c.Request().Context()
	userID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Message: "Invalid request",
			Code:    "INVALID_USER_ID",
			Details: "User ID must be a valid number.",
		})
	}
	req := new(dto.UpdateUserStatusRequest)
	if err := c.Bind(req); err != nil {
		return c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Message: "Invalid request body",
			Code:    "INVALID_REQUEST_BODY",
			Details: err.Error(),
		})
	}
	err = h.userService.UpdateUserActiveStatus(ctx, userID, req.IsActive)
	if err != nil {
		if errors.Is(err, service.ErrUserNotFound) {
			return c.JSON(http.StatusNotFound, dto.ErrorResponse{
				Message: "User not found",
				Code:    "USER_NOT_FOUND",
				Details: err.Error(),
			})
		}
		return c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Message: "An internal server error occurred",
			Code:    "INTERNAL_SERVER_ERROR",
			Details: err.Error(),
		})
	}
	return c.NoContent(http.StatusOK)
}

func (h *UserHandler) DeleteUserByID(c echo.Context) error {
	ctx := c.Request().Context()
	userID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Message: "Invalid request",
			Code:    "INVALID_USER_ID",
			Details: "User ID must be a valid number.",
		})
	}
	err = h.userService.DeleteUserByID(ctx, userID)
	if err != nil {
		if errors.Is(err, service.ErrUserNotFound) {
			return c.JSON(http.StatusNotFound, dto.ErrorResponse{
				Message: "User not found",
				Code:    "USER_NOT_FOUND",
				Details: err.Error(),
			})
		}
		return c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Message: "An internal server error occurred",
			Code:    "INTERNAL_SERVER_ERROR",
			Details: err.Error(),
		})
	}
	return c.NoContent(http.StatusNoContent)
}

func (h *UserHandler) LoginUser(c echo.Context) error {
	ctx := c.Request().Context()
	req := new(dto.LoginRequest)
	if err := c.Bind(req); err != nil {
		return c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Message: "Invalid request body",
			Code:    "INVALID_REQUEST_BODY",
			Details: err.Error(),
		})
	}
	if err := c.Validate(req); err != nil {
		return err
	}
	user, err := h.userService.AuthenticateUser(ctx, req.Email, req.Password)
	if err != nil {
		if errors.Is(err, service.ErrInvalidCredentials) || errors.Is(err, service.ErrInactiveAccount) {
			return c.JSON(http.StatusUnauthorized, dto.ErrorResponse{
				Message: "Invalid email or password",
				Code:    "AUTHENTICATION_FAILED",
				Details: "Authentication failed due to invalid credentials or inactive account.",
			})
		}
		return c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Message: "An internal server error occurred",
			Code:    "INTERNAL_SERVER_ERROR",
			Details: err.Error(),
		})
	}
	return c.JSON(http.StatusOK, userModelToResponse(user))
}
