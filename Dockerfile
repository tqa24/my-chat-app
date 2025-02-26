# Stage 1: Build the Go application
FROM golang:1.23 AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
#Remove env from build stage.
RUN rm -f .env

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main ./cmd/main.go

# Stage 2: Build Frontend
FROM node:20-alpine AS frontend-builder
WORKDIR /frontend

COPY frontend/package*.json ./
RUN npm install
# RUN chmod +x ./node_modules/.bin/vue-cli-service
COPY frontend/ .
RUN npm run build

# Stage 3: Create a minimal image
FROM alpine:latest

# Add timezone data, PostgreSQL client, and other necessary tools
RUN apk --no-cache add ca-certificates tzdata postgresql-client

WORKDIR /root/

COPY --from=builder /app/main .
COPY --from=frontend-builder /frontend/dist ./frontend/dist
# Copy uploads directory
COPY --from=builder /app/uploads ./uploads

# Copy migrations *into the final image*
COPY --from=builder /app/migrations ./migrations

# Copy the entrypoint script AND set execute permissions
COPY entrypoint.sh /root/entrypoint.sh
RUN chmod +x /root/entrypoint.sh

#Not copy env file to image
#COPY --from=builder /app/.env .

EXPOSE 8080

# Use an entrypoint script
ENTRYPOINT ["/root/entrypoint.sh"]
CMD ["./main"]