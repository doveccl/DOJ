<?php
	require_once("query/message.php");
	
	$DOJSS = $_COOKIE['DOJSS'];
	$user = checkDOJSS($DOJSS);
	$rid = intval($_GET['rid']);
	if (!$rid)
		send(1, '');
	function judging()
	{
		sleep(1);
		exec("ps -ax | grep 'dojudge'", $out, $s);
		$out = join(" ", $out);
		return strpos($out, "python");
	}

	if ($user)
	{
		while (judging() !== false);
		$res = mysql_query("SELECT * FROM `submit` WHERE `id` = $rid");
		$r = mysql_fetch_object($res);
		mysql_free_result($res);
		send(0, sprintf($tip['finishedJudge'], $r->res));
	} else send(1, $err['wrongDOJSS']);
?>
