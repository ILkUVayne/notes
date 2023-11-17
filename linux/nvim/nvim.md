# neovim

## 安装

- 添加源

~~~bash
# stable version
sudo add-apt-repository ppa:neovim-ppa/stable

# dev version
sudo add-apt-repository ppa:neovim-ppa/unstable
~~~

- 更新源

~~~bash
sudo apt update

# 直接安装不会成功，会提示
# "The following packages have been kept back: ... "
sudo apt upgrade
~~~

- 安装

~~~bash
sudo apt install neovim
~~~

## 基础配置

> 基于lua

在 **~/.config/nvim/** 目录中添加配置文件

例如：

~/.config/nvim/init.lua

~~~bash
1 vim.o.tabstop = 4
2 vim.bo.tabstop = 4
3 vim.o.softtabstop = 4
4 vim.o.shiftwidth=4
5 vim.o.shiftround = true
6 vim.o.number = true
  
# 进入命令模式
:source %
~~~