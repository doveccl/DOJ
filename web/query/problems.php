<?php
	require_once("query/message.php");

	if (!isset($_GET['type']))
		send(1, '');
	
	$DOJSS = $_COOKIE['DOJSS'];
	$user = checkDOJSS($DOJSS);

	if ($user)
	{
		$type = $_GET['type'];
		if ($type == "num")
		{
			$res = mysql_query("SELECT COUNT(*) FROM `problems`");
			$num = mysql_fetch_array($res)[0];
			mysql_free_result($res);
			send(0, $num);
		}
		else if ($type == "tag")
		{
			if (!isset($_GET['tags']))
				send(1, '');

			$res = mysql_query("SELECT * FROM `problems`");
			$qt = explode('|', $_GET['tags']);
			$list = [];
			while ($p = mysql_fetch_object($res))
			{
				$flag = true;
				$at = explode('|', $p->tags);
				foreach ($qt as $t)
					if (!in_array($t, $at))
					{
						$flag = false;
						break;
					}
				if ($flag && ($p->name[0] != '$' || $user->admin))
					$list []= ['id' => $p->id, 'name' => $p->name];
			}
			
			send(0, $list);
		}
		else if (is_numeric($type))
		{
			$l = (int)$type;
			$r = $l + 100;
			$list = [];

			$res = mysql_query("SELECT * FROM `problems` WHERE `id` >= $l AND `id` < $r");
			while ($p = mysql_fetch_object($res))
				if ($p->name[0] != '$' || $user->admin)
					$list []= ["id" => $p->id, "name" => $p->name];
			mysql_free_result($res);
			send(0, json_encode($list));
		}
		else send(1, '');
	} else send(1, $err['wrongDOJSS']);
?>
