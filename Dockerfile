FROM alpine:latest

WORKDIR /root/

ARG BINARY
ARG TARGETOS
ARG TARGETARCH

COPY dist/${BINARY}-${TARGETOS}-${TARGETARCH}/ /root/${BINARY}

RUN chmod +x /root/${BINARY}

CMD ["./externalsecret-watcher"]
