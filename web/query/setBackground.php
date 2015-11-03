<?php
	$DOJSS = $_COOKIE['DOJSS'];
	$bg = (int)$_POST['bg'];
	$user = checkDOJSS($DOJSS);

	if ($user)
	{
		require_once("query/message.php");
		$uid = $user -> id;
		mysql_query("UPDATE `users` SET `bg` = $bg WHERE `id` = $uid ");
		if (mysql_affected_rows())
			send(0, $tip['changedBg']);
		else send(1, $err['notSaved']);
	} else send(1, $err['wrongDOJSS']);
?>