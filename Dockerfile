# get golang container
FROM golang:1.26.0 AS builder

# get args
ARG apiVersion=unknown

# create and set workingfolder
WORKDIR /go/src/

# copy go mod files and sourcecode
COPY go.mod go.sum ./
COPY src/ .

# install swag CLI (matches the swaggo runtime locked in go.mod), regenerate
# the OpenAPI spec from in-source annotations, then compile. swag init writes
# docs/swagger.{go,yaml,json}; openapi_handler.go embeds the YAML at build.
RUN go install github.com/swaggo/swag/cmd/swag@v1.16.4 && \
  go mod download && \
  /go/bin/swag init -g webserver.go -o docs --outputTypes go,yaml,json --parseDependency --parseInternal && \
  CGO_ENABLED=0 GOOS=linux go build \
  -a -installsuffix cgo -ldflags="-w -s \
  -X 'main.apiVersion=${apiVersion}' \
  " -o app ./...


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
