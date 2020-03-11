FROM golang:1.14-alpine AS builder

ENV PROJECT=$GOPATH/src/github.com/dizaknz/gkgraph/event
RUN mkdir -p $PROJECT
WORKDIR $PROJECT
COPY . $PROJECT/

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags '-extldflags "-static"' -o /event $PROJECT/cmd/event/main.go

FROM scratch

WORKDIR /
COPY --from=builder /event /

ENTRYPOINT ["/event"]
CMD ["$@"]
