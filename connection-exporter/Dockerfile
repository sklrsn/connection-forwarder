FROM --platform=amd64 jrottenberg/ffmpeg:6.0-ubuntu
RUN apt-get install -y wget

ENV GO_VERSION=1.21.0
RUN wget https://go.dev/dl/go${GO_VERSION}.linux-amd64.tar.gz && \
        rm -rf /usr/local/go && \
        tar -C /usr/local -xzf go${GO_VERSION}.linux-amd64.tar.gz && \
        rm -f go${GO_VERSION}.linux-amd64.tar.gz
ENV PATH="$PATH:/usr/local/go/bin"

WORKDIR /root/go/src/github.com/sklrsn/video-convertor/connection-exporter
ADD . .

RUN go mod download && \ 
        go mod vendor -v

RUN go build -o /opt/exporter/connection-exporter && \
        chmod +x /opt/exporter/connection-exporter

RUN mkdir -p /opt/exporter && \
        cp -f entrypoint.sh /opt/exporter/entrypoint.sh && \
        chmod +x /opt/exporter/entrypoint.sh

ENTRYPOINT [ "/opt/exporter/entrypoint.sh" ]
