# 概述

`DOJ` 是一个开源的在线评测系统，运行于Ubuntu系统。

`DOJ` 使用php作为后台语言，使用开源的 [Lo-runner](https://github.com/lodevil/Lo-runner) 作为评测内核。

# 使用条款

该项目使用 GPL 2.0 开原协议

如有商业用途请联系作者授权

# 安装

1. 建议使用Ubuntu 14.04或以上版本的操作系统作为OJ服务器

2. 如果你还没有安装git，请通过以下命令在终端中安装git：

	```bash
	sudo apt-get install git git-core
	```

3. 通过git命令克隆项目到本地：

	```bash
	git clone https://github.com/doveccl/DOJ.git
	```

4. 运行安装脚本：

	```bash
	cd doj
	bash ./install.sh
	```

5. 如果你之前没有安装过 mysql，安装脚本会帮助你安装 mysql ，你需要在安装过程中为 mysql 的 root 用户设置密码。

6. 安装脚本会提示你输入 mysql 用户名（默认为 root），然后你需要输入你设定的密码，安装脚本会自动检查用户名密码的正确性。

7. 等待安装脚本运行完成。如果你需要评测服务开机启动，请使用 root 权限编辑 /etc/rc.local 脚本：
	
	```bash
	sudo gedit /etc/rc.local
	```

	在该文件 `exit 0` 之前加入一行代码并保存：

	```bash
	sudo service doj start
	```

8. 在浏览器中输入 localhost 来测试 OJ 搭建情况（初始用户为管理员，用户名为 `admin`，密码为 `oj_admin`），并及时修改密码。

# 评测

如果你发现无论如何都无法AC，请在评测记录中找到该代码运行编号，并在终端中输入以下命令（命令中 %d 处写你获取的运行编号）：

```bash
dojudge %d config
sudo service doj start
```

如果你看到 `#0 {res: 1, ...}` 代表该语言已经被成功配置。

# 反馈

如有任何问题请通过以下邮箱反馈：

[i@doveccl.com](mailto:i@doveccl.com)