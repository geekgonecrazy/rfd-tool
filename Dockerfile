FROM golang:1.24-alpine AS build

RUN apk add --no-cache ca-certificates git
WORKDIR /go/src/github.com/geekgonecrazy/rfd-tool
COPY go.mod .
COPY go.sum .
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build ./cmd/rfd-server
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build ./cmd/rfd-client

FROM scratch AS runtime

ARG GIN_MODE=release
ENV GIN_MODE=$GIN_MODE

WORKDIR /usr/local/rfd-tool

COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=build /go/src/github.com/geekgonecrazy/rfd-tool/templates templates
COPY --from=build /go/src/github.com/geekgonecrazy/rfd-tool/assets assets

COPY --from=build /go/src/github.com/geekgonecrazy/rfd-tool/rfd-server .
COPY --from=build /go/src/github.com/geekgonecrazy/rfd-tool/rfd-client .

EXPOSE 5678
EXPOSE 8877

CMD ["./rfd-server"]
