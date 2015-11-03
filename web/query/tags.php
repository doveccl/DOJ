<?php
	require_once("query/message.php");

	$DOJSS = $_COOKIE['DOJSS'];
	$user = checkDOJSS($DOJSS);

	if ($user)
	{
		$res = mysql_query("SELECT * FROM `tags`");
		$tags = [];
		while ($group = mysql_fetch_object($res))
		{
			$group->set = explode('|', $group->set);
			$tags []= $group;
		}
		send(0, $tags);
	} else send(1, $err['wrongDOJSS']);
?>
