FROM golang:1.22.4-bullseye AS builder

WORKDIR /build

COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy the source code
COPY . .

RUN CGO_ENABLED=0 go build -o main ./cmd/example_many/main.go

FROM gcr.io/distroless/static:nonroot

WORKDIR /home/nonroot


COPY --from=builder /build/main ./main
CMD ["./main"]
