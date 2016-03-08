<?php
	function send_error($error) {
		$res = new stdClass();
		$res->success = false;
		$res->error = $error;
		DOJ::response($res);
	}

	if (isset($_GET['o']))
		if (is_numeric($_GET['o'])) {
			$offset = " OFFSET " . $_GET['o'];
		} else
			send_error(str::$wrong_args);
	else
		$offset = "";

	$sql = "SELECT id, name, sign, solve, submit FROM users ORDER BY solve DESC, submit LIMIT 51 $offset";
	$args = array();
	$r = DOJ::db_execute($sql, $args);

	$res = new stdClass();
	if ($r) {
		$res->success = true;
		$res->data = $r;
	} else {
		$res->success = false;
		$res->error = str::$no_data;
	}

	DOJ::response($res);
?>
