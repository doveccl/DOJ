<?php
	require_once("query/message.php");

	$DOJSS = $_COOKIE['DOJSS'];
	$name = safe($_POST['name']);
	$pwd = safe($_POST['password']);
	$user = checkDOJSS($DOJSS);

	if (!checkName($name))
		send(1, $err['invalidName']);

	if ($user)
	{
		if ($user -> name == $name)
			send(2, $warning['sameMsg']);
		if ($u = getUserByName($name))
			if ($u -> id != $user -> id)
				send(1, $err['sameName']);
		if (dc_decrypt($user -> password, $key_pwd) != $pwd)
			send(1, $err['wrongPwd']);

		$uid = $user -> id;
		mysql_query("UPDATE `users` SET 
			`name` = '$name'
		WHERE `id` = $uid ");
		if (mysql_affected_rows())
			send(0, $tip['changedName'], "\$('#myName').html('$name');");
		else send(1, $err['notSaved']);
	} else send(1, $err['wrongDOJSS']);
?>
