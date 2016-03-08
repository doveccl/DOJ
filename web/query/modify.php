<?php
	function send_error($error) {
		$res = new stdClass();
		$res->success = false;
		$res->error = $error;
		DOJ::response($res);
	}

	$res = new stdClass();
	$res->success = true;
	$res->logout = false;

	if (isset($_GET['sign'])) {
		$me->sign = $_GET['sign'];
		$res->tip = str::$modify_success;
		DOJ::response($res);
	}

	if (!$_GET['opwd'])
		send_error(str::$empty_password);
	if ($me->get_pwd() != $_GET['opwd'])
		send_error(str::$wrong_password);

	if ($_GET['npwd'])
		if (format::password($_GET['npwd'])) {
			$changepwd = true;
			if ($_GET['opwd'] != $_GET['npwd'])
				$res->logout = true;
		} else send_error(str::$password_format);

	if ($_GET['name'])
		if (format::name($_GET['name'])) {
			if (is_null((new user("name", $_GET['name']))->id)) {
				$changename = true;
				$res->logout = true;
			} else if ($me->name != $_GET['name'])
				send_error(str::$used_name);
		} else send_error(str::$name_format);

	if ($_GET['mail'])
		if (format::mail($_GET['mail'])) {
			if (is_null((new user("mail", $_GET['mail']))->id)) {
				$changemail = true;
				$res->logout = true;
			} else if ($me->mail != $_GET['mail'])
				send_error(str::$used_mail);
		} else send_error(str::$name_format);

	if ($changepwd)
		$me->set_pwd($_GET['npwd']);
	if ($changename)
		$me->name = $_GET['name'];
	if ($changemail)
		$me->mail = $_GET['mail'];

	DOJ::response($res);
?>