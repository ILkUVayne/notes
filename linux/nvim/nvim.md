# neovim

## 前置

~~~bash
sudo apt install ripgrep

sudo apt install fd-find
~~~

## 安装

### 1. apt安装

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

### 2. 官网下载安装

apt安装的版本一般较低

在neovim仓库下载最新release版本：**https://github.com/neovim/neovim/wiki/Installing-Neovim**

下载对应的版本：**https://github.com/neovim/neovim/releases/latest/download/nvim-linux64.tar.gz**

~~~bash
# 解压
tar -zxf nvim-linux64.tar.gz

# 建立软链
cd /usr/local/bin
sudo ln -s ~/nvim_src/nvim-linux64/bin/nvim ./nvim

# 查看版本
nvim -version
NVIM v0.9.4
Build type: Release
LuaJIT 2.1.1692716794

   system vimrc file: "$VIM/sysinit.vim"
  fall-back for $VIM: "/__w/neovim/neovim/build/nvim.AppDir/usr/share/nvim"

Run :checkhealth for more info
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