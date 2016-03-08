<!DOCTYPE html>
<html>
	<head>
		<link href="//cdn.bootcss.com/bootstrap/3.3.6/css/bootstrap.min.css" rel="stylesheet">

		<meta name="viewport" content="width=device-width, initial-scale=1">
		<meta http-equiv="x-ua-compatible" content="ie=edge">

		<title>Admin - DOJ</title>
		<meta charset="utf8">

		<style type="text/css">
			div {
				margin-bottom: 5px;
			}	
		</style>
	</head>
	<body>
		<div class="container">
			<h2>Admin DOJ</h2>

			<h4>Generate Invitation Code</h4>
			<div class="row">
				<div class="input-group col-sm-12 col-md-6">
					<input type="text" class="form-control" id="mail" placeholder="Email">
					<span class="input-group-btn">
						<button class="btn btn-default" type="button" id="getkey">Generate</button>
					</span>
				</div>
			</div>
			<div class="row">
				<div class="col-sm-12 col-md-6">
					<span id="key" style="word-break:break-all;"></span>
				</div>
			</div><hr>

			<h4>Rejudge Code</h4>
			<div class="row">
				<div class="input-group col-sm-12 col-md-6">
					<input type="text" class="form-control" id="rid" placeholder="Run ID">
					<span class="input-group-btn">
						<button class="btn btn-default" type="button" id="rerid">Rejudge</button>
					</span>
				</div>
			</div>
			<div class="row">
				<div class="input-group col-sm-12 col-md-6">
					<input type="text" class="form-control" id="pid" placeholder="Problem ID">
					<span class="input-group-btn">
						<button class="btn btn-default" type="button" id="repid">Rejudge</button>
					</span>
				</div>
			</div><hr>

			<h4>Upload Images</h4>
			<div class="row">
				<button class="btn btn-default" type="button" id="upload-img">Choose Images</button>
				<div class="input-group col-sm-12 col-md-6" id="upload-box" style="display:none;">
					<div>
						File Name: <span id="fname"></span>&nbsp;&nbsp;&nbsp;
						<button class="btn btn-default" type="button" id="upload-start">Start</button>
					</div>
					<div class="progress">
						<div class="progress-bar progress-bar-info progress-bar-striped" role="progressbar" aria-valuemin="0" aria-valuemax="100">
							<span id="progress"></span>%
						</div>
					</div>
				</div>
			</div>
		</div>

		<div class="modal fade" id="myModal" role="dialog">
			<div class="modal-dialog" role="document">
				<div class="modal-content">
					<div class="modal-header">
						<button type="button" class="close" data-dismiss="modal" aria-label="Close">
							<span aria-hidden="true">&times;</span>
						</button>
						<h4 class="modal-title">Message</h4>
					</div>
					<div class="modal-body" id="msg"></div>
				</div>
			</div>
		</div>

		<script src="//cdn.bootcss.com/jquery/1.12.0/jquery.min.js"></script>
		<script src="//cdn.bootcss.com/bootstrap/3.3.6/js/bootstrap.min.js"></script>
		<script src="//cdn.bootcss.com/plupload/2.1.8/plupload.full.min.js"></script>

		<script>
			$("#getkey").click(function() {
				$("#key").hide();
				$.ajax({
					url: "?key",
					type: "GET",
					data: { mail: $("#mail").val() },
					success: function(d) {
						$("#key").html(d);
						$("#key").slideDown("slow");
					}
				});
			});

			$("#rerid").click(function() {
				$.ajax({
					url: "?rejudge",
					type: "GET",
					data: { rid: $("#rid").val() },
					success: function(d) {
						$("#msg").html(d);
						$('#myModal').modal('show')
					}
				});
			});
			$("#repid").click(function() {
				$.ajax({
					url: "?rejudge",
					type: "GET",
					data: { pid: $("#pid").val() },
					success: function(d) {
						$("#msg").html(d);
						$('#myModal').modal('show')
					}
				});
			});

			var uploader = new plupload.Uploader({
				browse_button: "upload-img",
				url: "?uploadImg",
				multi_selection: false,
				filters: {
					mime_types : [{
						title: "Image files",
						extensions: "jpg,gif,png,bmp"
					}],
					prevent_duplicates : false
				},
				flash_swf_url: "//cdn.bootcss.com/plupload/2.1.8/Moxie.swf"
			});
			uploader.init();
			uploader.bind("FilesAdded", function(uploader, files) {
				$("#upload-box").slideDown();
				$("#fname").html(files[0].name);
				$("#progress").html("0");
				$(".progress-bar").css("width", "0");
			});
			$("#upload-start").click(function() {
				$(this).hide();
				uploader.start();
			});
			uploader.bind("UploadProgress", function(uploader, file) {
				$("#progress").html(file.percent);
				$(".progress-bar").css("width", file.percent + "%");
			});
			uploader.bind("FileUploaded", function(uploader, file, res) {
				$("#msg").html(res.response);
				setTimeout(function() {
					$('#myModal').modal('show');
					$("#upload-start").show();
					$("#upload-box").hide();
					$(".progress-bar").css("width", "0");
				}, 500);
			});
		</script>
	</body>
</html>
<?php die(); ?>