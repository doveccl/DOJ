<!DOCTYPE html>
<html>
	<head>
		<link type="text/css" rel="stylesheet" href="//cdn.bootcss.com/materialize/0.97.5/css/materialize.min.css">

		<meta name="viewport" content="width=device-width, initial-scale=1.0">
		<meta name="description" content="DOJ, an online judge for OIers">
		<meta name="keywords" content="OI, Online Judge, algorithm">
		<meta name="author" content="Doveccl">
		<meta charset="UTF-8">

		<link rel='shortcut icon' type='image/x-icon' href='favicon.ico'>
		<title><?=str::$err?></title>
	</head>

	<body class="grey lighten-3">
		<div class="container">
			<div class="hide-on-small-only" id="offset"></div>
			<div class="row" id="main">
				<div class="col s12 m8 offset-m2">
					<div class="card hoverable red darken-3">
						<div class="card-content white-text">
							<span class="card-title"><?=str::$err?></span>
							<p><?=str::$err_code?>: <?=$_e->getCode()?></p>
							<p id="msg"><?=str::$err_msg?>: <?=$_e->getMessage()?></p>
						</div>
						<div class="card-action">
							<a href="mailto:<?=DOJ::$admin_mail?>"><?=str::$mail_to_admin?></a>
							<a href="javascript:;" id="copy"><?=str::$copy_err_msg?></a>
							<a class="tooltipped" data-tooltip="<?=str::$copy_ok?>"></a>
						</div>
					</div>
				</div>
			</div>
		</div>

		<script src="//cdn.bootcss.com/jquery/1.11.3/jquery.min.js"></script>
		<script src="//cdn.bootcss.com/zeroclipboard/2.2.0/ZeroClipboard.min.js"></script>
		<script src="//cdn.bootcss.com/materialize/0.97.5/js/materialize.min.js"></script>
		<script>
			$(document).ready(function() {
				var tmp = $(window).height() - $("#main").height();
				$("#offset").height(tmp * 0.3);

				ZeroClipboard.config({
					swfPath: "//cdn.bootcss.com/zeroclipboard/2.2.0/ZeroClipboard.swf"
				});
				var client = new ZeroClipboard($("#copy"));
				client.on("copy", function (event) {
					var clipboard = event.clipboardData;
					clipboard.setData("text/plain", $("#msg").html());
					Materialize.toast("<?=str::$copy_ok?>", 2000);
				});
			});
		</script>
	</body>
</html>
<?php die(); ?>