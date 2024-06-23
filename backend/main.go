package main

import (
	"fmt"
	"math/rand"
	"net/http"
	"sync"
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
	mu               sync.Mutex
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
	<div id="game"
		hx-swap-oob="innerhtml"
		style="
			display: grid;
			gap: 2px;
			grid-template-columns: repeat(20, 15px);
			grid-template-rows: repeat(20, 15px);
		"
	>
		%s
	</div>
	`, paragraphs)

	return response
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

	state[208] = 1
	state[209] = 1
	state[210] = 1

	return state
}

func resetBoardGame() string {
	currentGameState = getInitialGameState()
	return drawBoard()
}

func getNumberOfAliveNeighbours(cellIndex int) int {
	topIndex := cellIndex - BOARD_COLUMNS
	bottomIndex := cellIndex + BOARD_COLUMNS
	rightIndex := cellIndex + 1
	leftIndex := cellIndex - 1
	topRightIndex := cellIndex - BOARD_COLUMNS + 1
	topLeftIndex := cellIndex - BOARD_COLUMNS - 1
	bottomRightIndex := cellIndex + BOARD_COLUMNS + 1
	bottomLeftIndex := cellIndex + BOARD_COLUMNS - 1

	neighbours := []int{topIndex, bottomIndex, rightIndex, leftIndex, topRightIndex, topLeftIndex, bottomLeftIndex, bottomRightIndex}

	aliveNeighbours := 0
	for _, value := range neighbours {
		if value < 0 || value >= GAME_BOARD_SIZE {
			continue
		}

		neighbour := currentGameState[value]
		aliveNeighbours += neighbour
	}

	return aliveNeighbours
}

func getCellStateBasedOnNeighbours(cellIndex int) int {
	aliveNeighbours := getNumberOfAliveNeighbours(cellIndex)
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

func setInitialState() {
	initialState := getBlinker()
	currentGameState = initialState
}

func runConwaysRulesAndReturnState(stop chan int, newData chan string) {
	mu.Lock()
	setInitialState()
	mu.Unlock()

	for {
		select {
		case <-stop:
			// break only breaks from the innermost loop (in this case would be select)
			// return breaks from all
			return
		default:
			mu.Lock()
			updateCurrentGameState()
			updatedData := drawBoard()
			mu.Unlock()

			newData <- updatedData
			time.Sleep(1 * time.Second)
		}
	}
}

// To test via CLI: wscat -H "Origin: http://localhost:4444" -c ws://localhost:4444/ws
func ws(c echo.Context, start, stop, reset chan int, newData chan string) error {
	upgrader.CheckOrigin = func(r *http.Request) bool { return true }
	ws, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		c.Logger().Error(err)
		return err
	}

	// defers are executed in LIFO fashion
	defer ws.Close()
	defer c.Logger().Info("CLOSING....")

	for {
		select {
		case <-newData:
			howManyAlive := 0
			for _, value := range currentGameState {
				if value == 1 {
					howManyAlive += 1
				}
			}
			c.Logger().Info(currentGameState)
			c.Logger().Info(howManyAlive)

			// Write
			err := ws.WriteMessage(websocket.TextMessage, []byte(<-newData))
			if err != nil {
				c.Logger().Error(err)
			}
		case <-start:
			go runConwaysRulesAndReturnState(stop, newData)
		case <-reset:
			stop <- 1
			mu.Lock()
			reset := resetBoardGame()
			mu.Unlock()

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
	reset := make(chan int)
	newData := make(chan string)

	e.GET("/start", func(c echo.Context) error {
		start <- 1
		return c.NoContent(http.StatusNoContent)
	})
	e.GET("/stop", func(c echo.Context) error {
		stop <- 1
		return c.NoContent(http.StatusNoContent)
	})
	e.GET("/reset", func(c echo.Context) error {
		reset <- 1
		return c.NoContent(http.StatusNoContent)
	})

	e.GET("/ws", func(c echo.Context) error {
		return ws(c, start, stop, reset, newData)
	})

	e.Logger.Fatal(e.Start(":4444"))
}
