FROM golang:1.24 as builder

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o main cmd/ws/main.go

FROM scratch

WORKDIR /app

COPY --from=builder /app/main .

EXPOSE 3000

CMD ["./main"]