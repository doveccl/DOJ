<?php
	class str {
		static $err = "抱歉，出错了";
		static $err_msg = "错误信息";
		static $err_code = "错误代码";

		static $mail_to_admin = "联系管理员";
		static $copy_err_msg = "复制错误信息";
		static $copy_ok = "复制成功！";

		static $login_title = "登录 - DOJ";
		static $name_or_mail = "用户名 / 邮箱";
		static $password = "密码";
		static $keep_login = "记住我";
		static $login = "登录";
		static $register = "注册";
		static $forget_password = "忘记密码";
		static $contact_admin = "请联系管理员";
		static $empty_user = "请填写用户名 / 邮箱";
		static $empty_password = "请填写密码";
		static $no_user = "用户不存在";
		static $wrong_password = "密码错误";

		static $register_title = "注册 - DOJ";
		static $name = "用户名";
		static $mail = "邮箱";
		static $repeat_password = "重复密码";
		static $invite_key = "邀请码";
		static $school = "学校";
		static $sign = "签名";
		static $empty_name = "请填写用户名";
		static $empty_mail = "请填写邮箱";
		static $empty_re_pwd = "请填写重复密码";
		static $empty_key = "请填写邀请码";
		static $name_format = "用户名须以字母开头，可含数字、下划线，3-15 个字符";
		static $mail_format = "邮箱格式错误";
		static $password_format = "密码长度为 6-20 个字符";
		static $password_different = "两次输入的密码不同";
		static $used_name = "该用户名已被注册";
		static $used_mail = "该邮箱已被注册";
		static $invalid_key = "无效的邀请码";
		static $finish_register = "注册成功，现在即可登录";
		static $cancel = "取消";

		static $db_failed = "数据库请求失败，请重试";
		static $post_failed = "请求失败，请重试";
		static $no_data = "没有查询到任何数据";
		static $wrong_args = "请求参数有误";

		static $home = "首页";
		static $problems = "题库";
		static $records = "记录";
		static $contests = "比赛";
		static $rank = "排名";

		static $setting = "设置";
		static $logout = "登出";

		static $personal_statistics = "个人数据统计";
		static $oj_statistics = "OJ 数据统计";
		static $announcement = "公告";
		static $recent_problems = "最近添加的题目";
		static $recent_contests = "最近举办的比赛";

		static $page_index = "分页索引";
		static $tags = "分类索引";
		static $load_more = "加载更多";

		static $problem_id = "题目编号";
		static $any = "任意";
		static $filter = "筛选";
		static $can_not_see_others = "比赛期间不可查看他人代码";

		static $can_not_view = "你没有权限查看";
		static $time = "时间限制";
		static $memory = "内存限制";
		static $description = "问题描述";
		static $input_format = "输入格式";
		static $output_format = "输出格式";
		static $sample_input = "样例输入";
		static $sample_output = "样例输出";
		static $range_hint = "数据范围和提示";
		static $submit = "提交";
		static $my_submit = "我的提交记录";
		static $problem_submit = "该题提交记录";
		static $close = "关闭";
		static $submit_too_fast = "提交过于频繁，请稍后再试";
		static $wait_for_start = "比赛开始后才能够提交";
		static $wait_for_finish = "请等待比赛结束再看结果";
		static $can_submit_once = "比赛期间每道题只能提交一次";

		static $upcoming = "将进行";
		static $in_process = "进行中";
		static $finished = "已结束";
		static $before_start = "距离比赛开始";
		static $before_end = "距离比赛结束";
		static $day = "日";
		static $hour = "时";
		static $minute = "分";
		static $second = "秒";
		static $contest_rank = "比赛排名";
		static $wait_for_count = "正在进行统计，请稍等";

		static $modify_info = "个人信息修改";
		static $modify_sign = "修改签名";
		static $old_password = "原密码（必填）";
		static $new_password = "新密码（留空则为不修改）";
		static $modify = "修改";
		static $modify_success = "修改成功";

		static $switch_problem_success = ["题目已启用", "题目已隐藏"];
		static $switch_error = "切换失败";
		static $add_problem = "添加题目";
		static $edit_tags = "编辑标签";
		static $add = "添加";
		static $problem_name = "题目名称";
		static $problem_tags = "标签（用 | 隔开）";
		static $empty_problem_name = "题目名称不可为空";
		static $not_exists = "不存在";
		static $wrong_time = "请输入有效的时间";
		static $wrong_memory = "请输入有效的内存";
		static $add_problem_success = "成功添加题目，请在上传数据后在题目列表中启用该题目";

		static $tag = "标签";
		static $tags_explain = "说明：一行代表一个标签分组，每一行冒号前面是分组名称，后面是用 | 隔开的不同的标签，请不要随意添加多余的空格";
		static $empty_tags = "请至少设置一组标签";
		static $wrong_tags = "标签格式错误";

		static $edit_problem = "编辑题目";
		static $download_data = "下载该题数据";
		static $zip_error = "压缩数据时发生严重错误，请检查数据是否完整";
		static $upload_data = "上传数据";
		static $size_limit = "大小限制";
		static $start_upload = "开始上传";
		static $upload_failed = "上传出错";
		static $upload_type_error = "上传类型出错";
		static $no_sample = "没有发现样例文件 (s.in | s.out)";
		static $only_sample = "没有找到从0开始编号的数据，只更新样例数据";
		static $data_found = "组数据已被上传";
		static $upload_success = "数据更新成功";
		static $data_explain = "题目数据应以zip格式上传，压缩包内s.in和s.out为样例输入输出文件，评测数据必须以数字作为名字，同名的.in和.out被视作同一组数据，数字必须连续且从0开始";

		static $switch_contest_success = ["比赛已启用", "比赛已隐藏"];
		static $add_contest = "添加比赛";
		static $start_date = "开始日期";
		static $start_time = "开始时间";
		static $year = "年";
		static $month = "月";
		static $contest_name = "比赛名称";
		static $contest_duration = "持续时间";
		static $empty_contest_name = "比赛名称不可为空";
		static $stime_before_now = "您准备设置一场开始于过去的比赛么？";
		static $wrong_start_time = "开始日期 / 时间输入有误";
		static $add_contest_success = "成功添加比赛";

		static $edit_contest = "编辑比赛";
		static $start_datetime = "开始日期时间";
		static $delete = "删除";
		static $no_problem = "没有此题";
		static $belong_to_this = "该题已经添加";
		static $belong_to_other = "该题属于其它比赛";

		static $admin = "管理";
		static $rejudge_success = "重评成功";
	}
?>