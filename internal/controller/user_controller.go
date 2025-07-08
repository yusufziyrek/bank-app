package controller

import (
	"context"
	"net/http"
	"time"

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
	ctx, cancel := context.WithTimeout(c.Request().Context(), 5*time.Second)
	defer cancel()

	users, err := u.svc.GetAllUsers(ctx)
	if err != nil {
		return sendError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Could not fetch users", err.Error())
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
	id, herr := parseID(c)
	if herr != nil {
		return c.JSON(herr.Code, herr.Message)
	}
	ctx, cancel := context.WithTimeout(c.Request().Context(), 5*time.Second)
	defer cancel()

	user, err := u.svc.GetUserByID(ctx, id)
	switch err {
	case service.ErrUserNotFound:
		return sendError(c, http.StatusNotFound, "USER_NOT_FOUND", err.Error(), "")
	case nil:
		return c.JSON(http.StatusOK, dto.UserResponseFromModel(user))
	default:
		return sendError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Could not fetch user", err.Error())
	}
}

func (u *UserController) UpdateEmail(c echo.Context) error {
	id, herr := parseID(c)
	if herr != nil {
		return c.JSON(herr.Code, herr.Message)
	}
	var req dto.UpdateUserEmailRequest
	if err := c.Bind(&req); err != nil {
		return sendError(c, http.StatusBadRequest, "INVALID_BODY", "Invalid JSON", err.Error())
	}
	if err := c.Validate(&req); err != nil {
		return err
	}
	if err := u.svc.UpdateUserEmail(c.Request().Context(), id, req.NewEmail); err != nil {
		switch err {
		case service.ErrUserNotFound:
			return sendError(c, http.StatusNotFound, "USER_NOT_FOUND", err.Error(), "")
		case service.ErrEmailAlreadyRegistered:
			return sendError(c, http.StatusConflict, "EMAIL_EXISTS", err.Error(), "")
		default:
			return sendError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Could not update email", err.Error())
		}
	}
	return c.NoContent(http.StatusNoContent)
}

func (u *UserController) UpdatePassword(c echo.Context) error {
	id, herr := parseID(c)
	if herr != nil {
		return c.JSON(herr.Code, herr.Message)
	}
	var req dto.UpdateUserPasswordRequest
	if err := c.Bind(&req); err != nil {
		return sendError(c, http.StatusBadRequest, "INVALID_BODY", "Invalid JSON", err.Error())
	}
	if err := c.Validate(&req); err != nil {
		return err
	}
	if err := u.svc.UpdateUserPassword(c.Request().Context(), id, req.NewPassword); err != nil {
		if err == service.ErrUserNotFound {
			return sendError(c, http.StatusNotFound, "USER_NOT_FOUND", err.Error(), "")
		}
		return sendError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Could not update password", err.Error())
	}
	return c.NoContent(http.StatusNoContent)
}

func (u *UserController) UpdateStatus(c echo.Context) error {
	id, herr := parseID(c)
	if herr != nil {
		return c.JSON(herr.Code, herr.Message)
	}
	var req dto.UpdateUserStatusRequest
	if err := c.Bind(&req); err != nil {
		return sendError(c, http.StatusBadRequest, "INVALID_BODY", "Invalid JSON", err.Error())
	}
	if err := u.svc.UpdateUserActiveStatus(c.Request().Context(), id, req.IsActive); err != nil {
		if err == service.ErrUserNotFound {
			return sendError(c, http.StatusNotFound, "USER_NOT_FOUND", err.Error(), "")
		}
		return sendError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Could not update status", err.Error())
	}
	return c.NoContent(http.StatusNoContent)
}

func (u *UserController) DeleteByID(c echo.Context) error {
	id, herr := parseID(c)
	if herr != nil {
		return c.JSON(herr.Code, herr.Message)
	}
	if err := u.svc.DeleteUserByID(c.Request().Context(), id); err != nil {
		if err == service.ErrUserNotFound {
			return sendError(c, http.StatusNotFound, "USER_NOT_FOUND", err.Error(), "")
		}
		return sendError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Could not delete user", err.Error())
	}
	return c.NoContent(http.StatusNoContent)
}
