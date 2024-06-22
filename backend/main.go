package main

import (
	"fmt"
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/labstack/gommon/log"
)

var (
	upgrader = websocket.Upgrader{}
)

func resetBoardGame() string {
	response := ""
	for i := range 20 * 20 {
		response += fmt.Sprintf(`<p id="cell-%d" style="background: black; width: 15px; height: 15px" class="grid-item"></p>`, i)
	}

	return response
}

func getDivStyle() string {
	return ""
	return `
	display: grid;
	gap: 2px;
	grid-template-columns: repeat(20, 15px);
	grid-template-rows: repeat(20, 15px);
	`
}

// To test via CLI: wscat -H "Origin: http://localhost:4444" -c ws://localhost:4444/ws
func ws(c echo.Context, start, stop chan int) error {
	upgrader.CheckOrigin = func(r *http.Request) bool { return true }
	ws, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		c.Logger().Error(err)
		return err
	}

	c.Logger().Info("CONNECTED....")

	// defers are executed in LIFO fashion
	defer ws.Close()
	defer c.Logger().Info("CLOSING....")

	reset := fmt.Sprintf(`
	<div id="game" style="%s" hx-swap-oob="innerhtml">
		%s
	</div>
	`, getDivStyle(), resetBoardGame())

	response := `
	<p id="cell-1" style="background: red; width: 15px; height: 15px" class="grid-item"></p>
	<p id="cell-3" style="background: red; width: 15px; height: 15px" class="grid-item"></p>
	<p id="cell-5" style="background: red; width: 15px; height: 15px" class="grid-item"></p>
	`

	for {
		select {
		case <-start:
			// Write
			err := ws.WriteMessage(websocket.TextMessage, []byte(response))
			if err != nil {
				c.Logger().Error(err)
			}
			c.Logger().Info("START")
		case <-stop:
			// Write
			err := ws.WriteMessage(websocket.TextMessage, []byte(reset))
			if err != nil {
				c.Logger().Error(err)
			}
			c.Logger().Info("STOP")
		}
	}

}

func main() {
	e := echo.New()

	// Middleware
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORS())

	e.Logger.SetLevel(log.INFO)

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
