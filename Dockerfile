# 多阶段构建：编译阶段与运行阶段分离，镜像只含二进制
FROM golang:1.27-alpine AS build

WORKDIR /src

# 先拷贝依赖清单，利用 Docker 层缓存
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/api ./cmd/api

# 运行阶段：最小镜像 + 非 root 用户
FROM alpine:3.22

RUN addgroup -S app && adduser -S app -G app
COPY --from=build /out/api /usr/local/bin/api

USER app
EXPOSE 8080

ENTRYPOINT ["api"]
