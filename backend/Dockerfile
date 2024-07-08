#---- Building Stage
FROM golang:1.22.4-alpine as build

ARG USERNAME=go_backend_user
RUN addgroup -S ${USERNAME} && adduser -S ${USERNAME} -G ${USERNAME}
USER $USERNAME

WORKDIR /build
ENV GO111MODULE=on \
    CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64

COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build --ldflags "-s -w" -o exec main.go

#---- Image Stage
FROM scratch

WORKDIR /app
COPY --from=build /etc/passwd /etc/passwd
USER $USERNAME

COPY --from=build ./build/exec ./
EXPOSE 4444

CMD ["./exec"]
