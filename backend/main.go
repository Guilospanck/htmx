package main

import (
	"fmt"
	"math/rand"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/labstack/gommon/log"
)

const BOARD_COLUMNS = 20
const GAME_BOARD_SIZE = BOARD_COLUMNS * BOARD_COLUMNS

var (
	upgrader         = websocket.Upgrader{}
	currentGameState = make([]int, 0, GAME_BOARD_SIZE)
)

func getColorBasedOnCellAliveOrDead(state int) string {
	if state == 0 {
		return "black"
	} else {
		return "white"
	}
}

func drawBoard() string {
	paragraphs := ""
	for idx, value := range currentGameState {
		paragraphs += fmt.Sprintf(`<p id="cell-%d" style="background: %s; width: 15px; height: 15px" class="grid-item"></p>`, idx, getColorBasedOnCellAliveOrDead(value))
	}

	response := fmt.Sprintf(`
	<div id="game" hx-swap-oob="innerhtml">
		%s
	</div>
	`, paragraphs)

	return response
}

func resetBoardGame() string {
	return drawBoard()
}

func getInitialGameState() []int {
	state := make([]int, 0, GAME_BOARD_SIZE)

	for range GAME_BOARD_SIZE {
		state = append(state, 0)
	}

	return state
}

func getRandomGameState() []int {
	state := make([]int, 0, GAME_BOARD_SIZE)
	for range GAME_BOARD_SIZE {
		state = append(state, rand.Intn(2))
	}

	return state
}

func getBlinker() []int {
	state := getInitialGameState()

	state[201] = 1
	state[202] = 1
	state[203] = 1

	return state
}

func getNumberOfAliveNeighbors(index int) int {
	topIndex := index - BOARD_COLUMNS
	bottomIndex := index + BOARD_COLUMNS
	rightIndex := index + 1
	leftIndex := index - 1
	topRightIndex := index - BOARD_COLUMNS + 1
	topLeftIndex := index - BOARD_COLUMNS - 1
	bottomRightIndex := index + BOARD_COLUMNS + 1
	bottomLeftIndex := index + BOARD_COLUMNS - 1

	neighbours := []int{topIndex, bottomIndex, rightIndex, leftIndex, topRightIndex, topLeftIndex, bottomLeftIndex, bottomRightIndex}

	aliveNeighbours := 0
	for _, value := range neighbours {
		if value < 0 || value > GAME_BOARD_SIZE-1 {
			continue
		}

		neighbour := currentGameState[value]
		aliveNeighbours += neighbour
	}

	return aliveNeighbours
}

func getCellStateBasedOnNeighbours(cellIndex int) int {
	aliveNeighbours := getNumberOfAliveNeighbors(cellIndex)
	currentStateOfCell := currentGameState[cellIndex]

	switch currentStateOfCell {
	case 1:
		if aliveNeighbours < 2 || aliveNeighbours > 3 {
			currentStateOfCell = 0
		}
	case 0:
		if aliveNeighbours == 3 {
			currentStateOfCell = 1
		}
	}

	return currentStateOfCell
}

func updateCurrentGameState() {
	for index := range currentGameState {
		currentGameState[index] = getCellStateBasedOnNeighbours(index)
	}
}

func runConwaysRulesAndReturnState(quit chan int, newData chan string) {
	for {
		select {
		case <-quit:
			break
		default:
			updateCurrentGameState()
			newData <- drawBoard()
			time.Sleep(2 * time.Second)
		}
	}
}

// To test via CLI: wscat -H "Origin: http://localhost:4444" -c ws://localhost:4444/ws
func ws(c echo.Context, start, stop chan int, newData chan string) error {
	initialState := getBlinker()
	currentGameState = initialState

	upgrader.CheckOrigin = func(r *http.Request) bool { return true }
	ws, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		c.Logger().Error(err)
		return err
	}

	// defers are executed in LIFO fashion
	defer ws.Close()
	defer c.Logger().Info("CLOSING....")

	reset := fmt.Sprintf(`
	<div id="game" hx-swap-oob="innerhtml">
		%s
	</div>
	`, resetBoardGame())

	for {
		select {
		case <-newData:
			// Write
			err := ws.WriteMessage(websocket.TextMessage, []byte(<-newData))
			if err != nil {
				c.Logger().Error(err)
			}
		case <-start:
			go runConwaysRulesAndReturnState(stop, newData)
		case <-stop:
			// Write
			err := ws.WriteMessage(websocket.TextMessage, []byte(reset))
			if err != nil {
				c.Logger().Error(err)
			}
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
	newData := make(chan string)

	e.GET("/start", func(c echo.Context) error {
		start <- 1
		return c.NoContent(http.StatusNoContent)
	})
	e.GET("/stop", func(c echo.Context) error {
		stop <- 1
		return c.NoContent(http.StatusNoContent)
	})

	e.GET("/ws", func(c echo.Context) error {
		return ws(c, start, stop, newData)
	})

	e.Logger.Fatal(e.Start(":4444"))
}
