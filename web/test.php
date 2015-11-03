<?php
	$a = exec("ps -ax | grep 'doj'", $out, $s);
	var_dump($a);
	echo strpos($out, "python");
	
		
