<?php
	function send_error($error) {
		$res = new stdClass();
		$res->success = false;
		$res->error = $error;
		DOJ::response($res);
	}

	$res = new stdClass();
	$res->success = true;

	if (!is_numeric($_POST["time"]))
		send_error(str::$wrong_time);
	if (!is_numeric($_POST["memory"]))
		send_error(str::$wrong_memory);

	if (!$_POST["name"])
		send_error(str::$empty_problem_name);

	if ($_POST["tags"]) {
		$tags = explode("|", $_POST["tags"]);
		foreach ($tags as $tag) {
			$sql = "SELECT COUNT(*) AS n FROM tags WHERE LOCATE(?, `set`) > 0";
			$args = array($tag);
			if (DOJ::db_execute($sql, $args)[0]->n == 0)
				send_error("\"$tag\"" . str::$not_exists);
		}
	}
	
	$sql = "INSERT INTO problems (name, `time`, memory, description, input, output, hint, tags, create_time) VALUES (?, ?, ?, ?, ?, ?, ?, ?, NOW())";
	$args = array($_POST["name"], $_POST["time"], $_POST["memory"], $_POST["description"], $_POST["input"], $_POST["output"], $_POST["hint"], $_POST["tags"]);

	if (DOJ::db_execute($sql, $args) > 0) {
		$res->pid = DOJ::$db->lastInsertId();
		$res->msg = str::$add_problem_success;
		mkdir(DOJ::$oj_data . "/" . $res->pid);
		chmod(DOJ::$oj_data . "/" . $res->pid, 0777);
		DOJ::response($res);
	} else send_error(str::$db_failed);
?>
