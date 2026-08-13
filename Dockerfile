FROM golang:1.26-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o /out/cloudbalancer ./cmd/cloudbalancer
RUN go build -o /out/demo-backend ./cmd/demo-backend

FROM alpine:3.22

RUN addgroup -S cloudbalancer && adduser -S cloudbalancer -G cloudbalancer

WORKDIR /app

COPY --from=builder /out/cloudbalancer /usr/local/bin/cloudbalancer
COPY --from=builder /out/demo-backend /usr/local/bin/demo-backend
COPY configs/config.docker.yaml /app/configs/config.docker.yaml

USER cloudbalancer

EXPOSE 8080

ENTRYPOINT ["cloudbalancer"]
