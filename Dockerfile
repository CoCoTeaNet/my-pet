# 前端构建
FROM node:22-alpine AS frontend
WORKDIR /app/frontend
COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci
COPY frontend/ ./
RUN npm run build

# 后端构建
FROM golang:1.26-alpine AS backend
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=frontend /app/frontend/dist ./frontend/dist
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o my-pet .

# 运行镜像
FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=backend /app/my-pet ./my-pet
ENV TZ=Asia/Shanghai PORT=8080 DB_PATH=/app/data/my-pet.db
VOLUME ["/app/data"]
EXPOSE 8080
ENTRYPOINT ["./my-pet"]
