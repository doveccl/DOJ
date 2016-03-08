<?php
	$key = new stdClass();

	if (empty($_GET["mail"]))
		die(str::$empty_mail);
	if (!format::mail($_GET["mail"]))
		die(str::$mail_format);

	$key->m = $_GET["mail"];
	$key->a = 0;

	if (isset($_GET["admin"]) && $_GET["admin"] == 1)
		$key->a = 1;

	$k = json_encode($key);
	die(dc_encrypt($k, DOJ::$key_reg));
?>
