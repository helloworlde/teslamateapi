# get golang container
FROM golang:1.26.0 AS builder

# get args
ARG apiVersion=unknown

# create and set workingfolder
WORKDIR /go/src/teslamateapi

# copy go mod files and download dependencies first for better layer caching
COPY go.mod go.sum ./
RUN go mod download

# copy sources
COPY cmd/ ./cmd/
COPY internal/ ./internal/

# compile the program
RUN CGO_ENABLED=0 GOOS=linux go build \
  -a -installsuffix cgo -ldflags="-w -s \
  -X 'github.com/tobiasehlert/teslamateapi/internal/config.APIVersion=${apiVersion}' \
  " -o /out/app ./cmd/teslamateapi


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
COPY --from=builder --chown=nonroot:nonroot --chmod=555 /out/app .

# expose port 8080
EXPOSE 8080

# run application
CMD ["./app"]
