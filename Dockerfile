FROM golang:1.12

WORKDIR /go/src/github.com/ilya-pirogov/youtube-downloader/
COPY . .
RUN go get -d -v ./... \
  && GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags '-w -extldflags "-static"' -a -o yd ./cmd/yd/main.go

FROM alpine
RUN apk --no-cache add ca-certificates ffmpeg

WORKDIR /app
VOLUME /app/out

COPY --from=0 /go/src/github.com/ilya-pirogov/youtube-downloader/yd .

EXPOSE 80

CMD ["./yd"]
