var pi = ["des", "in", "out", "hint"];
var p = [null, null, null, null];
var n = [null ,null, null, null];
var frame_first_load = true;
var pindex = 1000;

function showNote(d) {
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
	else
		$.Notify({
			caption: '警告',
			content: d.msg,
			type: 'warning',
			icon: '<span class="mif-warning"></span>'
		});

	eval(d.cmd);
}

function preview(type, id) {
	if (type == 1)
		$("#p" + pi[id] + "view").html(marked(p[id].getValue()));
	else
		$("#n" + pi[id] + "view").html(marked(n[id].getValue()));
	MathJax.Hub.Queue(["Typeset",MathJax.Hub]);
}

function changeProblem(pid) {
	$.get("?problem", {"pid": pid}, function(d){
		if (d.res)
			showNote(d);
		else {
			var pr = d.msg;
			$("#pid").html(pr.id);
			$("#ptitle").val(pr.name);
			$("#ptime").val(pr.time);
			$("#pmem").val(pr.memory);
			$("#ptags").val(pr.tags);
			p[0].setValue(pr.description);
			p[1].setValue(pr.input);
			p[2].setValue(pr.output);
			p[3].setValue(pr.hint);
			for (var i = 0; i < 4; i++)
				p[i].navigateFileStart();
			$("#plist").hide();
			$("#pchange").show();
			$(".pgroup.current").removeClass("current");
		}
	}, "json");
}
function saveChangeProblem()
{
	var pid = $("#pid").html();
	$.post("?changeProblem", {
		"pid": pid,
		"title": $("#ptitle").val(),
		"time": $("#ptime").val(),
		"memory": $("#pmem").val(),
		"tags": $("#ptags").val(),
		"description": p[0].getValue(),
		"input": p[1].getValue(),
		"output": p[2].getValue(),
		"hint": p[3].getValue()
	}, function(d) {
		if (d.res == 0)
			$("#pchange").hide();
		showNote(d);
	}, "json");
}

function addProblem()
{
	$.post("?addProblem", {
		"title": $("#nptitle").val(),
		"time": $("#nptime").val(),
		"memory": $("#npmem").val(),
		"tags": $("#nptags").val(),
		"description": n[0].getValue(),
		"input": n[1].getValue(),
		"output": n[2].getValue(),
		"hint": n[3].getValue()
	}, function(d) {
		if (d.res == 0) {
			$("#npid").html(parseInt($("#npid").html()) + 1);
			$("#nptitle").val('');
			$("#nptime").val('');
			$("#npmem").val('');
			$("#nptags").val('');
			for (var i = 0; i < 4; i++)
				n[i].setValue('');
		}
		showNote(d);
	}, "json");
}

function getKey()
{
	var flag = 0;
	if ($("#setAdmin").is(":checked"))
		flag = 1;

	$.get("?key", {
			mail: $("#mail").val(),
			admin: flag
	}, function(d) {
		if (d.res)
			showNote(d);
		else $("#key").val(d.msg);
	}, "json");
}

function reJudge(w, s)
{
	$.get("?reJudge", {way: w, id: $(s).val()}, showNote, "json");
}

$(function() {
	for (var  i =  0; i < 4; i++) {
		p[i] = ace.edit('p' + pi[i]);
		p[i].setTheme("ace/theme/xcode");
		p[i].getSession().setMode("ace/mode/markdown");
		p[i].setFontSize(16);
		p[i].$blockScrolling = Infinity;
	}

	for (var  i =  0; i < 4; i++) {
		n[i] = ace.edit('n' + pi[i]);
		n[i].setTheme("ace/theme/xcode");
		n[i].getSession().setMode("ace/mode/markdown");
		n[i].setFontSize(16);
		n[i].$blockScrolling = Infinity;
	}

	$("[href=#problems]").click(function() {
		$.get("?problems", {"type": "num"}, function(d) {
			if (d.res)
				showNote(d);
			else {
				var x = parseInt(d.msg), index = '';
				var groupNum = Math.ceil(x / 100);
				for (var i = 0; i < groupNum; i++) {
					var head = 1000 + i * 100;
					index += '<span class="item pgroup" data-head="' + head + '">P' + head + ' &gt;</span>';
				}
				$("#pindex").html(index);
				$(".pgroup").click(function() {
					$(".pgroup.current").removeClass("current");
					$(this).addClass("current");

					$.get("?problems", {'type': $(this).data('head')}, function(d) {
						if (d.res)
							showNote(d);
						else {
							var x = eval(d.msg), plist = '|题号|名称|修改|\n|---|---|---|\n';
							for (var i = 0; i < x.length; i++) {
								var id = x[i].id, t = x[i].name;
								plist += '|' + id + '|' + t + '|<button onclick="changeProblem(' + id + ')">点此修改</button>|\n';
							}
							plist = marked(plist);
							$("#plist").html(plist);
							$("#plist").show();
							$("#pchange").hide();
						}
					}, "json");

				});
			}
		}, "json");
	});

	$("[href=#addProblem]").click(function() {
		$.get("?problems", {"type": "num"}, function(d) {
			if (d.res)
				showNote(d);
			else $("#npid").html(parseInt(d.msg) + 1000);
		}, "json");
	});

	$("a[href=#datas]").click(function() {
		$("#data-iframe").hide();
		var h = $(window).height() - 70;
		$('#data-iframe').height(h);
		$('#data-iframe').attr('src', document.location.protocol + '//' + window.location.host + '/file/');
		$("#data-iframe").show();
	});
	$("a[href!=#datas]").click(function() {
		$("#data-iframe").hide();
		$('#data-iframe').removeAttr('src');
	});
});

$(function(){
	marked.setOptions({
		highlight: function (code, lan) {
			console.log(code);
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
