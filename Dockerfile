FROM golang:alpine

COPY . /go/src/app
WORKDIR /go/src/app

RUN go build -v
RUN go install -v

EXPOSE 8000

CMD ["app"]
