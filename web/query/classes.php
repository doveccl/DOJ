<?php
	class user {
		private $id, $name, $mail;
		private $reg_time;
		private $admin;

		private $sign, $school;

		private $solve, $submit;

		function __construct($key = null, $val = null) {
			$this->id = null;
			$need_confirm = false;
			$pwd = null;

			if (isset($_COOKIE['DOJSS']) && is_null($key)) {
				$need_confirm = true;
				$dojss = $_COOKIE['DOJSS'];
				$o = dc_decrypt($dojss, DOJ::$key_log);
				$o = json_decode($o);
				if (!empty($o)) {
					if (isset($o->i)) {
						$key = "id";
						$val = $o->i;
					}
					$pwd = isset($o->p) ? $o->p : "";
				}
			}

			if (!is_null($key)) {
				$sql = "SELECT * FROM users WHERE $key = ?";
				$args = array($val);
				$r = DOJ::db_execute($sql, $args);
				if ($r = $r[0])
					foreach ($r as $k => $v)
						if (property_exists($this, $k))
							$this->$k = $v;

				if ($need_confirm)
					if (md5($this->get_pwd(true)) != $pwd)
						$this->id = null;
			}
		}

		function __get($key) {
			return $this->$key;
		}

		function __set($key, $val) {
			if (property_exists($this, $key)) {
				$sql = "UPDATE users SET $key = ? WHERE id = ?";
				$args = array($val, $this->id);
				$r = DOJ::db_execute($sql, $args);
				if ($r > 0)
					$this->$key = $val;
			}
		}

		function update() {
			$sql = "SELECT COUNT(DISTINCT pid) AS ac FROM submit WHERE uid = ? AND res = 1";
			$args = array($this->id);
			$ac = DOJ::db_execute($sql, $args)[0]->ac;
			$sql = "SELECT COUNT(*) AS tot FROM submit WHERE uid = ?";
			$tot = DOJ::db_execute($sql, $args)[0]->tot;
			$this->__set("solve", $ac);
			$this->__set("submit", $tot);
		}

		function get_pwd($enc = false) {
			$sql = "SELECT password FROM users WHERE id = ?";
			$args = array($this->id);
			$r = DOJ::db_execute($sql, $args);
			if ($r = $r[0]) {
				$enc_pwd = $r->password;
				if ($enc)
					return $enc_pwd;
				return dc_decrypt($enc_pwd, DOJ::$key_pwd);
			}
			return false;
		}

		function set_pwd($pwd) {
			$enc_pwd = dc_encrypt($pwd, DOJ::$key_pwd);
			$sql = "UPDATE users SET password = ? WHERE id = ?";
			$args = array($enc_pwd, $this->id);
			$r = DOJ::db_execute($sql, $args);
			return $r > 0;
		}
	}

	class problem {
		private $id, $name, $cid;
		private $description, $input, $output, $hint;
		private $sin, $sout;
		private $time, $memory;

		private $create_time;
		private $tags;

		private $hide;

		private $raw;

		function __construct($pid) {
			$this->id = null;

			if (!is_null($pid)) {
				$sql = "SELECT * FROM problems WHERE id = ?";
				$args = array($pid);
				$r = DOJ::db_execute($sql, $args);
				if ($r = $r[0])
					foreach ($r as $k => $v)
						if (property_exists($this, $k))
							$this->$k = $v;

				$this->raw = $r;
			}
		}

		function __get($key) {
			return $this->$key;
		}

		function __set($key, $val) {
			if (property_exists($this, $key)) {
				$sql = "UPDATE problems SET $key = ? WHERE id = ?";
				$args = array($val, $this->id);
				$r = DOJ::db_execute($sql, $args);
				if ($r > 0) {
					$this->$key = $val;
					$this->raw->$key = $val;
				}
			}
		}
	}

	class record {
		private $id, $pid, $uid, $cid;
		private $submit_time;
		private $language;

		private $res, $result;
		private $code;

		private $raw;

		function __construct($rid) {
			$this->id = null;

			if (!is_null($rid)) {
				$sql = "SELECT * FROM submit WHERE id = ?";
				$args = array($rid);
				$r = DOJ::db_execute($sql, $args);
				if ($r = $r[0])
					foreach ($r as $k => $v)
						if (property_exists($this, $k))
							$this->$k = $v;

				$this->raw = $r;
			}
		}

		function __get($key) {
			return $this->$key;
		}

		function __set($key, $val) {
			if (property_exists($this, $key)) {
				$sql = "UPDATE submit SET $key = ? WHERE id = ?";
				$args = array($val, $this->id);
				$r = DOJ::db_execute($sql, $args);
				if ($r > 0) {
					$this->$key = $val;
					$this->raw->$key = $val;
				}
			}
		}
	}

	class contest {
		private $id, $name;
		private $start_time, $time;

		private $result;

		private $hide;

		private $raw;

		function __construct($cid) {
			$this->id = null;

			if (!is_null($cid)) {
				$sql = "SELECT * FROM contests WHERE id = ?";
				$args = array($cid);
				$r = DOJ::db_execute($sql, $args);
				if ($r = $r[0])
					foreach ($r as $k => $v)
						if (property_exists($this, $k))
							$this->$k = $v;

				$this->raw = $r;
			}
		}

		function __get($key) {
			return $this->$key;
		}

		function __set($key, $val) {
			if (property_exists($this, $key)) {
				$sql = "UPDATE contests SET $key = ? WHERE id = ?";
				$args = array($val, $this->id);
				$r = DOJ::db_execute($sql, $args);
				if ($r > 0) {
					$this->$key = $val;
					$this->raw->$key = $val;
				}
			}
		}
	}
?>
