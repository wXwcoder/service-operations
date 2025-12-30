# 官方教程
#https://docs.docker.com/engine/install/debian/

#1. 卸载旧版本
for pkg in docker.io docker-doc docker-compose docker-compose-v2 podman-docker containerd runc; do
  sudo apt-get remove $pkg
done

#2. 设置仓库
# 安装依赖包
sudo apt-get update
sudo apt-get install ca-certificates curl

#3. 添加Docker官方GPG密钥
# 创建keyring目录
sudo install -m 0755 -d /etc/apt/keyrings

# 下载并安装GPG密钥
curl -fsSL https://download.docker.com/linux/debian/gpg | sudo gpg --dearmor -o /etc/apt/keyrings/docker.gpg
sudo chmod a+r /etc/apt/keyrings/docker.gpg


#4. 添加Docker APT仓库
# 根据你的Debian版本设置仓库
# 将下面的 "<release>" 替换为你的Debian代号

echo \
  "deb [arch="$(dpkg --print-architecture)" signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/debian \
  "$(. /etc/os-release && echo "$VERSION_CODENAME")" stable" | \
  sudo tee /etc/apt/sources.list.d/docker.list > /dev/null


#5. 安装Docker引擎
# 更新APT包索引
sudo apt-get update

# 安装最新版本
sudo apt-get install docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin

# 或者安装特定版本
# 首先查看可用版本
apt-cache madison docker-ce | awk '{ print $3 }'

# 安装特定版本（例如 5:26.1.0-1~debian.12~bookworm）
VERSION_STRING=5:26.1.0-1~debian.12~bookworm
sudo apt-get install docker-ce=$VERSION_STRING docker-ce-cli=$VERSION_STRING containerd.io docker-buildx-plugin docker-compose-plugin


#5. 配置Docker开机自启
# 设置Docker开机启动
sudo systemctl enable docker.service
sudo systemctl enable containerd.service

# 启动Docker服务（如果尚未运行）
sudo systemctl start docker.service

#6. 卸载Docker包
sudo apt-get purge docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin docker-ce-rootless-extras