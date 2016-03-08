SET NAMES utf8;
SET foreign_key_checks = 0;
SET time_zone = '+08:00';
SET sql_mode = 'NO_AUTO_VALUE_ON_ZERO';

CREATE DATABASE IF NOT EXISTS `doj`;
USE `doj`;

CREATE TABLE `contests` (
  `id` int(11) NOT NULL AUTO_INCREMENT,
  `name` text NOT NULL,
  `start_time` datetime NOT NULL,
  `time` int(11) NOT NULL,
  `result` longtext NOT NULL,
  `hide` tinyint(4) NOT NULL DEFAULT '0',
  PRIMARY KEY (`id`)
) ENGINE=MyISAM DEFAULT CHARSET=utf8;


CREATE TABLE `problems` (
  `id` int(11) NOT NULL AUTO_INCREMENT,
  `cid` int(11) NOT NULL DEFAULT '0',
  `name` varchar(20) NOT NULL,
  `description` text NOT NULL,
  `input` text NOT NULL,
  `output` text NOT NULL,
  `hint` text NOT NULL,
  `time` int(11) NOT NULL,
  `memory` int(11) NOT NULL,
  `create_time` datetime NOT NULL,
  `tags` text NOT NULL,
  `hide` tinyint(4) NOT NULL DEFAULT '0',
  PRIMARY KEY (`id`)
) ENGINE=MyISAM DEFAULT CHARSET=utf8;

INSERT INTO `problems` (`id`, `cid`, `name`, `description`, `input`, `output`, `hint`, `time`, `memory`, `create_time`, `tags`, `hide`) VALUES
(1000,	1,	'A+B Problem',	'给你两个数 $a$ 和 $b$，请输出他们的和。',	'一行，两个用空格隔开的整数 $a$ 和 $b$。',	'一个整数，表示 $a+b$。',	'对于100%的数据，有 $0 \\leq a, b \\leq 10^9$。\r\n\r\n此处放出各种语言代码：\r\n\r\n##### C\r\n\r\n>```c\r\n#include <stdio.h>\r\nint main()\r\n{\r\n	int a, b;\r\n	scanf(\"%d%d\", &a, &b);\r\n	printf(\"%d\", a + b);\r\n	return 0;\r\n}\r\n```\r\n\r\n##### C++\r\n\r\n>```cpp\r\n#include <iostream>\r\nusing namespace std;\r\nint main()\r\n{\r\n	int a, b;\r\n	cin >> a >> b;\r\n	cout << a + b;\r\n	return 0;\r\n}\r\n```\r\n\r\n##### Pascal\r\n\r\n>```delphi\r\nprogram P1000;\r\nvar a, b: longint;\r\nbegin\r\n	read(a, b);\r\n	writeln(a + b);\r\nend.\r\n```\r\n\r\n##### Python 2.x\r\n\r\n>```python\r\na, b = raw_input().split(\" \")\r\nprint int(a) + int(b)\r\n```',	1000,	64000,	'2015-08-30 20:23:46',	'水题',	0),
(1001,	1,	'Hello World !',	'同样是一道水题，和A+B Problem一样，为初学者而设，用于熟悉评测环境。\n你只需要输出 `Hello World !` 即可。',	'此题没有输入格式。',	'输出一行，一个字符串 `Hello World !` 。',	'和上一题一样，放出各种语言代码，以供参考：\n\n#### C\n\n>```c\n#include <stdio.h>\nint main()\n{\n	printf(\"Hello World !\");\n}\n```\n\n#### C++\n\n>```cpp\n#include <iostream>\nusing namespace std;\nint main()\n{\n	cout << \"Hello World !\";\n}\n```\n\n#### Pascal\n\n>```delphi\nprogram P1001;\nbegin\n	write(\'Hello World !\');\nend.\n```\n\n#### Python 2.x\n\n>```python\nprint \"Hello World !\"\n```',	1000,	64000,	'2015-08-30 20:39:30',	'水题',	0);

CREATE TABLE `submit` (
  `id` int(11) NOT NULL AUTO_INCREMENT,
  `uid` int(11) NOT NULL,
  `pid` int(11) NOT NULL,
  `cid` int(11) NOT NULL DEFAULT '0',
  `submit_time` datetime NOT NULL,
  `language` int(10) unsigned NOT NULL,
  `res` smallint(6) NOT NULL,
  `result` text NOT NULL,
  `code` text NOT NULL,
  PRIMARY KEY (`id`),
  KEY `uid` (`uid`),
  KEY `cid` (`cid`)
) ENGINE=MyISAM DEFAULT CHARSET=utf8;


CREATE TABLE `tags` (
  `group` varchar(20) NOT NULL,
  `set` text NOT NULL
) ENGINE=MyISAM DEFAULT CHARSET=utf8;

INSERT INTO `tags` (`group`, `set`) VALUES
('比赛',	'NOIP|APIO|CTSC|NOI|IOI|WC|省选'),
('年份',	'2001|2002|2003|2004|2005|2006|2007|2008|2009|2010|2011|2012|2013|2014|2015|2016'),
('地区',	'上海|四川|安徽|山东|广东|江苏|浙江|湖南|重庆'),
('基础算法',	'枚举|贪心|递归|递推|模拟|排序|哈希'),
('图论',	'最短路|差分约束|网络流|费用流|二分图|拓扑排序|强连通分量|割点和桥|欧拉回路|2-SAT'),
('计算几何',	'凸包|半平面交|最远点对|最小圆覆盖|坐标离散|扫描线算法'),
('树结构',	'树的遍历|生成树|公共祖先|树上倍增|树链剖分'),
('动态规划',	'动态规划|背包DP|线性DP|环形DP|树形DP|斜率DP|插头DP|状态压缩'),
('数学',	'GCD&LCM|素数判定|扩展欧几里德|排列组合|生成函数|容斥原理|康托展开|Polya定理|Fibonacci数列|Catalan数列|博弈论|高斯消元|矩阵乘法'),
('字符串',	'KMP|Trie树|AC自动机|Manacher|正则表达式'),
('数据结构',	'栈|队列|链表|并查集|堆|线段树|左偏树|平衡树|树状数组|树套树|KD-Tree|块状链表|后缀树'),
('搜索',	'DFS|BFS|剪枝|启发式搜索|记忆化搜索'),
('其它',	'二分查找|高精度|FFT|水题');

CREATE TABLE `users` (
  `id` int(11) NOT NULL AUTO_INCREMENT,
  `name` varchar(16) NOT NULL,
  `mail` varchar(40) NOT NULL,
  `password` varchar(200) NOT NULL,
  `reg_time` datetime NOT NULL,
  `sign` varchar(100) NOT NULL DEFAULT '我是大神犇！',
  `school` varchar(100) NOT NULL DEFAULT '清华大学',
  `admin` tinyint(4) NOT NULL DEFAULT '0',
  `solve` int(11) NOT NULL DEFAULT '0',
  `submit` int(11) NOT NULL DEFAULT '0',
  PRIMARY KEY (`id`),
  KEY `name` (`name`),
  KEY `mail` (`mail`)
) ENGINE=MyISAM DEFAULT CHARSET=utf8;

INSERT INTO `users` (`id`, `name`, `mail`, `password`, `reg_time`, `admin`, `solve`, `submit`) VALUES
(1,	'admin',	'admin@doj.com',	'biyfhilxdhzfehacohsflhnxghaxchmfgijxnhmnkipoaiphhhphbhwpkhzlmhcjciyvdizkphjefhpfjibteimbiipcjhzofihtihsw',	'0000-00-00', 2, 0, 0);
