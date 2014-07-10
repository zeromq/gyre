FROM debian:7.4

MAINTAINER Armen Baghumian <armen@OpenSourceClub.org>

WORKDIR zeromq-4.0.4
RUN apt-get update -y
RUN apt-get install -y curl git mercurial file build-essential libtool autoconf pkg-config

RUN curl -o /tmp/go.tar.gz https://go.googlecode.com/files/go1.2.linux-amd64.tar.gz
RUN tar -C /usr/local -zxvf /tmp/go.tar.gz
RUN rm /tmp/go.tar.gz
RUN /usr/local/go/bin/go version

ENV GOROOT /usr/local/go
ENV GOPATH /var/local/gopath
ENV PATH $GOROOT/bin:$GOPATH/bin:$PATH

RUN curl -o /tmp/zeromq.tar.gz http://download.zeromq.org/zeromq-4.0.4.tar.gz
RUN tar -C /tmp -zxvf /tmp/zeromq.tar.gz
RUN rm /tmp/zeromq.tar.gz
WORKDIR /tmp/zeromq-4.0.4
RUN ./autogen.sh && ./configure && make && make install
RUN ldconfig

RUN mkdir -p $GOPATH/src
RUN mkdir -p $GOPATH/bin
RUN mkdir -p $GOPATH/pkg

RUN go get code.google.com/p/go.net/ipv4
RUN go get code.google.com/p/go.net/ipv6
RUN go get github.com/pebbe/zmq4

COPY . /var/local/gopath/src/github.com/zeromq/gyre/
WORKDIR /var/local/gopath/src/github.com/zeromq/gyre/
CMD go run cmd/monitor/monitor.go
