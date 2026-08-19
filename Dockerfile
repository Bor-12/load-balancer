FROM golang:1.26-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o /out/loadbalancer ./cmd/loadbalancer
RUN go build -o /out/demo-backend ./cmd/demo-backend

FROM alpine:3.22

RUN addgroup -S loadbalancer && adduser -S loadbalancer -G loadbalancer

WORKDIR /app

COPY --from=builder /out/loadbalancer /usr/local/bin/loadbalancer
COPY --from=builder /out/demo-backend /usr/local/bin/demo-backend

USER loadbalancer

EXPOSE 8080

ENTRYPOINT ["loadbalancer"]
