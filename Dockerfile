FROM golang:alpine as builder
WORKDIR /go/src/github.com/mehrdadrad/tcpprobe
COPY . .
RUN CGO_ENABLED=0 go build

FROM alpine 

COPY --from=builder /go/src/github.com/mehrdadrad/tcpprobe/tcpprobe /usr/bin/

EXPOSE 8081

ENTRYPOINT ["tcpprobe"]
