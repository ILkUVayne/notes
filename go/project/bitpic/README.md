# 黑白像素点图片解析为文字文件

> 记一次线上面试题实现方式

> go version: 1.21

## 题目描述

现有一份由某段文字转换生成的黑白两色的图片文件，现需要编写一个程序通过这个图片还原对于的文字信息

## 解题思路

大致思路就是提取图片文件的像素点信息，因为只有黑白两色，理解为只有0、1的二进制字符串，故直接转换为0、1数组即可，需要把bit数组转换为byte数组，最后在转换utf8即可，具体实现思路可以查看代码。同时还补充实现了通过文字生成图片功能