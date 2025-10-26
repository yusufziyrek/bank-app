package controller

import (
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/yusufziyrek/bank-app/internal/controller/dto"
	"github.com/yusufziyrek/bank-app/internal/service"
)

type UserController struct {
	svc service.UserService
}

func NewUserController(svc service.UserService) *UserController {
	return &UserController{svc: svc}
}

func (u *UserController) GetAll(c echo.Context) error {
	ctx, cancel := withTimeout(c.Request().Context())
	defer cancel()

	users, err := u.svc.GetAllUsers(ctx)
	if err != nil {
		return handleServiceError(c, err, "fetch users")
	}

	resp := dto.UsersResponse{
		Users: make([]dto.UserResponse, len(users)),
		Count: len(users),
	}
	for i, usr := range users {
		resp.Users[i] = dto.UserResponseFromModel(usr)
	}
	return c.JSON(http.StatusOK, resp)
}

func (u *UserController) GetByID(c echo.Context) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		return sendError(c, http.StatusBadRequest,
			ErrInvalidUserID, "Invalid user ID", "ID must be a positive integer")
	}

	ctx, cancel := withTimeout(c.Request().Context())
	defer cancel()

	user, err := u.svc.GetUserByID(ctx, id)
	if err != nil {
		return handleServiceError(c, err, "fetch user")
	}

	return c.JSON(http.StatusOK, dto.UserResponseFromModel(user))
}

func (u *UserController) MyProfile(c echo.Context) error {
	userID, err := getUserIDFromToken(c)
	if err != nil {
		return sendError(c, http.StatusUnauthorized,
			ErrUnauthorized, "User not authenticated", err.Error())
	}

	ctx, cancel := withTimeout(c.Request().Context())
	defer cancel()

	user, err := u.svc.GetUserByID(ctx, userID)
	if err != nil {
		return handleServiceError(c, err, "fetch user profile")
	}

	return c.JSON(http.StatusOK, dto.UserResponseFromModel(user))
}

func (u *UserController) UpdateEmail(c echo.Context) error {
	userID, err := getUserIDFromToken(c)
	if err != nil {
		return sendError(c, http.StatusUnauthorized, "UNAUTHORIZED", "User not authenticated", err.Error())
	}
	var req dto.UpdateUserEmailRequest
	if ok := bindAndValidate(c, &req); !ok {
		return nil
	}
	ctx, cancel := withTimeout(c.Request().Context())
	defer cancel()

	if err := u.svc.UpdateUserEmail(ctx, userID, req.NewEmail); err != nil {
		return handleServiceError(c, err, "update email")
	}

	return c.NoContent(http.StatusNoContent)
}

func (u *UserController) UpdatePassword(c echo.Context) error {
	userID, err := getUserIDFromToken(c)
	if err != nil {
		return sendError(c, http.StatusUnauthorized, "UNAUTHORIZED", "User not authenticated", err.Error())
	}
	var req dto.UpdateUserPasswordRequest
	if ok := bindAndValidate(c, &req); !ok {
		return nil
	}
	ctx, cancel := withTimeout(c.Request().Context())
	defer cancel()

	if err := u.svc.UpdateUserPassword(ctx, userID, req.NewPassword); err != nil {
		return handleServiceError(c, err, "update password")
	}

	return c.NoContent(http.StatusNoContent)
}

func (u *UserController) UpdateStatus(c echo.Context) error {
	userID, err := getUserIDFromToken(c)
	if err != nil {
		return sendError(c, http.StatusUnauthorized, "UNAUTHORIZED", "User not authenticated", err.Error())
	}
	var req dto.UpdateUserStatusRequest
	if ok := bindAndValidate(c, &req); !ok {
		return nil
	}
	ctx, cancel := withTimeout(c.Request().Context())
	defer cancel()

	if err := u.svc.UpdateUserActiveStatus(ctx, userID, req.IsActive); err != nil {
		return handleServiceError(c, err, "update status")
	}

	return c.NoContent(http.StatusNoContent)
}

func (u *UserController) DeleteByID(c echo.Context) error {
	userID, err := getUserIDFromToken(c)
	if err != nil {
		return sendError(c, http.StatusUnauthorized, "UNAUTHORIZED", "User not authenticated", err.Error())
	}
	ctx, cancel := withTimeout(c.Request().Context())
	defer cancel()

	if err := u.svc.DeleteUserByID(ctx, userID); err != nil {
		return handleServiceError(c, err, "delete user")
	}

	return c.NoContent(http.StatusNoContent)
}
