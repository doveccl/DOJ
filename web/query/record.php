<?php
	if (isset($_GET['id'])) {
		$r = new record($_GET['id']);
		$res = new stdClass();
		$res->success = !is_null($r->id);
		if ($res->success) {
			foreach ($r->raw as $k => $v)
				$res->$k = $v;
			$res->result = json_decode($res->result);
			$res->code = htmlspecialchars($res->code);

			if ($r->cid != 0) {
				$c = new contest($r->cid);
				if (time() <= strtotime($c->start_time) + 60 * $c->time)
					if ($r->uid != $me->id)
						$res->code = str::$can_not_see_others;
			}
		}

		if (!$res->success)
			$res->error = str::$db_failed;
		
		DOJ::response($res);
	}
?>
