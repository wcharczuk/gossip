FROM docker.io/golang:latest AS builder

WORKDIR "/go/src"

ADD go.mod /go/src/go.mod
ADD go.sum /go/src/go.sum
ADD vendor /go/src/vendor
ADD main.go /go/src/main.go

ENV GOOS=linux
ENV GOARCH=arm64
RUN go build -o /go/bin/gossip main.go

FROM docker.io/alpine:latest
WORKDIR /app
COPY --from=builder /go/bin/gossip .
CMD ["./gossip"]

ENTRYPOINT /go/bin/gossip
EXPOSE 7946
