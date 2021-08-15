FROM alpine:latest

WORKDIR /app/

COPY ./aks-node-termination-handler /app/aks-node-termination-handler
COPY config.yaml /app/config.yaml

RUN addgroup -g 101 -S app \
&& adduser -u 101 -D -S -G app app

USER 101

ENTRYPOINT [ "/app/aks-node-termination-handler" ]