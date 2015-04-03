FROM scratch

MAINTAINER Armen Baghumian <armen@OpenSourceClub.org>

ENV ZSYS_INTERFACE eth0
ENV PATH .

ADD misc/bins-linux-x86_64.tar.gz /

CMD ["/chat"]
