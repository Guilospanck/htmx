package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/labstack/gommon/log"
)

const CELL_SIZE int = 15
const BOARD_COLUMNS int = 50
const GAME_BOARD_SIZE int = BOARD_COLUMNS * BOARD_COLUMNS

var (
	upgrader            = websocket.Upgrader{}
	mutCurrentGameState = make([]int, GAME_BOARD_SIZE)
	mu                  sync.Mutex
	isItRunning         atomic.Bool
	gameSpeed           atomic.Uint32
)

func getColorBasedOnCellAliveOrDead(state int) string {
	if state == 0 {
		return "black"
	} else {
		return "white"
	}
}

func drawBoard(gameState []int) string {
	marshalledGameState, _ := json.Marshal(gameState)

	response := fmt.Sprintf(`
		<script id="htmx_pls_here" hx-swap-oob="outerHTML">
			var NUMBER_OF_COLUMNS = 50;
			var NUMBER_OF_ROWS = 50;
			var CELL_SIZE = 15;
			var canvas = document.getElementById("game");
			var ctx = canvas.getContext("2d");

			ctx.lineWidth = "0.1";

			var data = %s
			var isItRunning = %t
			var CELL_COLOR = 'green';

			var i = 0;
			var j = 0;

			for(const strIndex in data) {
				let index = parseInt(strIndex);

				if(index %% NUMBER_OF_COLUMNS === 0 && index !== 0){
					// go down
					i = 0;
					j++;
				}

				let rowStep = i * CELL_SIZE;
				let columnStep = j * CELL_SIZE;

				const getColor = () => {
					if(data[index] === 1){
						return CELL_COLOR
					} else if(data[index] === 2){
						return 'red'
					} else if(data[index] === 3){
						return 'blue'
					} else {
						return 'black'
					}
				}

				let color = getColor()

				ctx.beginPath();
				ctx.fillStyle = color;
				ctx.strokeRect(rowStep, columnStep, CELL_SIZE, CELL_SIZE);
				ctx.fillRect(rowStep, columnStep, CELL_SIZE, CELL_SIZE);
				ctx.stroke();

				i++;
			}
		</script>
	`, string(marshalledGameState), isItRunning.Load())

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

	state[825] = 1
	state[875] = 1
	state[925] = 1
	state[924] = 1
	state[873] = 1

	return state
}

func getCrossSign() []int {
	state := getInitialGameState()

	halfRow := BOARD_COLUMNS / 2
	for i := halfRow; i <= GAME_BOARD_SIZE-halfRow; i += BOARD_COLUMNS {
		state[i] = 1
	}

	initialRange := GAME_BOARD_SIZE / 2
	for j := initialRange; j <= initialRange+BOARD_COLUMNS-1; j++ {
		state[j] = 1
	}

	return state
}

func getGosperGliderGun() []int {
	state := getInitialGameState()

	state[875] = 1
	state[923] = 1
	state[925] = 1
	state[963] = 1
	state[964] = 1
	state[971] = 1
	state[972] = 1
	state[985] = 1
	state[986] = 1
	state[1012] = 1
	state[1016] = 1
	state[1021] = 1
	state[1022] = 1
	state[1035] = 1
	state[1036] = 1
	state[1051] = 1
	state[1052] = 1
	state[1061] = 1
	state[1067] = 1
	state[1071] = 1
	state[1072] = 1
	state[1101] = 1
	state[1102] = 1
	state[1111] = 1
	state[1115] = 1
	state[1117] = 1
	state[1118] = 1
	state[1123] = 1
	state[1125] = 1
	state[1161] = 1
	state[1167] = 1
	state[1175] = 1
	state[1212] = 1
	state[1216] = 1
	state[1263] = 1
	state[1264] = 1

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

	c.Logger().Warn("Sending new data from reset...")
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

// f(x) = -999/99000*(slider - 1) + 1
// This equation maps from slider values (1 to 100)
// into duration of sleep (1s to 1ms)
func calculateSleep() time.Duration {
	rangeSpeed := gameSpeed.Load()

	fx := ((float64(rangeSpeed)-1)*(1/1000-1) + 99) * 1 / 99

	if rangeSpeed > 1 {
		return time.Duration(int64(fx*1000) * int64(time.Millisecond))
	}

	duration := time.Duration(int64(fx) * int64(time.Second))
	return duration
}

func sendUpdatedData(newData chan string) {
	updateCurrentGameState()
	currentGameState := getCurrentStateData()
	updatedData := drawBoard(currentGameState)

	newData <- updatedData
}

func runConwaysRulesAndReturnState(newData chan string) {
	for {
		sendUpdatedData(newData)
		if !isItRunning.Load() {
			return
		}
		time.Sleep(calculateSleep())
	}
}

// To test via CLI: wscat -H "Origin: http://localhost:4444" -c ws://localhost:4444/ws
func ws(c echo.Context, start <-chan int, reset <-chan int, newData chan string) error {
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
		case <-start:
			c.Logger().Warn("Starting...")
			isItRunning.Store(true)
			go runConwaysRulesAndReturnState(newData)
		case <-reset:
			c.Logger().Warn("Resetting...")
			isItRunning.Store(false)
			resetBoard(c, ws)
		case x := <-newData:
			c.Logger().Warn("Sending new data...")
			err := ws.WriteMessage(websocket.TextMessage, []byte(x))
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
	// e.Use(middleware.Recover())
	e.Use(middleware.CORS())

	e.Logger.SetLevel(log.INFO)

	start := make(chan int)
	reset := make(chan int)
	newData := make(chan string)

	gameSpeed.Store(1)

	e.GET("/start", func(c echo.Context) error {
		if isItRunning.Load() {
			return c.NoContent(http.StatusNoContent)
		}
		start <- 1
		return c.NoContent(http.StatusNoContent)
	})
	e.GET("/stop", func(c echo.Context) error {
		isItRunning.Store(false)
		return c.NoContent(http.StatusNoContent)
	})
	e.GET("/reset", func(c echo.Context) error {
		reset <- 1
		return c.NoContent(http.StatusNoContent)
	})
	e.GET("/ws", func(c echo.Context) error {
		return ws(c, start, reset, newData)
	})
	e.POST("/speed", func(c echo.Context) (err error) {
		type GameSpeed struct {
			Speed string `json:"speed" form:"speed" query:"speed"`
		}

		u := new(GameSpeed)
		if err = c.Bind(u); err != nil {
			return err
		}

		intSpeed, err := strconv.Atoi(u.Speed)
		if err != nil {
			c.Logger().Errorf("Could not convert to integer: %s", err.Error())
			return err
		}

		gameSpeed.Store(uint32(intSpeed))

		return c.NoContent(http.StatusOK)
	})
	e.POST("/update", func(c echo.Context) (err error) {
		type UpdatedData struct {
			Index int `json:"index" form:"index" query:"index"`
			Value int `json:"value" form:"value" query:"value"`
		}

		u := new(UpdatedData)
		if err = c.Bind(u); err != nil {
			return err
		}

		index := u.Index
		value := u.Value
		oldData := getCurrentStateData()
		oldData[index] = value
		setGameData(oldData)

		return c.NoContent(http.StatusOK)
	})

	e.POST("/pattern", func(c echo.Context) (err error) {
		type RequestFormData struct {
			Pattern string `json:"pattern" form:"pattern" query:"pattern"`
		}

		u := new(RequestFormData)
		if err = c.Bind(u); err != nil {
			return err
		}

		var newPattern []int

		switch u.Pattern {
		case "blinker":
			newPattern = getBlinker()
		case "glider":
			newPattern = getGlider()
		case "cross":
			newPattern = getCrossSign()
		case "gosper_gun":
			newPattern = getGosperGliderGun()
		default:
			newPattern = getBlinker()
		}

		setGameData(newPattern)
		sendUpdatedData(newData)

		return c.NoContent(http.StatusOK)
	})

	e.Logger.Fatal(e.Start(":4444"))
}
