# syntax=docker/dockerfile:1

# get golang container
FROM golang:1.26.0 AS builder

# get args
ARG apiVersion=unknown

# create and set workingfolder
WORKDIR /go/src/teslamateapi

# download modules first so this layer stays cached unless go.mod/go.sum change.
# Module path stays github.com/tobiasehlert/teslamateapi.
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
  go mod download

# copy the full source tree (cmd/, internal/, pkg/, docs/).
COPY cmd/ ./cmd/
COPY internal/ ./internal/
COPY pkg/ ./pkg/
COPY docs/ ./docs/

# docs/swagger.{go,yaml,json} are committed and embedded by docs/embed.go,
# so the build stage just needs `go build`.
# cache mounts persist the module + build caches; dropping `-a -installsuffix cgo`
# lets Go reuse compiled stdlib/deps instead of rebuilding everything each time.
RUN --mount=type=cache,target=/go/pkg/mod \
  --mount=type=cache,target=/root/.cache/go-build \
  CGO_ENABLED=0 GOOS=linux go build \
  -trimpath -ldflags="-w -s \
  -X 'main.apiVersion=${apiVersion}' \
  " -o /go/src/app ./cmd/teslamateapi


# get alpine container
FROM alpine:3.23.3 AS app

# create workdir
WORKDIR /opt/app

# add packages, create nonroot user and group
RUN apk --no-cache add ca-certificates tzdata && \
  addgroup -S nonroot && \
  adduser -S nonroot -G nonroot && \
  chown -R nonroot:nonroot .

# set user to nonroot
USER nonroot:nonroot

# copy binary from builder
COPY --from=builder --chown=nonroot:nonroot --chmod=555 /go/src/app .

# expose port 8080
EXPOSE 8080

# run application
CMD ["./app"]
