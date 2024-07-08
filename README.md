# Game of Life in HTMX and Go

Project being built during [Twitch streamings](https://www.twitch.tv/guilospanck) to learn how to use HTMX and Go together.

## Key techs and concepts

- [HTMX](https://htmx.org/)
- [HTML Canvas](https://developer.mozilla.org/en-US/docs/Web/API/Canvas_API)
- [Golang Echo web framework](https://echo.labstack.com/)
- [Gorilla Websockets](https://github.com/gorilla/websocket)

## Install

You only need to have [Go](https://go.dev/doc/install) installed or [Docker](https://www.docker.com/) installed.

## How to run

Change directory into the backend folder:
 
```sh
cd backend/
```

### Golang source

Start the `Go` backend

```sh
go run .
```

### Docker

Build the Docker image:

```sh
docker build -t {potato} .
```

Run the container in detached mode:

```sh
docker run -p 4444:4444 -d {potato}
```

### HTML/HTMX

Now open the `index.html` in your browser. That's it.
