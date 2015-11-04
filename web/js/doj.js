var language = ['C', 'C++', 'Pascal', 'Python'];
var mode = ['c_cpp', 'c_cpp', 'pascal', 'python'];
var res_flag = ['Pending', 'AC', 'WA', 'TLE', 'MLE', 'RE', 'CE', 'OLE', 'Others', 'Judging'];
var replyText, topicText, codeEditor;
var r_lim = {}, t_lim = {};

(function($) {
	$.StartScreen = function() {
		var plugin = this;
		var width = (window.innerWidth > 0) ? window.innerWidth : screen.width;

		plugin.init = function() {
			setTilesAreaSize();
			if (width > 640) addMouseWheel();
		};

		var setTilesAreaSize = function() {
			var groups = $(".tile-group");
			var tileAreaWidth = 80;
			$.each(groups, function(i, t) {
				if (width <= 640) {
					tileAreaWidth = width;
				} else {
					tileAreaWidth += $(t).outerWidth() + 80;
				}
			});
			$(".tile-area").css({
				width: tileAreaWidth
			});
		};

		var addMouseWheel = function () {
			$(".tile-area").mousewheel(function(event, delta, deltaX, deltaY) {
				var page = $(document);
				var scroll_value = delta * 50;
				page.scrollLeft(page.scrollLeft() - scroll_value);
			});
		};

		plugin.init();
	}
})(jQuery);

function isStr(str)
{
	return Object.prototype.toString.call(str) === "[object String]";
}

function htmlspecialchars(str)
{
	str = str.replace(/&/g, '&amp;');
	str = str.replace(/</g, '&lt;');
	str = str.replace(/>/g, '&gt;');
	str = str.replace(/"/g, '&quot;');
	str = str.replace(/'/g, '&#039;');
	return str;
}

function stripTags(str)
{
	str = str.replace(/<\s*(\S*?)\s*[^>]*>(.|\n)*?<\s*\/\s*\1\s*>/mg, '');
	str = str.replace(/<.*?\s*\/>/g, '');
	return str;
}

function markdown(str)
{
	str = stripTags(str);
//	str = htmlspecialchars(str);
	return marked(str);
}

function showSearch()
{
	var  charm = $("#charmSearch");

	if (charm.data('hidden') == undefined)
		charm.data('hidden', true);

	if (!charm.data('hidden')) {

		charm.animate({
			right: -300
		});

		charm.data('hidden', true);
	} else {
		charm.animate({
			right: 0
		});
		charm.data('hidden', false);
	}
}

function showSettings()
{
	var  charm = $("#charmSettings");

	if (charm.data('hidden') == undefined) {charm.data('hidden', true);}

	if (!charm.data('hidden')) {

		charm.animate({
			right: -300
		});

		charm.data('hidden', true);
	} else {
		charm.animate({
			right: 0
		});
		charm.data('hidden', false);
	}
}

function setSearchPlace(el)
{
	var a = $(el);
	var text = a.text();
	var toggle = a.parents('label').children('.dropdown-toggle');

	toggle.text(text);
}

function showDialog(id)
{
	var dialog = $(id).data('dialog');
	dialog.open();
}

function showNote(d)
{
	if (d.res == 0)
		$.Notify({
			caption: '成功',
			content: d.msg,
			type: 'success',
			icon: '<span class="mif-checkmark"></span>'
		});
	else if(d.res == 1)
		$.Notify({
			caption: '失败',
			content: d.msg,
			type: 'alert',
			icon: '<span class="mif-cross"></span>'
		});
	else if (d.res == 2)
		$.Notify({
			caption: '警告',
			content: d.msg,
			type: 'warning',
			icon: '<span class="mif-warning"></span>'
		});
	else
		$.Notify({
			caption: '提示',
			content: d.msg,
			type: 'info',
			icon: '<span class="mif-info"></span>'
		});

	eval(d.cmd);
}

function formSubmit(x)
{
	if (x == 1) {
		$.post('?changeMsg', {
				"sex": $("input[name='sex']:checked").val(),
				"birth": $("#birth").val(),
				"school": $("#school").val(),
				"sign": $("#sign").val()
			},showNote,"json"
		);
	} else if (x == 2) {
		var uname = $("#user_name").val();
		var pwd = $("#u_pwd").val();

		if (uname.length < 5) {
			showNote({"res": 1, "msg": $("#user_name").data("wrong")});
			return false;
		}
		if (pwd.length < 6) {
			showNote({"res": 1, "msg": $("#u_pwd").data("wrong")});
			return false;
		}

		$.post('?changeName', {
				"password": pwd,
				"name": uname
			}, showNote, "json"
		);

		$("#u_pwd").val("");
	} else if (x == 3) {
		var o_pwd = $("#o_pwd").val();
		var n_pwd = $("#n_pwd").val();

		if (o_pwd.length < 6) {
			showNote({"res": 1, "msg": $("#o_pwd").data("wrong")});
			return false;
		}
		if (n_pwd.length < 6) {
			showNote({"res": 1, "msg": $("#u_pwd").data("wrong")});
			return false;
		}

		$.post('?changePwd', {
				"opwd": o_pwd,
				"npwd": n_pwd
			}, showNote, "json"
		);

		$("#o_pwd").val("");
		$("#n_pwd").val("");
	} else if (x == 4){
		var mail = $("#mail").val();
		var pwd = $("#m_pwd").val();

		if (pwd.length < 6) {
			showNote({"res": 1, "msg": $("#u_pwd").data("wrong")});
			return false;
		}

		$.post('?changeMail', {
				"password": pwd,
				"mail": mail
			}, showNote, "json"
		);

		$("#m_pwd").val("");
	}

	return false;
}

function showProblem(pid, tags)
{
	$.get("?problem", {"pid": pid}, function(d){
		if (d.res)
			showNote(d);
		else {
			var p = d.msg;
			$("#description").html(markdown(p.description));
			$("#inFormat").html(markdown(p.input));
			$("#outFormat").html(markdown(p.output));
			$("#sampleIn").html(markdown(p.sampleIn));
			$("#sampleOut").html(markdown(p.sampleOut));
			$("#hint").html(markdown(p.hint));
			$("#time_limit").html(p.time);
			$("#memory_limit").html(p.memory);

			var pMsg = $("#problemMsg");
			pMsg.height(pMsg.width());
			pMsg.highcharts({
				chart: {
					type: 'pie',
					options3d: {
						enabled: true,
						alpha: 45,
						beta: 0
					},
					backgroundColor: '#FFF'
				},
				credits: {
					enabled: false
				},
				title: {
					text: '该题提交概况',
					style: {
						color: '#000',
						font: 'bold 16px "Trebuchet MS", Verdana, sans-serif'
					}
				},
				tooltip: {
					pointFormat: '<span style="color:{point.color};">\u25CF</span>{series.name}: <b>{point.y}</b><br><span style="color:{point.color};">\u25CF</span>占比: <b>{point.percentage:.2f}%</b>'
				},
				plotOptions: {
					pie: {
						allowPointSelect: true,
						cursor: 'pointer',
						depth: 20,
						dataLabels: {
							enabled: true,
							format: '{point.name}'
						}
					}
				},
				series: [{
					type: 'pie',
					name: '次数',
					data: [
						['AC', parseInt(p.AC)],
						['WA', parseInt(p.WA)],
						['TLE', parseInt(p.TLE)],
						['MLE', parseInt(p.MLE)],
						['RE', parseInt(p.RE)],
						['CE', parseInt(p.CE)]
					],
					colors: ['#66CC66', '#FF6666', '#FFFF66', '#66CCFF', '#FF9900', '#CCCCCC']
				}]
			});

			$("#nowPid").html(pid);
			$("#nowPt").html(p.name);
			var hash = 'p' + pid;
			if (typeof tags != "undefined")
				hash = 't' + tags + '&' + hash;
			window.location.hash = hash;
			MathJax.Hub.Queue(["Typeset",MathJax.Hub]);
		}
	}, "json");
}

function showProblemList(e)
{
	var list = $(e).next();
	if (list.html() != "") return ;

	list.html('<a>...</a>');

	$.get("?problems", {'type': list.data('head')}, function(d){
		if (d.res)
			showNote(d);
		else {
			var x = eval(d.msg), plist = '';
			for (var i = 0; i < x.length; i++) {
				var id = x[i].id, t = x[i].name;
				plist += '<li><a href="javascript:showProblem(' + id + ');"> \
							<small><small>P' + id + '</small></small> \
							<span>' + t + '</span> \
						</a></li>';
			}

			list.html(plist);
		}
	}, "json");
}

function showProblemIndex(d)
{
	if (d.res)
		showNote(d);
	else {
		var x = parseInt(d.msg), index = '';
		var groupNum = Math.ceil(x / 100);
		for (var i = 0; i < groupNum; i++)
		{
			var head = 1000 + i * 100;
			index += '<li> \
						<a class="dropdown-toggle" onclick="showProblemList(this)"> \
							<span class="mif-menu icon"></span> \
							<span>P' + head + '&nbsp;<b><big>-&gt;</big></b></span> \
						</a> \
						<ul class="d-menu" data-role="dropdown" data-head="' + head + '"></ul> \
					</li>';
		}
		$("#problemIndex").html(index);
	}

	$("#wait").css("display", "none");
	$("#pcontent").css("display", "block");
}

function showTagedProblems(ps, tags)
{
	showDialog('#problems');
	$("#wait").css("display", "block");
	$("#pcontent").css("display", "none");
	
	var plist = '<li class="menu-title">————' + tags + '</li>';
	for (var i = 0; i < ps.length; i++) {
		var id = ps[i].id, t = ps[i].name;
		plist += '<li><a href="javascript:showProblem(' + id + ', \'' + tags + '\');"> \
					<small><small>p' + id + '</small></small> \
					<span>' + t + '</span> \
				</a></li>';
	}

	$("#problemIndex").html(plist);
	$("#wait").css("display", "none");
	$("#pcontent").css("display", "block");
	
	var p, hash = window.location.hash;
	if (hash.match(/#t.*&p\d+/)) {
		var th = hash.match(/&p\d+/)[0];
		p = th.substring(2, th.length);
	}

	if (typeof p == "undefined")
		p = ps[0].id;
	showProblem(p, tags);
}

function getTagRes(qtag)
{
	$.get('?problems&type=tag', {"tags": qtag}, function(d) {
		if (d.res)
			showNote(d);
		else {
			var r = eval(d.msg);
			$("#tagResNum").html(r.length);
			$("#tagResNum").unbind("click");
			if (r.length) {
				if (window.location.hash.match(/#t.*&p\d+/))
					showTagedProblems(r, qtag);
				else
					$("#tagResNum").click(function() {
						showTagedProblems(r, qtag);
					});
			}
		}
	}, "json")
}

function showTags()
{
	$('#noTagRes').css("display", "block");
	$('#tagRes').css("display", "none");
	showDialog('#tags');
	$.get("?tags", {}, function(d) {
		if (d.res)
			showNote(d);
		else {
			var t = eval(d.msg), tags = '';
			for (var i = 0; i < t.length; i++) {
				var g = t[i];
				var httag =
				'<div class="panel margin10"> \
					<div class="heading"> \
						<span class="title">' + g.group + '</span> \
					</div> \
					<div class="content">';

				for (var j = 0; j < g.set.length; j++)
					httag += '<a class="ptag">' + g.set[j] + '</a>';

				httag += '</div></div>';
				tags += httag;
			}

			$('#pageTags').html(tags);

			$('.ptag').click(function(){
				var e = $(this);
				if (e.hasClass('taged')) {
					e.removeClass('taged');
				} else {
					e.addClass('taged');
				}

				var stags = $('.taged');
				if (stags.length) {
					$('#noTagRes').css("display", "none");
					$('#tagRes').css("display", "block");
				} else {
					$('#noTagRes').css("display", "block");
					$('#tagRes').css("display", "none");
				}

				if (!stags.length) return ;

				$('#tagResNum').html('...');

				var atag = [], qtag;
				if (stags.length) {
					for (var i = 0; i < stags.length; i++)
						atag.push(stags.eq(i).html());
						qtag = atag.join('|');
				}

				getTagRes(qtag);
			})
		}
	}, "json");
}

function allProblem()
{
	showDialog("#problems");
	$("#wait").css("display", "block");
	$("#pcontent").css("display", "none");

	var hash = window.location.hash;
	if (!hash.match(/#p\d+/))
		if ($("#nowPid").html().match(/\d+/))
			hash = $("#nowPid").html();
		else hash = "1000";
	else hash = hash.substring(2, hash.length);
	setTimeout("showProblem(" + hash + ")", 0);

	$.get("?problems", {"type": "num"}, showProblemIndex, "json");
}

function showSubmit()
{
	var pid = $("#nowPid").html();
	if (pid != '') {
		$("#subPid").html(pid);
	}
	showDialog("#submit");
}

function submit()
{
	var lan = $("#subLan").val();
	$("#submit").data('dialog').close();
	$.post("?submit", {
		"pid": $("#subPid").html(),
		"language": lan,
		"code": codeEditor.getValue()
	}, showNote, "json");
}

function getResult(rid)
{
	console.log(rid);
	$.get('?holdResult', {"rid": rid}, function(d) {
		if (d.res)
			showNote(d);
		else
		{
			var msg = d.msg.match(/res_flag\[\d+\]/)[0];
			msg = d.msg.replace(/res_flag\[\d+\]/, eval(msg));
			showNote({"res": 3, "msg": msg});
		}
	}, "json");
}

function contest()
{
	$.Notify({
		caption: '抱歉',
		content: '还没有比赛！',
		type: 'info'
	});
}

function rowRecord(r)
{
	var row = "<tr>"
	r.res = parseInt(r.res);
	var deres = r.result;
	if (r.res !=6)
		deres = eval(deres);
	var utime = 0, umemory = 0;
	var s_res = res_flag[r.res];
	if (typeof deres != 'undefined' && r.res != 6)
		for (var i = 0; i < deres.length; i++) {
			utime = Math.max(utime, deres[i].time),
			umemory = Math.max(umemory, deres[i].memory);
		}

	row += "<td><span class='run_id'>#" + r.id + "</span></td>";
	row += "<td class='" + s_res + "'>" + s_res  + "</td>";
	row += "<td>" + utime + "</td>";
	row += "<td>" + umemory + "</td>";
	row += "<td><span class='go_pro'>P" + r.pid + " " + r.pname + "</span></td>";
	row += "<td>" + language[r.language] + "</td>";
	row += "<td>" + r.uname + "</td>";
	row += "<td>" + r.submit_time + "</td>";

	return row + "</tr>";
}

function refreshRecord()
{
	var r_pid = $("input[name='r_pid']").val();
	var r_uname = $("input[name='r_uname']").val();
	r_lim = {};

	if (r_pid != '')
		r_lim.pid = r_pid;
	if (r_uname != '')
		r_lim.uname = r_uname;

	$.get("?records", r_lim, function(d){
		if (d.res !=0)
			showNote(d);
		else {
			$("#recordHead").nextAll().remove();

			var msg = eval(d.msg), rs = '';
			var x = eval(msg.list);

			for (var i = 0; i < x.length; i++)
				rs += rowRecord(x[i]);

			$("#recordHead").after(rs);
			
			if (msg.more) {
				$("#noRecord").css("display", "none");
				$("#moreRecord").css("display", "block");
			} else {
				$("#noRecord").css("display", "block");
				$("#moreRecord").css("display", "none");
			}
		}
	}, "json");
}

function moreRecord()
{
	r_lim.offset = $("#recordTable tr").length - 1;
	$.get("?records", r_lim, function(d){
		if (d.res !=0)
			showNote(d);
		else {
			var msg = eval(d.msg), rs = '';
			var x = eval(msg.list);

			for (var i = 0; i < x.length; i++)
				rs += rowRecord(x[i]);

			$("#recordTable").append(rs);
			
			if (msg.more) {
				$("#noRecord").css("display", "none");
				$("#moreRecord").css("display", "block");
			} else {
				$("#noRecord").css("display", "block");
				$("#moreRecord").css("display", "none");
			}
		}
	}, "json");
}

function record()
{
	showDialog("#records");
	$("input[name='r_pid']").val('');
	$("input[name='r_uname']").val('');
	refreshRecord();
}

function myRecord()
{
	showDialog("#records");
	var myName = $("#myName").html();
	$("input[name='r_pid']").val('');
	$("input[name='r_uname']").val(myName);
	refreshRecord();
}

function rowTopic(t)
{
	var btn = "<button class='command-button full-size' data-id='" + t.id + "' onclick='showTopic(this)'>";
	btn += "<img class='icon gravatar' src='" + t.gravatar + "'>";
	btn += t.title;
	btn += "<small>" + t.uname + " - " + t.time + "</small>";
	btn += "</button>";
	return btn;
}

function refreshTopic()
{
	$("#topic").fadeOut();

	var t_uname = $("input[name='t_uname']").val();
	t_lim = {};

	if (t_uname != '')
		t_lim.uname = t_uname;

	$.get("?topics", t_lim, function(d){
		if (d.res !=0)
			showNote(d);
		else {
			$("#topList").html('');
			$("#topicList").html('');
			
			$("#tlist").fadeIn();
			
			var msg = eval(d.msg), tA = '', tB = '';
			var a = eval(msg.listA);
			var b = eval(msg.listB);

			for (var i = 0; i < a.length; i++)
				tA += rowTopic(a[i]);
			for (var i = 0; i < b.length; i++)
				tB += rowTopic(b[i]);

			$("#topList").html(tA);
			$("#topicList").html(tB);

			if (msg.more)
				$("#moreTopic").css("display", "block");
			else
				$("#moreTopic").css("display", "none");
		}
	}, "json");
}

function moreTopic()
{
	t_lim.offset = $("#topicList button").length - 1;

	$.get("?topics", t_lim, function(d){
		if (d.res !=0)
			showNote(d);
		else {
			var msg = eval(d.msg);
			var tB = $("#topicList").html();
			var b = eval(msg.listB);

			for (var i = 0; i < b.length; i++)
				tB += rowTopic(b[i]);

			$("#topicList").html(tB);

			if (msg.more)
				$("#moreTopic").css("display", "block");
			else
				$("#moreTopic").css("display", "none");
		}
	}, "json");
}

function topic()
{
	showDialog("#topics");
	$("input[name='t_uname']").val('');
	refreshTopic();
}

function myTopic()
{
	showDialog("#topics");
	var myName = $("#myName").html();
	$("input[name='t_uname']").val(myName);
	refreshTopic();
}

function topicHTML(t)
{
	var ht = "<div class='panel'>";
	ht += "<div class='heading'>";
	ht += "<img class='icon gravatar no-padding' src='" + t.gravatar + "'>";
	ht += "<span class='title'>" + t.uname + "<small> -- " + t.time + "</small></span>";
	ht += "</div>";
	ht += "<div class='content'>";
	ht += markdown(t.content);
	ht += "</div>";
	ht += "</div><br>";
	return ht;
}

function preview(x)
{
	if (x == 1) {
		var ht_view = replyText.getValue();
		ht_view = markdown(ht_view);
		$("#rep_view").html(ht_view);
	} else {
		var ht_view = topicText.getValue();
		ht_view = markdown(ht_view);
		$("#topic_view").html(ht_view);
	}

	MathJax.Hub.Queue(["Typeset",MathJax.Hub]);
}

function showTopic(e)
{
	$("#tlist").fadeOut();

	var tid = $(e).data("id");
	$.get("?topic", {"id": tid}, function(d){
		if (d.res)
			showNote(d);
		else {
			var msg = eval(d.msg);
			var content = '';

			$("#topic").fadeIn();
			$("#topicTitle").html(msg[0].title);

			for (var i = 0; i < msg.length; i++)
				content += topicHTML(msg[i]);

			$("#tcontent").html(content);
			MathJax.Hub.Queue(["Typeset",MathJax.Hub]);

			$("#btn-reply").unbind('click');
			$("#btn-reply").click(function(){
				$.post("?postTopic&type=reply", {
					"tid": msg[0].id,
					"content": replyText.getValue()
				}, function(d){
					showNote(d);
					showTopic(e);
				}, "json");
			});
		}
	}, "json");
}

function newTopic()
{
	showDialog("#newTopic");

	$("#btn-topic").unbind("click");
	$("#btn-topic").click(function(){
		$.post("?postTopic&type=main", {
			"title": $("input[name='topicTitle']").val(),
			"content": topicText.getValue()
		}, function(d){
			showNote(d);
			refreshTopic();
			$('#newTopic').data('dialog').close();
		}, "json");
	});
}

function logout()
{
	$.cookie("DOJSS", null);
	location.reload();
}

$(function(){
	marked.setOptions({
		highlight: function (code, lan) {
			var htcode;
			if (typeof lan == "undefined")
				htcode = hljs.highlightAuto(code).value;
			else htcode = hljs.highlightAuto(code, [lan]).value;
			return htcode;
		}
	});

	MathJax.Hub.Config({
		showProcessingMessages: false,
		tex2jax: {
			inlineMath: [["$", "$"], ["\\(", "\\)"]],
			processEscapes: false
		},
		menuSettings: {
			zoom: "Click"
		}
	});
});

$(function(){
	var current_bg = $("body").css("background");

	$(".schemeButtons .button").hover(
			function(){
				var b = $(this);
				var bg = "url(bg/" + b.data("bg") + ".png) top center /cover no-repeat fixed";
				$("body").css("background", bg);
			},
			function(){
				$("body").css("background", current_bg);
			}
	);

	$(".schemeButtons .button").on("click", function(){
		var b = $(this);
		var scheme = "tile-area-scheme-" +  b.data('scheme');

		if (current_bg.indexOf("bg/" + b.data("bg") + ".png") >= 0)
		{
			showSettings();
			return;
		}

		current_bg = "url(bg/" + b.data("bg") + ".png) top center /cover no-repeat fixed";
		$("body").css("background", current_bg);		

		$.post("?changeBg", {
				"bg": b.data("bg")
			},showNote,"json"
		);

		showSettings();
	});
});

$(function(){
	$.StartScreen();

	var tiles = $(".tile, .tile-small, .tile-sqaure, .tile-wide, .tile-large, .tile-big, .tile-super");

	$.each(tiles, function(){
		var tile = $(this);
		setTimeout(function(){
			tile.css({
				opacity: 1,
				"-webkit-transform": "scale(1)",
				"transform": "scale(1)",
				"-webkit-transition": ".3s",
				"transition": ".3s"
			});
		}, Math.floor(Math.random() * 500));
	});

	$(".tile-group").animate({
		left: 0
	});
});

$(function() {
	var hash = window.location.hash;
	if (hash.match(/#p\d+/))
		setTimeout(allProblem, 0);
	else if (hash.match(/#t.*&p\d+/)) {
		var th = hash.match(/#t.*&/)[0];
		var t = th.substring(2, th.length - 1);
		setTimeout("getTagRes('" + t + "')", 0);
	}

	replyText = ace.edit("replyText");
	replyText.setTheme("ace/theme/xcode");
	replyText.getSession().setMode("ace/mode/markdown");
	replyText.$blockScrolling = Infinity;

	topicText = ace.edit("topicText");
	topicText.setTheme("ace/theme/xcode");
	topicText.getSession().setMode("ace/mode/markdown");
	topicText.$blockScrolling = Infinity;

	codeEditor = ace.edit("subCode");
	codeEditor.setTheme("ace/theme/xcode");
	codeEditor.getSession().setMode("ace/mode/c_cpp");
	codeEditor.$blockScrolling = Infinity;

	$("#subLan").change(function() {
		var lan = $(this).val();
		codeEditor.getSession().setMode("ace/mode/" + mode[lan]);
	});

	$("body").on("click", ".run_id", function() {
		var rid = $(this).html().match(/\d+/)[0];
		$.get('?result', {"rid": rid}, function(d) {
			if (d.res)
				showNote(d);
			else {
				var res = d.msg, r = res.result;
				if (res.res != 6)
					r = eval(r);

				$("#run_id").html(rid);
				$("#run_pid").html(res.pid);

				if ("code" in res)
					res.code = '#### Code:\n\n```\n' + res.code + '\n```';
				else res.code = '';
				$("#run_code").html(marked(res.code));

				if (res.res != 6) {
					var p_res = '#### Points:\n\n|数据点|结果|时间(ms)|内存(KB)|\n|:---|:---:|:---:|:---:|\n';
					for (var i = 0; i < r.length; i++)
						p_res += '|#' + i + '|' + res_flag[r[i].res] + '|' + r[i].time + '|' + r[i].memory + '|\n';
					$("#points_res").html(marked(p_res));
				} else {
					var p_res = "<h4>Compile Message</h4>";
					p_res += "<pre class='ce_msg'>" + r + "</pre>";
					$("#points_res").html(p_res);
				}

				showDialog("#result");
			}
		}, "json");
	});

	$("body").on("click", ".go_pro", function() {
		var pid = $(this).html().match(/^P\d+/)[0];
		pid = pid.match(/\d+/)[0];
		window.location.hash = 'p' + pid;
		setTimeout(allProblem, 0);
		$('#records').data('dialog').close();
	});
});
