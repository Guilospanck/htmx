package main

import (
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

var (
	upgrader = websocket.Upgrader{}
)

// To test via CLI: wscat -H "Origin: http://localhost:4444" -c ws://localhost:4444/ws
func ws(c echo.Context, start, stop chan int) error {
	upgrader.CheckOrigin = func(r *http.Request) bool { return true }
	ws, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		c.Logger().Error(err)
		return err
	}
	defer ws.Close()
	c.Logger().Info("\n> CONNECTED to ws \n")

	response := `<div id="game" hx-swap-oob="innerhtml">
	<p style="background: red; width: 10px; height: 10px"></p>
	<p style="background: black; width: 10px; height: 10px"></p>
	<p style="background: red; width: 10px; height: 10px"></p>
	</div>
	`

	reset := `<div id="game" hx-swap-oob="innerhtml">
	</div>
	`

	for {
		select {
		case <-start:
			// Write
			err := ws.WriteMessage(websocket.TextMessage, []byte(response))
			if err != nil {
				c.Logger().Error(err)
			}
		case <-stop:
			// Write
			err := ws.WriteMessage(websocket.TextMessage, []byte(reset))
			if err != nil {
				c.Logger().Error(err)
			}
			break
		}
	}

}

func main() {
	e := echo.New()

	// Middleware
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORS())

	start := make(chan int)
	stop := make(chan int)

	e.GET("/start", func(c echo.Context) error {
		start <- 1
		return c.NoContent(http.StatusNoContent)
	})
	e.GET("/stop", func(c echo.Context) error {
		stop <- 1
		return c.NoContent(http.StatusNoContent)
	})

	e.GET("/ws", func(c echo.Context) error {
		return ws(c, start, stop)
	})

	e.Logger.Fatal(e.Start(":4444"))
}
