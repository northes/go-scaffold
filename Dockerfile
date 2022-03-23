FROM golang:1.17-alpine3.13 as builder

RUN apk add make

WORKDIR /app/

COPY . .

RUN make download && make build

FROM alpine:3.14

ENV TZ=Asia/Shanghai
ENV ZONEINFO=/usr/local/go/lib/time/zoneinfo.zip

WORKDIR /app/

COPY --from=builder /usr/local/go/lib/time/zoneinfo.zip /usr/local/go/lib/time/zoneinfo.zip
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=builder /app/.air.conf.example /app/.air.conf
COPY --from=builder /app/etc/config.yaml.example /app/etc/config.yaml
COPY --from=builder /app/bin/app /app/bin/app

CMD ["./bin/app"]
