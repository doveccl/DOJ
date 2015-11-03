<?php
	require_once("query/message.php");

	$DOJSS = $_COOKIE['DOJSS'];
	$mail = safe($_POST['mail']);
	$pwd = safe($_POST['password']);
	$user = checkDOJSS($DOJSS);

	if (!checkEmail($mail))
		send(1, $err['wrongEmailFormat']);

	if ($user)
	{
		if ($user -> mail == $mail)
			send(2, $warning['sameMsg']);
		if (getUserByEmail($mail))
			send(1, $err['sameEmail']);
		if (dc_decrypt($user -> password, $key_pwd) != $pwd)
			send(1, $err['wrongPwd']);

		$uid = $user -> id;
		mysql_query("UPDATE `users` SET 
			`mail` = '$mail'
		WHERE `id` = $uid ");

		$gravatar = "//cn.gravatar.com/avatar/" . md5($mail) . "?d=mm";

		if (mysql_affected_rows())
			send(0, $tip['changedMail'], "$('#gravatar').attr('src', '$gravatar');");
		else send(1, $err['notSaved']);
	} else send(1, $err['wrongDOJSS']);
?>
