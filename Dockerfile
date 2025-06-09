########################################################################
# 1) BUILD STAGE – build static Linux binaries
########################################################################
FROM golang:1.23-alpine AS builder

WORKDIR /app

# Go modules cache
COPY go.mod go.sum ./
RUN go mod download

# Shared code (copied first for better caching)
COPY shared/ ./shared/

# Swagger docs (optional)
COPY docs/ ./docs/

# Build argument: which micro-service to compile as the main binary
ARG SERVICE_NAME

# Copy the specific service code
COPY ${SERVICE_NAME}/ ./${SERVICE_NAME}/

# CLI helpers in cmd/  (seed, reset-db, etc.)
COPY cmd/ ./cmd/

# Main service binary
RUN CGO_ENABLED=0 GOOS=linux go build -o main ./${SERVICE_NAME}

# Seed and reset-db binaries
RUN CGO_ENABLED=0 GOOS=linux go build -o seed     ./cmd/seed
RUN CGO_ENABLED=0 GOOS=linux go build -o reset-db ./cmd/reset-db

########################################################################
# 2) RUNTIME STAGE – distroless, non-root
########################################################################
FROM gcr.io/distroless/static-debian12:nonroot

# Main service binary
COPY --from=builder /app/main /main

# Helper binaries
COPY --from=builder /app/seed     /seed
COPY --from=builder /app/reset-db /reset-db

# Mail templates
COPY --from=builder /app/shared/mail_templates/ /mail_templates/

# The distroless image already runs as non-root
USER nonroot:nonroot

# Default port (overridden by docker-compose)
EXPOSE 8000

# Entry point
CMD ["/main"]