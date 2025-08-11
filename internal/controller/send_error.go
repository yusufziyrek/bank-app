package controller

import (
	"github.com/labstack/echo/v4"
)

func SendError(c echo.Context, status int, code, msg, details string) error {
	return sendError(c, status, code, msg, details)
}
