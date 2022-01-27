FROM ubuntu:jammy

RUN apt update && apt install -y iproute2 tcpdump bridge-utils \
    iputils-ping curl iptables

COPY mconsole /usr/bin/mconsole
COPY entrypoint.sh /entrypoint.sh
