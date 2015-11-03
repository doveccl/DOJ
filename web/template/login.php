<!DOCTYPE html>
<html>
<head lang="zh-cn">
	<meta charset="UTF-8">
	<meta http-equiv="X-UA-Compatible" content="IE=edge">	 
	<meta name="viewport" content="width=device-width, initial-scale=1.0,
maximum-scale=1.0, user-scalable=no">	 
	<meta name="description" content="DOJ, an online judge for OIers.">
	<meta name="keywords" content="OI, Online Judge, algorithm">
	<meta name="author" content="Doveccl">

	<link rel='shortcut icon' type='image/x-icon' href='favicon.ico' />

	<title>Login - DOJ</title>

	<link href="css/metro.min.css" rel="stylesheet">
	<link href="css/metro-icons.min.css" rel="stylesheet">

	<script src="//cdn.bootcss.com/jquery/1.11.3/jquery.min.js"></script>
	<script src="js/metro.min.js"></script>
 
	<style>
		.login-form {
			width: 400px;
			position: fixed;
			top: 50%;
			left: 50%;
			margin-left: -200px;
			background-color: #ffffff;
			opacity: 0;
			-webkit-transform: scale(.8);
			transform: scale(.8);
		}
	</style>

	<script>
		$(function(){
			var form = $(".login-form");

			form.css({
				opacity: 1,
				"margin-top": - form.height() * 0.618,
				"-webkit-transform": "scale(1)",
				"transform": "scale(1)",
				"-webkit-transition": ".5s",
				"transition": ".5s"
			});
		});
	</script>
</head>

<body class="bg-blue">
	<div class="login-form padding20 block-shadow">
		<form method="post" action="?submitlogin" data-role="validator" data-hint-mode="line">
			<h1 class="text-light">登陆 -- DOJ</h1>
			<hr class="thin"/><br>
			<div class="input-control text full-size" data-role="input">
				<label for="user_login">用户名 / 邮箱：</label>
				<input type="text" name="user" id="user_login" data-validate-func="minlength" data-validate-arg="5" data-validate-hint="不得少于5个字符！" maxlength="40" <?php if (isset($user)) echo "value='$user'";?> >
				<span class="input-state-error mif-warning" style="right:8px;"></span>
				<span class="input-state-success mif-checkmark" style="right: 8px;"></span>
				<button class="button helper-button clear"><span class="mif-cross"></span></button>
			</div>
			<br><br>
			<div class="input-control password full-size" data-role="input">
				<label for="user_password">密码：</label>
				<input type="password" name="password" id="user_password" data-validate-func="minlength" data-validate-arg="6" data-validate-hint="不得少于6个字符！" maxlength="20" <?php if (isset($pwd)) echo "value='$pwd'";?> >
				<span class="input-state-error mif-warning" style="right:8px;"></span>
				<span class="input-state-success mif-checkmark" style="right: 8px;"></span>				
				<button class="button helper-button reveal"><span class="mif-looks"></span></button>
			</div>
			<br>
			<label class="switch">
				<span class="caption">记住我：</span>
				<input type="checkbox" name="remember" checked>
				<span class="check"></span>
			</label>
			<br><br>
			<div class="form-actions">
				<button type="submit" class="button primary">登陆</button>
				<button type="button" class="button link loginhelp" data-role="popover" data-popover-mode="click" data-popover-position="right" data-popover-background="bg-green" data-popover-color="fg-white" data-popover-text="忘记密码请<a href='?lostpassword'><b>点这里</b></a><br><br>注册账号请<a href='?register'><b>点这里</b></a>"><span class="mif-question mif-lg"></span></button>
				<?php
					if (isset($error))
						echo "<b style='color:red'>$error</b><br>";
			   		if (isset($hint))
						echo "<b style='color:green'>$hint</b><br>";
				?>
			</div>
		</form>
	</div>
</body>
</html>