<?php
	function send_error($error) {
		$res = new stdClass();
		$res->success = false;
		$res->error = $error;
		DOJ::response($res);
	}

	if (!is_numeric($_POST['pid']))
		send_error(str::$wrong_args);
	if (!is_numeric($_POST['language']))
		send_error(str::$wrong_args);
	if (strlen($_POST['code']) < 10)
		send_error(str::$wrong_args);

	$sql = "SELECT MAX(submit_time) AS max FROM submit WHERE uid = ?";
	$args = array($me->id);
	if ($max = DOJ::db_execute($sql, $args)[0]->max)
		if (strtotime($max) > strtotime("-15 seconds"))
			send_error(str::$submit_too_fast);

	$lim = "";
	$arg = "";
	$p = new problem($_POST['pid']);
	if (is_null($p))
		send_error(str::$wrong_args);
	$cid = $p->cid;
	if ($cid != 0) {
		$c = new contest($cid);
		if (time() < strtotime($c->start_time))
			send_error(str::$wait_for_start);
		if (time() <= strtotime($c->start_time) + 60 * $c->time) {
			$sql = "SELECT COUNT(*) AS n FROM submit WHERE uid = ? AND cid = ?";
			$args = array($me->id, $cid);
			if (DOJ::db_execute($sql, $args)[0]->n > 0)
				send_error(str::$can_submit_once);

			$lim = ", res, cid";
			$arg = ", ?, ?";
		}
	}

	$sql = "INSERT INTO submit (uid, pid, language, code, submit_time $lim) VALUES (?, ?, ?, ?, NOW() $arg)";
	$args = array($me->id, $_POST['pid'], $_POST['language'], $_POST['code']);
	if ($lim != "") {
		$args []= 10;
		$args []= $cid;
	}

	DOJ::db_execute($sql, $args);

	$rid = DOJ::$db->lastInsertId();
	do {
		usleep(500);
		$sql = "SELECT res, result FROM submit WHERE id = ?";
		$args = array($rid);
		$r = DOJ::db_execute($sql, $args)[0];
	} while ($r->res == 0 || $r->res == 9);
	
	$state = DOJ::$res[$r->res];
	$r->result = json_decode($r->result);
	$pid = $_POST['pid'];

	$res = new stdClass();
	$res->success = true;
	$res->detail = "#$rid - $state  (P$pid)\n";

	if ($r->res == 6) {
		$res->detail .= "Compile Error:\n";
		$res->detail .= $r->result->ce;
	} else if ($r->res == 10) {
		$res->detail = str::$wait_for_finish;
	} else {
		$len = count($r->result);
		for ($i = 0; $i < $len; $i++) {
			$res->detail .= "#$i: " . DOJ::$res[$r->result[$i]->res];
			$res->detail .= "    " . $r->result[$i]->time . "ms";
			$res->detail .= "    " . $r->result[$i]->memory . "KB\n";
		}
	}

	$me->update();
	DOJ::response($res);
?>
