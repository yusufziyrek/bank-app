package main

import (
	"fmt"
	"github.com/labstack/echo/v4"
)

func main() {
	fmt.Println("Hello World")

	e := echo.New()
	e.Start("localhost:8080")
}
