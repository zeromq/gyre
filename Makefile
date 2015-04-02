GIT = $(shell git rev-parse --git-dir | xargs readlink -f)
ROOT = $(shell readlink -f ${GIT}/../)

docker-image:
	go build -a -ldflags '-extldflags "-lm -lstdc++ -lsodium -static"' github.com/zeromq/gyre/examples/chat 2>/dev/null
	go build -a -ldflags '-extldflags "-lm -lstdc++ -lsodium -static"' github.com/zeromq/gyre/examples/ping 2>/dev/null
	go build -a -ldflags '-extldflags "-lm -lstdc++ -lsodium -static"' github.com/zeromq/gyre/cmd/monitor 2>/dev/null
	docker build -t armen/gyre .
	zip release.zip chat ping monitor
	rm chat ping monitor

zre-msg:
	gsl -script:zproto_codec_go zre_msg.xml

gofmt-hook:
	cp ${ROOT}/misc/gofmt-hook/pre-commit ${GIT}/hooks/
	chmod +x ${GIT}/hooks/pre-commit
