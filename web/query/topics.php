<?php
	require_once("query/message.php");
	
	$DOJSS = $_COOKIE['DOJSS'];
	$user = checkDOJSS($DOJSS);

	if ($user)
	{
		$listA = $listB = [];

		$resA = mysql_query("SELECT * FROM `topics` WHERE `top` = 1 AND `main` = 1");
		while ($t = mysql_fetch_object($resA))
		{
			unset($t->content);
			unset($t->main);
			unset($t->top);
			$u = getUserByID($t->uid);
			$t->uname = $u->name;
			$mail = $u->mail;
			$gravatar = "//cn.gravatar.com/avatar/" . md5($mail) . "?d=mm";
			$t->gravatar = $gravatar;
			$listA []= $t;
		}
		
		mysql_free_result($resA);
		
		$offset = '';
		$limit = 'WHERE `top` = 0 AND `main` = 1';

		if (isset($_GET['uname']))
		{
			$uid = getUserByName($_GET['uname']);
			if ($uid) $uid = $uid->id;
			else $uid = 0;
			$limit .= " AND `uid` = $uid";
		}
		
		if (isset($_GET['offset']))
			$offset = "OFFSET " . (int)$_GET['offset'];
	
		$resB = mysql_query("SELECT * FROM `topics` $limit ORDER BY `time` DESC LIMIT 50 $offset");

		while ($t = mysql_fetch_object($resB))
		{
			unset($t->content);
			unset($t->main);
			unset($t->top);
			$u = getUserByID($t->uid);
			$t->uname = $u->name;
			$mail = $u->mail;
			$gravatar = "//cn.gravatar.com/avatar/" . md5($mail) . "?d=mm";
			$t->gravatar = $gravatar;
			$listB []= $t;
		}

		mysql_free_result($resB);

		if ($offset == '')
			$offset = "OFFSET 50";
		else
			$offset = "OFFSET " . ((int)$_GET['offset'] + 50);
		
		$msg = new stdClass();
		$resB = mysql_query("SELECT * FROM `topics` $limit ORDER BY `time` DESC LIMIT 1 $offset");		
		if (mysql_fetch_object($resB))
			$msg->more = true;
		else $msg->more = false;
		
		$msg->listA = $listA;
		$msg->listB = $listB;
		
		send(0, $msg);
	} else send(1, $err['wrongDOJSS']);
?>
