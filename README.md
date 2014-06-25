Gyre [![GoDoc](https://godoc.org/github.com/zeromq/gyre?status.png)](https://godoc.org/github.com/zeromq/gyre)
====

This is a Golang port of [Zyre](zyre.org) 2.0, an open-source framework for proximity-based
peer-to-peer applications, implementing the same [ZeroMQ Realtime Exchange Protocol](http://rfc.zeromq.org/spec:36).

## Description

Gyre does local area discovery and clustering. A Gyre node broadcasts
UDP beacons, and connects to peers that it finds. This class wraps a
Gyre node with a message-based API.

All incoming events are delivered via the recv call of a Gyre instance.
The first frame defines the type of the message, and following
frames provide further values:

    ENTER fromnode headers ipaddress
        a new peer has entered the network
    EXIT fromnode
        a peer has left the network
    JOIN fromnode groupname
        a peer has joined a specific group
    LEAVE fromnode groupname
        a peer has left a specific group
    WHISPER fromnode message
        a peer has sent this node a message
    SHOUT fromnode groupname message
        a peer has sent one of our groups a message

In SHOUT and WHISPER the message is a single frame in this version.
In ENTER, the headers frame contains a packed dictionary.

To join or leave a group, use the Join and Leave methods.
To set a header value, use the SetHeader method. To send a message
to a single peer, use Whisper method. To send a message to a group, use
Shout.

## Installation

### Install essential packages

    sudo apt-get install curl make git libtool build-essential dh-autoreconf pkg-config mercurial

OR

    sudo yum install curl make git libtool autoreconf pkgconfig mercurial file gcc gcc-c++

### Install golang

    curl -o /tmp/go.tar.gz https://go.googlecode.com/files/go1.2.linux-amd64.tar.gz
    sudo tar -C /usr/local -zxvf /tmp/go.tar.gz
    rm /tmp/go.tar.gz

OR

    sudo apt-get install golang

OR

    sudo yum install golang

Test the installation - You need at least version 1.2 example wont work with 1.0 and 1.1:

    go version
    
### Install zeromq

    curl -o /tmp/zeromq.tar.gz http://download.zeromq.org/zeromq-4.0.4.tar.gz
    sudo tar -C /tmp -zxvf /tmp/zeromq.tar.gz
    rm /tmp/zeromq.tar.gz
    cd /tmp/zeromq-4.0.4
    ./autogen.sh && ./configure && make && sudo make install
    export PKG_CONFIG_PATH=/tmp/zeromq-4.0.4/src/
    export LD_LIBRARY_PATH=/usr/local/lib/
    cd -

### Install Gyre

    mkdir ~/gopath
    export GOROOT=/usr/local/go # Ignore if installed with apt or yum
    export GOPATH=~/gopath
    export PATH=$GOROOT/bin:$GOPATH/bin:$PATH

    go get github.com/zeromq/gyre

## Example

Run following command in a terminal:

    cd $GOPATH/src/github.com/zeromq/gyre/
    go run examples/chat/chat.go -name yourname

Or

    git clone https://github.com/zeromq/gyre
    cd gyre
    go run examples/chat/chat.go -name yourname

And repeat the last command in another terminal or all the commands in another computer. Happy chatting!

## Api

View the API docs [![GoDoc](https://godoc.org/github.com/zeromq/gyre?status.png)](https://godoc.org/github.com/zeromq/gyre)

## Project Organization

Gyre is owned by all its authors and contributors. This is an open source
project licensed under the LGPLv3. To contribute to Gyre please read the
[C4.1 process](http://rfc.zeromq.org/spec:22) that we use.

To report an issue, use the [PYRE issue tracker](https://github.com/zeromq/gyre/issues) at github.com.
