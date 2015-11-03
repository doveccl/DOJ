<?php
	require_once('query/message.php');

	$DOJSS = $_COOKIE['DOJSS'];
	$user = checkDOJSS($DOJSS);

	if ($user && $user->admin > 0)
	{
		$pid = safe($_POST['pid']);
		$title = safe($_POST['title']);
		$time = safe($_POST['time']);
		$mem = safe($_POST['memory']);
		$des = safe($_POST['description']);
		$in = safe($_POST['input']);
		$out = safe($_POST['output']);
		$hint = safe($_POST['hint']);
		$tag = safe($_POST['tags']);

		$p = getProblemByID($pid);
		foreach ($p as $key => $value)
		{
			$p->$key = safe($value);
		}
		if ($title == $p->name && $time == $p->time && $mem == $p->memory && $des == $p->description && $in == $p->input && $out == $p->output && $hint == $p->hint && $tag == $p->tags)
			send(2, $warning['sameMsg']);

		mysql_query("UPDATE `problems` SET
			`name` = '$title',
			`time` = $time,
			`memory` = $mem,
			`description` = '$des',
			`input` = '$in',
			`output` = '$out',
			`hint` = '$hint',
			`tags` = '$tag'
		WHERE `id` = $pid ");
		if (mysql_affected_rows())
			send(0, $tip['changedProblem']);
		else send(1, $err['notSaved']);
	} else send(1, $err['notAdmin']);
?>