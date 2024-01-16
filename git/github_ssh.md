# github 添加ssh

> linux/wsl

## 生成ssh密钥

~~~bash
ssh-keygen -t ed25519 -C "1193055746@qq.com"
# 后续全部回车即可，成功后会在 ~/.ssh/ 生成对于密钥（可能在其他位置，会在生成过程中提示）
~~~

## 将 SSH 密钥添加到 ssh-agent

### 1. 在后台启动 ssh 代理

~~~bash
eval "$(ssh-agent -s)"
~~~

### 2. 将 SSH 私钥添加到 ssh-agent

~~~bash
# 生成的公钥 id_ed25519
ssh-add ~/.ssh/id_ed25519
~~~

### 3. 将 SSH 公钥添加到 GitHub 上的帐户

在github->Settings->SSH and GPG keys中添加生成的ssh公钥，位于 ~/.ssh/id_ed25519.pub

### 4. 测试

~~~bash
ssh -T git@github.com
Hi ILkUVayne! You've successfully authenticated, but GitHub does not provide shell access.
~~~

# FAQ

## fatal: Could not read from remote repository.

如果正确配置了ssh,可能是网络问题，我的解决办法是重启电脑（遇到过两次，都是这么解决的）

## ssh: connect to host github.com port 22: Connection timed out

1. 可能是22端口不可用，尝试换个端口

~~~bash
ssh -T -p 443 git@ssh.github.com
Hi ILkUVayne! You've successfully authenticated, but GitHub does not provide shell access.
~~~

2. 修改端口

~~~bash
# 创建config
vim ~/.ssh/config
~~~

添加配置

~~~editorconfig
Host github.com
User git
Hostname ssh.github.com
PreferredAuthentications publickey
IdentityFile ~/.ssh/id_ed25519
Port 443
~~~