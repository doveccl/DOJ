<?php
	if ($_FILES["file"]["error"] > 0)
		die(str::$err_msg . ": " . $_FILES["file"]["error"]);
	if (!$_FILES["file"]["name"])
		die(str::$upload_failed);
	if (!preg_match("/^image/" ,$_FILES["file"]["type"]))
		die(str::$upload_type_error);

	$md5 = md5_file($_FILES["file"]["tmp_name"]);
	copy($_FILES["file"]["tmp_name"], "upload/$md5");

	echo "Use <code>![DOJ Image](?img=$md5)</code> to insert this image:";
	die("<hr><img src=\"?img=$md5\">");
?>