<?php
    ini_set('display_errors','1');
    error_reporting(E_ALL);

	if (isset($_GET["img"])) {
		$name = $_GET["img"];
		if (is_file("upload/$name")) {
			header("Content-type: image");
			readfile("upload/$name");
		}
	}

	die();
?>
