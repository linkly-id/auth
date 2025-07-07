FROM golang:1.23.7-alpine3.20 as build
ENV GO111MODULE=on
ENV CGO_ENABLED=0
ENV GOOS=linux

RUN apk add --no-cache make git

WORKDIR /go/src/github.com/linkly-id/auth

# Pulling dependencies
COPY ./Makefile ./go.* ./
RUN make deps

# Building stuff
COPY . /go/src/github.com/linkly-id/auth

# Make sure you change the RELEASE_VERSION value before publishing an image.
RUN RELEASE_VERSION=unspecified make build

# Always use alpine:3 so the latest version is used. This will keep CA certs more up to date.
FROM alpine:3
RUN adduser -D -u 1000 linkly

RUN apk add --no-cache ca-certificates
COPY --from=build /go/src/github.com/linkly-id/auth/auth /usr/local/bin/auth
COPY --from=build /go/src/github.com/linkly-id/auth/migrations /usr/local/etc/auth/migrations/
RUN ln -s /usr/local/bin/auth /usr/local/bin/gotrue

ENV GOTRUE_DB_MIGRATIONS_PATH /usr/local/etc/auth/migrations

USER linkly
CMD ["auth"]
