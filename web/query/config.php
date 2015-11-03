<?php
	$forbidden = 
	'<!DOCTYPE html>
	<html xmlns="http://www.w3.org/1999/xhtml" xml:lang="en" lang="en">
	<head>
		<title>403 &dash; Forbidden</title>
		<meta http-equiv="Content-Type" content="text/html; charset=UTF-8"/>
		<meta name="description" content="You do not have permission to view this"/>
		<style type="text/css">
			body {font-size:14px; color:#777777; font-family:arial; text-align:center;}
			h1 {font-size:180px; color:#99A7AF; margin: 70px 0 0 0;}
			h2 {color: #DE6C5D; font-family: arial; font-size: 20px; font-weight: bold; letter-spacing: -1px; margin: -3px 0 39px;}
			p {width:320px; text-align:center; margin-left:auto;margin-right:auto; margin-top: 30px }
			div {width:320px; text-align:center; margin-left:auto;margin-right:auto;}
			a:link {color: #34536A;}
			a:visited {color: #34536A;}
			a:active {color: #34536A;}
			a:hover {color: #34536A;}
		</style>
	</head>
	<body>
		<h1>403</h1>
		<h2>Forbidden</h2>
		<div>
			Unfortunately, you do not have permission to view this.
		</div>
	</body>
	</html>';

	ini_set("display_errors","On");

	static $oj_data = "/home/doj";
	$config = file_get_contents("$oj_data/db.conf");
	$conf = json_decode($config);

	$db_host = $conf->db_host;
	$db_name = $conf->db_name;
	$db_user = $conf->db_user;
	$db_pwd = $conf->db_pwd;
	static $key_reg = 231; // 1110 0111 todo: enc invite key
	static $key_pwd = 113; // 0111 0001 todo: enc password
	static $key_log = 111; // 0110 1111 todo: enc DOJSS

	static $flag = ["WAIT", "AC", "WA", "TLE", "MLE", "RE", "CE", "TESTING"];
	static $language = ["c", "cpp", "pascal", "python"];
	static $lan_subfix = ["c", "cpp", "pas", "py"];
	if(!mysql_pconnect($db_host, $db_user, $db_pwd)) 
		die('Could not connect: ' . mysql_error());
	mysql_query("set names utf8");
	if(!mysql_select_db($db_name))
		die('Can\'t use database : ' . mysql_error());
	date_default_timezone_set("PRC"); //中华人民共和国
	mysql_query("SET time_zone ='+8:00'");
?>
