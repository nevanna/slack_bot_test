#FIRST STAGE
FROM golang:1.15.3-alpine
ENV GO111MODULE=on
ENV SLACK_CHANNEL=docker_file_channel
COPY . $GOPATH/src/slack_bot_test
WORKDIR $GOPATH/src/slack_bot_test/cmd/slack_bot_test/
RUN go get github.com/slack-go/slack@v0.7.2
RUN go get github.com/pkg/errors@v0.9.1
RUN go build -o /main main.go

#SECOND STAGE
FROM alpine
ENTRYPOINT ["./bot"]

#Copy from first stage

COPY --from=0 /main /bot