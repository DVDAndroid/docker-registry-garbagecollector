FROM golang:1.22-alpine as builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY main.go ./
RUN CGO_ENABLED=0 GOOS=linux go build -o docker-registry-garbagecollector

FROM scratch

COPY --from=builder /app/docker-registry-garbagecollector /app/docker-registry-garbagecollector

ENTRYPOINT ["/app/docker-registry-garbagecollector"]
