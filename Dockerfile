FROM node:22-alpine AS frontend-build
WORKDIR /src/frontend
COPY frontend/package*.json ./
RUN npm ci
COPY frontend/ ./
ARG VITE_BUSINESS_TIMEZONE=Asia/Almaty
ENV VITE_BUSINESS_TIMEZONE=$VITE_BUSINESS_TIMEZONE
RUN npm run build

FROM golang:1.25-alpine AS backend-build
WORKDIR /src/backend
RUN apk add --no-cache git
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ ./
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/server ./cmd

FROM alpine:3.22
WORKDIR /app
RUN apk add --no-cache ca-certificates tzdata
COPY --from=backend-build /out/server /app/server
COPY --from=frontend-build /src/frontend/dist /app/web
EXPOSE 8080
CMD ["/app/server"]

