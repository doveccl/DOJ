<?php
	$p = new problem($_POST["pid"]);
	if (!$p->id) die(str::$wrong_args);
	$pdata = DOJ::$oj_data . "/" . $p->id;

	if ($_FILES["file"]["error"] > 0)
		die(str::$err_msg . ": " . $_FILES["file"]["error"]);
	if (!$_FILES["file"]["name"])
		die(str::$upload_failed);

	$f = basename($_FILES["file"]["tmp_name"]);
	$path = dirname($_FILES["file"]["tmp_name"]);

	if (unzipData("$path/$f", "$path/$f.tmp")) {
		$tmp = "$path/$f.tmp";
		if (is_file("$tmp/s.in") && is_file("$tmp/s.out")) {
			$i = 0;
			if (is_file("$tmp/$i.in") && is_file("$tmp/$i.out")) {
				system("rm -rf $pdata/*");
				do {
					copy("$tmp/$i.in", "$pdata/$i.in");
					copy("$tmp/$i.out", "$pdata/$i.out");
					$i++;
				} while (is_file("$tmp/$i.in") && is_file("$tmp/$i.out"));
				echo "$i" . str::$data_found . "<br>";
			} else
				echo str::$only_sample . "<br>";
			file_put_contents("$pdata/s.in", file_get_contents("$tmp/s.in"));
			file_put_contents("$pdata/s.out", file_get_contents("$tmp/s.out"));
			echo str::$upload_success . " !";
		} else
			echo str::$no_sample;
		system("rm -rf $tmp");
	}

	die();
?>