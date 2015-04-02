Gyre [![GoDoc](https://godoc.org/github.com/zeromq/gyre?status.png)](https://godoc.org/github.com/zeromq/gyre) [![Build Status](https://travis-ci.org/zeromq/gyre.svg?branch=master)](https://travis-ci.org/zeromq/gyre)
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

## Example (docker)

Run following command in a terminal:

	docker run --rm -i --tty -t armen/gyre chat -name yourname

And repeat the above command in another terminal, the chat instances will discover eachother. Happy chatting!

## Api

View the API docs [![GoDoc](https://godoc.org/github.com/zeromq/gyre?status.png)](https://godoc.org/github.com/zeromq/gyre)

## Project Organization

Gyre is owned by all its authors and contributors. This is an open source
project licensed under the LGPLv3. To contribute to Gyre please read the
[C4.1 process](http://rfc.zeromq.org/spec:22) that we use.

To report an issue, use the [Gyre issue tracker](https://github.com/zeromq/gyre/issues) at github.com.
