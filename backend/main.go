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
func ws(c echo.Context, channel chan string) error {
	upgrader.CheckOrigin = func(r *http.Request) bool { return true }
	ws, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		return err
	}
	defer ws.Close()

	for response := range channel {
		// Write
		err := ws.WriteMessage(websocket.TextMessage, []byte(response))
		if err != nil {
			c.Logger().Error(err)
		}
	}

	return nil
}

func main() {
	e := echo.New()

	// Middleware
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORS())

	my_channel := make(chan string)

	e.GET("/ping", func(c echo.Context) error {

		response := `
		<div id="game" hx-swap-oob="innerHTML">
		<p style="background: red; width: 10px; height: 10px"></p>
		<p style="background: black; width: 10px; height: 10px"></p>
		<p style="background: red; width: 10px; height: 10px"></p>
		</div>
		`

		my_channel <- response
		return c.NoContent(http.StatusNoContent)
	})

	e.GET("/ws", func(c echo.Context) error {
		return ws(c, my_channel)
	})

	e.Logger.Fatal(e.Start(":4444"))
}
