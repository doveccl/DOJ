<?php
	require_once('query/config.php');
	require_once('query/dcen.php');
	require_once('query/function.php');

	if (isset($_GET['login']))
		require_once('template/login.php');
	else if (isset($_GET['submitlogin']))
		require_once('query/login.php');
	else if (isset($_GET['problems']))
		require_once('query/problems.php');
	else if (isset($_GET['problem']))
		require_once('query/problem.php');
	else if (isset($_GET['submit']))
		require_once('query/submit.php');
	else if (isset($_GET['tags']))
		require_once('query/tags.php');
	else if (isset($_GET['records']))
		require_once('query/records.php');
	else if (isset($_GET['result']))
		require_once('query/result.php');
	else if (isset($_GET['holdResult']))
		require_once('query/holdResult.php');
	else if (isset($_GET['topics']))
		require_once('query/topics.php');
	else if (isset($_GET['topic']))
		require_once('query/topic.php');
	else if (isset($_GET['postTopic']))
		require_once('query/postTopic.php');
	else if (isset($_GET['downloadCode']))
		require_once('query/download.php');
	else if (isset($_GET['changeBg']))
		require_once('query/setBackground.php');
	else if (isset($_GET['changeMsg']))
		require_once('query/setMessage.php');
	else if (isset($_GET['changeName']))
		require_once('query/setName.php');
	else if (isset($_GET['changePwd']))
		require_once('query/setPassword.php');
	else if (isset($_GET['changeMail']))
		require_once('query/setMail.php');
	else if (isset($_GET['changeProblem']))
		require_once('query/changeProblem.php');
	else if (isset($_GET['addProblem']))
		require_once('query/addProblem.php');
	else if (isset($_GET['lostpassword']))
		require_once('template/lostpwd.php');
	else if (isset($_GET['key']))
		require_once('query/key.php');
	else if (isset($_GET['reJudge']))
		require_once('query/reJudge.php');
	else if (isset($_GET['register']))
		require_once('template/register.php');
	else if (isset($_GET['submitregister']))
		require_once('query/register.php');
	else if (isset($_GET['admin']))
		require_once('template/admin.php');
	else if (isLogin())
		require_once('template/start.php');
	else header('Location:?login');
?>
