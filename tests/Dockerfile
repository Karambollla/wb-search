FROM golang:1.26

WORKDIR /tests
COPY go.mod go.sum* ./
RUN go mod download
COPY . .

ENTRYPOINT ["go", "test", "-race", "-v", "./..."]
