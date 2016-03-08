<?php
	class DOJ {
		public static $oj_data = "/home/doj";
		public static $db;

		public static $admin_mail = "admin@doveccl.com";
		public static $locale = "zh-cn";
		public static $key_reg = 0; // todo: enc invite key
		public static $key_pwd = 0; // todo: enc password
		public static $key_log = 0; // todo: enc DOJSS

		public static $lan = array("C", "C++", "Pascal", "Python", "C++11");
		public static $res = array("Pending", "AC", "WA", "TLE", "MLE", "RE", "CE", "OLE", "Others", "Judging", "Waiting");

		public static function init() {
			ini_set("display_errors", "Off");

			self::init_database();
		}

		private static function init_database() {
			$data = self::$oj_data;
			$conf = file_get_contents("$data/db.conf");
			$conf = json_decode($conf);

			$host = $conf->db_host;
			$name = $conf->db_name;
			$user = $conf->db_user;
			$pwd = $conf->db_pwd;

			$dsn = "mysql:host=$host;dbname=$name";
			$args = array(PDO::ATTR_PERSISTENT => true);
			try {
				self::$db = new PDO($dsn, $user, $pwd, $args);
			} catch (PDOException $_e) {
				require_once("template/error.php");
			}
			self::db_exec("set names utf8");
			self::db_exec("set time_zone ='+8:00'");
		}
		
		public static function db_execute($sql, $args) {
			$state = self::$db->prepare($sql);
			if ($state->execute($args)) {
				$state->setFetchMode(PDO::FETCH_OBJ);
				$res = $state->fetchAll();
				if (empty($res))
					return $state->rowCount();
				return $res;
			}
			return false;
		}

		public static function db_exec($sql) {
			return self::$db->exec($sql);
		}

		public static function db_query($sql) {
			$res = self::$db->query($sql);
			if (!$res) return false;
			$res->setFetchMode(PDO::FETCH_OBJ);
			return $res->fetchAll();
		}

		public static function response($res) {
			die(json_encode($res));
		}

		public static function deal_request() {
//			for ($i = 1; $i <= 15; $i++)
//				(new user("id", $i))->update();

			$pre_act = array("login", "register");
			foreach ($_GET as $k => $v)
				if (in_array($k, $pre_act))
					require_once("query/$k.php");

			$me = new user();
			if (is_null($me->id))
				require_once("template/login.php");

			$admin_act = array(
				"setProblem", "addProblem", "uploadImg",
				"editTags", "downloadData", "uploadData",
				"setContest", "addContest", "key", "rejudge"
			);
			foreach ($_GET as $k => $v)
				if (in_array($k, $admin_act))
					if ($me->admin)
						require_once("admin/$k.php");
					else die();

			if (isset($_GET['statistic'])) {
				$mysta = array();
				for ($i = 1; $i <= 6; $i++) {
					$sql = "SELECT COUNT(*) AS n FROM submit WHERE uid = ? AND res = ?";
					$args = array($me->id, $i);
					$res = self::db_execute($sql, $args);
					$mysta []= $res[0]->n;
				}

				$recentDates = [];
				for ($i = 6; $i >= 0; $i--)
				{
					$day = date("Y-m-d", strtotime("-$i day"));
					$recentDates []= $day;
				}

				$ojsta = array();
				for ($i = 0; $i < 7; $i++) {
					$sql = "SELECT COUNT(*) AS n FROM submit WHERE LEFT(submit_time, 10) = ?";
					$args = array($recentDates[$i]);
					$tot = self::db_execute($sql, $args)[0]->n;

					$sql = "SELECT COUNT(*) AS n FROM submit WHERE LEFT(submit_time, 10) = ? AND res = 1";
					$args = array($recentDates[$i]);
					$ac = self::db_execute($sql, $args)[0]->n;
					
					$ojsta []= array(substr($recentDates[$i], -5), $tot, $ac);
				}
				
				$r = new stdClass();
				$r->success = true;
				$r->i = $mysta;
				$r->oj = $ojsta;
				self::response($r);
			}

			if (isset($_GET['recentProblems'])) {
				$sql = "SELECT id, name, tags FROM problems WHERE hide = 0 ORDER BY id DESC LIMIT 5";
				$args = array();
				$ps = self::db_execute($sql, $args);

				$r = new stdClass();
				$r->success = true;
				$r->data = $ps;
				self::response($r);
			}

			if (isset($_GET['recentContests'])) {
				$sql = "SELECT * FROM contests WHERE hide = 0 ORDER BY id DESC LIMIT 5";
				$args = array();
				$cs = self::db_execute($sql, $args);
				if (!$cs) $cs = [];
				
				$len = count($cs);
				for ($i = 0; $i < $len; $i++) {
					$stime = strtotime($cs[$i]->start_time);
					$etime = strtotime("+" . $cs[$i]->time . " minutes", $stime);
					if (time() < $stime)
						$cs[$i]->state = str::$upcoming;
					else if (time() <= $etime)
						$cs[$i]->state = str::$in_process;
					else
						$cs[$i]->state = str::$finished;
				}

				$r = new stdClass();
				$r->success = true;
				$r->data = $cs;
				self::response($r);
			}

			$act = array(
				"problems", "problem",
				"records", "record",
				"contests", "contest",
				"submit", "modify", "rank", "img"
			);

			if (isset($_GET['records']))
				$me->update();

			foreach ($_GET as $k => $v)
				if (in_array($k, $act))
					require_once("query/$k.php");

			if (isset($_GET["admin"]) && $me->admin)
				require_once("template/admin.php");

			$me->update();
			require_once("template/main.php");
		}
	}

	date_default_timezone_set("PRC");

	require_once("query/locale/" . DOJ::$locale . ".php");
	require_once("query/classes.php");
	require_once("query/format.php");
	require_once("query/dcen.php");
	require_once("query/zip.php");
?>
