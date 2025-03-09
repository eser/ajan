# versions

FROM golang:1-bookworm AS upstream-builder
FROM gcr.io/distroless/base-debian12 AS upstream-runner

# Create a minimal image base-debian12 or static-debian12
# (see: https://github.com/GoogleContainerTools/distroless#why-should-i-use-distroless-images)

# ------------------
# builder-base image
# ------------------

FROM upstream-builder AS builder-base

# Install the Protocol Buffers Library and Compiler
RUN apt-get update && apt-get -y install --no-install-recommends \
  libprotobuf-dev \
  protobuf-compiler

ENV GO111MODULE=on
ENV CGO_ENABLED=0
ENV GOOS=linux

# Set working directory
WORKDIR /app

# Install dependencies first to improve caching
COPY ./go.mod ./go.sum ./
RUN go mod download && go mod verify

# Copy source code
COPY . .

# ------------------------
# runner-development image
# ------------------------

FROM builder-base AS runner-development

EXPOSE 8080
ENTRYPOINT ["go", "run", "./"]

# ------------------------
# builder-production image
# ------------------------

FROM builder-base AS builder-production

RUN go build -o ./app ./

# -----------------------
# runner-production image
# -----------------------

FROM upstream-runner AS runner-production

# Copy the binary from the builder-production container
COPY --from=builder-production --chown=nonroot:nonroot /app /
# COPY --from=builder-production /bin/sh /bin/sh
# COPY --from=builder-production /bin/cat /bin/cat

# Run as a non-root user
USER nonroot

EXPOSE 8080
ENTRYPOINT ["/app"]
