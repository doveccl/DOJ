<?php
	require_once("query/message.php");

	$DOJSS = $_COOKIE['DOJSS'];
	$opwd = safe($_POST['opwd']);
	$npwd = safe($_POST['npwd']);
	$user = checkDOJSS($DOJSS);

	if (!checkPwd($npwd))
		send(1, $err['invalidPwd']);

	if ($user)
	{
		if (dc_decrypt($user -> password, $key_pwd) != $opwd)
			send(1, $err['wrongPwd']);
		if ($opwd == $npwd)
			send(2, $warning['samePwd']);

		$uid = $user -> id;
		$pwd_enc = dc_encrypt($npwd, $key_pwd);

		mysql_query("UPDATE `users` SET 
			`password` = '$pwd_enc'
		WHERE `id` = $uid ");
		if (mysql_affected_rows())
			send(0, $tip['changedPwd'], "setTimeout(logout, 3000);");
		else send(1, $err['notSaved']);
	} else send(1, $err['wrongDOJSS']);
?>
