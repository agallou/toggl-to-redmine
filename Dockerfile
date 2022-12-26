FROM golang:1.4

RUN go get github.com/mattn/go-redmine

RUN go get github.com/roessland/gotoggl
