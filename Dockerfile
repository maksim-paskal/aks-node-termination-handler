FROM alpine:latest

WORKDIR /app/

COPY ./aks-node-termination-handler /app/aks-node-termination-handler

RUN apk upgrade \
&& addgroup -g 30523 -S app \
&& adduser -u 30523 -D -S -G app app

USER 30523

ENTRYPOINT [ "/app/aks-node-termination-handler" ]