FROM scratch

MAINTAINER Armen Baghumian <armen@OpenSourceClub.org>

ENV ZSYS_INTERFACE eth0
ENV PATH .

ADD https://github.com/armen/gyre/releases/download/v1.0.0/release.zip .

CMD "chat"
