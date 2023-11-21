# go 环境配置

## 相关工具安装

- goimports-reviser

~~~bash
go install github.com/incu6us/goimports-reviser/v3@latest
~~~

- gofumpt

~~~bash
go install mvdan.cc/gofumpt@latest
~~~

- golines

~~~bash
go install github.com/segmentio/golines@latest
~~~

- go-delve

~~~bash
go install github.com/go-delve/delve/cmd/dlv@latest
~~~

- gotests

~~~bash
go install github.com/cweill/gotests/gotests@latest
~~~

- impl

~~~bash
go install github.com/josharian/impl@latest
~~~

- lazygit

~~~bash
LAZYGIT_VERSION=$(curl -s "https://api.github.com/repos/jesseduffield/lazygit/releases/latest" | grep -Po '"tag_name": "v\K[^"]*')
curl -Lo lazygit.tar.gz "https://github.com/jesseduffield/lazygit/releases/latest/download/lazygit_${LAZYGIT_VERSION}_Linux_x86_64.tar.gz"
tar xf lazygit.tar.gz lazygit
sudo install lazygit /usr/local/bin
~~~
