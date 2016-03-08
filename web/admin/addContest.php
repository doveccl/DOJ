<?php
	function send_error($error) {
		$res = new stdClass();
		$res->success = false;
		$res->error = $error;
		DOJ::response($res);
	}

	$res = new stdClass();
	$res->success = true;

	if (!$_GET["name"])
		send_error(str::$empty_contest_name);

	if (!is_numeric($_GET["time"]))
		send_error(str::$wrong_time);

	if (!preg_match("/^\d{4}(-\d{2}){2} \d{2}(:\d{2}){2}$/", $_GET["stime"]))
		send_error(str::$wrong_start_time);
	$dt = explode(" ", $_GET["stime"]);
	list($y, $m, $d) = explode("-", $dt[0]);
	if (checkdate($m, $d, $y) === false)
		send_error(str::$wrong_start_time);
	if (strtotime($_GET["stime"]) === false)
		send_error(str::$wrong_start_time);

	if (strtotime($_GET["stime"]) <= time())
		send_error(str::$stime_before_now);

	$sql = "INSERT INTO contests (name, `time`, start_time) VALUES (?, ?, ?)";
	$args = array($_GET["name"], $_GET["time"], $_GET["stime"]);

	if (DOJ::db_execute($sql, $args) > 0) {
		$res->cid = DOJ::$db->lastInsertId();
		$res->msg = str::$add_contest_success;
		DOJ::response($res);
	} else send_error(str::$db_failed);
?>
