# Stage 1: build executable
FROM golang:1.12-alpine AS builder

WORKDIR /src

# Git is required for fetching the dependencies.
RUN apk add --no-cache git

# Fetch dependencies first; they are less susceptible to change on every build
# and will therefore be cached for speeding up the next build
COPY ./go.mod ./go.sum ./
RUN go mod download 

# Import the code
COPY ./ ./ 

# Build static executable /app
RUN CGO_ENABLED=0 go build \
    -installsuffix 'static' \
    -o /app .

# Stage 2: run container
FROM scratch AS runtime

ARG HTTP_PORT=8080

COPY --from=builder /app /app
COPY keystore /keystore
COPY password.txt password.txt

# Expose HTTP port
EXPOSE ${HTTP_PORT}

ENTRYPOINT ["/app"]
