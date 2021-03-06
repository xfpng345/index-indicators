FROM golang:1.15 as builder

ENV CGO_ENABLED=0
ENV GOOS=linux
ENV GOARCH=amd64
WORKDIR /usr/src/app/server

COPY server/go.mod server/go.sum ./
RUN go mod download
COPY ./server .

RUN make

# -----------------------------------
FROM alpine

RUN apk add --no-cache ca-certificates && \
    apk --update add tzdata && \
    cp /usr/share/zoneinfo/Asia/Tokyo /etc/localtime && \
    apk del tzdata && \
    rm -rf /var/cache/apk/*

COPY --from=builder usr/src/app/server/main ./

CMD ["./main"]


EXPOSE 8080
