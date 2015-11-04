<?php
	require_once("query/message.php");
	
	$DOJSS = $_COOKIE['DOJSS'];
	$user = checkDOJSS($DOJSS);
	$rid = intval($_GET['rid']);
	if (!$rid)
		send(1, '');
	
	if ($user)
	{
		$res = mysql_query("SELECT * FROM `submit` WHERE `id` = $rid");
		$r = mysql_fetch_object($res);
		mysql_free_result($res);

		$msg = new stdClass();
		$u = getUserByID($r->uid);
		if ($user->admin > $u->admin || $user->id == $u->id)
			$msg->code = $r->code;
		$msg->pid = $r->pid;
		$msg->res = $r->res;
		$msg->result = $r->result;

		send(0, $msg);
	} else send(1, $err['wrongDOJSS']);
?>
