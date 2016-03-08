<?php	
	if (is_numeric($_GET["rid"])) {
		$sql = "UPDATE submit SET res = 0 WHERE id = ?";
		$args = array($_GET["rid"]);
		DOJ::db_execute($sql, $args);
		die(str::$rejudge_success);
	}

	if (is_numeric($_GET["pid"])) {
		$sql = "UPDATE submit SET res = 0 WHERE pid = ?";
		$args = array($_GET["pid"]);
		DOJ::db_execute($sql, $args);
		die(str::$rejudge_success);
	}

	die(str::$wrong_args);
?>
