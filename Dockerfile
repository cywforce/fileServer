FROM alpine:latest as certs

# Install the CA certificates
RUN apk --update add ca-certificates


FROM scratch AS prod

MAINTAINER Alsmile "alsmile123@qq.com"
ENV REFRESHED_AT 2019-08-02

# 从certs阶段拷贝CA证书
COPY --from=certs /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt

# 拷贝主程序
COPY . .

# WORKDIR server
RUN chmod +x ./fileServer
EXPOSE 8201
ENTRYPOINT ["./fileServer"]

# docker build -t registry.local/fileServer:0.1 .
# docker run --name fileServer -d -p 8201:8201 [-v /etc/le5leFileServer.yaml:/etc/le5leFileServer.yaml] <image name:tag>
