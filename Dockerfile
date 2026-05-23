FROM golang:1.26-alpine AS build

WORKDIR /src
COPY go.mod go.sum* ./
RUN go mod download
COPY . .
RUN go build -o /out/wb-search ./cmd/server

FROM alpine:3.22

RUN adduser -D -H app
USER app
COPY --from=build /out/wb-search /usr/local/bin/wb-search
EXPOSE 8080
EXPOSE 9090
ENTRYPOINT ["wb-search"]
