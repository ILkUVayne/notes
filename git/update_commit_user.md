# 修改历史commit user

## 使用脚本自动更改

### 1. 克隆需修改仓库裸露的Git存储库

~~~bash
$ git clone --bare git@github.com:ILkUVayne/gii.git
~~~

### 2. 进入裸Git仓库

~~~bash
$ cd gii.git
~~~

### 3. 添加更新脚本

使用以下脚本，把相关参数改为自己的。

- OLD_EMAIL：旧邮箱，需要被修改掉的commit 邮箱

- CORRECT_NAME： 新用户名

- CORRECT_EMAIL： 新邮箱
~~~bash
#!/bin/sh

git filter-branch --env-filter '

OLD_EMAIL="luoyang19950909@163.com"
CORRECT_NAME="luoyang"
CORRECT_EMAIL="1193055746@qq.com"

if [ "$GIT_COMMITTER_EMAIL" = "$OLD_EMAIL" ]
then
export GIT_COMMITTER_NAME="$CORRECT_NAME"
export GIT_COMMITTER_EMAIL="$CORRECT_EMAIL"
fi
if [ "$GIT_AUTHOR_EMAIL" = "$OLD_EMAIL" ]
then
export GIT_AUTHOR_NAME="$CORRECT_NAME"
export GIT_AUTHOR_EMAIL="$CORRECT_EMAIL"
fi
' --tag-name-filter cat -- --branches --tags
~~~

~~~bash
$ vim amend.sh
~~~

### 4. 执行脚本

~~~bash
sh ./amend.sh
~~~

### 5. 推送远程仓库，并删除本地裸Git库

~~~bash
$ git push --force --tags origin 'refs/heads/*'
$ cd ..
$ rm -rf repo.git
~~~