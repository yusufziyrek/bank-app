package controller

import (
	"net/http"

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
	id, herr := parseID(c)
	if herr != nil {
		return c.JSON(herr.Code, herr.Message)
	}

	ctx, cancel := withTimeout(c.Request().Context())
	defer cancel()

	user, err := u.svc.GetUserByID(ctx, id)
	if err != nil {
		return handleServiceError(c, err, "fetch user")
	}

	return c.JSON(http.StatusOK, dto.UserResponseFromModel(user))
}

func (u *UserController) UpdateEmail(c echo.Context) error {
	id, herr := parseID(c)
	if herr != nil {
		return c.JSON(herr.Code, herr.Message)
	}

	var req dto.UpdateUserEmailRequest
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}

	ctx, cancel := withTimeout(c.Request().Context())
	defer cancel()

	if err := u.svc.UpdateUserEmail(ctx, id, req.NewEmail); err != nil {
		return handleServiceError(c, err, "update email")
	}

	return c.NoContent(http.StatusNoContent)
}

func (u *UserController) UpdatePassword(c echo.Context) error {
	id, herr := parseID(c)
	if herr != nil {
		return c.JSON(herr.Code, herr.Message)
	}

	var req dto.UpdateUserPasswordRequest
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}

	ctx, cancel := withTimeout(c.Request().Context())
	defer cancel()

	if err := u.svc.UpdateUserPassword(ctx, id, req.NewPassword); err != nil {
		return handleServiceError(c, err, "update password")
	}

	return c.NoContent(http.StatusNoContent)
}

func (u *UserController) UpdateStatus(c echo.Context) error {
	id, herr := parseID(c)
	if herr != nil {
		return c.JSON(herr.Code, herr.Message)
	}

	var req dto.UpdateUserStatusRequest
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}

	ctx, cancel := withTimeout(c.Request().Context())
	defer cancel()

	if err := u.svc.UpdateUserActiveStatus(ctx, id, req.IsActive); err != nil {
		return handleServiceError(c, err, "update status")
	}

	return c.NoContent(http.StatusNoContent)
}

func (u *UserController) DeleteByID(c echo.Context) error {
	id, herr := parseID(c)
	if herr != nil {
		return c.JSON(herr.Code, herr.Message)
	}

	ctx, cancel := withTimeout(c.Request().Context())
	defer cancel()

	if err := u.svc.DeleteUserByID(ctx, id); err != nil {
		return handleServiceError(c, err, "delete user")
	}

	return c.NoContent(http.StatusNoContent)
}
