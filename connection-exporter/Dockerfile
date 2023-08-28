FROM --platform=amd64 ubuntu:22.04
RUN apt-get update -y && apt-get upgrade -y 
RUN apt-get install -y gcc \
        nano \
        vim \
        curl \
        wget \
        g++ \
        libcairo2-dev \
        libjpeg-turbo8-dev \
        libpng-dev \
        libtool-bin \
        libossp-uuid-dev

RUN apt-get install -y libavcodec-dev \
        libavformat-dev \
        libavutil-dev \
        libswscale-dev \
        build-essential \
        libpango1.0-dev \
        libssh2-1-dev \
        libvncserver-dev \
        libtelnet-dev \
        libpulse-dev \
        libvorbis-dev \
        libwebp-dev

RUN apt-get -y install software-properties-common && \
        add-apt-repository -y ppa:remmina-ppa-team/remmina-next-daily && \
        apt-get update -y && \
        apt-get -y install freerdp2-dev freerdp2-x11

ARG guacamole_version=1.5.3
RUN wget https://archive.apache.org/dist/guacamole/${guacamole_version}/source/guacamole-server-$guacamole_version.tar.gz && \
        tar -xvzf guacamole-server-$guacamole_version.tar.gz && \
        cd guacamole-server-$guacamole_version/ && \
        ./configure --with-init-dir=/etc/init.d && \
        make -j$(nproc) && \
        make -j$(nproc) install && \
        ldconfig  -v

ARG golang_version=1.21.0
RUN wget https://go.dev/dl/go${golang_version}.linux-amd64.tar.gz && \
        rm -rf /usr/local/go && \
        tar -C /usr/local -xzf go${golang_version}.linux-amd64.tar.gz && \
        rm -f go${golang_version}.linux-amd64.tar.gz
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
