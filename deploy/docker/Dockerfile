# Stage 1: 构建 React 前端
FROM node:22-alpine AS frontend-builder
WORKDIR /app/frontend
RUN corepack enable && corepack prepare pnpm@10.23.0 --activate
COPY frontend/package.json frontend/pnpm-lock.yaml ./
RUN pnpm install --frozen-lockfile
COPY frontend/ ./
RUN pnpm build

# Stage 2: 编译 Go 后端
FROM golang:1.26-alpine AS backend-builder
RUN apk add --no-cache gcc musl-dev sqlite-dev
WORKDIR /app/backend
ENV GOPROXY=https://goproxy.cn,direct
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ ./
# 将前端编译成果拷贝到后端相对路径以备 Go 托管
COPY --from=frontend-builder /app/frontend/dist ./web/dist
RUN CGO_ENABLED=1 GOOS=linux go build -o /app/ledger-two ./cmd/server

# Stage 3: 最终部署镜像
FROM alpine:latest
RUN apk add --no-cache ca-certificates tzdata sqlite
WORKDIR /app
# 拷贝编译完成的可执行文件与前端静态资源目录
COPY --from=backend-builder /app/ledger-two /app/ledger-two
COPY --from=backend-builder /app/backend/web/dist /app/web/dist
EXPOSE 8080
CMD ["/app/ledger-two"]

