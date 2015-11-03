<?php
	require_once('query/message.php');

	$DOJSS = $_COOKIE['DOJSS'];
	$user = checkDOJSS($DOJSS);
	$mail = $_GET['mail'];

	if ($user && $user->admin > 0)
	{
		send(0, dc_encrypt($mail, $key_reg));
	} else send(1, $err['notAdmin']);
?>