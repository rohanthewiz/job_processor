
FROM golang:1.24-bookworm AS builder
WORKDIR /root
ADD . .

# Install libpcap development files
RUN apt-get update

RUN go env

RUN go mod tidy

RUN CGO_ENABLED=1 GOOS=linux go build -o app

FROM ubuntu:24.04

#ARG ENV_NAME
#ENV APP_ENV=$ENV_NAME

RUN  apt-get -y update  &&  apt-get -y install ca-certificates

#RUN mkdir -p /etc/pki/tls/certs
#RUN ln -s /etc/ssl/certs/ca-certificates.crt /etc/pki/tls/certs/ca-bundle.crt

# Create a non-root user
RUN groupadd appuser -g 1001 && useradd -u 1001 -g 1001 -m -d /home/appuser appuser

WORKDIR /home/appuser

COPY --from=builder /root/app /home/appuser/app
RUN chmod +x /home/appuser/app

EXPOSE 8000
ENTRYPOINT [ "./app" ]
