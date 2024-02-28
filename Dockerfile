FROM golang:1.20

WORKDIR /opt/app

COPY . .

RUN go build ./proxy.go

EXPOSE 8080

CMD ["./proxy"]