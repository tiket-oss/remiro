FROM golang:1.12

ENV GO111MODULE=on

WORKDIR /app

COPY . .

RUN go build -o remiro 

ENTRYPOINT ["/app/remiro"]