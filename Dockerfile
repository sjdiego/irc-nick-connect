FROM golang:1.17.5-bullseye

RUN mkdir /app

ADD . /app

WORKDIR /app

RUN go mod download

RUN go build -o main .

CMD ["/app/main"]
