# 添加 Docker 安装源
yum-config-manager --add-repo http://mirrors.aliyun.com/docker-ce/linux/centos/docker-ce.repo
#安装所需组件
yum install -y docker-ce

#配置本地仓库
vim /etc/docker/daemon.json
{
"insecure-registries": ["127.0.0.1:5000"],
"registry-mirrors": ["https://4abdkxlk.mirror.aliyuncs.com"]
}

# 设置Docker开机启动
systemctl enable docker
systemctl daemon-reload
systemctl restart docker.service

# 官方教程
#https://docs.docker.com/engine/install/centos/