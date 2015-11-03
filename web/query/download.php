<?php
	require_once("query/message.php");
	$DOJSS = $_COOKIE['DOJSS'];
	$user = checkDOJSS($DOJSS);

	if ($user)
	{
		$uid = $user -> id;
		$res = mysql_query("SELECT * FROM `submit` WHERE `uid` = $uid ");
		$noCode = $warning['noCode'];
		if (!($r = mysql_fetch_object($res)))
			echo "<meta charset='utf-8'> $noCode";
		else
		{
			mkdir('temp');
			chdir("temp");
			set_time_limit(0);

			$zip = new ZipArchive();
			$zip->open('code.zip', ZipArchive::OVERWRITE);

			while ($r)
			{
				$runid = $r->id;
				$pid = $r->pid;
				$result = $flag[$r->res];
				$score = $r->score;
				$subfix = $lan_subfix[$r->language];
				
				$file_name = "RUN$runid-P$pid-$result-$score.$subfix";
				file_put_contents($file_name, $r->code);
				$zip->addFile($file_name);
				
				$r = mysql_fetch_object($res);
			}

			$zip->close();
			
			header("Content-Type: application/force-download");
			header("Content-Transfer-Encoding: binary");
			header("Content-Type: application/zip");
			header("Content-Disposition: attachment; filename=code.zip");
			header("Content-Length: ".filesize("code.zip"));
			readfile('code.zip');
			
			delDirAndFile('../temp');
		}
	} else send(1, $err['wrongDOJSS']);
?>