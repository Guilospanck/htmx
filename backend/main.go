package main

import (
	"fmt"
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

var (
	upgrader = websocket.Upgrader{}
)

// To test via CLI: wscat -H "Origin: http://localhost:4444" -c ws://localhost:4444/ws
func ws(c echo.Context) error {
	response := `
	<div id="game" hx-swap-oob="innerHTML">
		<p style="background: red; width: 10px; height: 10px"></p>
		<p style="background: black; width: 10px; height: 10px"></p>
		<p style="background: red; width: 10px; height: 10px"></p>
	</div>
	`

	ws, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		return err
	}
	defer ws.Close()

	for {
		// Write
		err := ws.WriteMessage(websocket.TextMessage, []byte(response))
		if err != nil {
			c.Logger().Error(err)
		}

		// Read
		_, msg, err := ws.ReadMessage()
		if err != nil {
			c.Logger().Error(err)
		}
		fmt.Printf("%s\n", msg)
	}
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
