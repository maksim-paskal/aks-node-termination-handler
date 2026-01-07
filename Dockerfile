FROM alpine:latest@sha256:865b95f46d98cf867a156fe4a135ad3fe50d2056aa3f25ed31662dff6da4eb62

WORKDIR /app/

COPY ./aks-node-termination-handler /app/aks-node-termination-handler

RUN apk upgrade \
&& addgroup -g 30523 -S app \
&& adduser -u 30523 -D -S -G app app

USER 30523

ENTRYPOINT [ "/app/aks-node-termination-handler" ]