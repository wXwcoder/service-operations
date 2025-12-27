#!/usr/bin/env bash
#
# 启动 Alloy 容器
#
docker run \
    -itd \
    --name=alloy \
    --restart=unless-stopped \
    -v /data/alloy/config/config.alloy:/etc/alloy/config.alloy \
    -v /data/alloy/data:/var/lib/alloy/data \
    -v /val/log:/val/log:ro \
    -p 12345:12345 \
    registry.cn-beijing.aliyuncs.com/vtm/alloy:latest \
    run --server.http.listen-addr=0.0.0.0:12345 --storage.path=/var/lib/alloy/data \
    /etc/alloy/config.alloy