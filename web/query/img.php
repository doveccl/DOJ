<?php	
	if (isset($_GET["img"])) {
		$name = $_GET["img"];
		if (is_file("upload/$name")) {
			$finfo = finfo_open(FILEINFO_MIME);
			$mimetype = finfo_file($finfo, "upload/$name");
			finfo_close($finfo);
			header("Content-type: $mimetype");
			readfile("upload/$name");
		}
	}

	die();
?>
