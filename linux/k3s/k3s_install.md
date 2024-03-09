# k3s install

## 下载

### 1.官方下载

国内下载可能非常慢

~~~bash
curl -sfL https://get.k3s.io | sh -
~~~

### 2.国内源安装

~~~bash
curl –sfL \
     https://rancher-mirror.oss-cn-beijing.aliyuncs.com/k3s/k3s-install.sh | \
     INSTALL_K3S_MIRROR=cn sh -s - \
     --system-default-registry "registry.cn-hangzhou.aliyuncs.com" \
     --write-kubeconfig ~/.kube/config \
     --write-kubeconfig-mode 666 \
     --disable traefik
~~~

## 验证

~~~bash
$ kubectl get nodes
NAME                      STATUS   ROLES                  AGE     VERSION
izuf6ja8r1bc2wwsrg54vhz   Ready    control-plane,master   3m17s   v1.28.7+k3s1
$ kubectl -n kube-system get pods
NAME                                      READY   STATUS    RESTARTS   AGE
local-path-provisioner-7f4c755b68-7rjzr   1/1     Running   0          7m19s
coredns-58c9946f4-59dd4                   1/1     Running   0          7m19s
metrics-server-595fb6fd99-4cg7t           1/1     Running   0          7m19s
~~~

## 配置kubectl命令补全

~~~bash
# bash
source <(kubectl completion bash)

# zsh
source <(kubectl completion zsh)
~~~

## 外网访问

如果是阿里云服务器，需要添加安全组（6443端口）

添加tls-san配置

~~~bash
# xxx.xxx.xxx.158 是云服务器公网地址
# vim /etc/systemd/system/multi-user.target.wants/k3s.service
ExecStart=/usr/local/bin/k3s server '--tls-san' 'xxx.xxx.xxx.158'
~~~

## 修改docker运行时

> 省略下载安装docker

修改配置：

~~~bash
sudo vim /etc/systemd/system/multi-user.target.wants/k3s.service
~~~

需要修改ExecStart的值，将其修改为：

~~~bash
/usr/local/bin/k3s server --docker --no-deploy traefik
~~~

重启服务

~~~bash
sudo systemctl daemon-reload
sudo systemctl restart k3s 

# 然后查看节点是否启动正常
sudo k3s kubectl get no
~~~
