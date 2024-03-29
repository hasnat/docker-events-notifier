FROM golang:1.15 as builder

WORKDIR $GOPATH/src/github.com/hasnat/docker-events-notifier

COPY main.go .

RUN go get -d -v ./...

RUN go install -v ./...

ENV DOCKER_API_VERSION=1.43
ENV RLOG_LOG_LEVEL=WARN
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o /go/bin/docker-events-notifier .

FROM alpine:latest



WORKDIR /etc/docker-events-notifier

COPY templates/ /etc/docker-events-notifier/templates/
COPY --from=builder /go/bin/docker-events-notifier .

CMD ["./docker-events-notifier"]
