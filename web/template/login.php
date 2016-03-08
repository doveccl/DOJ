<!DOCTYPE html>
<html>
	<head>
		<link type="text/css" rel="stylesheet" href="//cdn.bootcss.com/material-design-icons/2.1.3/iconfont/material-icons.css">
		<link type="text/css" rel="stylesheet" href="//cdn.bootcss.com/materialize/0.97.5/css/materialize.min.css">

		<meta name="viewport" content="width=device-width, initial-scale=1.0">
		<meta name="description" content="DOJ, an online judge for OIers">
		<meta name="keywords" content="OI, Online Judge, algorithm">
		<meta name="author" content="Doveccl">
		<meta charset="UTF-8">

		<link rel='shortcut icon' type='image/x-icon' href='favicon.ico'>
		<title><?=str::$login_title?></title>
	</head>

	<body class="grey lighten-3 hide">
		<div class="container">
			<div class="hide-on-small-only" id="offset"></div>
			<div class="row" id="main">
				<div class="col s12 m8 offset-m2">
					
					<div class="card hoverable">
						<div class="card-content grey-text">
							<span class="card-title"><?=str::$login_title?></span>
							<div class="input-field">
								<i class="material-icons prefix">account_circle</i>
								<input id="user" type="text" class="validate">
								<label for="user"><?=str::$name_or_mail?></label>
							</div>
							<div class="input-field">
								<i class="material-icons prefix">vpn_key</i>
								<input id="password" type="password" class="validate">
								<label for="password"><?=str::$password?></label>
							</div>
							<div class="switch">
								<label>
									<b><?=str::$keep_login?></b>
									<input type="checkbox" id="rem_pwd" checked>
									<span class="lever"></span>
								</label>
							</div>
						</div>
						<div class="card-action">

							<button class="waves-effect waves-light btn" id="login">
								<i class="material-icons right hide-on-small-only">send</i>
								<?=str::$login?>
							</button>
							<span>&nbsp;&nbsp;&nbsp;</span>
							<a class="waves-effect waves-dark" id="apply" href="#reg-modal"><?=str::$register?></a>
							<a class="waves-effect waves-dark" id="forget"><?=str::$forget_password?></a>
							
							<div class="preloader-wrapper small right hide" id="wait">
								<div class="spinner-layer spinner-blue">
									<div class="circle-clipper left">
										<div class="circle"></div>
									</div>
									<div class="gap-patch">
										<div class="circle"></div>
									</div>
									<div class="circle-clipper right">
										<div class="circle"></div>
									</div>
								</div>
								<div class="spinner-layer spinner-red">
									<div class="circle-clipper left">
										<div class="circle"></div>
									</div>
									<div class="gap-patch">
										<div class="circle"></div>
									</div>
									<div class="circle-clipper right">
										<div class="circle"></div>
									</div>
								</div>
								<div class="spinner-layer spinner-yellow">
									<div class="circle-clipper left">
										<div class="circle"></div>
									</div>
									<div class="gap-patch">
										<div class="circle"></div>
									</div>
									<div class="circle-clipper right">
										<div class="circle"></div>
									</div>
								</div>
								<div class="spinner-layer spinner-green">
									<div class="circle-clipper left">
										<div class="circle"></div>
									</div>
									<div class="gap-patch">
										<div class="circle"></div>
									</div>
									<div class="circle-clipper right">
										<div class="circle"></div>
									</div>
								</div>
							</div>

						</div>
					</div>

					<div class="card-panel hoverable" id="error">
						<b class="red-text" id="err_msg"></b>
					</div>

				</div>
			</div>
		</div>

		<div id="reg-modal" class="modal modal-fixed-footer">
			<div class="modal-content grey-text">
				<h4><?=str::$register_title?></h4>
				
				<div class="input-field">
					<i class="material-icons prefix">person</i>
					<input id="name" type="text" class="validate">
					<label for="name"><?=str::$name?></label>
				</div>
				<div class="input-field">
					<i class="material-icons prefix">email</i>
					<input id="mail" type="email" class="validate">
					<label for="mail"><?=str::$mail?></label>
				</div>
				<div class="input-field">
					<i class="material-icons prefix">lock</i>
					<input id="pwd" type="password" class="validate">
					<label for="pwd"><?=str::$password?></label>
				</div>
				<div class="input-field">
					<i class="material-icons prefix">replay</i>
					<input id="re_pwd" type="password" class="validate">
					<label for="re_pwd"><?=str::$repeat_password?></label>
				</div>
				<div class="input-field">
					<i class="material-icons prefix">school</i>
					<input id="school" type="text" class="validate">
					<label for="school"><?=str::$school?></label>
				</div>
				<div class="input-field">
					<i class="material-icons prefix">create</i>
					<textarea id="sign" class="materialize-textarea"></textarea>
					<label for="sign"><?=str::$sign?></label>
				</div>
				<div class="input-field">
					<i class="material-icons prefix">verified_user</i>
					<textarea id="key" class="materialize-textarea"></textarea>
					<label for="key"><?=str::$invite_key?></label>
				</div>

			</div>
			<div class="modal-footer grey-text">
				<button class="modal-action waves-effect btn" id="register">
					<i class="material-icons right hide-on-small-only">send</i>
					<?=str::$register?>
				</button>
				<a class="modal-action modal-close btn-flat waves-effect" id="cancel">
					<b class="orange-text"><?=str::$cancel?></b>
				</a>
			</div>
		</div>

		<script src="//cdn.bootcss.com/jquery/1.11.3/jquery.min.js"></script>
		<script src="//cdn.bootcss.com/jquery-cookie/1.4.1/jquery.cookie.min.js"></script>
		<script src="//cdn.bootcss.com/materialize/0.97.5/js/materialize.min.js"></script>

		<script>
			function enable_submit() {
				$("#login").removeClass("disabled");
				$("#login").attr("disabled", false);
				$("#wait").removeClass("active");
				$("#wait").addClass("hide");
			}

			function disable_submit() {
				$("#login").addClass("disabled");
				$("#login").attr("disabled", true);
				$("#wait").addClass("active");
				$("#wait").removeClass("hide");
			}

			function show_error(err) {
				$("#err_msg").html(err);
				$("#error").slideDown("slow", enable_submit);
			}

			$(document).ready(function() {
				$("#error").hide();
				$("#offset").height($(window).height());
				$("body").removeClass("hide");
				var tmp = $(window).height() - $("#main").height();
				$("#offset").animate({height: tmp * 0.3}, "slow");

				$("#login").click(function() {
					disable_submit();
					$("#error").hide();

					$.ajax({
						type: "post",
						url: "?login",
						data: {
							user: $("#user").val(),
							password: $("#password").val()
						},
						success: function(d) {
							if (d.success) {
								if ($("#rem_pwd").is(":checked"))
									$.cookie("DOJSS", d.dojss, {expires: 30});
								else
									$.cookie("DOJSS", d.dojss);
								window.location.reload();
							} else
								show_error(d.error);
						},
						error: function(e) {
							console.log(e);
							show_error("<?=str::$post_failed?>");
						},
						dataType: "json"
					});
				});

				$("#register").click(function() {
					$("#cancel").hide();
					$("#register").addClass("disabled");
					$("#register").attr("disabled", true);
					
					$.ajax({
						type: "post",
						url: "?register",
						data: {
							name: $("#name").val(),
							mail: $("#mail").val(),
							pwd: $("#pwd").val(),
							re_pwd: $("#re_pwd").val(),
							school: $("#school").val(),
							sign: $("#sign").val(),
							key: $("#key").val()
						},
						success: function(d) {
							if (d.success) {
								$("#cancel").show();
								$("#reg-modal").closeModal();
								$("#user").val($("#name").val());
								$("#user:first").focus();
								show_error("<b class='green-text'><?=str::$finish_register?></b>");
							} else {
								Materialize.toast(d.error, 3000);
								$("#register").removeClass("disabled");
								$("#register").attr("disabled", false);
								$("#cancel").show();
							}
						},
						error: function(e) {
							Materialize.toast("<?=str::$post_failed?>", 3000);
							$("#register").removeClass("disabled");
							$("#register").attr("disabled", false);
							$("#cancel").show();
						},
						dataType: "json"
					});
				});

				$("#apply").leanModal({
					dismissible: false,
					ready: function() {
						$("#error").hide();
					}
				});

				$("#forget").click(function() {
					Materialize.toast("<?=str::$contact_admin?>", 3000);
				});

				$("#password").bind('keypress', function(e) {
					if(e.keyCode == '13')
						$("#login").trigger('click');
				});
			});
		</script>
	</body>
</html>
<?php die(); ?>