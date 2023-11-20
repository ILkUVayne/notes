# E492: Not an editor command: ^M on Linux subsystem

~~~bash
Error detected while processing /mnt/c/Users/user/.vim/plugged/c-support/ftdetect/template.vim:
line   18:
E492: Not an editor command: ^M
Error detected while processing /mnt/c/Users/user/.vim/plugged/vim-go/ftdetect/gofiletype.vim:
line    2:
E492: Not an editor command: ^M
line    4:
E15: Invalid expression: &cpo^M
line    5:
E488: Trailing characters: cpo&vim^M
line    6:
E492: Not an editor command: ^M
line   11:
E492: Not an editor command: ^M
line   20:
E492: Not an editor command: ^M
~~~

造成原因：

显然你得到了“Windows”运行时文件集（带有<CR><NL>行尾），并且 Vim 认为它运行在“类 Unix”操作系统上：

对于具有类似 Dos <EOL>( <CR><NL>) 的系统，当读取
带有 ":source" 的文件以及 vimrc 文件时，<EOL>可能会
进行自动检测：

当“fileformats”为空时，不会自动检测。
将使用Dos格式。
当“文件格式”设置为一个或多个名称时，
将完成自动检测。这是根据<NL>文件中的第一个：如果
<CR>前面有a，则使用Dos格式，否则
使用Unix格式。
然而，所有这一切都只发生在“for Windows”的 Vim 中。OTOH，任何 Vim 都可以使用“for Unix”运行时文件（只有<NL>行尾）。因此，不仅是您的 vimrc，而且要获取的任何脚本都必须在每行末尾有一个<NL>without 。<CR>

解决方法：

~~~bash
git config --global core.autocrlf false 
~~~

## Can not open brower by using cmd.exe command

wsl2下 markdown-preview 插件不能打开浏览器

解决方法：

导出win的环境变量：

~~~bash
# 放入.zshrc
export PATH=$PATH:/mnt/c/Windows/System32/
~~~

如果打不开预览或者没有反应，可能是 **:call mkdp#util#install()** 命令没有执行成功

手动执行：

~~~vim
:call mkdp#util#install()
~~~

git相关issues [git issues](https://github.com/iamcco/markdown-preview.nvim/issues/309)
