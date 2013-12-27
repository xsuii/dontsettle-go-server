dontsettle-go-server
====================

This is a server write by go which build for "2013 summer's android game develope".

### Installation
		go get -u github.com/xsuii/dontsettle-go-server
		go get -u github.com/go-sql-driver/mysql
		go get -u github.com/cihub/seelog

### 兼容性
该服务器主要在ubuntu12.04上开发，所以在linux上运行效果最佳（主要是日志系统方面）。

### Third Part API

* Log System - [https://github.com/cihub/seelog] [1]

### Go Package List

* seelog		: [https://github.com/cihub/seelog] [1]
* mysql-driver	: [https://github.com/go-sql-driver/mysql] [2]


[1]: https://github.com/cihub/seelog "seelog"
[2]: https://github.com/go-sql-driver/mysql "mysql"