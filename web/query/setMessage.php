<?php
	require_once("query/message.php");

	$DOJSS = $_COOKIE['DOJSS'];
	$sex = (int)$_POST['sex'];
	$school = safe($_POST['school']);
	$birth = safe($_POST['birth']);
	$sign = safe($_POST['sign']);
	$user = checkDOJSS($DOJSS);

	if ($user -> sex == $sex && $user -> school == $school && $user -> birth == $birth && $user -> sign == $sign)
		send(2, $warning['sameMsg']);

	if ($user)
	{
		$uid = $user -> id;
		mysql_query("UPDATE `users` SET 
			`sex` = $sex,
			`birth` = '$birth',
			`school` = '$school',
			`sign` = '$sign'
		WHERE `id` = $uid ");
		if (mysql_affected_rows())
			send(0, $tip['changedMsg']);
		else send(1, $err['notSaved']);
	} else send(1, $err['wrongDOJSS']);
?>
