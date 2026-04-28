FROM golang:1.25-alpine AS builder

ENV GOPROXY=https://goproxy.cn,direct

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -buildvcs=false -o /finance-invoice .

FROM alpine:3.21

RUN sed -i 's#dl-cdn.alpinelinux.org#mirrors.aliyun.com#g' /etc/apk/repositories && \
    apk add --no-cache ca-certificates tzdata curl && \
    sed -i 's#https://mirrors.aliyun.com#http://mirrors.cloud.aliyuncs.com#g' /etc/apk/repositories  && \
    echo -e '\n\n# septnet CA' >> /etc/ssl/certs/ca-certificates.crt && curl 'https://7netpublic.oss-cn-hangzhou.aliyuncs.com/dev/ca/septnet-ca.crt' >> /etc/ssl/certs/ca-certificates.crt

ENV TZ=Asia/Shanghai

WORKDIR /app
COPY --from=builder /finance-invoice .
COPY web/ web/
COPY users.csv .

EXPOSE 8080

ENTRYPOINT ["./finance-invoice"]
