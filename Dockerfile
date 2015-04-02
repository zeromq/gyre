FROM scratch

MAINTAINER Armen Baghumian <armen@OpenSourceClub.org>

ENV ZSYS_INTERFACE eth0
ENV PATH .

COPY ./chat chat
COPY ./monitor monitor
COPY ./ping ping

CMD "chat"
