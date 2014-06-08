GIT = $(shell git rev-parse --git-dir | xargs readlink -f)
ROOT = $(shell readlink -f ${GIT}/../)

zre-msg:
	gsl -script:misc/scripts/codec_go zre_msg.xml

gofmt-hook:
	cp ${ROOT}/misc/gofmt-hook/pre-commit ${GIT}/hooks/
	chmod +x ${GIT}/hooks/pre-commit
