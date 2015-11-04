<!DOCTYPE html>
<?php
	$user = checkDOJSS($_COOKIE['DOJSS']);
	$uname = $user -> name;
	$uid = $user -> id;
	$bg = $user -> bg;
	$admin = $user -> admin;
	$gravatar = "//cn.gravatar.com/avatar/" . md5($user -> mail) . "?d=mm";
?>
<html>
<head lang="zh-cn">
	<meta charset="UTF-8">
	<meta http-equiv="X-UA-Compatible" content="IE=edge">
	<meta name="viewport" content="width=device-width, initial-scale=1.0, maximum-scale=1.0, user-scalable=no">
	<meta name="description" content="DOJ, an online judge for OIers.">
	<meta name="keywords" content="OI, Online Judge, algorithm">
	<meta name="author" content="Doveccl">

	<link rel='shortcut icon' type='image/x-icon' href='favicon.ico' />
	<title>Welcome to DOJ !</title>

	<link href="//cdn.bootcss.com/highlight.js/8.9.1/styles/github.min.css" rel="stylesheet">
	<link href="css/metro-icons.min.css" rel="stylesheet">
	<link href="css/metro.min.css" rel="stylesheet">
	<link href="css/doj.css" rel="stylesheet">

	<!--[if lt IE9]> 
		<script src="//cdn.bootcss.com/html5shiv/r29/html5.min.js"></script>
	<![endif]-->

	<script src="//cdn.bootcss.com/jquery/1.11.3/jquery.min.js"></script>
	<script src="//cdn.bootcss.com/jquery-cookie/1.4.1/jquery.cookie.min.js"></script>
	<script src="//cdn.bootcss.com/highcharts/4.1.9/highcharts.js"></script>
	<script src="//cdn.bootcss.com/highcharts/4.1.9/highcharts-3d.js"></script>
	<script src="//cdn.bootcss.com/highcharts/4.1.9/themes/gray.js"></script>
	<script src="//cdn.bootcss.com/marked/0.3.5/marked.min.js"></script>
	<script src="//cdn.bootcss.com/highlight.js/8.9.1/highlight.min.js"></script>
	<script src="//cdn.bootcss.com/mathjax/2.6.0-beta.1/MathJax.js?config=TeX-AMS-MML_SVG"></script>
	<script src="js/metro.min.js"></script>
	<script src="js/Ace/ace.js"></script>
	<script src="js/doj.js"></script>

</head>
<body style="<?php echo "background:url(bg/$bg.png) top center /cover no-repeat fixed";?>;" >
	<div class="charm right-side padding20 bg-grayDark" id="charmSearch">
		<button class="square-button bg-transparent fg-white no-border place-right small-button" onclick="showSearch()"><span class="mif-cross"></span></button>
		<h1 class="text-light">搜索</h1>
		<hr class="thin"/>
		<br />
		<div class="input-control text full-size">
			<label>
				<span class="dropdown-toggle drop-marker-light">所有项目</span>
				<ul class="d-menu" data-role="dropdown">
					<li><a onclick="setSearchPlace(this)">所有项目</a></li>
					<li><a onclick="setSearchPlace(this)">用户</a></li>
					<li><a onclick="setSearchPlace(this)">题目</a></li>
					<li><a onclick="setSearchPlace(this)">博客</a></li>
				</ul>
			</label>
			<input type="text">
			<button class="button"><span class="mif-search"></span></button>
		</div>
	</div>

	<div class="charm right-side padding20 bg-grayDark" id="charmSettings">
		<button class="square-button bg-transparent fg-white no-border place-right small-button" onclick="showSettings()"><span class="mif-cross"></span></button>
		<h1 class="text-light">背景设置</h1>
		<hr class="thin"/>
		<br />
		<div class="schemeButtons">
			<?php 
				for ($i = 1; $i < 27; $i++) echo "<img class=\"no-padding bg-img button square-button\" src=\"bg/sm/$i.png\" data-bg=\"$i\"/>\n";
			?>
		</div>
	</div>

	<div class="tile-area">
		<div class="tile-head">
			<div class="tile-area-title" style="font-size:22px;">
				<h1 style="color:white;">开始</h1>
			</div>

			<div class="tile-area-controls">
				<button class="image-button icon-right bg-transparent fg-white bg-hover-dark no-border dropdown-toggle" id="user-dropdown">
					<span class="sub-header no-margin text-light" id="myName"><?php echo $uname;?></span>
					<img class="icon no-padding gravatar" id="gravatar" src="<?php echo $gravatar;?>" style="border-radius:50px;"></img>
				</button>
				<ul class="d-menu" data-role="dropdown" style="display:none;">
					<li><a href="javascript:showDialog('#changeMsg');">修改资料</a></li>
					<li><a href="javascript:showDialog('#changeName');">修改用户名</a></li>
					<li><a href="javascript:showDialog('#changePwd');">修改密码</a></li>
					<li><a href="javascript:showDialog('#changeMail');">修改邮箱</a></li>
				</ul>

				<!--button class="square-button bg-transparent fg-white bg-hover-dark no-border" onclick="showSearch()">
					<span class="mif-search"></span>
				</button-->
				<button class="square-button bg-transparent fg-white bg-hover-dark no-border" onclick="showSettings()">
					<span class="mif-cog"></span>
				</button>
				<a href="javascript:logout();" title="注销登录" class="square-button bg-transparent fg-white bg-hover-dark no-border">
					<span class="mif-switch"></span>
				</a>
				<?php if ($admin != 0): ?>
					<a href="?admin" target="_blank" title="后台管理" class="square-button bg-transparent fg-white bg-hover-dark no-border">
						<span class="mif-equalizer"></span>
					</a>
				<?php endif; ?>
			</div>
		</div>

		<div class="tile-group double">
			<span class="tile-group-title">题库</span>

			<div class="tile-container">
				<div class="tile-large fg-white" onclick="allProblem()" style="background-image:url(bg/tile/1.jpg);background-size:cover;background-repeat:no-repeat;" data-role="tile">
					<div class="tile-content iconic">
						<span class="icon mif-file-code"></span>
					</div>
					<span class="tile-label">所有题目</span>
				</div>

				<div class="tile fg-white" onclick="showTags()" style="background-image:url(bg/tile/2.jpg);background-size:cover;background-repeat:no-repeat;" data-role="tile">
					<div class="tile-content iconic">
						<span class="icon mif-embed"></span>
					</div>
					<span class="tile-label">试题分类</span>
				</div>

				<div class="tile fg-white" onclick="contest()" style="background-image:url(bg/tile/3.jpg);background-size:cover;background-repeat:no-repeat;" data-role="tile">
					<div class="tile-content iconic">
						<span class="icon mif-list-numbered"></span>
					</div>
					<span class="tile-label">比赛</span>
				</div>
			</div>
		</div>

		<div class="tile-group double">
			<span class="tile-group-title">记录</span>
			<div class="tile-container">
				<div class="tile fg-white" onclick="record()" style="background-image:url(bg/tile/4.jpg);background-size:cover;background-repeat:no-repeat;" data-role="tile">
					<div class="tile-content iconic">
						<span class="icon mif-library"></span>
					</div>
					<span class="tile-label">评测大厅</span>
				</div>
				<div class="tile fg-white" onclick="myRecord()" style="background-image:url(bg/tile/5.jpg);background-size:cover;background-repeat:no-repeat;" data-role="tile">
					<div class="tile-content iconic">
						<span class="icon mif-pencil"></span>
					</div>
					<span class="tile-label">我的评测记录</span>
				</div>
				<div class="tile-large fg-white" style="cursor:default;" data-role="tile">
					<div class="tile-content">
						<div class="carousel" data-role="carousel" data-controls="false">
							<div class="slide" id="myChart" style="width:100%;height:100%;"></div> 
							<div class="slide" id="dojChart" style="width:100%;height:100%;"></div> 
						</div>
					</div>
				</div>
			</div>
		</div>

		<div class="tile-group double">
			<span class="tile-group-title">讨论</span>
			<div class="tile-container">
				<div class="tile-large fg-white" onclick="topic()" style="background-image:url(bg/tile/6.jpg);background-size:cover;background-repeat:no-repeat;" data-role="tile">
					<div class="tile-content iconic">
						<span class="icon mif-bubbles"></span>
					</div>
					<span class="tile-label">所有话题</span>
				</div>
				<div class="tile-wide fg-white" onclick="myTopic()" style="background-image:url(bg/tile/7.jpg);background-size:cover;background-repeat:no-repeat;" data-role="tile">
					<div class="tile-content iconic">
						<span class="icon mif-bubble"></span>
					</div>
					<span class="tile-label">我的话题</span>
				</div>
			</div>
		</div>

	</div>
	
	<div data-role="dialog" id="changeMsg" class="padding20" data-close-button="true" data-overlay="true" data-type="info" data-overlay-color="op-dark">
		<form style="width:500px;" onSubmit="return formSubmit(1);">
			<h2>修改资料</h2>
			<hr class="thin"/><br>
			<div class="input-control text full-size" data-role="datepicker" data-format="yyyy-mm-dd" data-locale="zhCN" data-preset="<?php echo $user -> birth;?>">
				<label style="display:block;">生日</label>
				<input type="text" name="birth" id="birth">
				<button class="button"><span class="mif-calendar"></span></button>
			</div>
			<br><br>
			<div class="input-control text full-size" data-role="input">
				<label for="school">学校</label>
				<input type="text" name="school" id="school" maxlength="100" value="<?php echo $user -> school;?>">
				<button class="button helper-button clear"><span class="mif-cross"></span></button>
			</div>
			<br><br>
			<div class="input-control text full-size" data-role="input">
				<label for="sign">个性签名</label>
				<input type="text" name="sign" id="sign" maxlength="100" value="<?php echo $user -> sign;?>">
				<button class="button helper-button clear"><span class="mif-cross"></span></button>
			</div><br><br>
			<div><!--Form底部Div-->
				<div style="display:inline;">
					<label style="display:block;">性别</label>
					<label class="input-control radio small-check no-margin-bottom">
						<input type="radio" name="sex" value="1" <?php if ($user -> sex) echo "checked";?> >
						<span class="check"></span>
						<span class="caption">男</span>
					</label>&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;
					<label class="input-control radio small-check no-margin-bottom">
						<input type="radio" name="sex" value="0" <?php if (!$user -> sex) echo "checked";?> >
						<span class="check"></span>
						<span class="caption">女</span>
					</label>
				</div>
				<div class="form-actions" style="display:inline;">
					<button type="submit" class="button success place-right">保存修改</button>
				</div>
			</div>
		</form>
	</div>

	<div data-role="dialog" id="changeName" class="padding20" data-close-button="true" data-overlay="true" data-type="info" data-overlay-color="op-dark">
		<form style="width:300px;" onSubmit="return formSubmit(2);">
			<h2>修改用户名</h2>
			<hr class="thin"/><br>
			<div class="input-control password full-size" data-role="input">
				<label for="u_pwd">请先确认你的密码</label>
				<input type="password" name="u_pwd" id="u_pwd" maxlength="20" data-wrong="密码长度应大于6！">
				<button class="button helper-button reveal"><span class="mif-looks"></span></button>
			</div><br><br>
			<div class="input-control text full-size" data-role="input">
				<label for="user_name">新的用户名</label>
				<input type="text" name="user_name" id="user_name" maxlength="16" data-wrong="用户名长度应大于5！" value="<?php echo $uname;?>">
				<button class="button helper-button clear"><span class="mif-cross"></span></button>
			</div>
			<div class="form-actions">
				<button type="submit" class="button success place-right">保存修改</button>
			</div>
		</form>
	</div>

	<div data-role="dialog" id="changePwd" class="padding20" data-close-button="true" data-overlay="true" data-type="info" data-overlay-color="op-dark">
		<form style="width:300px;" onSubmit="return formSubmit(3);">
			<h2>修改密码</h2>
			<hr class="thin"/><br>
			<div class="input-control password full-size" data-role="input">
				<label for="o_pwd">原密码</label>
				<input type="password" name="o_pwd" id="o_pwd" maxlength="20" data-wrong="原密码长度应大于6！">
				<button class="button helper-button reveal"><span class="mif-looks"></span></button>
			</div><br><br>
			<div class="input-control password full-size" data-role="input">
				<label for="n_pwd">新密码</label>
				<input type="password" name="n_pwd" id="n_pwd" maxlength="20" data-wrong="新密码长度应大于6！">
				<button class="button helper-button reveal"><span class="mif-looks"></span></button>
			</div>
			<div class="form-actions">
				<button type="submit" class="button success place-right">保存修改</button>
			</div>
		</form>
	</div>

	<div data-role="dialog" id="changeMail" class="padding20" data-close-button="true" data-overlay="true" data-type="info" data-overlay-color="op-dark">
		<form style="width:300px;" onSubmit="return formSubmit(4);">
			<h2>修改邮箱</h2>
			<hr class="thin"/><br>
			<div class="input-control password full-size" data-role="input">
				<label for="m_pwd">请先确认你的密码</label>
				<input type="password" name="m_pwd" id="m_pwd" maxlength="20" data-wrong="密码长度应大于6！">
				<button class="button helper-button reveal"><span class="mif-looks"></span></button>
			</div><br><br>
			<div class="input-control text full-size" data-role="input">
				<label for="mail">新的邮箱</label>
				<input type="text" name="mail" id="mail" maxlength="40" value="<?php echo $user->mail;?>">
				<button class="button helper-button reveal"><span class="mif-looks"></span></button>
			</div>
			<div class="form-actions">
				<button type="submit" class="button success place-right">保存修改</button>
			</div>
		</form>
	</div>

	<div data-role="dialog" id="tags" data-close-button="true" data-overlay="true" data-overlay-color="op-dark">
		<div id="tagHead">
			<sapn id="noTagRes">可点击下方标签来筛选题目</sapn>
			<sapn id="tagRes">筛选出的题目个数：<a id="tagResNum"></a></sapn>
		</div>
		<div id="pageTags" class="overflow-down" style="width:670px;height:400px;">
			<div data-role="preloader" data-type="metro" style="top:50%;"></div>
		</div>
	</div>

	<div data-role="dialog" id="problems" data-windows-style="true" data-height="100%">
		<div id="wait" data-role="preloader" data-style="dark" style="margin:-32px;top:50%;left:50%;"></div>
		<div id="pcontent" class="grid condensed full-height no-margin">

			<div class="row cells12 full-height">
				<div class="cell colspan2 full-height">
					<ul class="v-menu subdown overflow-down side-list">
						<li><a href="javascript:window.location.hash='',$('#problems').data('dialog').close();"><span class="mif-home icon"></span>回到主页</a></li>
						<li class="divider"></li>
						<li class="menu-title">
							P<span id="nowPid"></span>
							&nbsp;-&nbsp;
							<span id="nowPt"></span>
						</li>
						<li style="background:#FFF">
							<div class="padding10">
								时间限制：<span id="time_limit"></span> ms<br>
								内存限制：<span id="memory_limit"></span> KB
								<div id="problemMsg" style="width:100%;"></div>
							</div>
						</li>
						<li><a href="javascript:showSubmit();"><span class="mif-upload2 icon"></span>提交代码</a></li>
						<li class="divider"></li>
						<li class="menu-title">题目列表</li>
						<div id="problemIndex"></div>
					</ul>
				</div>
				<div class="cell colspan10 overflow-down" id="problem">

					<div class="quote" style="border-color:#CE352C;">
						<h3 class="no-margin-top">题目描述</h3>
						<hr class="ribbed-red">
						<div id="description"><div class="pwait" data-role="preloader" data-style="dark"></div></div>
					</div>
					<div class="quote" style="border-color:#FA6800;">
						<h3 class="no-margin-top">输入格式</h3>
						<hr class="ribbed-orange">
						<div id="inFormat"><div class="pwait" data-role="preloader" data-style="dark"></div></div>
					</div>
					<div class="quote" style="border-color:#E3C800;">
						<h3 class="no-margin-top">输出格式</h3>
						<hr class="ribbed-yellow">
						<div id="outFormat"><div class="pwait" data-role="preloader" data-style="dark"></div></div>
					</div>
					<div class="quote" style="border-color:#60A917;">
						<h3 class="no-margin-top">样例输入</h3>
						<hr class="ribbed-green">
						<div id="sampleIn"><div class="pwait" data-role="preloader" data-style="dark"></div></div>
					</div>
					<div class="quote" style="border-color:#00AFF0;">
						<h3 class="no-margin-top">样例输出</h3>
						<hr class="ribbed-blue">
						<div id="sampleOut"><div class="pwait" data-role="preloader" data-style="dark"></div></div>
					</div>
					<div class="quote" style="border-color:#AA00FF;">
						<h3 class="no-margin-top">数据范围 &amp; 提示</h3>
						<hr class="ribbed-violet">
						<div id="hint"><div class="pwait" data-role="preloader" data-style="dark"></div></div>
					</div>

				</div>
			</div>

		</div>
	</div>

	<div data-role="dialog" id="records" data-windows-style="true" data-height="100%">
		<div id="rcontent" class="grid condensed full-height no-margin">
			<div class="row cells12 full-height">
				<div class="cell colspan3">
					<div class="quote" style="border-color:#FA6800;">
						<button class="image-button warning full-size" onclick="$('#records').data('dialog').close()">
							回到主页<span class="icon mif-home bg-darkOrange"></span>
						</button>
					</div>
					<div class="quote" style="border-color:#60A917;">
						<div class="input-control modern text iconic">
							<input type="text" name="r_pid">
							<span class="label">题目ID</span>
							<span class="informer">在此键入题目ID以便筛选记录</span>
							<span class="placeholder"></span>
							<span class="icon mif-list"></span>
						</div>
						<div class="input-control modern text iconic">
							<input type="text" name="r_uname">
							<span class="label">用户名</span>
							<span class="informer">在此键入用户名以便筛选记录</span>
							<span class="placeholder"></span>
							<span class="icon mif-user"></span>
						</div><br><br>
						<button class="button loading-cube lighten success full-size" onclick="refreshRecord()">
							过滤结果 / 刷新表格
						</button>
					</div>
					<div class="quote" style="border-color:#00AFF0;">
						<a href="?downloadCode" target="_blank">
							<button class="button loading-pulse lighten info full-size">
								打包下载我提交的代码
							</button>
						</a>
					</div>
				</div>
				<div class="cell colspan9 overflow-down">
					<div class="quote" style="border-color:#E3C800;">
						<table class="table striped hovered" id="recordTable">
							<tr id="recordHead">
								<th>运行ID</th>
								<th>状态</th>
								<th>用时(ms)</th>
								<th>内存(KB)</th>
								<th>题目</th>
								<th>语言</th>
								<th>用户</th>
								<th>提交时间</th>
							</tr>
						</table>
						<div id="noRecord" style="display:none;">没有更多记录了</div>
						<div id="moreRecord" onclick="moreRecord()" style="display:none;">点击加载更多记录...</div>
					</div>
				</div>
			</div>
		</div>
	</div>

	<div data-role="dialog" id="topics" data-windows-style="true" data-height="100%">
		<div id="rcontent" class="grid condensed full-height no-margin">
			<div class="row cells4 full-height">
				<div class="cell">
					<div class="quote" style="border-color:#CE352C;">
						<button class="image-button alert full-size" onclick="$('#topics').data('dialog').close()">
							回到主页<span class="icon mif-home bg-darkRed"></span>
						</button>
					</div>
					<div class="quote" style="border-color:#60A917;">
						<div class="input-control modern text iconic full-size">
							<input type="text" name="t_uname">
							<span class="label">用户名</span>
							<span class="informer">在此键入用户名以便筛选话题</span>
							<span class="placeholder"></span>
							<span class="icon mif-user"></span>
						</div><br><br>
						<button class="button loading-cube lighten success full-size" onclick="refreshTopic()">
							过滤话题 / 刷新列表
						</button>
					</div>
					<div class="quote" style="border-color:#FA6800;">
						<button class="button loading-pulse lighten warning full-size" onclick="newTopic()">
							发表新的话题
						</button>
					</div>
				</div>
				<div class="cell colspan3 overflow-down">
					<div class="quote" id="topic" style="border-color:#00AFF0;">
						<h4 id="topicTitle"></h4>
						<hr class="ribbed-blue">
						<div id="tcontent"></div>
						<h5>发表回复：</h5>
						<div class="tabcontrol2" data-role="tabControl">
							<ul class="tabs">
								<li><a href="#md_code">编辑回复</a></li>
								<li><a href="#pre_view" onclick="preview(1)">预览</a></li>
							</ul>
							<div class="frames">
								<div class="frame" id="md_code">
									<pre id="replyText"></pre>
								</div>
								<div class="frame" id="pre_view">
									<div class="overflow-down" id="rep_view"></div>
								</div>
							</div>
						</div>
						<button class="button" id="btn-reply">回复</button>
					</div>
					<div class="quote" id="tlist" style="border-color:#E3C800;">
						<h4 class="no-margin-top">置顶话题</h4>
						<hr class="ribbed-yellow">
						<div id="topList"></div>
						<h4 class="no-margin-top">普通话题</h4>
						<hr class="ribbed-yellow">
						<div id="topicList"></div>
						<div id="moreTopic" onclick="moreTopic()" style="display:none;">点击加载更多话题...</div>
					</div>
				</div>
			</div>
		</div>
	</div>

	<div data-role="dialog" id="newTopic" data-close-button="true" data-width="60%">
		<div class="padding20">
			<h4 class="no-margin-top">发表新话题</h4>
			<label>标题：</label>
			<div class="input-control text" style="width:60%;">
				<input type="text" name="topicTitle">
			</div><br>
			<div class="tabcontrol2" data-role="tabControl">
				<ul class="tabs">
					<li><a href="#mdcode">话题正文</a></li>
					<li><a href="#preview" onclick="preview(2)">预览</a></li>
				</ul>
				<div class="frames">
					<div class="frame" id="mdcode">
						<pre id="topicText"></pre>
					</div>
					<div class="frame" id="preview">
						<div class="overflow-down" id="topic_view"></div>
					</div>
				</div>
			</div>
			<button class="button" id="btn-topic">发表话题</button>
		</div>
	</div>

	<div data-role="dialog" id="submit" data-close-button="true" data-width="60%">
		<div class="padding20">
			<h4 class="no-margin-top">
				P<span id="subPid"></span>
			</h4>
			<pre id="subCode"></pre>
			<hr>
			<label>语言：</label>
			<div class="input-control select">
				<select id="subLan">
					<option value="0">C</option>
					<option value="1">C++</option>
					<option value="2">Pascal</option>
					<option value="3">Python</option>
				</select>
			</div>
			<button class="button place-right" id="btn-submit" onclick="submit()">提交</button>
		</div>
	</div>

	<div data-role="dialog" id="result" data-close-button="true" data-width="60%">
		<div class="padding20">
			<h3 class="no-margin-top">
				<b>RunID #<span id="run_id"></span></b><small> - P<span id="run_pid"></span></small>
			</h3><hr>
			<div id="points_res"></div>
			<div id="run_code"></div>
		</div>
	</div>

	<script>
		$("#myChart").highcharts({
			chart: {
				type: 'pie',
				options3d: {
					enabled: true,
					alpha: 45,
					beta: 0
				}
			},
			credits: {
				enabled: false
			},
			title: {
				text: '我的提交概况'
			},
			tooltip: {
				pointFormat: '<span style="color:{point.color};">\u25CF</span>{series.name}: <b>{point.y}</b><br><span style="color:{point.color};">\u25CF</span>占比: <b>{point.percentage:.2f}%</b>'
			},
			plotOptions: {
				pie: {
					allowPointSelect: true,
					cursor: 'pointer',
					depth: 20,
					dataLabels: {
						enabled: true,
						format: '{point.name}'
					}
				}
			},
			series: [{
				<?php
					$subres = array_pad([], 10, 0);
					$res = mysql_query("SELECT * FROM `submit` WHERE `uid` = $uid");
					while ($a = mysql_fetch_object($res))
						$subres[$a->res]++;
					mysql_free_result($res);
				?>
				type: 'pie',
				name: '次数',
				data: [
					['正确', <?php echo $subres[1]; ?>],
					['答案错误', <?php echo $subres[2]; ?>],
					['时间超限', <?php echo $subres[3]; ?>],
					['内存超限', <?php echo $subres[4]; ?>],
					['运行错误', <?php echo $subres[5]; ?>],
					['编译错误', <?php echo $subres[6]; ?>]
				],
				colors: ['#66CC66', '#FF6666', '#FFFF66', '#66CCFF', '#FF9900', '#CCCCCC']
			}]
		});

		$('#dojChart').highcharts({
			chart: {
				type: 'line'
			},
			credits: {
				enabled: false
			},
			title: {
				text: 'DOJ提交概况'
			},
			subtitle: {
				text: '近一周数据'
			},
			xAxis: {
				categories: [<?php
					$recentDates = [];
					for ($i = 6; $i > 0; $i--)
					{
						$day = date("Y-m-d", strtotime("-$i day"));
						$recentDates []= $day;
						echo "'$day',";
					}
					$recentDates []= date("Y-m-d");
					echo date("'Y-m-d'");
				?>]
			},
			yAxis: {
				title: {
					text: '次数'
				}
			},
			tooltip: {
				enabled: true,
				formatter: function() {
					var item = this.series.name;
					if (this.series.color == '#66CC66')
						item = 'AC';
					return '<b>'+ item +'数</b><br/>'+ this.x +': '+ this.y;
				}
			},
			series: [{
				name: 'AC\u3000\u3000\u3000\u3000\u3000',
				data: [<?php
					function getACNum($d)
					{
						$res = mysql_query("SELECT COUNT(*) FROM `submit` WHERE LEFT(`submit_time`, 10) = '$d' AND `res` = 1 ");
						$a = mysql_fetch_array($res);
						mysql_free_result($res);
						return $a[0];
					}
					for ($i = 0; $i < 6; $i++)
						echo getACNum($recentDates[$i]).",";
					echo getACNum($recentDates[6]);
				?>],
				color: '#66CC66'
			}, {
				name: '总提交',
				data: [<?php
					function getNum($d)
					{
						$res = mysql_query("SELECT COUNT(*) FROM `submit` WHERE LEFT(`submit_time`, 10) = '$d' ");
						$a = mysql_fetch_array($res);
						mysql_free_result($res);
						return $a[0];
					}
					for ($i = 0; $i < 6; $i++)
						echo getNum($recentDates[$i]).",";
					echo getNum($recentDates[6]);
				?>],
				color: '#FFFF66'
			}]
		});
	</script>
</body>
</html>
