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

    curl -o /tmp/zeromq.tar.gz http://download.zeromq.org/zeromq-4.1.4.tar.gz
    sudo tar -C /tmp -zxvf /tmp/zeromq.tar.gz
    rm /tmp/zeromq.tar.gz
    cd /tmp/zeromq-4.1.4
    ./autogen.sh && ./configure && make && sudo make install
    sudo ldconfig
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

And repeat the last command in another terminal or all the commands in another computer.
