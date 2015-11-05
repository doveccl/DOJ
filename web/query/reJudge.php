<?php
	require_once('query/message.php');

	$DOJSS = $_COOKIE['DOJSS'];
	$user = checkDOJSS($DOJSS);
	$way = intval($_GET['way']);
	$id = intval($_GET['id']);
	if ($way == 1)
		$way = 'id';
	else $way = 'pid';

	if ($user && $user->admin > 0)
	{
		mysql_query("UPDATE `submit` SET `res` = 0 WHERE `$way` = $id ");
		send(0, $tip['reJudged'] . $way . $id);
	} else send(1, $err['notAdmin']);
?>