<?php
	try {
		$pid = (new problem($_GET["pid"]))->id;
		if (!is_null($pid)) {
			$name = DOJ::$oj_data . "/tmp.zip";
			if (zipData($pid, $name)) {
				header("Content-Type: application/zip");
				header("Content-Disposition: attachment; filename=$pid.zip");
				readfile($name);
			} else throw new Exception(str::$zip_error);
		} else throw new Exception(str::$wrong_args);
	} catch (Exception $_e) {
		require_once("template/error.php");
	}
?>