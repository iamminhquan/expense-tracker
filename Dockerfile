FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o /out/server ./cmd/server

FROM alpine:3.19
WORKDIR /app
COPY --from=build /out/server ./server
COPY internal/database/migrations ./internal/database/migrations
COPY internal/web/templates ./internal/web/templates
EXPOSE 8080
CMD ["./server"]
