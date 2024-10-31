FROM alpine:latest

WORKDIR /root/

ARG BINARY
ARG TARGETOS
ARG TARGETARCH

RUN ls -lah dist/

#COPY dist/${TARGETOS}-${TARGETARCH}/${BINARY} /root/${BINARY}
#
#RUN chmod +x /root/${BINARY}
#
#CMD ["./externalsecret-watcher"]
