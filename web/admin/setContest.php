<?php
	function send_error($error) {
		$res = new stdClass();
		$res->success = false;
		$res->error = $error;
		DOJ::response($res);
	}

	$res = new stdClass();
	$res->success = true;
	$res->msg = str::$modify_success;

	$c = new contest($_GET["id"]);
	if (is_null($c->id))
		send_error(str::$wrong_args);

	if (isset($_GET["switch"])) {
		$c->hide = $_GET["switch"];
		if ($c->hide == $_GET["switch"])
			$res->msg = str::$switch_contest_success[$c->hide];
		else send_error(str::$switch_error);
		DOJ::response($res);
	}

	if (isset($_GET["ap"])) {
		$p = new problem($_GET["ap"]);
		if (is_null($p->id))
			send_error(str::$no_problem);
		if ($p->cid == $c->id)
			send_error(str::$belong_to_this);
		if ($p->cid != 0)
			send_error(str::$belong_to_other);
		$p->cid = $c->id;
		DOJ::response($res);
	}

	if (isset($_GET["dp"])) {
		$p = new problem($_GET["dp"]);
		if (is_null($p->id))
			send_error(str::$no_problem);
		if ($p->cid != $c->id)
			send_error(str::$no_problem);
		$p->cid = 0;
		DOJ::response($res);
	}

	if (!preg_match("/^\d{4}(-\d{2}){2} \d{2}(:\d{2}){2}$/", $_GET["stime"]))
		send_error(str::$wrong_start_time);
	$dt = explode(" ", $_GET["stime"]);
	list($y, $m, $d) = explode("-", $dt[0]);
	if (checkdate($m, $d, $y) === false)
		send_error(str::$wrong_start_time);
	if (strtotime($_GET["stime"]) === false)
		send_error(str::$wrong_start_time);

	if (strtotime($_GET["stime"]) <= time())
		if (strtotime($c->start_time) > time())
			send_error(str::$stime_before_now);

	if (strtotime($_GET["stime"]) > time())
		$c->result = "";

	if ($_GET["name"] && is_numeric($_GET["time"])) {
		$c->name = $_GET["name"];
		$c->time = $_GET["time"];
		$c->start_time = $_GET["stime"];

		$sql = "UPDATE submit SET cid = 0 WHERE cid = ? AND submit_time < ?";
		$args = array($c->id, $c->start_time);
		DOJ::db_execute($sql, $args);

		DOJ::response($res);
	}

	send_error(str::$wrong_args);
?>
