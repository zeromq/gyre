Gyre
====

This is a Golang port of [Zyre](zyre.org) 1.0, implementing the same [ZRE protocol](http://rfc.zeromq.org/spec:20).

# Gyre - an open-source framework for proximity-based peer-to-peer applications

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

In SHOUT and WHISPER the message is a single frame in this version
. In ENTER, the headers frame contains a packed dictionary.

To join or leave a group, use the Join and Leave methods.
To set a header value, use the SetHeader method. To send a message
to a single peer, use Whisper method. To send a message to a group, use
Shout.

## Installation

    go get github.com/armen/gyre

## Api

View the API docs [here](http://godoc.org/github.com/armen/gyre).

## Example

Run following command in a terminal:

    git clone https://github.com/armen/gyre
    cd gyre
    go run examples/chat.go -name yourname

And repeat the run command in another terminal or all the commands in another computer. Happy chatting!
