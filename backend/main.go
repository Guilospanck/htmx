package main

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func main() {
	e := echo.New()

	// Middleware
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORS())

	e.GET("/ping", func(c echo.Context) error {
		response := "<p>I AM TESTING</p>"
		// c.Response().Header().Set("Access-Control-Allow-Origin", "*")
		return c.String(http.StatusOK, response)
	})

	e.Logger.Fatal(e.Start(":4444"))
}
