<?php
	require_once("query/message.php");

	$DOJSS = $_COOKIE['DOJSS'];
	$user = checkDOJSS($DOJSS);

	if (!isset($_GET['pid']))
		send(1, '');
	else $pid = $_GET['pid'];

	if ($user)
	{
		$p = getProblemByID($pid);
		if ($p)
		{
			$p->sampleIn = $p->sampleOut = '';
			if (file_exists("$oj_data/$pid/0.in"))
				$p->sampleIn = file_get_contents("$oj_data/$pid/0.in");
			if (file_exists("$oj_data/$pid/0.out"))
				$p->sampleOut = file_get_contents("$oj_data/$pid/0.out");

			if ($p->sampleIn == '')
				$p->sampleIn = "`N/A`";
			else $p->sampleIn = '>' . nl2br($p->sampleIn);
			if ($p->sampleOut == '')
				$p->sampleOut = "`N/A`";
			else $p->sampleOut = '>' . nl2br($p->sampleOut);
			
			$pid = $p->id;
			$situ = ["", "AC", "WA", "TLE", "MLE", "RE", "CE"];
			for ($i = 1; $i <= 6; $i++)
			{
				$res = mysql_query("SELECT COUNT(*) FROM `submit` WHERE `pid` = $pid AND `res` = $i");
				$p->$situ[$i] = mysql_fetch_array($res)[0];
				mysql_free_result($res);
			}


			send(0, $p);
		}
		else send(1, $err['noProblem']);
	} else send(1, $err['wrongDOJSS']);
?>
