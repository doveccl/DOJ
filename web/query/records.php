<?php
	function send_error($error) {
		$res = new stdClass();
		$res->success = false;
		$res->error = $error;
		DOJ::response($res);
	}

	$limits = array();
	$args = array();
	if (isset($_GET['u'])) {
		$u = new user('name', $_GET['u']);
		if (!is_null($u->id)) {
			$limits []= "uid = ?";
			$args []= $u->id;
		} else
			send_error(str::$wrong_args);
	}
	if (isset($_GET['p'])) {
		$p = new problem($_GET['p']);
		if (!is_null($p->id)) {
			$limits []= "pid = ?";
			$args []= $p->id;
		} else
			send_error(str::$wrong_args);
	}
	if (isset($_GET['r']))
		if (is_numeric($_GET['r'])) {
			$limits []= "res = ?"; 
			$args []= $_GET['r'];
		} else
			send_error(str::$wrong_args);
	if (isset($_GET['l']))
		if (is_numeric($_GET['l'])) {
			$limits []= "language = ?";
			$args []= $_GET['l'];
		} else
			send_error(str::$wrong_args);

	if (isset($_GET['o']))
		if (is_numeric($_GET['o'])) {
			$offset = " OFFSET " . $_GET['o'];
		} else
			send_error(str::$wrong_args);
	else
		$offset = "";

	$limits = implode(" AND ", $limits);
	if ($limits != "")
		$limits = "WHERE " . $limits;

	$sql = "SELECT * FROM submit $limits ORDER BY id DESC LIMIT 51 $offset";
	$r = DOJ::db_execute($sql, $args);
	$res = new stdClass();
	if ($r) {
		$res->success = true;
		$len = count($r);
		for ($i = 0; $i < $len; $i++) {
			$r[$i]->uname = (new user("id", $r[$i]->uid))->name;
			$r[$i]->pname = (new problem($r[$i]->pid))->name;
			$r[$i]->result = json_decode($r[$i]->result);
			$utime = $umemory = 0;
			if (0 < $r[$i]->res && $r[$i]->res < 6) {
				$rlen = count($r[$i]->result);
				for ($j = 0; $j < $rlen; $j++) {
					$utime = max($utime, $r[$i]->result[$j]->time);
					$umemory = max($umemory, $r[$i]->result[$j]->memory);
				}
			}
			$r[$i]->utime = $utime;
			$r[$i]->umemory = $umemory;
			unset($r[$i]->result);
			unset($r[$i]->code);
		}
		$res->data = $r;
	} else {
		$res->success = false;
		$res->error = str::$no_data;
	}

	DOJ::response($res);
?>
