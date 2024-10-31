FROM alpine:latest

WORKDIR /root/

ARG BINARY
ARG TARGETOS
ARG TARGETARCH

COPY dist/${TARGETOS}-${TARGETARCH}/${BINARY} /root/${BINARY}

RUN chmod +x /root/${BINARY}

CMD ["./externalsecret-watcher"]
