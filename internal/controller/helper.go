package controller

import (
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/yusufziyrek/bank-app/internal/controller/dto"
)

func parseID(c echo.Context) (int64, *echo.HTTPError) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return 0, echo.NewHTTPError(http.StatusBadRequest, dto.ErrorResponse{
			Message: "Invalid user ID", Code: "INVALID_USER_ID", Details: "ID must be a number",
		})
	}
	return id, nil
}

func sendError(c echo.Context, status int, code, msg, details string) error {
	return c.JSON(status, dto.ErrorResponse{
		Message: msg, Code: code, Details: details,
	})
}
