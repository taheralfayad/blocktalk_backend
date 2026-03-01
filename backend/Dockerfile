FROM golang:1.24-alpine

RUN go env -w GOFLAGS=-buildvcs=false

WORKDIR /app

RUN apk add --no-cache curl git bash

RUN curl -sSfL https://raw.githubusercontent.com/cosmtrek/air/master/install.sh | sh -s -- -b /usr/local/bin

COPY . .

RUN go mod tidy && go build -buildvcs=false -o main .

CMD ["air"]
