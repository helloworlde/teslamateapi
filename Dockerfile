# get golang container
FROM golang:1.26.0 AS builder

# get args
ARG apiVersion=unknown

# create and set workingfolder
WORKDIR /go/src/teslamateapi

# copy go mod files and full source tree (cmd/, internal/, pkg/, docs/).
# Module path stays github.com/tobiasehlert/teslamateapi.
COPY go.mod go.sum ./
COPY cmd/ ./cmd/
COPY internal/ ./internal/
COPY pkg/ ./pkg/
COPY docs/ ./docs/

# install swag CLI (matches the swaggo runtime locked in go.mod), regenerate
# the OpenAPI spec from in-source annotations, then compile. swag init writes
# docs/swagger.{go,yaml,json}; docs/embed.go embeds the YAML at build time.
RUN go install github.com/swaggo/swag/cmd/swag@v1.16.4 && \
  go mod download && \
  /go/bin/swag init --dir cmd/teslamateapi,internal,pkg/dto -g main.go -o docs --outputTypes go,yaml,json --parseDependency --parseInternal && \
  CGO_ENABLED=0 GOOS=linux go build \
  -a -installsuffix cgo -ldflags="-w -s \
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
