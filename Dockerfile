FROM golang:1.21-alpine AS builder
RUN apk add --no-cache git nodejs npm make gcc musl-dev
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
WORKDIR /app/frontend
RUN npm install
RUN npm run build
WORKDIR /app
RUN go install github.com/wailsapp/wails/v2/cmd/wails@latest
RUN /go/bin/wails build -clean -skipbindings -o tellonym-checker

FROM alpine:latest
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=builder /app/build/bin/tellonym-checker .
COPY --from=builder /app/configs ./configs
EXPOSE 8080
EXPOSE 2112
CMD ["./tellonym-checker"]
