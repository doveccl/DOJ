<?php
	function send_error($error) {
		$res = new stdClass();
		$res->success = false;
		$res->error = $error;
		DOJ::response($res);
	}

	if (isset($_GET['id'])) {
		$p = new problem($_GET['id']);
		$res = new stdClass();
		$res->success = !is_null($p->id);
		if ($res->success)
			foreach ($p->raw as $k => $v)
				$res->$k = $v;

		if (!$res->success)
			send_error(str::$db_failed);
		if ($res->hide && !$me->admin)
			send_error(str::$can_not_view);
		if ($res->cid && !$me->admin) {
			$c = new contest($res->cid);
			if (time() < strtotime($c->start_time))
				send_error(str::$can_not_view);
		}

		$pid = $res->id;
		$res->sin = @file_get_contents(DOJ::$oj_data . "/$pid/s.in");
		$res->sout = @file_get_contents(DOJ::$oj_data . "/$pid/s.out");

		$state = array();
		for ($i = 1; $i <= 6; $i++) {
			$sql = "SELECT COUNT(*) AS n FROM submit WHERE pid = ? AND res = ?";
			$args = array($res->id, $i);
			$state []= DOJ::db_execute($sql, $args)[0]->n;
		}
		$res->res = $state;

		DOJ::response($res);
	}
?>
