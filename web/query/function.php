<?php
	function send($res, $msg, $cmd = "")
	{
		$r = new stdClass();
		$r->res = $res;
		$r->msg = $msg;
		$r->cmd = $cmd;
		die(json_encode($r));
	}

	function safe($value)
	{
		if (get_magic_quotes_gpc())
			$value = stripslashes($value);
		if (!is_numeric($value))
			$value = mysql_real_escape_string($value);
		return $value;
	}

	function getUserByID($id)
	{
		$id = (int)$id;
		$res = mysql_query("SELECT * FROM `users` WHERE `id`=$id ");
		$o = mysql_fetch_object($res);
		mysql_free_result($res);
		return ($o);
	}

	function getUserByName($name)
	{
		$name = mysql_escape_string($name);
		$res = mysql_query("SELECT * FROM `users` WHERE `name`='$name' ");
		$o = mysql_fetch_object($res);
		mysql_free_result($res);
		return $o;
	}
	
	function getUserByEmail($mail)
	{
		$mail = mysql_escape_string($mail);
		$res = mysql_query("SELECT * FROM `users` WHERE `mail`='$mail' ");
		$o = mysql_fetch_object($res);
		mysql_free_result($res);
		return $o;
	}
	
	function checkName($name)
	{
		$exp = '/^[a-zA-Z0-9_]{5,16}$/';
		return preg_match($exp, $name);
	}
	
	function checkPwd($pwd)
	{
		$len = strlen($pwd);
		if ($len < 6 || $len > 20)
			return false;
		for ($i = 0; $i < $len; $i++)
			if ($pwd[$i] < ' ' || $pwd[$i] > '~' )
				return false;
		return true;
	}
	
	function checkEmail($email)
	{
		$exp = '/^[\w\-\.]+@[\w\-\.]+(\.\w+)+$/';
		return preg_match($exp, $email);
	}
	
	function checkKey($k, $m)
	{
		global $key_reg;
		return dc_decrypt($k, $key_reg) == $m;
	}
	
	function checkUserPwd($user, $pwd)
	{
		global $key_pwd;
		if (!($u = getUserByName($user)))
			$u = getUserByEmail($user);
		if (dc_decrypt($u->password, $key_pwd) == $pwd)
			return $u;
		else return false;
	}

	function DOJSS($id, $pass)
	{
		global $key_log;
		$o = new stdClass();
		$o->i = $id;
		$o->p = md5($pass);
		return dc_encrypt(json_encode($o), $key_log);
	}

	function checkDOJSS($dojss)
	{
		global $key_log;
		$res = json_decode(dc_decrypt($dojss, $key_log));
		if ($res == '')
			return false;

		if (isset($res->i))
			$i = $res->i;
		else $i = "";
		if (isset($res->p))
			$p = $res->p;
		else $p = "";

		$o = getUserByID($i);
		if ($o == null)
			return false;
		if (md5($o->password) != $p)
			return false;
		return $o;
	}

	function isLogin()
	{
		if (!isset($_COOKIE['DOJSS']))
			return false;
		return checkDOJSS($_COOKIE['DOJSS']);
	}

	function getProblemByID($id)
	{
		$id = (int)$id;
		$res = mysql_query("SELECT * FROM `problems` WHERE `id`=$id ");
		$o = mysql_fetch_object($res);
		mysql_free_result($res);
		return $o;
	}
	
	function delDirAndFile($dirName)
	{
		if ($handle = opendir("$dirName"))
		{
			while ($item = readdir($handle))
			{
				if ($item != "." && $item != "..")
				{
					if (is_dir("$dirName/$item"))
					{
						delDirAndFile("$dirName/$item");
					}
					else unlink("$dirName/$item" );
				}
			}

			closedir($handle);
			rmdir($dirName);
		}
	}

?>