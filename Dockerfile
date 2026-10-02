# Go API, commands and mail/cleanup worker
FROM golang:1.25-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY cmd ./cmd
COPY internal ./internal
RUN CGO_ENABLED=0 go build -o /out/api ./cmd/api
RUN CGO_ENABLED=0 go build -o /out/worker ./cmd/worker
RUN CGO_ENABLED=0 go build -o /out/manage ./cmd/manage
RUN CGO_ENABLED=0 go build -o /out/migrate ./cmd/migrate
RUN CGO_ENABLED=0 go build -o /out/import ./cmd/import
FROM alpine:3.22
RUN apk add --no-cache ca-certificates tzdata && adduser -D -u 10001 sso
COPY --from=build /out/ /usr/local/bin/
WORKDIR /app
USER sso
EXPOSE 8080
CMD ["api"]
