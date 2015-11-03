<?php
	require_once("query/message.php");

	$DOJSS = $_COOKIE['DOJSS'];
	$user = checkDOJSS($DOJSS);

	if (!isset($_POST['pid']))
		send(1, '');
	else $pid = safe($_POST['pid']);

	if (!isset($_POST['language']))
		send(1, '');
	else $lan = safe($_POST['language']);

	if (!isset($_POST['code']))
		send(1, '');
	else $code = safe($_POST['code']);
	if (strlen($code) < 20)
		send(2, $warning['shortCode']);
	
	if ($user)
	{
		$uid = $user->id;
		mysql_query(
			"INSERT INTO `submit` (`uid`,`pid`,`submit_time`,`language`,`code`)
			VALUES ($uid,$pid,NOW(),$lan,'$code')"
		) or send(1, $err['submitFail']);

		$rid = mysql_insert_id();
		send(0, $tip['submitSuccess'], "getResult($rid)");
	} else send(1, $err['wrongDOJSS']);
?>
