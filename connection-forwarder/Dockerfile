FROM golang:1.21-bookworm

WORKDIR /root/go/src/github.com/sklrsn/video-convertor/connection-forwarder
ADD . .

RUN go mod download && \ 
    go mod vendor -v

RUN go build -o /usr/local/bin/connection-forwarder && \
    chmod +x /usr/local/bin/connection-forwarder

ENTRYPOINT [ "/usr/local/bin/connection-forwarder" ]