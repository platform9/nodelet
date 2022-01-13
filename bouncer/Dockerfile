FROM alpine:latest as certs
RUN apk --no-cache --update upgrade && apk --no-cache add ca-certificates

FROM scratch
COPY bouncerd /
COPY config /etc/config
COPY --from=certs /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
