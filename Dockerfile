FROM golang:alpine

COPY . /go/src/app
WORKDIR /go/src/app

RUN	apk add --no-cache git
RUN	go get -insecure goji.io	&& \
	go get gopkg.in/mgo.v2 		&& \
	go build -v 			&& \
	go install -v

EXPOSE 8000

CMD ["app"]
