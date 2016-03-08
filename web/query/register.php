<?php
	function send_error($error) {
		$res = new stdClass();
		$res->success = false;
		$res->error = $error;
		DOJ::response($res);
	}

	if (empty($_POST["name"]))
		send_error(str::$empty_name);
	if (empty($_POST["mail"]))
		send_error(str::$empty_mail);
	if (empty($_POST["pwd"]))
		send_error(str::$empty_password);
	if (empty($_POST["re_pwd"]))
		send_error(str::$empty_re_pwd);
	if (empty($_POST["key"]))
		send_error(str::$empty_key);

	$u = new user("name", $_POST["name"]);
	if (!is_null($u->id))
		send_error(str::$used_name);
	$u = new user("mail", $_POST["mail"]);
	if (!is_null($u->id))
		send_error(str::$used_mail);

	if (!format::name($_POST["name"]))
		send_error(str::$name_format);
	if (!format::mail($_POST["mail"]))
		send_error(str::$mail_format);
	if (!format::password($_POST["pwd"]))
		send_error(str::$password_format);

	if ($_POST["pwd"] != $_POST["re_pwd"])
		send_error(str::$password_different);

	$pwd_enc = dc_encrypt($_POST["pwd"], DOJ::$key_pwd);

	$key = dc_decrypt($_POST["key"], DOJ::$key_reg);
	$key = json_decode($key);
	if (is_null($key) || $_POST["mail"] != $key->m)
		send_error(str::$invalid_key);

	if (is_numeric($key->a))
		$admin = $key->a;
	else $admin = 0;

	if (!isset($_POST["sign"]))
		$_POST["sign"] = "";
	if (!isset($_POST["school"]))
		$_POST["school"] = "";

	$sql = "INSERT INTO users (name, mail, password, sign, school, admin, reg_time) VALUES (?, ?, ?, ?, ?, ?, NOW())";
	$args = array($_POST["name"], $_POST["mail"], $pwd_enc, $_POST["sign"], $_POST["school"], $admin);
	$r = DOJ::db_execute($sql, $args);
	if ($r > 0) {
		$res = new stdClass();
		$res->success = true;
		DOJ::response($res);
	}

	send_error(str::$db_failed);
?>
