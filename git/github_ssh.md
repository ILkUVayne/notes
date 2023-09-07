# github 添加ssh

> linux/wsl

## 生成ssh密钥

~~~bash
$ ssh-keygen -t ed25519 -C "1193055746@qq.com"
# 后续全部回车即可，成功后会在 ~/.ssh/ 生成对于密钥（可能在其他位置，会在生成过程中提示）
~~~

## 将 SSH 密钥添加到 ssh-agent

### 1. 在后台启动 ssh 代理

~~~bash
$ eval "$(ssh-agent -s)"
~~~

### 2. 将 SSH 私钥添加到 ssh-agent

~~~bash
# 生成的公钥 id_ed25519
ssh-add ~/.ssh/id_ed25519
~~~

### 3. 将 SSH 公钥添加到 GitHub 上的帐户

在github->Settings->SSH and GPG keys中添加生成的ssh公钥，位于 ~/.ssh/id_ed25519.pub