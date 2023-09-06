# apt update 报错403问题

> system version: Ubuntu 20 LTS

# 报错描述

~~~bash
$ sudo apt update
Hit:1 http://mirrors.aliyun.com/ubuntu focal InRelease
Hit:2 http://mirrors.aliyun.com/ubuntu focal-updates InRelease
Hit:3 http://mirrors.aliyun.com/ubuntu focal-backports InRelease
Hit:4 http://mirrors.aliyun.com/ubuntu focal-security InRelease
Hit:5 https://download.docker.com/linux/ubuntu focal InRelease
Get:6 https://deb.nodesource.com/node_18.x focal InRelease [4583 B]
Err:7 http://ppa.launchpad.net/dawidd0811/neofetch/ubuntu focal InRelease
  403  Forbidden [IP: 185.125.190.52 80]
Hit:8 http://ppa.launchpad.net/ondrej/php/ubuntu focal InRelease
Reading package lists... Done
E: Failed to fetch http://ppa.launchpad.net/dawidd0811/neofetch/ubuntu/dists/focal/InRelease  403  Forbidden [IP: 185.125.190.52 80]
E: The repository 'http://ppa.launchpad.net/dawidd0811/neofetch/ubuntu focal InRelease' is not signed.
N: Updating from such a repository can't be done securely, and is therefore disabled by default.
N: See apt-secure(8) manpage for repository creation and user configuration details.
~~~

# 解决方法

~~~bash
$ cd /etc/apt/sources.list.d
# 删除403报错的源即可
$ sudo rm -rf dawidd0811-ubuntu-neofetch-focal.list
~~~