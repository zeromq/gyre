FROM scratch

MAINTAINER Armen Baghumian <armen@OpenSourceClub.org>

ENV ZSYS_INTERFACE eth0
ENV PATH .

ADD misc/release.tar.gz /

CMD ["/chat"]
