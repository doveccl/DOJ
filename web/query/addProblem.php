<?php
	require_once('query/message.php');

	$DOJSS = $_COOKIE['DOJSS'];
	$user = checkDOJSS($DOJSS);

	if ($user && $user->admin > 0)
	{
		$title = safe($_POST['title']);
		$time = safe($_POST['time']);
		$mem = safe($_POST['memory']);
		$des = safe($_POST['description']);
		$in = safe($_POST['input']);
		$out = safe($_POST['output']);
		$hint = safe($_POST['hint']);
		$tag = safe($_POST['tags']);

		mysql_query("INSERT INTO `problems`
			(`name`, `time`, `memory`, `description`, `input`, `output`, `hint`, `tags`, `create_time`)
		VALUES ('$title', $time, $mem, '$des', '$in', '$out', '$hint', '$tag', NOW() ) ");
		if (mysql_affected_rows()) {
			$npid = mysql_insert_id();
			mkdir("$oj_data/$npid");
			send(0, $tip['addedProblem']);
		} else send(1, $err['notSaved']);
	} else send(1, $err['notAdmin']);
?>