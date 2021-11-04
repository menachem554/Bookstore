FROM golang:1.12-alpine

RUN apk add --no-cache git

# Set the Current Working Directory inside the container
WORKDIR /app/Bookstore

# We want to populate the module cache based on the go.{mod,sum} files.
COPY go.mod .
COPY go.sum .

RUN go mod download

COPY . .

# Build the Go app
RUN go build -o ./out/Bookstore .


# This container exposes port 9090 to the outside world
EXPOSE 9090

# Run the binary program produced by `go install`
CMD ["./out/Bookstore"]