<?php
	function send_error($error) {
		$res = new stdClass();
		$res->success = false;
		$res->error = $error;
		DOJ::response($res);
	}

	if (isset($_GET['max'])) {
		$sql = "SELECT MAX(id) AS max FROM problems WHERE hide = 0";
		$args = array();
		$r = DOJ::db_execute($sql, $args);

		if (!$r) send_error(str::$db_failed);

		$res = new stdClass();
		$res->success = true;
		$res->max = $r[0]->max;
		DOJ::response($res);
	}

	if (isset($_GET['tags'])) {
		$sql = "SELECT * FROM tags";
		$args = array();
		$r = DOJ::db_execute($sql, $args);

		if (!$r) send_error(str::$db_failed);

		$len = count($r);
		for ($i = 0; $i < $len; $i++)
			$r[$i]->set = explode("|", $r[$i]->set);

		$res = new stdClass();
		$res->success = true;
		$res->data = $r;
		DOJ::response($res);
	}

	$limits = array();
	$args = array();
	if (isset($_GET['t'])) {
		$ts = explode("|", $_GET["t"]);
		$len = count($ts);
		for ($i = 0; $i < $len; $i++) {
			$limits []= "LOCATE(?, tags) > 0";
			$args []= $ts[$i];
		}
	}

	if (isset($_GET['o']))
		if (is_numeric($_GET['o']))
			$offset = " OFFSET " . $_GET['o'];
		else
			send_error(str::$wrong_args);
	else
		$offset = "";

	$limits = implode(" AND ", $limits);
	if ($limits != "")
		$limits = "WHERE " . $limits;

	$sql = "SELECT id, name, tags, hide FROM problems $limits ORDER BY id LIMIT 51 $offset";
	$r = DOJ::db_execute($sql, $args);
	
	if (!$r) send_error(str::$no_data);

	$len = count($r);
	for ($i = 0; $i < $len; $i++) {
		$sql = "SELECT MIN(res) AS min FROM submit WHERE uid = ? AND pid = ? AND res > 0";
		$args = array($me->id, $r[$i]->id);
		$min = DOJ::db_execute($sql, $args)[0]->min;
		if ($min <= 6)
			$r[$i]->flag = $min;
	}

	$res = new stdClass();
	$res->success = true;
	$res->data = $r;
	DOJ::response($res);
?>
