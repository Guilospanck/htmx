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
	upgrader            = websocket.Upgrader{}
	mutCurrentGameState = make([]int, GAME_BOARD_SIZE)
	mu                  sync.Mutex
	isItRunning         bool = false
)

func getColorBasedOnCellAliveOrDead(state int) string {
	if state == 0 {
		return "black"
	} else {
		return "white"
	}
}

func drawBoard(gameState []int) string {
	paragraphs := ""
	for idx, value := range gameState {
		paragraphs += fmt.Sprintf(`<div id="cell-%d" style="background: %s; width: 15px; height: 15px" class="grid-item"></div>`, idx, getColorBasedOnCellAliveOrDead(value))
	}

	response := fmt.Sprintf(`
	<div id="game" hx-swap-oob="innerhtml">
		%s
	</div>
	`, paragraphs)

	return response
}

func getInitialGameState() []int {
	state := make([]int, GAME_BOARD_SIZE)

	for index := range GAME_BOARD_SIZE {
		state[index] = 0
	}

	return state
}

func getRandomGameState() []int {
	state := make([]int, GAME_BOARD_SIZE)
	for index := range GAME_BOARD_SIZE {
		state[index] = rand.Intn(2)
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

func getGlider() []int {
	state := getInitialGameState()

	state[170] = 1
	state[190] = 1
	state[210] = 1
	state[209] = 1
	state[188] = 1

	return state
}

func getNumberOfAliveNeighbours(cellIndex int, readOnlyCurrentState []int) int {
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

		neighbour := readOnlyCurrentState[value]
		aliveNeighbours += neighbour
	}

	return aliveNeighbours
}

func getCellStateBasedOnNeighbours(cellIndex int, readOnlyCurrentState []int) int {
	aliveNeighbours := getNumberOfAliveNeighbours(cellIndex, readOnlyCurrentState)
	currentStateOfCell := readOnlyCurrentState[cellIndex]

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

func resetBoard(c echo.Context, ws *websocket.Conn) {
	initialState := getGlider()
	setGameData(initialState)
	initialBoard := drawBoard(initialState)

	c.Logger().Info("Sending new data from reset...")
	err := ws.WriteMessage(websocket.TextMessage, []byte(initialBoard))
	if err != nil {
		c.Logger().Error(err)
	}
}
func setGameData(state []int) {
	mu.Lock()
	defer mu.Unlock()
	copy(mutCurrentGameState, state)
}

func getCurrentStateData() []int {
	mu.Lock()
	defer mu.Unlock()
	readOnlyCurrentState := make([]int, GAME_BOARD_SIZE)
	copy(readOnlyCurrentState, mutCurrentGameState)
	return readOnlyCurrentState
}

func updateCurrentGameState() {
	readOnlyCurrentState := getCurrentStateData()
	mutableCopy := getCurrentStateData()

	for index := range readOnlyCurrentState {
		mutableCopy[index] = getCellStateBasedOnNeighbours(index, readOnlyCurrentState)
	}

	setGameData(mutableCopy)
}

func runConwaysRulesAndReturnState(c echo.Context, stop chan int, newData chan string) {
	for {
		select {
		case <-stop:
			// 'break' only breaks from the innermost loop (in this case would be select)
			// 'return' breaks from all
			c.Logger().Info("Stopping...")
			isItRunning = false
			return
		default:
			updateCurrentGameState()
			currentGameState := getCurrentStateData()
			updatedData := drawBoard(currentGameState)

			newData <- updatedData
			time.Sleep(1 * time.Second)
		}
	}
}

// To test via CLI: wscat -H "Origin: http://localhost:4444" -c ws://localhost:4444/ws
func ws(c echo.Context, start <-chan int, stop chan int, reset <-chan int, newData chan string) error {
	upgrader.CheckOrigin = func(r *http.Request) bool { return true }
	ws, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		c.Logger().Error(err)
		return err
	}

	// defers are executed in LIFO fashion
	defer ws.Close()

	resetBoard(c, ws)

	for {
		select {
		case x := <-newData:
			c.Logger().Info("Sending new data...")
			err := ws.WriteMessage(websocket.TextMessage, []byte(x))
			if err != nil {
				c.Logger().Error(err)
			}
		case <-start:
			isItRunning = true
			c.Logger().Info("Starting...")
			go runConwaysRulesAndReturnState(c, stop, newData)
		case <-reset:
			c.Logger().Info("Resetting...")
			// ERROR: Here is where lies the error...
			if isItRunning {
				stop <- 1
			}
			resetBoard(c, ws)
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
