<!DOCTYPE html>
<html>
<head lang="zh-cn">
	<meta charset="UTF-8">
	<meta http-equiv="X-UA-Compatible" content="IE=edge">
	<meta name="viewport" content="width=device-width, initial-scale=1.0, maximum-scale=1.0, user-scalable=no">
	<meta name="description" content="DOJ, an online judge for OIers.">
	<meta name="keywords" content="OI, Online Judge, algorithm">
	<meta name="author" content="Doveccl">

	<link rel='shortcut icon' type='image/x-icon' href='favicon.ico' />

	<title>Register - DOJ</title>

	<link href="css/metro.min.css" rel="stylesheet">
	<link href="css/metro-icons.min.css" rel="stylesheet">

	<script src="//cdn.bootcss.com/jquery/1.11.3/jquery.min.js"></script>
	<script src="js/metro.min.js"></script>
 
	<style>
		.reg-form {
			width: 600px;
			position: fixed;
			top: 50%;
			left: 50%;
			margin-left: -300px;
			background-color: #ffffff;
			opacity: 0;
			-webkit-transform: scale(.8);
			transform: scale(.8);
		}
	</style>

	<script>
		$(function(){
			var form = $(".reg-form");

			form.css({
				opacity: 1,
				"margin-top": - form.height() * 0.57,
				"-webkit-transform": "scale(1)",
				"transform": "scale(1)",
				"-webkit-transition": ".5s",
				"transition": ".5s"
			});
		});
	</script>
</head>

<body class="bg-green">
	<div class="reg-form padding20 block-shadow">
		<form method="post" action="?submitregister" data-role="validator" data-hint-mode="line">
			<a class="mif-arrow-left mif-lg button cycle-button no-margin-bottom" href="/"></a>
			<h1 class="text-light v-align-bottom no-margin-bottom" style="display:inline-block;">注册 -- DOJ</h1>
			<hr class="thin"/><br>
			<div class="input-control text full-size" data-role="input">
				<label for="user_login">用户名：</label>
				<input type="text" name="name" id="user_login" data-role="popover" data-popover-position="right" data-popover-text="用户名要求5-16个字符<br>只可包含字母、数字和下划线" data-popover-background="bg-blue" data-popover-color="fg-white" data-popover-mode="focus" data-validate-func="minlength" data-validate-arg="5" data-validate-hint="不得少于5个字符！" maxlength="16" <?php if (isset($name)) echo "value='$name'";?> >
				<span class="input-state-error mif-warning" style="right:8px;"></span>
				<span class="input-state-success mif-checkmark" style="right: 8px;"></span>
				<button class="button helper-button clear"><span class="mif-cross"></span></button>
			</div>
			<br><br>
			<div class="input-control text full-size" data-role="input">
				<label for="user_email">邮箱：</label>
				<input type="text" name="email" id="user_email" data-role="popover" data-popover-position="right" data-popover-text="邮箱用于身份验证和密码找回<br>管理员根据邮箱发放邀请码" data-popover-background="bg-blue" data-popover-color="fg-white" data-popover-mode="focus" data-validate-func="email" data-validate-hint="邮箱格式不正确！" <?php if (isset($mail)) echo "value='$mail'";?> >
				<span class="input-state-error mif-warning" style="right:8px;"></span>
				<span class="input-state-success mif-checkmark" style="right: 8px;"></span>
				<button class="button helper-button clear"><span class="mif-cross"></span></button>
			</div>
			<br><br>
			<div class="input-control password full-size" data-role="input">
				<label for="user_password">密码：</label>
				<input type="password" name="password" id="user_password" data-role="popover" data-popover-position="right" data-popover-text="密码不可少于6位" data-popover-background="bg-blue" data-popover-color="fg-white" data-popover-mode="focus" data-validate-func="minlength" data-validate-arg="6" data-validate-hint="不得少于6个字符！" maxlength="20" <?php if (isset($pwd)) echo "value='$pwd'";?> >
				<span class="input-state-error mif-warning" style="right:8px;"></span>
				<span class="input-state-success mif-checkmark" style="right: 8px;"></span>
				<button class="button helper-button reveal">
				<span class="mif-looks"></span></button>
			</div>
			<br><br>
			<label for="user_key">邀请码：</label>
			<div class="input-control textarea full-size">
				<textarea name="key" id="user_key" data-role="popover" data-popover-position="right" data-popover-text="邀请码需由管理员发放" data-popover-background="bg-blue" data-popover-color="fg-white" data-popover-mode="focus" data-validate-func="required" data-validate-hint="邀请码不可为空！"><?php if (isset($key)) echo $key;?></textarea>
				<span class="input-state-error mif-warning" style="right:8px;"></span>
				<span class="input-state-success mif-checkmark" style="right: 8px;"></span>
			</div><br><br>
			<div class="form-actions">
				<button type="submit" class="button primary">注册</button>
				&nbsp;&nbsp;&nbsp;&nbsp;
				<?php
					if (isset($error))
						echo "<b style='color:red'>$error</b><br>";
				?>
			</div>	
		</form>
	</div>
</body>
</html>