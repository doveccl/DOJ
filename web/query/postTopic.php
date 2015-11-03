<?php
	require_once("query/message.php");
	
	$DOJSS = $_COOKIE['DOJSS'];
	$user = checkDOJSS($DOJSS);
	
	if (!isset($_GET['type']))
		send(1, '');
	
	if ($user)
	{
		if ($_GET['type'] == "reply")
		{
			if (!isset($_POST['tid']))
				send(1, '');
			$tid = $_POST['tid'];
			
			if (!isset($_POST['content']))
				send(1, '');
			$content = trim($_POST['content']);
			if ($content == '')
				send(1, $err['invalidPost']);
			$content = safe($content);
			
			$res = mysql_query("SELECT * FROM `topics` WHERE `id` = $tid LIMIT 1");
			if (!mysql_fetch_object($res))
			{
				mysql_free_result($res);
				send(1, '');
			}
			mysql_free_result($res);
			
			$uid = $user->id;
			
			$res = mysql_query("SELECT * FROM `topics` WHERE `uid` = $uid ORDER BY `time` DESC LIMIT 1");
			$t = mysql_fetch_object($res);
			mysql_free_result($res);
			if ($t)
			{
				$lastP = strtotime($t->time);
				if ($lastP > strtotime("-30 seconds"))
					send(2, $warning['fastPost']);
			}
			
			$sql = "INSERT INTO `topics`
			(`id`, `uid`, `content`, `time`, `main`) VALUE
			($tid, $uid, '$content', NOW(), 0)";
			
			if (mysql_query($sql))
				send(0, $tip['postedReply']);
			else send(1, $sql.$err['notSaved']);
		}
		else if ($_GET['type'] == "main")
		{
			if (!isset($_POST['title']))
				send(1, '');
			$title = trim($_POST['title']);
			if ($title == '')
				send(1, $err['invalidTitle']);
			$title = safe($title);

			if (!isset($_POST['content']))
				send(1, '');
			$content = trim($_POST['content']);
			if ($content == '')
				send(1, $err['invalidPost']);
			$content = safe($content);
			
			$uid = $user->id;
			
			$res = mysql_query("SELECT * FROM `topics` WHERE `uid` = $uid ORDER BY `time` DESC LIMIT 1");
			$t = mysql_fetch_object($res);
			mysql_free_result($res);
			if ($t)
			{
				$lastP = strtotime($t->time);
				if ($lastP > strtotime("-30 seconds"))
					send(2, $warning['fastPost']);
			}
			
			$res = mysql_query("SELECT * FROM `topics` ORDER BY `id` DESC LIMIT 1");
			$t = mysql_fetch_object($res);
			mysql_free_result($res);
			$tid = $t->id + 1;
			
			$sql = "INSERT INTO `topics`
			(`id`, `uid`, `title`, `content`, `time`, `main`) VALUE
			($tid, $uid, '$title', '$content', NOW(), 1)";

			if (mysql_query($sql))
				send(0, $tip['postedTopic']);
			else send(1, $sql.$err['notSaved']);
		}
		else send(1, '');
	} else send(1, $err['wrongDOJSS']);
?>
