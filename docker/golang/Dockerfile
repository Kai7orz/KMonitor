FROM golang:1.25 AS builder
WORKDIR /src/app
COPY ./backend/golang/src/go.mod .
COPY  ./backend/golang/src/go.sum .
RUN go mod download
COPY ./backend/golang/src/ .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o server .

# ---- Runner ----
FROM gcr.io/distroless/base-debian12
WORKDIR /src/app

COPY ./backend/golang/src/.env .
ENV PORT=8080
COPY --from=builder /src/app .
EXPOSE 8080
CMD ["./server"]

