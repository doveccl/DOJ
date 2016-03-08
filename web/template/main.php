<!DOCTYPE html>
<?php
	$admin = ($me->admin > 0);
	$hash = md5($me->mail);
	$head = "//cn.gravatar.com/avatar/$hash?d=mm&s=256";
?>
<html>
	<head>
		<link type="text/css" rel="stylesheet" href="//cdn.bootcss.com/material-design-icons/2.1.3/iconfont/material-icons.css">
		<link type="text/css" rel="stylesheet" href="//cdn.bootcss.com/materialize/0.97.5/css/materialize.min.css">
		<link type="text/css" rel="stylesheet" href="//cdn.bootcss.com/highlight.js/9.1.0/styles/github.min.css">
		<link type="text/css" rel="stylesheet" href="css/doj.css">
		<?php if ($admin): ?>
		<link type="text/css" rel="stylesheet" href="css/doj-admin.css">
		<?php endif; ?>

		<meta http-equiv="Pragma" content="no-cache">
		<meta http-equiv="Cache-control" content="no-cache">
		<meta http-equiv="Cache" content="no-cache">
		<meta name="viewport" content="width=device-width, initial-scale=1.0">
		<meta name="description" content="DOJ, an online judge for OIers">
		<meta name="keywords" content="OI, Online Judge, algorithm">
		<meta name="author" content="Doveccl">
		<meta charset="UTF-8">

		<link rel='shortcut icon' type='image/x-icon' href='favicon.ico'>
		<title>DOJ</title>
	</head>

	<body class="grey lighten-3" onhashchange="changeUI();">

		<div class="navbar-fixed">
			<ul id="user-dropdown" class="dropdown-content">
				<?php if ($admin): ?>
				<li><a href="?admin" target="_blank"><?=str::$admin?></a></li>
				<?php endif; ?>
				<li><a href="#modify"><?=str::$setting?></a></li>
				<li class="divider"></li>
				<li><a class="logout"><?=str::$logout?></a></li>
			</ul>

			<nav class="blue">
				<div class="box" id="top">
					<b class="hide-on-large-only brand-mobi go-home">DOJ</b>
					<ul class="hide-on-med-and-down">
						<b class="left brand logo go-home">DOJ</b>
						<li class="mproblems"><a class="waves-effect" href="#problems"><?=str::$problems?></a></li>
						<li class="mrecords"><a class="waves-effect" href="#records"><?=str::$records?></a></li>
						<li class="mcontests"><a class="waves-effect" href="#contests"><?=str::$contests?></a></li>
						<li class="mrank"><a class="waves-effect" href="#rank"><?=str::$rank?></a></li>
					</ul>
					<ul class="right hide-on-med-and-down">
						<li><a class="dropdown-button" data-activates="user-dropdown">
							<span id="uname"><?=$me->name?></span>
							<img src="<?=$head?>" class="logo head right">
						</a></li>
					</ul>
					<ul id="slide-out" class="side-nav">
						<div class="row green banner">
							<div class="col s6">
								<img src="<?=$head?>" class="head head-mobi">
								<b class="head-name"><?=$me->name?></b>
								<span class="head-name grey-text text-lighten-2"><?=$me->sign?></span>
							</div>
							<div class="col s4 logout">
								<i class="material-icons right">exit_to_app</i>
							</div>
						</div>

						<li class="mproblems"><a class="waves-effect" href="#problems">
							<i class="material-icons left"><small>subject</small></i>
							<?=str::$problems?>
						</a></li>
						<li class="mrecords"><a class="waves-effect" href="#records">
							<i class="material-icons left"><small>event</small></i>
							<?=str::$records?>
						</a></li>
						<li class="mcontests"><a class="waves-effect" href="#contests">
							<i class="material-icons left"><small>games</small></i>
							<?=str::$contests?>
						</a></li>
						<li class="mrank"><a class="waves-effect" href="#rank">
							<i class="material-icons left"><small>assessment</small></i>
							<?=str::$rank?>
						</a></li>
						<li class="divider"></li>
						<li><a class="waves-effect" href="#modify">
							<i class="material-icons left"><small>settings</small></i>
							<?=str::$setting?>
						</a></li>
					</ul>
					<a data-activates="slide-out" class="button-collapse">
						<i class="mdi-navigation-menu"></i>
					</a>
				</div>
			</nav>
			<div class="progress" id="loading" style="display:none;">
				<div class="indeterminate"></div>
			</div>
		</div>

		<div class="main">

			<div class="content row" id="Home" style="display:none;">
				<div class="col s12 m4">
					<div class="card">
						<div class="card-content">
							<span class="card-title"><?=str::$personal_statistics?></span>
							<div class="row nomargin">
								<div class="col s9">
									<canvas id="myChart"></canvas>
								</div>
								<div class="col s3" id="myLegend">
									<ul class="doughnut-legend">
										<li><span style="background-color:#FF6666"></span>AC</li>
										<li><span style="background-color:#66CC66"></span>WA</li>
										<li><span style="background-color:#FFCC00"></span>TLE</li>
										<li><span style="background-color:#66CCFF"></span>MLE</li>
										<li><span style="background-color:#FF66FF"></span>RE</li>
										<li><span style="background-color:#999999"></span>CE</li>
									</ul>
								</div>
							</div>
						</div>
					</div>

					<div class="card">
						<div class="card-content">
							<span class="card-title"><?=str::$oj_statistics?></span>
							<div class="row nomargin">
								<div class="col s12">
									<canvas id="ojChart"></canvas>
								</div>
								<div class="col s12 line-legend">
									<div class="box">
										<span style="background-color:#97BBCD"></span>
										<font>Accept</font>
									</div>
									<div class="box">
										<span style="background-color:#DCDCDC"></span>
										<font>Submit</font>
									</div>
								</div>
							</div>
						</div>
					</div>
				</div>

				<div class="col s12 m8">
					<div class="card">
						<div class="card-content">
							<span class="card-title"><?=str::$announcement?></span>
							<div><?php
								echo file_get_contents("template/announcement.html");
							?></div>
						</div>
					</div>

					<div class="card">
						<div class="card-content">
							<span class="card-title"><?=str::$recent_problems?></span>
							<div class="hide" id="rpim">
								<a href="#problem/%pid%" class="collection-item">
									<b>P%pid%</b><i class="space"></i>
									<span>%pname%</span>
									<span class="badge hide-on-small-only">%tags%</span>
								</a>
							</div>
							<div class="collection" id="rp"></div>
						</div>
					</div>

					<div class="card">
						<div class="card-content">
							<span class="card-title"><?=str::$recent_contests?></span>
							<div class="hide" id="rcim">
								<a href="#contest/%cid%" class="collection-item">
									<b>#%cid%</b><i class="space"></i>
									<span>%cname%</span><sup>%state%</sup><i class="space"></i>
									<i class="space"></i><i>%time% min</i>
									<span class="badge hide-on-small-only">%stime%</span>
								</a>
							</div>
							<div class="collection" id="rc"></div>
						</div>
					</div>
				</div>
			</div>

			<div class="content row" id="Problems" style="display:none;">
				<div class="col s12 m7 offset-m1">
					<?php if ($admin): ?>
					<div class="card">
						<div class="card-content row center-align">
							<div class="col s6">
								<a class="waves-effect btn yellow darken-1" id="apm-open">
									<?=str::$add_problem?>
									<i class="material-icons right">add</i>
								</a>
							</div>
							<div class="col s6">
								<a class="waves-effect btn green lighten-2" id="etm-open">
									<?=str::$edit_tags?>
									<i class="material-icons right">mode_edit</i>
								</a>
							</div>
						</div>
					</div>
					<?php endif; ?>
					<div class="card">
						<div class="card-content" id="page_index">
							<b><?=str::$page_index?> :</b>&nbsp;&nbsp;
						</div>
					</div>
					<div class="card">
						<div class="card-content">
							<div class="hide" id="pim">
								<a href="#problem/%pid%" class="collection-item">
									<?php if ($admin): ?>
									<input type="checkbox" id="P%pid%" class="p-show">
									<label for="P%pid%"></label>
									<?php endif; ?>
									<b>P%pid%</b><i class="space"></i>
									<span>%pname%</span>
									<span class="badge hide-on-small-only">%tags%</span>
								</a>
							</div>
							<div class="hide" id="pmm">
								<a class="collection-item more">
									<?=str::$load_more?>
								</a>
							</div>
							<div  class="hide" id="pml">
								<div class="preloader-wrapper tiny active">
									<div class="spinner-layer spinner-blue-only">
										<div class="circle-clipper left">
											<div class="circle"></div>
										</div>
										<div class="gap-patch">
											<div class="circle"></div>
										</div>
										<div class="circle-clipper right">
											<div class="circle"></div>
										</div>
									</div>
								</div>
							</div>
							<div class="collection" id="plist"></div>
						</div>
					</div>
				</div>

				<div class="col s12 m3">
					<div class="card">
						<div class="card-content">
							<span class="card-title"><?=str::$tags?></span>
							<div class="hide" id="tim">
								<h6>
									<b>%group%</b>
									<i class="material-icons tiny right">
										add_circle_outline
									</i>
								</h6>
								<blockquote class="tags-set" style="display:none;">
									%set%
								</blockquote>
							</div>
							<div id="tlist"></div>
						</div>
					</div>
				</div>		
			</div>

			<div class="content row" id="Records" style="display:none;">
				<div class="col s12 l2 offset-l1">
					<div class="card">
						<div class="card-content row">
							<div class="input-field col s12 m6 l12">
								<input id="r-pid" type="text" class="validate">
								<label for="r-pid"><?=str::$problem_id?></label>
							</div>
							<div class="input-field col s12 m6 l12">
								<input id="r-uname" type="text" class="validate">
								<label for="r-uname"><?=str::$name?></label>
							</div>
							<div class="input-field col s12 m6 l12">
								<select id="r-language">
									<option value="-1" selected><?=str::$any?></option>
								</select>
							</div>
							<div class="input-field col s12 m6 l12">
								<select id="r-result">
									<option value="-1" selected><?=str::$any?></option>
								</select>
							</div>
							<div class="input-field col s12">
								<button class="waves-effect btn" id="r-filter"><?=str::$filter?></button>
							</div>							
						</div>
					</div>
				</div>

				<div class="col s12 l8">
					<div class="hide" id="rim">
						<li data-id="%rid%">
							<div class="collapsible-header">
								<span class="%res%">#%rid%</span>
								<sup class="%res%"> %res%</sup>
								<b class="space"></b>
								<span>%uname%</span>
								<b class="space"></b>
								<a href="#problem/%pid%">P%pid% %pname%</a>
								<b class="space"></b>
								<span>%language%</span>
								<b class="space"></b>
								<span>%utime% ms</span>
								<b class="space"></b>
								<span>%umemory% KB</span>
								<b class="space"></b><b class="space"></b>
								<span class="hide-on-small-only">%stime%</span>
							</div>
							<div class="collapsible-body"></div>
						</li>
					</div>
					<div  class="hide" id="rwm">
						<div class="center-align container">
							<div class="preloader-wrapper tiny active">
								<div class="spinner-layer spinner-blue-only">
									<div class="circle-clipper left">
										<div class="circle"></div>
									</div>
									<div class="gap-patch">
										<div class="circle"></div>
									</div>
									<div class="circle-clipper right">
										<div class="circle"></div>
									</div>
								</div>
							</div>
						</div>
					</div>
					<div  class="hide" id="rdim">
						<li class="collection-item">
							<span class="%res%">
								<span>#%kth%</span>
								<i class="space"></i>
								<span>%res%</span>
								<i class="space"></i>
							</span>
							<span>%utime% ms</span>
							<i class="space"></i>
							<span>%umemory% KB</span>
						</li>
					</div>
					<div class="hide" id="rdim-ce">
						<li class="collection-item">
							<pre class="ce-msg">%ce%</pre>
						</li>
					</div>
					<div class="hide" id="rmm">
						<li class="collection-item more">
							<?=str::$load_more?>
						</li>
					</div>
					<div class="hide" id="rdm">
						<div class="box ritem">
							<ul class="collection with-header">
								<li class="collection-header">
									<h5>Points</h5>
								</li>
								<blockquote>
									%detail%
								</blockquote>
							</ul>
							<ul class="collection with-header">
								<li class="collection-header">
									<h5>Code</h5>
								</li>
								<blockquote>
									<pre><code class="%lan%">%code%</code></pre>
								</blockquote>
							</ul>
						</div>
					</div>
					<ul class="collapsible popout" id="rlist" data-collapsible="accordion"></ul>
				</div>
			</div>

			<div class="content row" id="Contests" style="display:none;">
				<div class="col s12 m10 offset-m1 l8 offset-l2">
					<div class="card">
						<div class="card-content">
							<span class="card-title"><?=str::$contests?></span>
							<?php if ($admin): ?>
							<a class="waves-effect btn right yellow darken-1" id="acm-open">
								<?=str::$add_contest?>
								<i class="material-icons right">add</i>
							</a>
							<?php endif; ?>
							<div class="hide" id="cim">
								<a href="#contest/%cid%" class="collection-item">
									<?php if ($admin): ?>
									<input type="checkbox" id="C%cid%" class="c-show">
									<label for="C%cid%"></label>
									<?php endif; ?>
									<b>#%cid%</b><i class="space"></i>
									<span>%cname%</span><sup>%state%</sup><i class="space"></i>
									<i class="space"></i><i>%time% min</i>
									<span class="badge hide-on-small-only">%stime%</span>
								</a>
							</div>
							<div class="hide" id="cmm">
								<a class="collection-item more">
									<?=str::$load_more?>
								</a>
							</div>
							<div  class="hide" id="cml">
								<div class="preloader-wrapper tiny active">
									<div class="spinner-layer spinner-blue-only">
										<div class="circle-clipper left">
											<div class="circle"></div>
										</div>
										<div class="gap-patch">
											<div class="circle"></div>
										</div>
										<div class="circle-clipper right">
											<div class="circle"></div>
										</div>
									</div>
								</div>
							</div>
							<div class="collection" id="clist"></div>							
						</div>
					</div>
				</div>
			</div>

			<div class="content row" id="Rank" style="display:none;">
				<div class="col s12 l10 offset-l1">
					<div class="card">
						<div class="card-content row">
							<span class="card-title"><?=str::$rank?></span>
							<div class="hide" id="uim">
								<a class="collection-item">
									<span>No.%rank%</span>
									<i class="space"></i>
									<span>%name%</span>
									<span class="hide-on-small-only">
										<i class="space"></i>
										<i class="space"></i>
										<span>%sign%</span>
									</span>
									<span class="badge">
										<span>%solve% AC</span> /
										<span>%submit%</span>
										&nbsp;&nbsp;&nbsp;
										<span class="hide-on-small-only">(%radio%%)</span>
									</span>
								</a>
							</div>
							<div class="hide" id="umm">
								<a class="collection-item more">
									<?=str::$load_more?>
								</a>
							</div>
							<div  class="hide" id="uml">
								<div class="preloader-wrapper tiny active">
									<div class="spinner-layer spinner-blue-only">
										<div class="circle-clipper left">
											<div class="circle"></div>
										</div>
										<div class="gap-patch">
											<div class="circle"></div>
										</div>
										<div class="circle-clipper right">
											<div class="circle"></div>
										</div>
									</div>
								</div>
							</div>
							<div class="collection" id="ranklist"></div>
						</div>
					</div>
				</div>
			</div>

			<div class="content row" id="Problem" style="display:none;">
				<div class="col s12 m7 offset-m1">
					<div class="card">
						<div class="card-content">
							<span class="card-title"><?=str::$description?></span>
							<?php if ($admin): ?>
							<i class="material-icons edit" data-m="description">edit</i>
							<pre class="hide"></pre>
							<?php endif; ?>
							<div id="p-description"></div>
						</div>
					</div>
					<div class="card">
						<div class="card-content">
							<span class="card-title"><?=str::$input_format?></span>
							<?php if ($admin): ?>
							<i class="material-icons edit" data-m="input">edit</i>
							<pre class="hide"></pre>
							<?php endif; ?>
							<div id="p-input"></div>
						</div>
					</div>
					<div class="card">
						<div class="card-content">
							<span class="card-title"><?=str::$output_format?></span>
							<?php if ($admin): ?>
							<i class="material-icons edit" data-m="output">edit</i>
							<pre class="hide"></pre>
							<?php endif; ?>
							<div id="p-output"></div>
						</div>
					</div>
					<div class="card">
						<div class="card-content">
							<span class="card-title"><?=str::$sample_input?></span>
							<div id="p-sinput">
								<blockquote><pre></pre></blockquote>
							</div>
						</div>
					</div>
					<div class="card">
						<div class="card-content">
							<span class="card-title"><?=str::$sample_output?></span>
							<div id="p-soutput">
								<blockquote><pre></pre></blockquote>
							</div>
						</div>
					</div>
					<div class="card">
						<div class="card-content">
							<span class="card-title"><?=str::$range_hint?></span>
							<?php if ($admin): ?>
							<i class="material-icons edit" data-m="hint">edit</i>
							<pre class="hide"></pre>
							<?php endif; ?>
							<div id="p-hint"></div>
						</div>
					</div>
				</div>

				<div class="col s12 m3">
					<div class="card">
						<div class="card-content">
							<span class="card-title">
								P<span id="pid"></span>
							</span>
							<span id="pname"></span>
							<?php if ($admin): ?>
							<i class="material-icons edit" data-m="base">edit</i>
							<?php endif; ?>
							<div class="row">
								<div class="col s12">
									<?=str::$time?> :
									<span id="ptime"></span> ms
								</div>
								<div class="col s12">
									<?=str::$memory?> :
									<span id="pmemory"></span> KB
								</div>
							</div>
							<div class="row nomargin">
								<div class="col s12">
									<canvas id="pChart"></canvas>
									<h5 class="center-align">
										<small>No Submission</small>
									</h5>
								</div>
								<div class="col s12" id="pLegend">
									<ul class="doughnut-legend">
										<li><span style="background-color:#FF6666"></span>AC</li>
										<li><span style="background-color:#66CC66"></span>WA</li>
										<li><span style="background-color:#FFCC00"></span>TLE</li>
										<li><span style="background-color:#66CCFF"></span>MLE</li>
										<li><span style="background-color:#FF66FF"></span>RE</li>
										<li><span style="background-color:#999999"></span>CE</li>
									</ul>
								</div>
							</div>
						</div>
					</div>

					<?php if ($admin): ?>
					<div class="card">
						<div class="card-content row">
							<div class="row"><?=str::$data_explain?></div>
							<a class="btn yellow darken-1 col s12" id="download-data">
								<i class="material-icons right">file_download</i>
								<?=str::$download_data?>
							</a>
							<div class="row"></div>
							<span><?=str::$size_limit?>: <?=ini_get("upload_max_filesize")?></span>
							<a class="btn green lighten-2 col s12" id="upload-data">
								<i class="material-icons right">file_upload</i>
								<?=str::$upload_data?>
							</a>
							<div style="display:none;">
								<div class="progress" id="ulp">
									<div class="determinate"></div>
								</div>
								<div class="center-align">
									<span id="ulpn"></span>:
									<span id="ulpp"></span>%
									<a class="btn-flat orange-text" id="upload-start">
										<?=str::$start_upload?>
									</a>
								</div>
							</div>
						</div>
					</div>
					<?php endif; ?>
				</div>

				<div class="fixed-action-btn" id="p-fab">
					<a class="btn-floating btn-large blue">
						<i class="large material-icons">add</i>
					</a>
					<ul>
						<li><a class="btn-floating red tooltipped"
							data-position="left" data-delay="50" 
							data-tooltip="<?=str::$submit?>" id="p-submit">
							<i class="material-icons">mode_edit</i>
						</a></li>
						<li><a class="btn-floating yellow darken-1 tooltipped"
							data-position="left" data-delay="50" 
							data-tooltip="<?=str::$my_submit?>" id="my-submit">
							<i class="material-icons">event</i>
						</a></li>
						<li><a class="btn-floating green tooltipped"
							data-position="left" data-delay="50" 
							data-tooltip="<?=str::$problem_submit?>" id="problem-submit">
							<i class="material-icons">date_range</i>
						</a></li>
					</ul>
				</div>
			</div>

			<div class="content row" id="Contest" style="display:none;">
				<dir class="col s12 m8 offset-m2">
					<div class="card">
						<div class="card-content">
							<span class="hide" id="cid"></span>
							<span class="hide" id="ctime"></span>
							<span class="hide" id="cstime"></span>
							<span class="card-title" id="cname"></span>
							<?php if ($admin): ?>
							<i class="material-icons edit">edit</i>
							<?php endif; ?>
							<div class="hide" id="cpim">
								<a href="#problem/%pid%" class="collection-item">
									<b>P%pid%</b><i class="space"></i>
									<span>%pname%</span>
								</a>
							</div>
							<div class="collection" id="cplist"></div>
							<?php if ($admin): ?>
							<div class="row">
								<div class="col s12 l6">
									<div class="row">
										<input type="text" class="validate col s8" placeholder="<?=str::$problem_id?>">
										<button class="btn-floating waves-effect waves-light green" id="cap"><i class="material-icons">add</i></button>
									</div>
								</div>
								<div class="col s12 l6">
									<div class="row">
										<input type="text" class="validate col s8" placeholder="<?=str::$problem_id?>">
										<button class="btn-floating waves-effect waves-light red" id="cdp"><i class="material-icons">delete</i></button>
									</div>
								</div>
							</div>
							<?php endif; ?>
						</div>
					</div>
					<div class="card" id="ccd">
						<div class="card-content">
							<span class="card-title" id="cdt"></span>
							<div id="cdm" class="hide">
								<div class="row center-align grey-text">
									<div class="col s3">
										<h3>%D</h3><?=str::$day?>
									</div>
									<div class="col s3">
										<h3>%H</h3><?=str::$hour?>
									</div>
									<div class="col s3">
										<h3>%M</h3><?=str::$minute?>
									</div>
									<div class="col s3">
										<h3>%S</h3><?=str::$second?>
									</div>
								</div>
							</div>
							<div class="cd"></div>
						</div>
					</div>
					<div class="card" id="cres">
						<div class="card-content">
							<span class="card-title"><?=str::$contest_rank?></span>
							<div class="result"></div>
						</div>
					</div>
				</dir>
			</div>

			<div class="content row" id="Modify" style="display:none;">
				<div class="col s12 m8 offset-m2">
					<ul class="tabs row">
						<li class="tab col s3"><a class="active" href="#modi-1">
							<?=str::$modify_info?>
						</a></li>
						<li class="tab col s3"><a href="#modi-2">
							<?=str::$modify_sign?>
						</a></li>
					</ul>
					<div class="card" id="modi-1">
						<div class="card-content">
							<div class="input-field">
								<input id="old-password" type="password" class="validate">
								<label for="old-password"><?=str::$old_password?></label>
							</div>
							<div class="input-field">
								<input id="new-password" type="password" class="validate">
								<label for="new-password"><?=str::$new_password?></label>
							</div>
							<div class="input-field">
								<input id="m-uname" type="text" class="validate" value="<?=$me->name?>">
								<label for="m-uname"><?=str::$name?></label>
							</div>
							<div class="input-field">
								<input id="m-mail" type="text" class="validate" value="<?=$me->mail?>">
								<label for="m-mail"><?=str::$mail?></label>
							</div>
							<div class="input-field">
								<button class="waves-effect waves-light btn" id="modi-info">
									<?=str::$modify;?>
								</button>
								<div class="preloader-wrapper right small">
									<div class="spinner-layer spinner-blue-only">
										<div class="circle-clipper left">
											<div class="circle"></div>
										</div>
										<div class="gap-patch">
											<div class="circle"></div>
										</div>
										<div class="circle-clipper right">
											<div class="circle"></div>
										</div>
									</div>
								</div>
							</div>
						</div>
					</div>
					<div class="card" id="modi-2">
						<div class="card-content">
							<div class="input-field">
								<textarea id="m-sign" class="materialize-textarea"><?=$me->sign?></textarea>
								<label for="m-sign"><?=str::$sign?></label>
							</div>
							<div class="input-field">
								<button class="waves-effect waves-light btn" id="modi-sign">
									<?=str::$modify;?>
								</button>
								<div class="preloader-wrapper right small">
									<div class="spinner-layer spinner-blue-only">
										<div class="circle-clipper left">
											<div class="circle"></div>
										</div>
										<div class="gap-patch">
											<div class="circle"></div>
										</div>
										<div class="circle-clipper right">
											<div class="circle"></div>
										</div>
									</div>
								</div>
							</div>
						</div>
					</div>
				</div>
			</div>

		</div>

		<div class="footer-copyright grey lighten-3">
			<p class="center-align grey-text text-darken-5">
				<span>Copyright © 2016 DOJ All Rights Reserved</span>
				<a></a>
			</p>
		</div>

		<div id="submit-modal" class="modal modal-fixed-footer">
			<div class="modal-content">
				<h4>
					<?=str::$submit?><i class="space"></i>
					<div class="preloader-wrapper tiny" id="submit-wait">
						<div class="spinner-layer spinner-blue-only">
							<div class="circle-clipper left">
								<div class="circle"></div>
							</div>
							<div class="gap-patch">
								<div class="circle"></div>
							</div>
							<div class="circle-clipper right">
								<div class="circle"></div>
							</div>
						</div>
					</div>
				</h4>
				<textarea id="p-code" spellcheck="false"></textarea>
			</div>
			<div class="modal-footer">
				<div>
					<select class="browser-default" id="p-language"></select>
					<button class="modal-action waves-effect btn" id="submit-p">
						<i class="material-icons right hide-on-small-only">send</i>
						<?=str::$submit?>
					</button>
					<a class="modal-action modal-close btn-flat waves-effect">
						<b class="orange-text"><?=str::$cancel?></b>
					</a>
				</div>
				<a class="modal-action modal-close btn-flat waves-effect" id="p-close">
					<b class="orange-text"><?=str::$close?></b>
				</a>
			</div>
		</div>
		
		<div class="hide" id="gtm">
			<div class="fixed-action-btn" id="go-top">
				<a class="btn-floating btn-large green">
					<i class="large material-icons">arrow_upward</i>
				</a>
			</div>
		</div>

		<?php if ($admin): ?>
		<div id="apm" class="modal modal-fixed-footer">
			<div class="modal-content">
				<h4>
					<?=str::$add_problem?><i class="space"></i>
					<div class="preloader-wrapper tiny" id="apw">
						<div class="spinner-layer spinner-blue-only">
							<div class="circle-clipper left">
								<div class="circle"></div>
							</div>
							<div class="gap-patch">
								<div class="circle"></div>
							</div>
							<div class="circle-clipper right">
								<div class="circle"></div>
							</div>
						</div>
					</div>
				</h4>
				<div class="row">
					<div class="input-field col s6">
						<input type="text" placeholder="ms" id="aptime" class="validate">
						<label for="aptime"><?=str::$time?></label>
					</div>
					<div class="input-field col s6">
						<input type="text" placeholder="KB" id="apmemory" class="validate">
						<label for="apmemory"><?=str::$memory?></label>
					</div>
				</div>
				<div class="input-field row">
					<input type="text" id="apn" class="validate">
					<label for="apn"><?=str::$problem_name?></label>
				</div>
				<div class="input-field row">
					<input type="text" id="apt" class="validate">
					<label for="apt"><?=str::$problem_tags?></label>
				</div>
				<div class="input-field row">
					<textarea id="apd" class="materialize-textarea" spellcheck="false"></textarea>
					<label for="apd"><?=str::$description?></label>
				</div>
				<div class="input-field row">
					<textarea id="api" class="materialize-textarea" spellcheck="false"></textarea>
					<label for="api"><?=str::$input_format?></label>
				</div>
				<div class="input-field row">
					<textarea id="apo" class="materialize-textarea" spellcheck="false"></textarea>
					<label for="apo"><?=str::$output_format?></label>
				</div>
				<div class="input-field row">
					<textarea id="aph" class="materialize-textarea" spellcheck="false"></textarea>
					<label for="aph"><?=str::$range_hint?></label>
				</div>
			</div>
			<div class="modal-footer">
				<button class="modal-action waves-effect btn" id="add-problem">
					<i class="material-icons right hide-on-small-only">send</i>
					<?=str::$add?>
				</button>
				<a class="modal-action modal-close btn-flat waves-effect">
					<b class="orange-text"><?=str::$cancel?></b>
				</a>
			</div>
		</div>
		<div id="etm" class="modal modal-fixed-footer">
			<div class="modal-content">
				<h4>
					<?=str::$edit_tags?><i class="space"></i>
					<div class="preloader-wrapper tiny" id="etw">
						<div class="spinner-layer spinner-blue-only">
							<div class="circle-clipper left">
								<div class="circle"></div>
							</div>
							<div class="gap-patch">
								<div class="circle"></div>
							</div>
							<div class="circle-clipper right">
								<div class="circle"></div>
							</div>
						</div>
					</div>
				</h4>
				<textarea class="tooltipped" data-position="top" data-delay="50"
					data-tooltip="<?=str::$tags_explain?>" id="ett" spellcheck="false">
				</textarea>
			</div>
			<div class="modal-footer">
				<button class="modal-action waves-effect btn" id="edit-tags">
					<i class="material-icons right hide-on-small-only">send</i>
					<?=str::$modify?>
				</button>
				<a class="modal-action modal-close btn-flat waves-effect">
					<b class="orange-text"><?=str::$cancel?></b>
				</a>
			</div>
		</div>

		<div id="mpbm" class="modal">
			<div class="modal-content">
				<h4>
					<?=str::$edit_problem?><i class="space"></i>
					<div class="preloader-wrapper tiny" id="mpbw">
						<div class="spinner-layer spinner-blue-only">
							<div class="circle-clipper left">
								<div class="circle"></div>
							</div>
							<div class="gap-patch">
								<div class="circle"></div>
							</div>
							<div class="circle-clipper right">
								<div class="circle"></div>
							</div>
						</div>
					</div>
				</h4>
				<div class="row">
					<div class="input-field col s6">
						<input type="text" placeholder="ms" id="mptime" class="validate">
						<label for="mptime"><?=str::$time?></label>
					</div>
					<div class="input-field col s6">
						<input type="text" placeholder="KB" id="mpmemory" class="validate">
						<label for="mpmemory"><?=str::$memory?></label>
					</div>
				</div>
				<div class="input-field row">
					<input type="text" placeholder="" id="mpn" class="validate">
					<label for="mpn"><?=str::$problem_name?></label>
				</div>
				<div class="input-field row">
					<div class="hide" id="p-tags"></div>
					<input type="text" placeholder="" id="mpt" class="validate">
					<label for="mpt"><?=str::$tag?></label>
				</div>
			</div>
			<div class="modal-footer">
				<button class="modal-action waves-effect btn" id="edit-base">
					<i class="material-icons right hide-on-small-only">send</i>
					<?=str::$modify?>
				</button>
				<a class="modal-action modal-close btn-flat waves-effect">
					<b class="orange-text"><?=str::$cancel?></b>
				</a>
			</div>
		</div>
		<div id="mpm" class="modal modal-fixed-footer">
			<div class="modal-content">
				<h4>
					<?=str::$modify?>
					<small class="grey-text" id="mpxl"></small>
					<i class="space"></i>
					<div class="preloader-wrapper tiny" id="mpw">
						<div class="spinner-layer spinner-blue-only">
							<div class="circle-clipper left">
								<div class="circle"></div>
							</div>
							<div class="gap-patch">
								<div class="circle"></div>
							</div>
							<div class="circle-clipper right">
								<div class="circle"></div>
							</div>
						</div>
					</div>
				</h4>
				<span class="hide" id="mpkey"></span>
				<textarea id="mpx" spellcheck="false"></textarea>
			</div>
			<div class="modal-footer">
				<button class="modal-action waves-effect btn" id="edit-problem">
					<i class="material-icons right hide-on-small-only">send</i>
					<?=str::$modify?>
				</button>
				<a class="modal-action modal-close btn-flat waves-effect">
					<b class="orange-text"><?=str::$cancel?></b>
				</a>
			</div>
		</div>

		<div id="acm" class="modal modal-fixed-footer">
			<div class="modal-content">
				<h4>
					<?=str::$add_contest?><i class="space"></i>
					<div class="preloader-wrapper tiny" id="acw">
						<div class="spinner-layer spinner-blue-only">
							<div class="circle-clipper left">
								<div class="circle"></div>
							</div>
							<div class="gap-patch">
								<div class="circle"></div>
							</div>
							<div class="circle-clipper right">
								<div class="circle"></div>
							</div>
						</div>
					</div>
				</h4>
				<div class="row">
					<div class="input-field col s8">
						<input type="text" id="acn" class="validate">
						<label for="acn"><?=str::$contest_name?></label>
					</div>
					<div class="input-field col s4">
						<input type="text" id="act" placeholder="min" class="validate">
						<label for="act"><?=str::$contest_duration?></label>
					</div>
				</div>
				<div class="row">
					<div class="col s12"><?=str::$start_date?></div>
					<div class="input-field col s4">
						<input type="text" id="acdy" class="validate" value="<?=date('Y')?>">
						<label for="acdy"><?=str::$year?></label>
					</div>
					<div class="input-field col s4">
						<input type="text" id="acdm" class="validate" value="<?=date('m')?>">
						<label for="acdm"><?=str::$month?></label>
					</div>
					<div class="input-field col s4">
						<input type="text" id="acdd" class="validate" value="<?=date('d')?>">
						<label for="acdd"><?=str::$day?></label>
					</div>
				</div>
				<div class="row">
					<div class="col s12"><?=str::$start_time?></div>
					<div class="input-field col s4">
						<input type="text" id="acth" class="validate">
						<label for="acth"><?=str::$hour?></label>
					</div>
					<div class="input-field col s4">
						<input type="text" id="actm" class="validate">
						<label for="actm"><?=str::$minute?></label>
					</div>
					<div class="input-field col s4">
						<input type="text" id="acts" class="validate" value="00">
						<label for="acts"><?=str::$second?></label>
					</div>
				</div>
			</div>
			<div class="modal-footer">
				<button class="modal-action waves-effect btn" id="add-contest">
					<i class="material-icons right hide-on-small-only">send</i>
					<?=str::$add?>
				</button>
				<a class="modal-action modal-close btn-flat waves-effect">
					<b class="orange-text"><?=str::$cancel?></b>
				</a>
			</div>
		</div>
		<div id="mcm" class="modal">
			<div class="modal-content">
				<h4>
					<?=str::$edit_contest?><i class="space"></i>
					<div class="preloader-wrapper tiny" id="mcw">
						<div class="spinner-layer spinner-blue-only">
							<div class="circle-clipper left">
								<div class="circle"></div>
							</div>
							<div class="gap-patch">
								<div class="circle"></div>
							</div>
							<div class="circle-clipper right">
								<div class="circle"></div>
							</div>
						</div>
					</div>
				</h4>
				<div class="row">
					<div class="input-field col s8">
						<input type="text" id="mcn" placeholder="" class="validate">
						<label for="mcn"><?=str::$contest_name?></label>
					</div>
					<div class="input-field col s4">
						<input type="text" id="mct" placeholder="min" class="validate">
						<label for="mct"><?=str::$contest_duration?></label>
					</div>
				</div>
				<div class="input-field row">
					<input type="text" id="mcst" placeholder="" class="validate">
					<label for="mcst"><?=str::$start_datetime?></label>
				</div>
			</div>
			<div class="modal-footer">
				<button class="modal-action waves-effect btn" id="edit-contest">
					<i class="material-icons right hide-on-small-only">send</i>
					<?=str::$modify?>
				</button>
				<a class="modal-action modal-close btn-flat waves-effect">
					<b class="orange-text"><?=str::$cancel?></b>
				</a>
			</div>
		</div>
		<?php endif; ?>

		<script src="//cdn.bootcss.com/jquery/1.11.3/jquery.min.js"></script>
		<script src="//cdn.bootcss.com/jquery-cookie/1.4.1/jquery.cookie.min.js"></script>
		<script src="//cdn.bootcss.com/materialize/0.97.5/js/materialize.min.js"></script>
		<script src="//cdn.bootcss.com/Chart.js/1.0.2/Chart.min.js"></script>
		<script src="//cdn.bootcss.com/marked/0.3.5/marked.min.js"></script>
		<script src="//cdn.bootcss.com/highlight.js/9.1.0/highlight.min.js"></script>
		<script src="//cdn.bootcss.com/mathjax/2.6.0/MathJax.js?config=TeX-AMS-MML_HTMLorMML"></script>
		<script src="//cdn.bootcss.com/scrollup/2.4.0/jquery.scrollUp.min.js"></script>
		<script src="//cdn.bootcss.com/jquery.countdown/2.1.0/jquery.countdown.min.js"></script>
		<script src="js/doj-common.js"></script>
		<?php if ($admin): ?>
		<script src="//cdn.bootcss.com/plupload/2.1.8/plupload.full.min.js"></script>
		<script src="js/doj-admin.js"></script>
		<?php endif; ?>
		<script src="js/doj.js"></script>
	</body>
</html>
<?php die(); ?>