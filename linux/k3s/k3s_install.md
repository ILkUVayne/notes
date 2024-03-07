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