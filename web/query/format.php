<?php
	class format {
		static $re_name = "/^[a-zA-Z][a-zA-Z0-9_]{2,14}$/";
		static $re_pwd = "/^[ -~]{6,20}$/";

		public static function name($name) {
			if (is_string($name))
				return preg_match(self::$re_name, $name);
			return false;
		}

		public static function password($pwd) {
			if (is_string($pwd))
				return preg_match(self::$re_pwd, $pwd);
			return false;
		}

		public static function mail($mail) {
			if (is_string($mail))
				return filter_var($mail, FILTER_VALIDATE_EMAIL);
			return false;
		}
	}
