FROM golang:1.24-alpine

WORKDIR /app

RUN apk add --no-cache curl git bash

RUN curl -sSfL https://raw.githubusercontent.com/cosmtrek/air/master/install.sh | sh -s -- -b /usr/local/bin

COPY . .

RUN go mod tidy && go build -o main .

CMD ["air"]