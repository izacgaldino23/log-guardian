FROM golang:1.25.4-alpine AS builder

WORKDIR /app

COPY go.mod .
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 go build -o /log-guardian ./src/cmd/main.go

FROM scratch AS final

COPY --from=builder /log-guardian /log-guardian

ENTRYPOINT ["/log-guardian"]