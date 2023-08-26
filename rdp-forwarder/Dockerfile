FROM golang:1.21-bookworm

RUN apt-get update -y && \
    apt-get upgrade -y && \
    apt-get autoremove -y

# RUN apt-get install -y build-essential wget net-tools lsof
# RUN apt-get install -y freerdp2-dev libavcodec-dev libavformat-dev libavutil-dev \
#     libswscale-dev libcairo2-dev libjpeg62-turbo-dev libjpeg-dev libpng-dev libtool-bin \
#     libpango1.0-dev libpango1.0-0 libssh2-1 libwebsockets17 libwebsocketpp-dev libossp-uuid-dev \
#     libssl-dev libwebp-dev libvorbis-dev libpulse-dev libwebsockets-dev libvncserver-dev \
#     libssh2-1-dev openssl
# RUN wget https://dlcdn.apache.org/guacamole/1.5.3/source/guacamole-server-1.5.3.tar.gz && \
#     tar -xvzf guacamole-server-1.5.3.tar.gz && \
#     cd guacamole-server-*/ && \
#     export CFLAGS="-Wno-error" && \
#     ./configure --with-systemd-dir=/etc/systemd/system/ && \
#     make -j $(nproc) && \
#     make install && \
#     ldconfig -v

WORKDIR /root/go/src/github.com/sklrsn/video-convertor/connection-forwarder
ADD . .

RUN go mod download && \ 
    go mod vendor -v

RUN go build -o /opt/forwarder/connection-forwarder *.go && \
    chmod +x /opt/forwarder/connection-forwarder

RUN mkdir -p /opt/forwarder && \
    cp -f entrypoint.sh /opt/forwarder/entrypoint.sh && \
    chmod +x /opt/forwarder/entrypoint.sh

ENTRYPOINT [ "/opt/forwarder/entrypoint.sh" ]