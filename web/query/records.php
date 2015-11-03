<?php
	require_once("query/message.php");
	
	$DOJSS = $_COOKIE['DOJSS'];
	$user = checkDOJSS($DOJSS);

	if ($user)
	{
		$offset = $limit = '';
		$list = $la = [];
		
		if (isset($_GET['pid']))
		{
			$pid = (int)$_GET['pid'];
			$la [] = "`pid` = $pid";
		}
		
		if (isset($_GET['uname']))
		{
			$uid = getUserByName($_GET['uname']);
			if ($uid)
				$uid = $uid->id;
			else $uid = 0;
			$la [] = "`uid` = $uid";
		}
		
		if ($la) $limit = "WHERE " . join(" AND ", $la);
		
		if (isset($_GET['offset']))
			$offset = "OFFSET " . (int)$_GET['offset'];

		$res = mysql_query("SELECT * FROM `submit` $limit ORDER BY `id` DESC LIMIT 50 $offset");

		while ($r = mysql_fetch_object($res)) {
			unset($r->code);
			$r->uname = getUserByID($r->uid)->name;
			$r->pname = getProblemByID($r->pid)->name;
			$list []= $r;
		}

		mysql_free_result($res);

		if ($offset == '')
			$offset = "OFFSET 50";
		else
			$offset = "OFFSET " . ((int)$_GET['offset'] + 50);
		
		$msg = new stdClass();
		$res = mysql_query("SELECT * FROM `submit` $limit ORDER BY `id` DESC LIMIT 1 $offset");		
		if (mysql_fetch_object($res))
			$msg->more = true;
		else $msg->more = false;
		
		$msg->list = $list;
		
		send(0, $msg);
	} else send(1, $err['wrongDOJSS']);
?>
