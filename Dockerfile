FROM golang:1.26.11-alpine AS build

ARG VERSION=dev

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN mkdir -p /out \
    && CGO_ENABLED=0 GOOS=linux go build \
    -trimpath \
    -ldflags="-s -w -X github.com/JiffyCRM/user/internal/version.Version=${VERSION}" \
    -o /out/user ./cmd/server \
    && CGO_ENABLED=0 GOOS=linux go build \
    -trimpath \
    -ldflags="-s -w" \
    -o /out/user-migrate ./cmd/migrate

FROM alpine:3.22

RUN addgroup -S -g 65532 user \
    && adduser -S -u 65532 -G user user

COPY --from=build /out/user /usr/local/bin/user
COPY --from=build /out/user-migrate /usr/local/bin/user-migrate

USER 65532:65532
EXPOSE 50053
ENTRYPOINT ["/usr/local/bin/user"]