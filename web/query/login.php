<?php
	require_once('query/message.php');

	$msg = $_POST;
	if (isset($msg['user']))
		$user = $msg['user'];
	if (isset($msg['password']))
		$pwd = $msg['password'];
	$rem = isset($msg['remember']);

	if (getUserByName($user) || getUserByEmail($user))
		if ($r = checkUserPwd($user, $pwd))
		{
			if ($rem)
				$time = time() + 3600*24*365;
			else
				$time = 0;
			setcookie("DOJSS", DOJSS($r->id, $r->password), $time);
			header("Location:/");
		}
		else $error = $err['wrongPwd'];
	else $error = $err['noUser'];

	require_once('template/login.php');
?>
