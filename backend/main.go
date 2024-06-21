package main

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"golang.org/x/net/websocket"
)

// To test via CLI: wscat -H "Origin: http://localhost:4444" -c ws://localhost:4444/ws
func ws(c echo.Context) error {
	response := `
	<div>
	<p style="background: red; width: 10px; height: 10px"></p>
	<p style="background: black; width: 10px; height: 10px"></p>
	<p style="background: red; width: 10px; height: 10px"></p>
	</div>
	`
	websocket.Handler(func(ws_connection *websocket.Conn) {
		defer ws_connection.Close()
		for {
			// Write
			err := websocket.Message.Send(ws_connection, response)
			if err != nil {
				c.Logger().Error(err)
			}
		}
	}).ServeHTTP(c.Response(), c.Request())
	return nil
}

func main() {
	e := echo.New()

	// Middleware
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORS())

	e.GET("/ping", func(c echo.Context) error {
		response := "<p>I AM TESTING</p>"
		return c.String(http.StatusOK, response)
	})

	e.GET("/ws", ws)

	e.Logger.Fatal(e.Start(":4444"))
}
