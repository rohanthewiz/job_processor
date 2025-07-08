FROM golang:1.24-alpine3.21 AS buildstg
WORKDIR /root
ADD . .

RUN go env

RUN go build -o app

FROM alpine:3.21

# Create a non-root user
RUN addgroup -g 1001 jpro
RUN adduser -D -u 1001 -G jpro jpro

WORKDIR /home/jpro

COPY --from=buildstg /root/app /home/jpro/app
COPY --from=buildstg /root/cfg /home/jpro/cfg

RUN chown -R jpro:jpro /home/jpro/
USER jpro

EXPOSE 8000
ENTRYPOINT [ "./app" ]
