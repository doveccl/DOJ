<?php
	function cmp($a, $b) {
		if ($a->score == $b->score)
			return $a->time > $b->time;
		
		return $a->score < $b->score;
	}

	$sql = "SELECT COUNT(*) AS n FROM submit WHERE cid = ? AND (res = 0 OR res = 9)";
	$args = array($c->id);
	if (DOJ::db_execute($sql, $args)[0]->n == 0) {
		$sql = "SELECT * FROM submit WHERE cid = ?";
		$args = array($c->id);
		$sr = DOJ::db_execute($sql, $args);
		if (!$sr) $sr = array();
		
		$list = array();
		$len = count($sr);
		$plen = count($ps);
		for ($i = 0; $i < $len; $i++) {
			$u = new user("id", $sr[$i]->uid);
			if (!isset($list[$u->name])) {
				$list[$u->name] = new stdClass();
				$list[$u->name]->name = $u->name;
				$list[$u->name]->score = 0;
				$list[$u->name]->time = 0;
				for ($j = 0; $j < $plen; $j++) {
					$pid = $ps[$j]->id;
					$list[$u->name]->$pid = 0;
				}
			}

			if ($sr[$i]->res != 6) {
				$si = json_decode($sr[$i]->result);
				$slen = count($si);
				for ($j = 0; $j < $slen; $j++) {
					if ($si[$j]->res == 1) {
						$pid = $sr[$i]->pid;
						$list[$u->name]->score += 100 / $slen;
						$list[$u->name]->$pid += 100 / $slen;
					}
					$list[$u->name]->time += $si[$j]->time;
				}
			}
		}
		usort($list, 'cmp');
		
		$cres = "|Rank|User|Score|Time (ms)";
		for ($i = 0; $i < $plen; $i++)
			$cres .= "|" . $ps[$i]->name;
		$cres .= "|\n";
		$cres .= "|:---:|:---:|:---:|:---:";
		for ($i = 0; $i < $plen; $i++)
			$cres .= "|:---:";
		$cres .= "|\n";

		$rlen = count($list);
		for ($i = 0; $i < $rlen; $i++) {
			if (!is_int($list[$i]->score))
				$list[$i]->score = round($list[$i]->score, 2);

			$cres .= "|" . ($i + 1);
			$cres .= "|" . $list[$i]->name;
			$cres .= "|" . $list[$i]->score;
			$cres .= "|" . $list[$i]->time;

			for ($j = 0; $j < $plen; $j++) {
				$pid = $ps[$j]->id;
				if (!is_int($list[$i]->$pid))
					$list[$i]->$pid = round($list[$i]->$pid, 2);
				$cres .= "|" . $list[$i]->$pid;
			}
			$cres .= "|\n";
		}
		
		$sql = "UPDATE contests SET result = ? WHERE id = ?";
		$args = array($cres, $c->id);
		DOJ::db_execute($sql, $args);

		$data->wait = false;
		$data->result = $cres;
	}
?>