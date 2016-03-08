var state = ['Pending', 'Accept', 'Wrong Answer', 'Time Limit Exceeded', 'Memory Limit Exceeded', 'Runtime Error', 'Compiler Error', 'Output Limit Exceed', 'Others', 'Judging', 'Waiting'];
var abbrs  = ['Pending', 'AC', 'WA', 'TLE', 'MLE', 'RE', 'CE', 'OLE', 'Others', 'Judging', 'Waiting']
var language = ['C', 'C++', 'Pascal', 'Python', 'C++11'];
var lan = ['c', 'cpp', 'delphi', 'python', 'cpp']

function showLoad() { $("#loading").clearQueue();$("#loading").fadeIn(); }
function hideLoad() { setTimeout('$("#loading").slideUp("slow");', 600); }

function makeToast(msg) { Materialize.toast(msg, 5000); }

function makeSpace() {
	return '<i class="space"></i>'
}

function makeA(href, content) {
	return '<a href="' + href + '">' + content + '</a>';
}

function makeTag(s) {
	return '<div class="tag">' + s + '</div>';
}

function makeSup(content, clas) {
	return '<sup class="' + clas + '">' + content + "</sup>"
}

function makeOption(value, content) {
	return '<option value="' + value + '">' + content + '</option>';
}

function makeMath(content) {
	return '<span class="hpl_mathjax_inline">' + content + '</span>';
}

function displayMath(block) {
	$(block + " code").each(function() {
		if ($(this).html().match(/^\$(.*)\$$/)) {
			$(this).replaceWith(makeMath($(this).html()));
			MathJax.Hub.Queue(["Typeset", MathJax.Hub, $(this).get(0)]);
		}
	});
	MathJax.Hub.Queue(["Typeset", MathJax.Hub, block]);
}

function fill(num, length) {
	var numstr = num.toString();
	var len = length - numstr.length;
	for (var i = 0; i < len; i++)
		numstr = "0" + numstr;
	return numstr;
}