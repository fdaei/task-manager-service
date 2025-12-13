FROM golang:1.24-alpine AS builder

WORKDIR /app
RUN apk add --no-cache git

COPY go.mod ./
COPY go.sum ./

RUN go mod download

COPY . .

ENV CGO_ENABLED=0 GOOS=linux
RUN go build -o /out/taskservice ./cmd/taskservice

FROM alpine:3.19 AS runner
RUN apk add --no-cache ca-certificates
WORKDIR /app

COPY --from=builder /out/taskservice /app/taskservice

ENV DATABASE_URL="" \
    REDIS_HOST=""

EXPOSE 8080

ENTRYPOINT ["/app/taskservice"]
CMD ["serve"]

FROM golang:1.24-alpine AS dev

WORKDIR /app
RUN apk add --no-cache git

COPY go.mod ./
COPY go.sum ./

ARG GOPROXY=https://proxy.golang.org,direct
ARG GOSUMDB=sum.golang.org
ENV GOPROXY=${GOPROXY}
ENV GOSUMDB=${GOSUMDB}
RUN go mod download
RUN go install github.com/air-verse/air@v1.61.6

COPY . .

ENV GOCACHE=/tmp/.gocache
CMD ["air", "-c", ".air.toml"]
