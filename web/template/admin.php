<?php
	require_once("query/message.php");

	$DOJSS = $_COOKIE['DOJSS'];
	$user = checkDOJSS($DOJSS);

	if ($user->admin == 0)
		die(str_replace("Forbidden", $err['notAdmin'], $forbidden));
?>
<!DOCTYPE html>
<html>
	<head>
		<meta charset="utf-8" />
		<title>Admin - DOJ</title>
		<meta http-equiv="X-UA-Compatible" content="IE=edge">
		<meta name="viewport" content="width=device-width, initial-scale=1.0, maximum-scale=1.0, user-scalable=no">
		<meta name="description" content="DOJ Manage UI.">
		<meta name="keywords" content="OI, Online Judge, algorithm">
		<meta name="author" content="Doveccl">

		<link rel='shortcut icon' type='image/x-icon' href='favicon.ico' />

		<link href="//cdn.bootcss.com/highlight.js/8.9.1/styles/github.min.css" rel="stylesheet">
		<link href="css/metro-icons.min.css" rel="stylesheet">
		<link href="css/metro.min.css" rel="stylesheet">
		<link href="css/doj-admin.css" rel="stylesheet">

		<!--[if lt IE9]> 
			<script src="//cdn.bootcss.com/html5shiv/r29/html5.min.js"></script>
		<![endif]-->

		<script src="//cdn.bootcss.com/jquery/1.11.3/jquery.min.js"></script>
		<script src="//cdn.bootcss.com/marked/0.3.5/marked.min.js"></script>
		<script src="//cdn.bootcss.com/highlight.js/8.9.1/highlight.min.js"></script>
		<script src="//cdn.bootcss.com/mathjax/2.6.0-beta.1/MathJax.js?config=TeX-AMS-MML_SVG"></script>
		<script src="js/metro.min.js"></script>
		<script src="js/Ace/ace.js"></script>
		<script src="js/doj-admin.js"></script>
	</head>
	<body>
		<div class="tabcontrol2" data-role="tabControl">
			<ul class="tabs">
				<li><a href="#home">首页</a></li>
				<li><a href="#problems">题目管理</a></li>
				<li><a href="#addProblem">添加题目</a></li>
				<li><a href="#datas">数据管理</a></li>
				<li><a href="#getKey">生成邀请码</a></li>
				<li><a href="#reJudge">代码重判</a></li>
			</ul>
			<div class="frames">
				<div class="frame" id="home">
					<div style="text-align:center;">
						<h2>欢迎使用DOJ后台管理程序</h2>
						<br><br><br><br><br>
						<h4>你可以通过点击上方标签来选择需要管理的项目</h4>
					</div>
				</div>
				<div class="frame" id="problems">
					题目索引（<b>需要隐藏题目请在题目的标题前加上一个</b><code>$</code>）：<div class="pagination rounded" id="pindex"></div>
					<div id="plist"></div>
					<div id="pchange" style="display:none;">
						<h4>
							修改 - P<span id="pid"></span>&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;
							<button class="button" onclick="saveChangeProblem()">点此保存修改</button>
						</h4>
						<table>
							<tr>
								<th><label for="ptitle">标题：</label></th>
								<th><label for="ptime">时间限制 (ms)：</label></th>
								<th><label for="pmem">内存限制 (KB)：</label></th>
								<th><label for="ptags">标签（用|隔开)</label></th>
							</tr>
							<tr>
								<td><div class="input-control text full-size" data-role="input">
									<input type="text" id="ptitle">
								</div></td>
								<td><div class="input-control text full-size" data-role="input">
									<input type="text" id="ptime">
								</div></td>
								<td><div class="input-control text full-size" data-role="input">
									<input type="text" id="pmem">
								</div></td>
								<td><div class="input-control text full-size" data-role="input">
									<input type="text" id="ptags">
								</div></td>
							</tr>
						</table>
						<div class="tabcontrol2" data-role="tabControl">
							<ul class="tabs">
								<li><a href="#mpdes">题目描述</a></li>
								<li><a href="#ppdes" onclick="preview(1,0)">预览</a></li>
							</ul>
							<div class="frames">
								<div class="frame" id="mpdes">
									<pre class="ace" id="pdes"></pre>
								</div>
								<div class="frame" id="ppdes">
									<div class="overflow-down" id="pdesview"></div>
								</div>
							</div>
						</div>

						<div class="tabcontrol2" data-role="tabControl">
							<ul class="tabs">
								<li><a href="#mpin">输入格式</a></li>
								<li><a href="#ppin" onclick="preview(1,1)">预览</a></li>
							</ul>
							<div class="frames">
								<div class="frame" id="mpin">
									<pre class="ace" id="pin"></pre>
								</div>
								<div class="frame" id="ppin">
									<div class="overflow-down" id="pinview"></div>
								</div>
							</div>
						</div>

						<div class="tabcontrol2" data-role="tabControl">
							<ul class="tabs">
								<li><a href="#mpout">输出格式</a></li>
								<li><a href="#ppout" onclick="preview(1,2)">预览</a></li>
							</ul>
							<div class="frames">
								<div class="frame" id="mpout">
									<pre class="ace" id="pout"></pre>
								</div>
								<div class="frame" id="ppout">
									<div class="overflow-down" id="poutview"></div>
								</div>
							</div>
						</div>

						<div class="tabcontrol2" data-role="tabControl">
							<ul class="tabs">
								<li><a href="#mphint">数据范围 &amp; 提示</a></li>
								<li><a href="#pphint" onclick="preview(1,3)">预览</a></li>
							</ul>
							<div class="frames">
								<div class="frame" id="mphint">
									<pre class="ace" id="phint"></pre>
								</div>
								<div class="frame" id="pphint">
									<div class="overflow-down" id="phintview"></div>
								</div>
							</div>
						</div>
					</div>
				</div>
				<div class="frame" id="addProblem">
					<h4>
						添加题目 - P<span id="npid"></span>&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;
						<button class="button" onclick="addProblem()">点此添加题目</button>
					</h4>
					<table>
						<tr>
							<th><label for="nptitle">标题：</label></th>
							<th><label for="nptime">时间限制 (ms)：</label></th>
							<th><label for="npmem">内存限制 (KB)：</label></th>
							<th><label for="nptags">标签（用|隔开)</label></th>
						</tr>
						<tr>
							<td><div class="input-control text full-size" data-role="input">
								<input type="text" id="nptitle">
							</div></td>
							<td><div class="input-control text full-size" data-role="input">
								<input type="text" id="nptime">
							</div></td>
							<td><div class="input-control text full-size" data-role="input">
								<input type="text" id="npmem">
							</div></td>
							<td><div class="input-control text full-size" data-role="input">
								<input type="text" id="nptags">
							</div></td>
						</tr>
					</table>
					<div class="tabcontrol2" data-role="tabControl">
						<ul class="tabs">
							<li><a href="#mndes">题目描述</a></li>
							<li><a href="#pndes" onclick="preview(2,0)">预览</a></li>
						</ul>
						<div class="frames">
							<div class="frame" id="mndes">
								<pre class="ace" id="ndes"></pre>
							</div>
							<div class="frame" id="pndes">
								<div class="overflow-down" id="ndesview"></div>
							</div>
						</div>
					</div>

					<div class="tabcontrol2" data-role="tabControl">
						<ul class="tabs">
							<li><a href="#mnin">输入格式</a></li>
							<li><a href="#pnin" onclick="preview(2,1)">预览</a></li>
						</ul>
						<div class="frames">
							<div class="frame" id="mnin">
								<pre class="ace" id="nin"></pre>
							</div>
							<div class="frame" id="pnin">
								<div class="overflow-down" id="ninview"></div>
							</div>
						</div>
					</div>

					<div class="tabcontrol2" data-role="tabControl">
						<ul class="tabs">
							<li><a href="#mnout">输出格式</a></li>
							<li><a href="#pnout" onclick="preview(2,2)">预览</a></li>
						</ul>
						<div class="frames">
							<div class="frame" id="mnout">
								<pre class="ace" id="nout"></pre>
							</div>
							<div class="frame" id="pnout">
								<div class="overflow-down" id="noutview"></div>
							</div>
						</div>
					</div>

					<div class="tabcontrol2" data-role="tabControl">
						<ul class="tabs">
							<li><a href="#mnhint">数据范围 &amp; 提示</a></li>
							<li><a href="#pnhint" onclick="preview(2,3)">预览</a></li>
						</ul>
						<div class="frames">
							<div class="frame" id="mnhint">
								<pre class="ace" id="nhint"></pre>
							</div>
							<div class="frame" id="pnhint">
								<div class="overflow-down" id="nhintview"></div>
							</div>
						</div>
					</div>
				</div>
				<div class="frame no-padding" id="datas">
					<font color="red">
						请不要随意删除以题号命名的文件夹，
						上传数据应上传至题号对应的目录，
						上传数据类型只能是<b> %d.in </b>和<b> %d.out </b>，
						此处的%d指代数字，样例数据的名字只能为<b> 0.in </b>和<b> 0.out </b>，
						上传的数据必须一个<b> %d.in </b>对应<b> %d.out </b>，
						并且数据命名应为从0开始的连续的数字，（例如 0.in,0.out | 1.in,1.out | 2.in,2.out | ...）
					</font>
				</div>
				<div class="frame" id="getKey">
					<div class="input-control text" id="mail-input" data-role="input">
						<label fpr="mail">注册使用的邮箱：</label>
						<input type="text" id="mail">
						<button class="button" onclick="getKey()"><span class="mif-key"></span></button>
					</div><br>
					<label class="input-control checkbox small-check">
					    <input type="checkbox" id="setAdmin">
					    <span class="check"></span>
					    <span class="caption">设为管理员（拥有后台管理权限）</span>
					</label><br><br>
					<textarea id="key"></textarea>
				</div>
				<div class="frame" id="reJudge">
					<div class="input-control text" data-role="input">
						<label fpr="mail">运行ID：</label>
						<input type="text" id="run_id">
						<button class="button" onclick="reJudge(1,'#run_id')"><span class="mif-loop2"></span></button>
					</div>
					<div class="input-control text" data-role="input">
						<label fpr="mail">题目ID：</label>
						<input type="text" id="pro_id">
						<button class="button" onclick="reJudge(2,'#pro_id')"><span class="mif-loop2"></span></button>
					</div>
				</div>
			</div>
		</div>
		<iframe class="full-size" id="data-iframe" frameborder="no" border="0"></iframe>
	</body>
</html>
