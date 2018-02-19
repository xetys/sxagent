FROM golang:1.9.2 as builder

WORKDIR /go/src/app

COPY . .

RUN go get -d -v ./...

#disable crosscompiling
ENV CGO_ENABLED=0

#compile linux only
ENV GOOS=linux

RUN go build -ldflags '-w -s' -a -installsuffix cgo -o sxagent

FROM scratch
COPY --from=builder /go/src/app/sxagent /sxagent

CMD ["/sxagent"]
EXPOSE 8080