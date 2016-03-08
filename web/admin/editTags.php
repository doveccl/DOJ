<?php
	function send_error($error) {
		$res = new stdClass();
		$res->success = false;
		$res->error = $error;
		DOJ::response($res);
	}

	$res = new stdClass();
	$res->success = true;

	if (!isset($_POST["tags"]))
		send_error(str::$wrong_args);

	$tags = trim($_POST["tags"]);
	$tags = explode("\n", $tags);

	if (empty($tags))
		send_error(str::$empty_tags);

	$len = count($tags);
	for ($i = 0; $i < $len; $i++)
		if (!preg_match("/^[^:\|]+:[^:\|]+(\|[^:\|]+)*$/", $tags[$i]))
			send_error(str::$wrong_tags);

	if (DOJ::db_exec("DELETE FROM tags") !== false) {
		$sql = "INSERT INTO tags (`group`, `set`) VALUES (?, ?)";
		$args = explode(":", $tags[0]);
		for ($i = 1; $i < $len; $i++) {
			$sql .= ", (?, ?)";
			$args = array_merge($args, explode(":", $tags[$i]));
		}
		DOJ::db_execute($sql, $args);
	}

	$res->msg = str::$modify_success;
	DOJ::response($res);
?>