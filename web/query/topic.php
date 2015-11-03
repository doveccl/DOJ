<?php
	require_once("query/message.php");
	
	$DOJSS = $_COOKIE['DOJSS'];
	$user = checkDOJSS($DOJSS);
	
	if (!isset($_GET['id']))
		send(1, '');
	$tid = $_GET['id'];
	
	if ($user)
	{
		$list = [];
		$res = mysql_query("SELECT * FROM `topics` WHERE `id` = $tid ORDER BY `time`");
		
		while ($t = mysql_fetch_object($res))
		{
			unset($t->top);
			$u = getUserByID($t->uid);
			$t->uname = $u->name;
			$mail = $u->mail;
			$gravatar = "//cn.gravatar.com/avatar/" . md5($mail) . "?d=mm";
			$t->gravatar = $gravatar;
			$list []= $t;
		}
		
		mysql_free_result($res);
		
		send(0, $list);
	} else send(1, $err['wrongDOJSS']);
?>
