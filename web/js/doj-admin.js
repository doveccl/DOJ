var admin = true;

$("#apm-open").click(function() { $("#apm").openModal(); });
$("#acm-open").click(function() { $("#acm").openModal(); });
$("#etm-open").click(function() {
	var ett_init = [];
	$("#tlist h6").each(function() {
		var group = [];
		$(this).next().children().each(function() {
			group.push($(this).html());
		});
		ett_init.push($(this).children("b").html() + ":" + group.join("|"));
	});
	$("#ett").html(ett_init.join("\n"));
	$("#etm").openModal();
});

$("#add-problem").click(function() {
	$(this).attr("disabled", true);
	$("#apw").addClass("active");
	$.ajax({
		url: "?addProblem",
		type: "POST",
		data: {
			name: $("#apn").val(),
			time: $("#aptime").val(),
			memory: $("#apmemory").val(),
			tags: $("#apt").val(),
			description: $("#apd").val(),
			input: $("#api").val(),
			output: $("#apo").val(),
			hint: $("#aph").val()
		},
		success: function(d) {
			if (d.success) {
				var hash = "#problem/" + d.pid;
				makeToast(d.msg);
				$("#apm").closeModal();
				location.hash = hash;
			} else
				makeToast(d.error);

			setTimeout(function() {
				$("#add-problem").attr("disabled", false);
				$("#apw").removeClass("active");
			}, 500);
		},
		error: function(req, msg) {
			console.error(msg);
		},
		dataType: "json"
	});
});

$("#add-contest").click(function() {
	$(this).attr("disabled", true);
	$("#acw").addClass("active");
	var stime = $("#acdy").val() + "-";
	stime += fill($("#acdm").val(), 2) + "-";
	stime += fill($("#acdd").val(), 2) + " ";
	stime += fill($("#acth").val(), 2) + ":";
	stime += fill($("#actm").val(), 2) + ":";
	stime += fill($("#acts").val(), 2);
	$.ajax({
		url: "?addContest",
		type: "GET",
		data: {
			name: $("#acn").val(),
			time: $("#act").val(),
			stime: stime
		},
		success: function(d) {
			if (d.success) {
				var hash = "#contest/" + d.cid;
				makeToast(d.msg);
				$("#acm").closeModal();
				location.hash = hash;
			} else
				makeToast(d.error);

			setTimeout(function() {
				$("#add-contest").attr("disabled", false);
				$("#acw").removeClass("active");
			}, 500);
		},
		error: function(req, msg) {
			console.error(msg);
		},
		dataType: "json"
	});
});

$("#edit-tags").click(function() {
	$(this).attr("disabled", true);
	$("#etw").addClass("active");
	$.ajax({
		url: "?editTags",
		type: "POST",
		data: {tags: $("#ett").val()},
		success: function(d) {
			if (d.success) {
				update_problem_tags();
				makeToast(d.msg);
				$("#etm").closeModal();
			} else
				makeToast(d.error);

			setTimeout(function() {
				$("#edit-tags").attr("disabled", false);
				$("#etw").removeClass("active");
			}, 500);
		},
		error: function(req, msg) {
			console.error(msg);
		},
		dataType: "json"
	});
});

function init_problems_switch() {
	$("#plist .collection-item:not(.init)").each(function() {
		$(this).addClass("init");
		if ($(this).hasClass("hide"))
			$(this).addClass("show");
		else
			$(this).children("input").attr("checked", true);

		$(this).children("label").unbind("click").click(function() {
			$(this).attr("disabled", true);
			var checkbox = $(this).prev();
			var pid = $(this).attr("for").replace("P", "");
			var item = $(this).parent();
			var ishide = item.hasClass("hide");

			$.ajax({
				url: "?setProblem",
				type: "GET",
				data: {
					id: pid,
					switch: ishide ? 0 : 1
				},
				success: function(d) {
					if (d.success) {
						makeToast(d.msg);
						checkbox.attr("checked", ishide);
						if (ishide)
							item.removeClass("hide");
						else {
							item.addClass("hide");
							item.addClass("show");
						}
					} else {
						makeToast(d.error);
						checkbox.attr("checked", !ishide);
					}
					checkbox.next().attr("disabled", false);
				},
				error: function(req, msg) {
					console.error(msg);
				},
				dataType: "json"
			});
		});
	});
}

function init_contests_switch() {
	$("#clist .collection-item:not(.init)").each(function() {
		$(this).addClass("init");
		if ($(this).hasClass("hide"))
			$(this).addClass("show");
		else
			$(this).children("input").attr("checked", true);

		$(this).children("label").unbind("click").click(function() {
			$(this).attr("disabled", true);
			var checkbox = $(this).prev();
			var cid = $(this).attr("for").replace("C", "");
			var item = $(this).parent();
			var ishide = item.hasClass("hide");

			$.ajax({
				url: "?setContest",
				type: "GET",
				data: {
					id: cid,
					switch: ishide ? 0 : 1
				},
				success: function(d) {
					if (d.success) {
						makeToast(d.msg);
						checkbox.attr("checked", ishide);
						if (ishide)
							item.removeClass("hide");
						else {
							item.addClass("hide");
							item.addClass("show");
						}
					} else {
						makeToast(d.error);
						checkbox.attr("checked", !ishide);
					}
					checkbox.next().attr("disabled", false);
				},
				error: function(req, msg) {
					console.error(msg);
				},
				dataType: "json"
			});
		});
	});
}

$("#edit-base").click(function() {
	var pid = $("#pid").html();
	$(this).attr("disabled", true);
	$("#mpbw").addClass("active");
	$.ajax({
		url: "?setProblem",
		type: "GET",
		data: {
			id: pid,
			name: $("#mpn").val(),
			time: $("#mptime").val(),
			memory: $("#mpmemory").val(),
			tags: $("#mpt").val()
		},
		success: function(d) {
			if (d.success) {
				makeToast(d.msg);
				$("#mpbm").closeModal();
				$("#pname").html($("#mpn").val());
				$("#ptime").html($("#mptime").val());
				$("#pmemory").html($("#mpmemory").val());
				changeUI();
			} else
				makeToast(d.error);

			setTimeout(function() {
				$("#edit-base").attr("disabled", false);
				$("#mpbw").removeClass("active");
			}, 500);
		},
		error: function(req, msg) {
			console.error(msg);
		},
		dataType: "json"
	});
});
$("#edit-problem").click(function() {
	var data = { id: $("#pid").html() };
	var key = $("#mpkey").html();
	data[key] = $("#mpx").val();
	$(this).attr("disabled", true);
	$("#mpw").addClass("active");
	$.ajax({
		url: "?setProblem",
		type: "POST",
		data: data,
		success: function(d) {
			if (d.success) {
				makeToast(d.msg);
				$("#p-" + key).prev().text($("#mpx").val());
				$("#p-" + key).html(marked($("#mpx").val()));
				$("#mpm").closeModal();
				displayMath("#p-" + key);
			} else
				makeToast(d.error);

			setTimeout(function() {
				$("#edit-problem").attr("disabled", false);
				$("#mpw").removeClass("active");
			}, 500);
		},
		error: function(req, msg) {
			console.error(msg);
		},
		dataType: "json"
	});
});
$("#Problem .edit").click(function() {
	if ($(this).data("m") == "base") {
		$("#mpn").val($("#pname").html());
		$("#mpt").val($("#p-tags").html());
		$("#mptime").val($("#ptime").html());
		$("#mpmemory").val($("#pmemory").html());
		$("#mpbm").openModal();
	} else {
		$("#mpxl").html($(this).prev().html());
		$("#mpx").prop("value", $(this).next().text());
		$("#mpkey").html($(this).data("m"));
		$("#mpm").openModal();
	}
});

$("#edit-contest").click(function() {
	$(this).attr("disabled", true);
	$("#mcw").addClass("active");
	$.ajax({
		url: "?setContest",
		type: "GET",
		data: {
			id: $("#cid").html(),
			name: $("#mcn").val(),
			time: $("#mct").val(),
			stime: $("#mcst").val()
		},
		success: function(d) {
			if (d.success) {
				makeToast(d.msg);
				$("#mcm").closeModal();
				changeUI();
			} else
				makeToast(d.error);

			setTimeout(function() {
				$("#edit-contest").attr("disabled", false);
				$("#mcw").removeClass("active");
			}, 500);
		},
		error: function(req, msg) {
			console.error(msg);
		},
		dataType: "json"
	});
});
$("#Contest .edit").click(function() {
	$("#mcn").val($("#cname").html());
	$("#mct").val($("#ctime").html());
	$("#mcst").val($("#cstime").html());
	$("#mcm").openModal();
});

$("#cap").click(function() {
	$(this).attr("disabled", true);
	$.ajax({
		url: "?setContest",
		type: "GET",
		data: {
			id: $("#cid").html(),
			ap: $("#cap").prev().val()
		},
		success: function(d) {
			if (d.success) {
				makeToast(d.msg);
				changeUI();
			} else
				makeToast(d.error);

			$("#cap").attr("disabled", false);
		},
		error: function(req, msg) {
			console.error(msg);
		},
		dataType: "json"
	});
});
$("#cdp").click(function() {
	$(this).attr("disabled", true);
	$.ajax({
		url: "?setContest",
		type: "GET",
		data: {
			id: $("#cid").html(),
			dp: $("#cdp").prev().val()
		},
		success: function(d) {
			if (d.success) {
				makeToast(d.msg);
				changeUI();
			} else
				makeToast(d.error);

			$("#cdp").attr("disabled", false);
		},
		error: function(req, msg) {
			console.error(msg);
		},
		dataType: "json"
	});
});

$("#download-data").click(function() {
	location.href = "?downloadData&pid=" + $("#pid").html();
});

var uploader = new plupload.Uploader({
	browse_button: "upload-data",
	url: "?uploadData",
	multi_selection: false,
	filters: {
		mime_types : [{
			title: "Zipped Data",
			extensions: "zip"
		}],
		prevent_duplicates : false
	},
	flash_swf_url: "//cdn.bootcss.com/plupload/2.1.8/Moxie.swf"
});
uploader.init();
uploader.bind("FilesAdded", function(uploader, files) {
	$("#upload-start").show();
	$("#ulpn").html(files[0].name);
	$("#ulpp").html("0");
	$("#ulp div").css("width", "0");
	$("#upload-data").next().slideDown();
});
$("#upload-start").click(function() {
	$(this).hide();
	uploader.setOption({
		multipart_params: {
			pid: $("#pid").html()
		}
	});
	uploader.start();
});
uploader.bind("UploadProgress", function(uploader, file) {
	$("#ulpp").html(file.percent);
	$("#ulp div").css("width", file.percent + "%");
});
uploader.bind("FileUploaded", function(uploader, file, res) {
	makeToast(res.response);
	if (res.response.indexOf("!") > -1)
		changeUI();
	setTimeout(function() {
		$("#upload-data").next().slideUp("slow");
	}, 500);
});
