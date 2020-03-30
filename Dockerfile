FROM golang:1.14-alpine as builder

RUN apk update && apk add --no-cache git
WORKDIR $GOPATH/src/main/app
COPY go.mod go.sum ./
RUN go mod download
RUN go mod verify
COPY main.go .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -a -installsuffix cgo -o /go/bin/docker-keeper

FROM scratch

COPY --from=builder /go/bin/docker-keeper /docker-keeper
EXPOSE 8000
ENTRYPOINT ["/docker-keeper"]
