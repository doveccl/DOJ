<?php
	function send_error($error) {
		$res = new stdClass();
		$res->success = false;
		$res->error = $error;
		DOJ::response($res);
	}

	if (empty($_POST["user"]))
		send_error(str::$empty_user);
	if (empty($_POST["password"]))
		send_error(str::$empty_password);

	if (format::mail($_POST["user"]))
		$u = new user("mail", $_POST["user"]);
	else
		$u = new user("name", $_POST["user"]);

	if (is_null($u->id))
		send_error(str::$no_user);

	$pwd = $u->get_pwd();
	if ($pwd != $_POST["password"])
		send_error(str::$wrong_password);

	$res = new stdClass();
	$res->success = true;
	$dojss = array (
		"i" => $u->id,
		"p" => md5($u->get_pwd(true))
	);
	$dojss = json_encode($dojss);
	$res->dojss = dc_encrypt($dojss, DOJ::$key_log);
	DOJ::response($res);
?>
