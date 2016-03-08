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

	$p = new problem($_REQUEST["id"]);
	if (is_null($p->id))
		send_error(str::$wrong_args);

	if (isset($_GET["switch"])) {
		$p->hide = $_GET["switch"];
		if ($p->hide == $_GET["switch"])
			$res->msg = str::$switch_problem_success[$p->hide];
		else send_error(str::$switch_error);
		DOJ::response($res);
	}

	if ($_GET["name"] && is_numeric($_GET["time"]) && is_numeric($_GET["memory"]) && $_GET["tags"]) {
		$p->name = $_GET["name"];
		$p->time = $_GET["time"];
		$p->memory = $_GET["memory"];
		$p->tags = $_GET["tags"];
		DOJ::response($res);
	}

	$keys = ["description", "input", "output", "hint"];
	foreach ($keys as $v)
		if (isset($_POST[$v])) {
			$p->$v = $_POST[$v];
			if ($p->$v == $_POST[$v])
				DOJ::response($res);
			else
				send_error(str::$db_failed);
		}

	send_error(str::$wrong_args);
?>
