FROM golang:1.12 as builder

WORKDIR /project
COPY go.mod .
COPY go.sum .
RUN go mod download
ADD . .
RUN CGO_ENABLED=0 go build -o stats_collector cmd/collector/collector.go

FROM alpine:latest
WORKDIR /project
COPY --from="builder" /project/stats_collector stats_collector
ENTRYPOINT ["/project/stats_collector"]
