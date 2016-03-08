<?php	
	function zipData($pid, $name) {
		$path = DOJ::$oj_data . "/$pid";
		$zip = new ZipArchive();
		if ($zip->open("$name", ZipArchive::OVERWRITE) === TRUE) {
			if (!is_dir($path))
				return false;
			$dir = dir($path);
			while ($f = $dir->read())
				if (
					is_file("$path/$f") && 
					preg_match("/^(s|\d+)\.(in|out)$/", $f)
				)
					$zip->addFile("$path/$f", "$f");

			$zip->close();
			return true;
		}
		return false;
	}

	function unzipData($name, $path) {
		$zip = new ZipArchive(); 
		$res = $zip->open($name);
		if ($res === true) {
			$zip->extractTo($path);
			$zip->close();
			return true;
		} else
			return false;
	}
?>
