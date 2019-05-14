# Build backend
FROM golang:1.12 as builder

ENV CGO_ENABLED=0

WORKDIR /temp

COPY . .

RUN cd /temp && \
	go test -mod=vendor ./... && \
	go build -o habr-bot -mod=vendor ./cmd/habrahabr-bot/main.go


FROM alpine

RUN apk update && apk upgrade && \
	apk add --no-cache ca-certificates tzdata

# Change timezone
ENV TZ Europe/Moscow

WORKDIR /app

COPY --from=builder /temp/habr-bot .

ENTRYPOINT [ "./habr-bot" ]