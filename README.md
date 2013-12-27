dontsettle-go-server
====================

This is a server write by go which build for "2013 summer's android game develope".

Installation
------------

	go get -u github.com/xsuii/dontsettle-go-server
	go get -u github.com/go-sql-driver/mysql
	go get -u github.com/cihub/seelog
数据库使用mysql，在服务器目录"./mysql/model/"中包含了mysql-workbench的数据库建模文件"2013SummerAndroidGame.mwb"，直接导入并恢复即可；目录"./mysql/script/"中"init.sql"为数据库数据初始化脚本文件。

Documentation
-------------

聊天系统：在相应输入框中输入id（一对一对应用户id，一对多对应组id，默认为广播），便可以和对应的用户（组）进行聊天；文件传输一定要指定目标用户id，仅支持一对一传输文件。

兼容性
---
该服务器主要在ubuntu12.04上开发，所以在linux上运行效果最佳（主要是日志系统方面）。

### Third Part API

* Log System - [https://github.com/cihub/seelog] [1]

### Go Package List

* seelog		: [https://github.com/cihub/seelog] [1]
* mysql-driver	: [https://github.com/go-sql-driver/mysql] [2]


[1]: https://github.com/cihub/seelog "seelog"
[2]: https://github.com/go-sql-driver/mysql "mysql"