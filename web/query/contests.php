<?php
	function send_error($error) {
		$res = new stdClass();
		$res->success = false;
		$res->error = $error;
		DOJ::response($res);
	}

	if (isset($_GET['o']))
		if (is_numeric($_GET['o']))
			$offset = " OFFSET " . $_GET['o'];
		else
			send_error(str::$wrong_args);
	else
		$offset = "";

	$sql = "SELECT * FROM contests ORDER BY id DESC LIMIT 51 $offset";
	$args = array();
	$r = DOJ::db_execute($sql, $args);
	
	if (!$r) send_error(str::$no_data);

	$len = count($r);
	for ($i = 0; $i < $len; $i++) {
		$stime = strtotime($r[$i]->start_time);
		$etime = strtotime("+" . $r[$i]->time . " minutes", $stime);
		if (time() < $stime)
			$r[$i]->state = str::$upcoming;
		else if (time() <= $etime)
			$r[$i]->state = str::$in_process;
		else
			$r[$i]->state = str::$finished;
	}

	$res = new stdClass();
	$res->success = true;
	$res->data = $r;
	DOJ::response($res);
?>
