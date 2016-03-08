<?php
	function send_error($error) {
		$res = new stdClass();
		$res->success = false;
		$res->error = $error;
		DOJ::response($res);
	}

	if (!isset($_GET['id']))
		send_error(str::$wrong_args);
	$c = new contest($_GET['id']);
	if (is_null($c->id))
		send_error(str::$wrong_args);

	if ($c->hide && !$me->admin)
		send_error(str::$can_not_view);

	$sql = "SELECT id, name FROM problems WHERE cid = ? ORDER BY id";
	$args = array($_GET['id']);
	$ps = DOJ::db_execute($sql, $args);
	if (!$ps) $ps = array();

	$data = new stdClass();
	$data->id = $c->id;
	$data->name = $c->name;
	$data->start_time = $c->start_time;
	$data->time = $c->time;
	$data->stime = date("Y/m/d H:i:s", strtotime($c->start_time));
	$data->etime = date("Y/m/d H:i:s", strtotime($c->start_time) + 60 * $c->time);
	if (time() < strtotime($c->start_time)) {
		$data->cd = 1;
		$data->cd_text = str::$before_start;
	} else if (time() < strtotime($c->start_time) + 60 * $c->time) {
		$data->cd = 2;
		$data->cd_text = str::$before_end;
	} else {
		$data->cd = 0;
		if ($c->result == "") {
			$data->wait = str::$wait_for_count;
			$sql = "SELECT COUNT(*) AS n FROM submit WHERE cid = ? AND res = ?";
			$args = array($c->id, 10);
			if (DOJ::db_execute($sql, $args)[0]->n > 0) {
				$sql = "UPDATE submit SET res = 0 WHERE cid = ? AND res = ?";
				$args = array($c->id, 10);
				DOJ::db_execute($sql, $args);
			}
			require_once("query/contest_result.php");
		} else {
			$data->wait = false;
			$data->result = $c->result;
		}
	}

	$data->ps = $ps;

	$res = new stdClass();
	$res->success = true;
	$res->data = $data;
	DOJ::response($res);
?>
