var color = ['#666' ,'#FF6666', '#66CC66', '#FFCC00', '#66CCFF', '#FF66FF', '#999999'];
var hcolor = ['#666' ,'#FF8888', '#88EE88', '#FFCC66', '#88EEFF', '#FF99FF', '#CCCCCC'];

var page_num = 50;

var can_update_tags = true;

if (typeof admin == "undefined")
	var admin = false;

$(".button-collapse").sideNav();
$(".dropdown-button").dropdown();

$(".go-home").click(function() {
	location.hash = "#home";
});
$(".logout").click(function() {
	$.cookie("DOJSS", null);
	location.reload();
});

for (var i = 0; i < language.length; i++)
	$("#r-language").append(makeOption(i, language[i]));
for (var i = 0; i < language.length; i++)
	$("#p-language").append(makeOption(i, language[i]));
for (var i = 0; i < abbrs.length; i++)
	$("#r-result").append(makeOption(i, abbrs[i]));

$("#p-language").children("option").eq(1).attr("selected", true);
$("select").material_select();

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
	messageStyle: "none",
	tex2jax: {
		inlineMath: [["$", "$"], ["\\(", "\\)"]]
	},
	menuSettings: {
		zoom: "Click"
	}
});

$.scrollUp({
	scrollName: 'scrollUp', // Element ID
	topDistance: '300', // Distance from top before showing element (px)
	topSpeed: 300, // Speed back to top (ms)
	animation: 'fade', // Fade, slide, none
	animationInSpeed: 200, // Animation in speed (ms)
	animationOutSpeed: 200, // Animation out speed (ms)
	scrollText: $("#gtm").html(), // Text for element
	activeOverlay: false, // Set CSS color to display scrollUp active point, e.g '#00FFFF'
});

$("#scrollUp").css("z-index", 10);
$("#p-fab").css("z-index", 11);

var mitem = [["b", "problems"], ["o", "records"], ["t", "contests"], ["k", "rank"]];
function changeUI() {
	var hash = location.hash;
	hash = hash.replace(/#/g, "");
	hash = hash.split("\/");

	showLoad();
	$(".content").hide();
	$(".button-collapse").sideNav("hide");
	$("nav .active").removeClass("active");
	for (var i = 0; i < 4; i++)
		if (hash[0][3] == mitem[i][0])
			$(".m" + mitem[i][1]).addClass("active");

	if (hash[0] == "home") {
		showHome();
	} else if (hash[0] == "problems") {
		showProblems(hash);
	} else if (hash[0] == "records") {
		showRecords(hash);
	} else if (hash[0] == "contests") {
		showContests();
	} else if (hash[0] == "rank") {
		showRank();
	} else if (hash[0] == "problem") {
		showProblem(hash);
	} else if (hash[0] == "contest") {
		showContest(hash);
	} else if (hash[0] == "modify") {
		showSetting();
	} else {
		location.hash = "#home";
	}
}

changeUI();

function update_statistic(callback) {
	$("#myChart").height($("#myChart").width() * 0.7);
	$("#ojChart").height($("#ojChart").width() * 0.65);

	$.ajax({
		url: "?statistic",
		type: "GET",
		success: function(d) {
			if (d.success) {
				var myctx = document.getElementById("myChart").getContext("2d");
				var ojctx = document.getElementById("ojChart").getContext("2d");

				var myData = [];
				for (var i = 1; i <= 6; i++)
					myData.push({
						value: d.i[i - 1],
						color: color[i],
						highlight: hcolor[i],
						label: state[i]
					});

				var times = [], tot = [], ac = [];
				for (var i = 0; i < 7; i++) {
					times.push(d.oj[i][0]);
					tot.push(d.oj[i][1]);
					ac.push(d.oj[i][2]);
				}
				var ojData = {
					labels: times,
					datasets: [
						{
							label: "Submit",
							fillColor: "rgba(220, 220, 220, 0.2)",
							strokeColor: "rgba(220, 220, 220, 1)",
							pointColor: "rgba(220, 220, 220, 1)",
							pointStrokeColor: "#FFF",
							pointHighlightFill: "#FFF",
							pointHighlightStroke: "rgba(220, 220, 220, 1)",
							data: tot
						},
						{
							label: "Accept",
							fillColor: "rgba(151, 187, 205, 0.2)",
							strokeColor: "rgba(151, 187, 205, 1)",
							pointColor: "rgba(151, 187, 205, 1)",
							pointStrokeColor: "#FFF",
							pointHighlightFill: "#FFF",
							pointHighlightStroke: "rgba(151, 187, 205, 1)",
							data: ac
						}
					]
				};

				var myChart = new Chart(myctx).Doughnut(myData, {animateScale: true});
				var ojChart = new Chart(ojctx).Line(ojData);
			}

			if (typeof callback == "function")
				callback();
		},
		error: function(req, msg) {
			console.error(msg);
		},
		dataType : "json"
	});
}
function update_recent_problems(callback) {
	$.ajax({
		url: "?recentProblems",
		type: "GET",
		success: function(d) {
			if (d.success) {
				var ps = d.data;
				var list = "";
				for (var i = 0; i < ps.length; i++) {
					var item = $("#rpim").html();
					item = item.replace(/%pid%/g, ps[i].id);
					if (ps[i].flag) {
						var f = ps[i].flag;
						ps[i].name += makeSup(abbrs[f], abbrs[f]);
					}
					item = item.replace(/%pname%/g, ps[i].name);
					var tags = ps[i].tags.split("|"), tlist = "";
					for (var j = 0; tags[j] != "" && j < tags.length; j++)
						tlist += makeTag(tags[j]);
					item = item.replace(/%tags%/g, tlist);
					list += item;
				}
				$("#rp").html(list);
				for (var i = 1; i <= 6; i++)
					$("#rp ." + abbrs[i]).css("color", color[i]);
			}

			if (typeof callback == "function")
				callback();
		},
		error: function(req, msg) {
			console.error(msg);
		},
		dataType: "json"
	});
}
function update_recent_contests(callback) {
	$.ajax({
		url: "?recentContests",
		type: "GET",
		success: function(d) {
			if (d.success) {
				var cs = d.data;
				var list = "";
				for (var i = 0; i < cs.length; i++) {
					var item = $("#rcim").html();
					item = item.replace(/%cid%/g, cs[i].id);
					item = item.replace(/%cname%/g, cs[i].name);
					item = item.replace(/%state%/g, cs[i].state);
					item = item.replace(/%time%/g, cs[i].time);
					item = item.replace(/%stime%/g, cs[i].start_time);
					list += item;
				}
				$("#rc").html(list);
			}

			if (typeof callback == "function")
				callback();
		},
		error: function(req, msg) {
			console.error(msg);
		},
		dataType: "json"
	});
}
function showHome() {
	var loaded = 0;
	$("#Home").show();
	update_statistic(function() { if (3 == ++loaded) hideLoad(); });
	update_recent_problems(function() { if (3 == ++loaded) hideLoad(); });
	update_recent_contests(function() { if (3 == ++loaded) hideLoad(); });
}

function update_problem_index(callback) {
	$.ajax({
		url: "?problems&max",
		type: "GET",
		success: function(d) {
			if (d.success) {
				var page = (d.max - 999) / page_num + 1;
				page = parseInt(page);
				var list = makeA("#problems", "Page1");
				for (var i = 1, s, p; i < page; i++)
					s = page_num * i, p = i + 1,
					list += makeA("#problems/o/" + s, "Page" + p + " ");
				$("#page_index a").remove();
				$("#page_index").append(list);
			} else
				makeToast(d.error);

			if (typeof callback == "function")
				callback();
		},
		error: function(req, msg) {
			console.error(msg);
		},
		dataType: "json"
	});
}
function bind_more_problems(data, button) {
	button.click(function() {
		button.html($("#pml").html());
		$.ajax({
			url: "?problems",
			type: "GET",
			data: data,
			success: function(d) {
				if (d.success) {
					var ps = d.data;
					var list = "";
					for (var i = 0; i < Math.min(ps.length, page_num); i++) {
						var item = $("#pim").html();
						item = item.replace(/%pid%/g, ps[i].id);
						if (ps[i].flag) {
							var f = ps[i].flag;
							ps[i].name += makeSup(abbrs[f], abbrs[f]);
						}
						item = item.replace(/%pname%/g, ps[i].name);
						var tags = ps[i].tags.split("|"), tlist = "";
						for (var j = 0; tags[j] != "" && j < tags.length; j++)
							tlist += makeTag(tags[j]);
						item = item.replace(/%tags%/g, tlist);
						if (ps[i].hide == "1")
							item = $(item).addClass("hide").prop("outerHTML");
						list += item;
					}
					if (ps.length > page_num)
						list += $("#pmm").html();

					setTimeout(function() {
						button.prop("outerHTML", list);
						for (var i = 1; i <= 6; i++)
							$("#plist ." + abbrs[i]).css("color", color[i]);

						data.o += page_num;
						if (ps.length > page_num)
							bind_more_problems(data, $("#plist .more"));

						if (admin) init_problems_switch();
					}, 500);
				} else
					makeToast(d.error);
			},
			error: function(req, msg) {
				console.error(msg);
			},
			dataType: "json"
		});
	});
}
function update_problem_list(data, callback) {
	$.ajax({
		url: "?problems",
		type: "GET",
		data: data,
		success: function(d) {
			if (d.success) {
				var ps = d.data;
				var list = "";
				for (var i = 0; i < Math.min(ps.length, page_num); i++) {
					var item = $("#pim").html();
					item = item.replace(/%pid%/g, ps[i].id);
					if (ps[i].flag) {
						var f = ps[i].flag;
						ps[i].name += makeSup(abbrs[f], abbrs[f]);
					}
					item = item.replace(/%pname%/g, ps[i].name);
					var tags = ps[i].tags.split("|"), tlist = "";
					for (var j = 0; tags[j] != "" && j < tags.length; j++)
						tlist += makeTag(tags[j]);
					item = item.replace(/%tags%/g, tlist);
					if (ps[i].hide == "1")
						item = $(item).addClass("hide").prop("outerHTML");
					list += item;
				}
				if (ps.length > page_num)
					list += $("#pmm").html();

				$("#plist").html(list);
				for (var i = 1; i <= 6; i++)
					$("#plist ." + abbrs[i]).css("color", color[i]);

				data.o = parseInt(data.o) ? parseInt(data.o) : 0;
				data.o += page_num;
				if (ps.length > page_num)
					bind_more_problems(data, $("#plist .more"));
			} else
				makeToast(d.error);

			if (typeof callback == "function")
				callback();
		},
		error: function(req, msg) {
			console.error(msg);
		},
		dataType: "json"
	});
}
function bind_tags() {
	can_update_tags = false;
	$(".tags-set .tag").click(function() {
		if ($(this).hasClass("selected"))
			$(this).removeClass("selected");
		else
			$(this).addClass("selected");
		var tags = [];
		$(".tags-set .tag.selected").each(function() {
			tags.push($(this).html());
		});
		var ts = tags.join("|"), hash = "#problems";
		if (ts != "")
			hash += "/t/" + ts;
		location.hash = hash;
	});
}
function update_problem_tags(callback) {
	$.ajax({
		url: "?problems&tags",
		type: "GET",
		success: function(d) {
			if (d.success) {
				var ts = d.data, list = "";
				for (var i = 0; i < ts.length; i++) {
					var item = $("#tim").html(), set = "";
					item = item.replace("%group%", ts[i].group);
					for (var j = 0; j < ts[i].set.length; j++)
						set += makeTag(ts[i].set[j]) + " ";
					item = item.replace("%set%", set);
					list += item;
				}
				$("#tlist").html(list);
				$("#tlist h6").click(function() {
					$(this).next().slideToggle("slow");
					$(this).children("i").fadeToggle("slow");
				});
				$("#tlist h6:lt(2)").trigger("click");
				bind_tags();
			} else
				makeToast(d.error);

			if (typeof callback == "function")
				callback();
		},
		error: function(req, msg) {
			console.error(msg);
		},
		dataType: "json"
	});
}
function showProblems(hash) {
	var loaded = 0;
	$("#Problems").show();
	update_problem_index(function() { if (3 == ++loaded) hideLoad(); });
	var data = {};
	if (hash.length > 2)
		if (hash[1] == 'o' || hash[1] == 't')
			data[hash[1]] = hash[2];

	update_problem_list(data, function() {
		if (3 == ++loaded) hideLoad();
		if (admin) init_problems_switch();
	});
	if (can_update_tags)
		update_problem_tags(function() {
			if (3 == ++loaded) hideLoad();
			if (data.t == null)
				$(".tags-set .tag.selected").removeClass("selected");
			else {
				var set = data.t.split("|");
				$(".tags-set .tag").each(function() {
					if (set.indexOf($(this).html()) > -1)
						$(this).addClass("selected");
					else
						$(this).removeClass("selected");
				});
			}
		})
	else if (3 == ++loaded)
		hideLoad();
}

$("#r-filter").click(function() {
	$(this).addClass("disabled");
	$(this).attr("disabled", true);

	var list = ["#records"];
	if ($("#r-pid").val() != "")
		list.push("p/" + $("#r-pid").val());
	if ($("#r-uname").val() != "")
		list.push("u/" + $("#r-uname").val());
	if ($("#r-language").val() != -1)
		list.push("l/" + $("#r-language").val());
	if ($("#r-result").val() != -1)
		list.push("r/" + $("#r-result").val());

	if (location.hash != list.join("/"))
		location.hash = list.join("/");
	else changeUI();
});
function bind_record_click() {
	$("#rlist li").unbind("click").click(function() {
		var r = $(this).children(".collapsible-body");
		var rid = $(this).data("id");
		if (r.html() == "" || r.html() == $("#rwm").html()) {
			r.html($("#rwm").html());
			$.ajax({
				url: "?record",
				type: "GET",
				data: {id: rid},
				success: function(d) {
					if (d.success) {
						var ht = $("#rdm").html(), points = "", t = "";
						if (d.result) {
							if (d.res == 6) {
								ht = ht.replace("Points", "Compile Error");
								t = $("#rdim-ce").html();
								t = t.replace(/%ce%/, d.result.ce);
							} else {
								var res = d.result;
								for (var i = 0; i < res.length; i++) {
									var item = $("#rdim").html();
									item = item.replace(/%kth%/g, i);
									item = item.replace(/%res%/g, abbrs[res[i].res]);
									item = item.replace(/%utime%/g, res[i].time);
									item = item.replace(/%umemory%/g, res[i].memory);
									t += item;
								}
							}
						}

						ht = ht.replace(/%detail%/, t);
						ht = ht.replace(/%lan%/, lan[d.language]);
						ht = ht.replace("%code%", d.code.replace(/\$/g, "&#36;"));
						setTimeout(function() {
							r.html(ht);
							for (var i = 1; i <= 6; i++)
								$(".ritem ." + abbrs[i]).css("color", color[i]);
							r.find("blockquote").css("border-color", hcolor[d.res]);
							r.find("pre code").each(function(i, block) {
								if (!$(this).hasClass("hljs"))
									hljs.highlightBlock(block);
							});
						}, 500);
					}
				},
				error: function(req, msg) {
					console.error(msg);
				},
				dataType: "json"
			});
		}
	});
}
function bind_more_records(data, button) {
	button.click(function() {
		button.html($("#rwm").html());
		$.ajax({
			url: "?records",
			type: "GET",
			data: data,
			success: function(d) {
				if (d.success) {
					var rs = d.data, list = "";
					for (var i = 0; i < Math.min(rs.length, page_num); i++) {
						var item = $("#rim").html();
						item = item.replace(/%rid%/g, rs[i].id);
						item = item.replace(/%res%/g, abbrs[rs[i].res]);
						item = item.replace(/%uname%/g, rs[i].uname);
						item = item.replace(/%pid%/g, rs[i].pid);
						item = item.replace(/%pname%/g, rs[i].pname);
						item = item.replace(/%utime%/g, rs[i].utime);
						item = item.replace(/%umemory%/g, rs[i].umemory);
						item = item.replace(/%stime%/g, rs[i].submit_time);
						item = item.replace(/%language%/g, language[rs[i].language]);
						list += item;
					}
					if (rs.length > page_num)
						list += $("#rmm").html();

					setTimeout(function() {
						button.prop("outerHTML", list);
						for (var i = 1; i <= 6; i++) {
							$("#rlist span." + abbrs[i]).css("color", hcolor[i]);
							$("#rlist sup." + abbrs[i]).css("color", color[i]);
						}
						$('.collapsible').collapsible();
						bind_record_click();

						data.o += page_num;
						if (rs.length > page_num)
							bind_more_records(data, $("#rlist .more"));
					}, 500);
				} else
					makeToast(d.error);
				if (typeof callback == "function")
					callback();
			},
			error: function(req, msg) {
				console.error(msg);
			},
			dataType: "json"
		});
	});
}
function update_records_list(data, callback) {
	$.ajax({
		url: "?records",
		type: "GET",
		data: data,
		success: function(d) {
			if (d.success) {
				var rs = d.data, list = "";
				for (var i = 0; i < Math.min(rs.length, page_num); i++) {
					var item = $("#rim").html();
					item = item.replace(/%rid%/g, rs[i].id);
					item = item.replace(/%res%/g, abbrs[rs[i].res]);
					item = item.replace(/%uname%/g, rs[i].uname);
					item = item.replace(/%pid%/g, rs[i].pid);
					item = item.replace(/%pname%/g, rs[i].pname);
					item = item.replace(/%utime%/g, rs[i].utime);
					item = item.replace(/%umemory%/g, rs[i].umemory);
					item = item.replace(/%stime%/g, rs[i].submit_time);
					item = item.replace(/%language%/g, language[rs[i].language]);
					list += item;
				}
				if (rs.length > page_num)
					list += $("#rmm").html();

				$("#rlist").html(list);
				for (var i = 1; i <= 6; i++) {
					$("#rlist span." + abbrs[i]).css("color", hcolor[i]);
					$("#rlist sup." + abbrs[i]).css("color", color[i]);
				}
				$('.collapsible').collapsible();
				bind_record_click();
                
                data.o = page_num;
				if (rs.length > page_num)
					bind_more_records(data, $("#rlist .more"));
			} else
				makeToast(d.error);
			if (typeof callback == "function")
				callback();
		},
		error: function(req, msg) {
			console.error(msg);
		},
		dataType: "json"
	});
}
function showRecords(hash) {
	$("#Records").show();

	var data = {};
	for (var i = 1; i < hash.length - 1; i += 2)
		if (['p', 'u', 'l', 'r'].indexOf(hash[i]) != -1)
			data[hash[i]] = hash[i + 1];

	if (data.p != null)
		$("#r-pid").val(data.p);
	else $("#r-pid").val("");
	if (data.u != null)
		$("#r-uname").val(data.u);
	else $("#r-uname").val("");
	if (data.l != null)
		$("#r-language").val(data.l);
	else $("#r-language").val(-1);
	if (data.r != null)
		$("#r-result").val(data.r);
	else $("#r-result").val(-1);

	update_records_list(data, function() {
		$("#r-filter").removeClass("disabled");
		$("#r-filter").attr("disabled", false);
		hideLoad();
	});
}

function bind_more_contests(data, button) {
	button.click(function() {
		button.html($("#cml").html());
		$.ajax({
			url: "?contests",
			type: "GET",
			data: data,
			success: function(d) {
				if (d.success) {
					var cs = d.data;
					var list = "";
					for (var i = 0; i < Math.min(cs.length, page_num); i++) {
						var item = $("#cim").html();
						item = item.replace(/%cid%/g, cs[i].id);
						item = item.replace(/%cname%/g, cs[i].name);
						item = item.replace(/%state%/g, cs[i].state);
						item = item.replace(/%time%/g, cs[i].time);
						item = item.replace(/%stime%/g, cs[i].start_time);
						if (cs[i].hide == "1")
							item = $(item).addClass("hide").prop("outerHTML");
						list += item;
					}
					if (cs.length > page_num)
						list += $("#cmm").html();
					setTimeout(function() {
						button.prop("outerHTML", list);

						data.o += page_num;
						if (cs.length > page_num)
							bind_more_contests(data, $("#clist .more"));

						if (admin) init_contests_switch();
					}, 500);
				} else
					makeToast(d.error);

				if (typeof callback == "function")
					callback();
			},
			error: function(req, msg) {
				console.error(msg);
			},
			dataType: "json"
		});
	});
}
function update_contest_list(callback) {
	$.ajax({
		url: "?contests",
		type: "GET",
		success: function(d) {
			if (d.success) {
				var cs = d.data;
				var list = "";
				for (var i = 0; i < Math.min(cs.length, page_num); i++) {
					var item = $("#cim").html();
					item = item.replace(/%cid%/g, cs[i].id);
					item = item.replace(/%cname%/g, cs[i].name);
					item = item.replace(/%state%/g, cs[i].state);
					item = item.replace(/%time%/g, cs[i].time);
					item = item.replace(/%stime%/g, cs[i].start_time);
					if (cs[i].hide == "1")
						item = $(item).addClass("hide").prop("outerHTML");
					list += item;
				}
				if (cs.length > page_num)
					list += $("#cmm").html();

				$("#clist").html(list);

				if (cs.length > page_num)
					bind_more_contests({o: page_num}, $("#clist .more"));

				if (admin) init_contests_switch();
			} else
				makeToast(d.error);

			if (typeof callback == "function")
				callback();
		},
		error: function(req, msg) {
			console.error(msg);
		},
		dataType: "json"
	});
}
function showContests() {
	$("#Contests").show();
	update_contest_list(function() {
		hideLoad();
	});
}

function bind_more_rank(data, button) {
	button.click(function() {
		button.html($("#uml").html());
		$.ajax({
			url: "?rank",
			data: data,
			type: "GET",
			success: function(d) {
				if (d.success) {
					var us = d.data, list = "";
					for (var i = 0 ; i < Math.min(us.length, page_num); i++) {
						var item = $("#uim").html();
						item = item.replace(/%rank%/g, data.o + i + 1);
						item = item.replace(/%name%/g, us[i].name);
						item = item.replace(/%sign%/g, us[i].sign);
						item = item.replace(/%solve%/g, us[i].solve);
						item = item.replace(/%submit%/g, us[i].submit);
						var radio = (100 * us[i].solve / us[i].submit).toFixed(2);
						if (isNaN(radio)) radio = 0;
						item = item.replace(/%radio%/g, radio);
						list += item;
					}
					if (us.length > page_num)
						list += $("#umm").html();

					setTimeout(function() {
						button.prop("outerHTML", list);

						data.o += page_num;
						if (us.length > page_num)
							bind_more_rank(data, $("#ranklist .more"));
					}, 500);
				} else
					makeToast(d.error);

				if (typeof callback == "function")
					callback();
			},
			error: function(req, msg) {
				console.error(msg);
			},
			dataType: "json"
		});
	});
}
function update_rank_list(callback) {
	$.ajax({
		url: "?rank",
		type: "GET",
		success: function(d) {
			if (d.success) {
				var us = d.data, list = "";
				for (var i = 0 ; i < Math.min(us.length, page_num); i++) {
					var item = $("#uim").html();
					item = item.replace(/%rank%/g, i + 1);
					item = item.replace(/%name%/g, us[i].name);
					item = item.replace(/%sign%/g, us[i].sign);
					item = item.replace(/%solve%/g, us[i].solve);
					item = item.replace(/%submit%/g, us[i].submit);
					var radio = (100 * us[i].solve / us[i].submit).toFixed(2);
					if (isNaN(radio)) radio = 0;
					item = item.replace(/%radio%/g, radio);
					list += item;
				}
				if (us.length > page_num)
					list += $("#umm").html();

				$("#ranklist").html(list);

				if (us.length > page_num)
					bind_more_rank({o: page_num}, $("#ranklist .more"));
			} else
				makeToast(d.error);

			if (typeof callback == "function")
				callback();
		},
		error: function(req, msg) {
			console.error(msg);
		},
		dataType: "json"
	});
}
function showRank() {
	$("#Rank").show();
	update_rank_list(function() {
		hideLoad();
	});
}

var pChart = null, pl;
var pcan = document.getElementById("pChart");
var pctx = pcan.getContext("2d");
function update_problem_statistic(data) {
	var pData = [], nodata = true;
	for (var i = 1; i <= 6; i++) {
		if (data[i - 1] != 0)
			nodata = false;
		pData.push({
			value: data[i - 1],
			color: color[i],
			highlight: hcolor[i],
			label: state[i]
		});
	}

	if (nodata) {
		$("#pChart").height(0);
		$("#pChart").next().show();
	} else {
		$("#pChart").height($("#pChart").width() * 0.7);
		$("#pChart").next().hide();

		if (pChart != null)
			pChart.destroy();
		pChart = new Chart(pctx).Doughnut(pData, {animateScale: true});
	}
}
function bind_submit_problem(pid) {
	$("#submit-modal").openModal();
	$("#submit-p").parent().fadeIn();
	$("#p-close").hide();
	$("#p-code").val("");
	$("#submit-p").unbind("click").click(function() {
		$("#submit-p").parent().fadeOut();
		$("#submit-wait").addClass("active");
		$.ajax({
			url: "?submit",
			type: "POST",
			data: {
				pid: pid,
				language: $("#p-language").val(),
				code: $("#p-code").val()
			},
			success: function(d) {
				if (d.success) {
					$("#p-code").val(d.detail);
					$("#p-close").show();
				} else {
					setTimeout(function() {
						$("#submit-p").parent().fadeIn();
						makeToast(d.error);
					}, 500);
				}
				setTimeout(function() {
					$("#submit-wait").removeClass("active");
				}, 500);
			},
			error: function(req, msg) {
				console.error(msg);
			},
			dataType: "json"
		});
	});
}
function showProblem(hash) {
	if (hash.length > 1) {
		var pid = hash[1];
		$.ajax({
			url: "?problem",
			type: "GET",
			data: {id: pid},
			success: function(d) {
				if (d.success) {
					$("#Problem").show();

					$("#pid").html(pid);
					$("#pname").html(d.name);
					$("#ptime").html(d.time);
					$("#pmemory").html(d.memory);
					$("#p-description").html(marked(d.description));
					if (admin) $("#p-description").prev().text(d.description);
					$("#p-input").html(marked(d.input));
					if (admin) $("#p-input").prev().text(d.input);
					$("#p-output").html(marked(d.output));
					if (admin) $("#p-output").prev().text(d.output);
					$("#p-sinput pre").html(d.sin);
					$("#p-soutput pre").html(d.sout);
					$("#p-hint").html(marked(d.hint));
					if (admin) $("#p-hint").prev().text(d.hint);
					if (admin) $("#p-tags").html(d.tags);

					displayMath("#Problem");
					update_problem_statistic(d.res);

					$("#my-submit").attr("href", "#records/p/" + $("#pid").html() + "/u/" + $("#uname").html());
					$("#problem-submit").attr("href", "#records/p/" + $("#pid").html());
					$("#p-submit").unbind("click").click(function() {
						bind_submit_problem(pid);
					});
				} else
					makeToast(d.error);

				hideLoad();
			},
			error: function(req, msg) {
				console.error(msg);
			},
			dataType: "json"
		});
	}
}

function update_contest_problems(ps) {
	var list = "";
	for (var i = 0; i < ps.length; i++) {
		var item = $("#cpim").html();
		item = item.replace(/%pid%/g, ps[i].id);
		item = item.replace(/%pname%/g, ps[i].name);
		list += item;
	}
	$("#cplist").html(list);
}
function showContest(hash) {
	if (hash.length > 1) {
		var cid = hash[1];
		$.ajax({
			url: "?contest",
			type: "GET",
			data: {id: cid},
			success: function(d) {
				if (d.success) {
					$("#Contest").show();
					var data = d.data;
					$("#cid").html(data.id);
					$("#cname").html(data.name);
					$("#ctime").html(data.time);
					$("#cstime").html(data.start_time);
					update_contest_problems(data.ps);

					if (data.cd == 0){
						$("#ccd").hide();
						$("#cres").show();

						if (data.wait)
							$("#cres .result").html(data.wait);
						else
							$("#cres .result").html(marked(data.result));
					} else {
						$("#ccd").show();
						$("#cres").hide();
						$("#cdt").html(data.cd_text);
						if (data.cd == 1)
							$("#ccd .cd").countdown(data.stime, function(event) {
								$(this).html(
									event.strftime($("#cdm").html())
								);
							}).on('finish.countdown', function() {
								changeUI();
							});
						else
							$("#ccd .cd").countdown(data.etime, function(event) {
								$(this).html(
									event.strftime($("#cdm").html())
								);
							}).on('finish.countdown', function() {
								changeUI();
							});	
					}
				} else
					makeToast(d.error);

				hideLoad();
			},
			error: function(req, msg) {
				console.error(msg);
			},
			dataType: "json"
		});
	}
}

function showSetting() {
	$("#Modify").show();
	hideLoad();
}
$("#modi-info").click(function() {
	$(this).attr("disabled", true);
	$(this).next().addClass("active");
	$.ajax({
		url: "?modify",
		type: "GET",
		data: {
			opwd: $("#old-password").val(),
			npwd: $("#new-password").val(),
			name: $("#m-uname").val(),
			mail: $("#m-mail").val()
		},
		success: function(d) {
			if (d.success) {
				if (d.logout) {
					$.cookie("DOJSS", null);
					location.reload();
				}
			} else
				makeToast(d.error);

			setTimeout(function() {
				$("#modi-info").attr("disabled", false);
				$("#modi-info").next().removeClass("active");
			}, 500);
		},
		error: function(req, msg) {
			console.error(msg);
		},
		dataType: "json"
	});
});
$("#modi-sign").click(function() {
	$(this).attr("disabled", true);
	$(this).next().addClass("active");
	$.ajax({
		url: "?modify",
		type: "GET",
		data: {
			sign: $("#m-sign").val()
		},
		success: function(d) {
			if (d.success) {
				makeToast(d.tip);
			} else
				makeToast(d.error);

			setTimeout(function() {
				$("#modi-sign").attr("disabled", false);
				$("#modi-sign").next().removeClass("active");
			}, 500);
		},
		error: function(req, msg) {
			console.error(msg);
		},
		dataType: "json"
	});
});