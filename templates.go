package bulktracker

import (
	"html/template"
)

const PageHeader = `
<!DOCTYPE html>
<html lang="en">
  <head>
    <meta charset="utf-8">
    <meta http-equiv="X-UA-Compatible" content="IE=edge">
    <meta name="viewport" content="width=device-width, initial-scale=1">

    <title>BulkTracker</title>

    <link href="/static/bootstrap.min.css" rel="stylesheet">
    <link href="/static/dataTables.bootstrap.css" rel="stylesheet">
    <!--<style type="text/css">
      .btn-primary {
        color: #fff;
	background-color: #f62711;
	border-color: #e6230b;
      }
    </style>-->
  </head>
  <body>
  <script src="//ajax.googleapis.com/ajax/libs/jquery/1.11.1/jquery.min.js"></script>
  <script src="/static/bootstrap.min.js"></script>
  <script src="//cdn.datatables.net/1.10.1/js/jquery.dataTables.min.js"></script>
  <script src="/static/dataTables.bootstrap.js"></script>
  <script type="text/javascript" src="https://www.google.com/jsapi"></script>
  <div style="background:#F26711; padding: 20px; margin-bottom: 20px">
  <div class="pull-left" style="padding-right: 20px">
    <img src="/static/pkgsrc-white.png" width="64px" height="64px">
  </div>
  <h1 style="color: white">BulkTracker
    <small style="color: white">pkgsrc bulk build status</small>
  </h1>
  </div>
  <div class="container">
  <div class="row">
`

const PageFooter = `
  </div>
  </div>
  </body>
</html>
`

const StartPageLead = `
  <div class="jumbotron">
    <p>BulkTracker is a web app to follow bulk build status in pkgsrc,
      the <a href="http://www.NetBSD.org/">NetBSD</a> package collection.
    </p>
    <p>
      <a class="btn btn-primary btn-lg" role="button" href="http://www.pkgsrc.org/">Learn more about pkgsrc</a>
    </p>
  </div>

  <h2>Recent Builds &nbsp; <a href="/builds" class="btn btn-primary">Show all</a></h2>
`

const tableBegin = `
  <table class="table">
    <thead>
      <tr>
	{{range .}}
	<th>{{.}}</th>
	{{end}}
      </tr>
    </thead>
    <tbody>`

var TableBegin = template.Must(template.New("TableBegin").Parse(tableBegin))

const TableEnd = `
    </tbody>
  </table>
`

const tableBuilds = `
      <tr>
	<td>
	  <a href="/build/{{.Key}}">{{.Build.Date}}</a>
	</td>
	<td>
	  <a href="/build/{{.Key}}">{{.Build.Platform}}</a>
	</td>
	<td>
	  <span class="text-danger">{{.Build.NumFailed}} failed</span> /
	  <span class="text-warning">{{.Build.NumIndirectFailed}} indirect-failed</span> /
	  <span class="text-success">{{.Build.NumOK}} ok</span>
	</td>
	<td>{{.Build.User}}</td>
      </tr>`

var TableBuilds = template.Must(template.New("TableBuilds").Parse(tableBuilds))

const tablePkgs = `
      <tr>
	<td>
	  <a href="/pkg/{{.Key}}">{{.Pkg.Category}}{{.Pkg.Dir}}</a>
	</td>
	<td>
	  <a href="/pkg/{{.Key}}">{{.Pkg.PkgName}}</a>
	</td>
	{{if eq .Pkg.BuildStatus 0}}
	<td class="success text-success">ok</td>
	{{else if eq .Pkg.BuildStatus 1}}
	<td class="info text-info">prefailed</td>
	{{else if eq .Pkg.BuildStatus 2}}
	<td class="danger text-danger">failed</td>
	{{else if eq .Pkg.BuildStatus 3}}
	<td class="warning text-warning">indirect-failed</td>
	{{else if eq .Pkg.BuildStatus 4}}
	<td class="info text-info">indirect-prefailed</td>
	{{end}}
	<td>
	  {{.Pkg.Breaks}}
	</td>
      </tr>`

var TablePkgs = template.Must(template.New("TablePkgs").Parse(tablePkgs))

const bulkBuildInfo = `
      <div class="col-md-8">
        <dl class="dl-horizontal" style="font-size: 120%">
	  <dt>Platform</dt>
	  <dd>{{.Platform}}</dd>
	  <dt>Compiler</dt>
	  <dd>{{.Compiler}}</dd>
	  <dt>Timestamp</dt>
	  <dd>{{.Date}}</dd>
	  <dt>User</dt>
	  <dd>{{.User}}</dd>
	</dl>
      </div>
      <div class="col-md-4"><div id="bulk-pie"></div></div>
    </div>
    <div class="row">
      <script type="text/javascript">
	google.load('visualization', '1.0', {'packages':['corechart']});

	function drawBulkPiechart() {
	  // Create and populate the data table.
	  var data = google.visualization.arrayToDataTable([
	    ['Number', 'Status'],
	    ['ok', {{.NumOK}}],
	    ['prefailed', {{.NumPrefailed}}],
	    ['indirect-failed', {{.NumIndirectFailed}}],
	    ['failed', {{.NumFailed}}],
	  ]);

	  // Create and draw the visualization.
	  new google.visualization.PieChart(document.getElementById('bulk-pie')).
	    draw(data, {
	      pieHole: 0.4,
	      pieSliceText: 'value',
	      chartArea: {
		width: 120,
		height: 120,
	      },
	      title: 'Build Status',
	      legend: { position: 'none' },
	      slices: {
		0: { color: 'green' },
		1: { color: 'blue' },
		2: { color: 'orange' },
		3: { color: 'red' },
	      },
	    });
	}
	google.setOnLoadCallback(drawBulkPiechart);
      </script>`

var BulkBuildInfo = template.Must(template.New("BulkBuildInfo").Parse(bulkBuildInfo))

const pkgInfo = `
    <dl class="dl-horizontal" style="font-size: 120%">
      <dt>Package location</dt>
      <dd>
        {{.Pkg.Category}}{{.Pkg.Dir}}
	<a href="http://pkgsrc.se/{{.Pkg.Category}}{{.Pkg.Dir}}" class="btn btn-default">pkgsrc.se</a>
      </dd>
      <dt>Package name</dt>
      <dd>{{.Pkg.PkgName}}</dd>
      <dt>Build Status</dt>
      {{if eq .Pkg.BuildStatus 0}}
      <dd><span class="label label-success">ok</span></dd>
      {{else if eq .Pkg.BuildStatus 1}}
      <dd><span class="label label-info">prefailed</span></dd>
      {{else if eq .Pkg.BuildStatus 2}}
      <dd><span class="label label-danger">failed</span></dd>
      {{else if eq .Pkg.BuildStatus 3}}
      <dd><span class="label label-warning">indirect-failed</span></dd>
      {{else if eq .Pkg.BuildStatus 4}}
      <dd><span class="label label-info">indirect-prefailed</span></dd>
      {{end}}
      {{if eq .Pkg.BuildStatus 2}}
      <dt>Build Logs</dt>
      <dd>
        <div class="btn-group btn-group-sm">
	  <a href="{{.Build.BaseURL}}/{{.Pkg.PkgName}}/work.log" class="btn btn-default">work</a>
	  <a href="{{.Build.BaseURL}}/{{.Pkg.PkgName}}/pre-clean.log" class="btn btn-default">pre-clean</a>
	  <a href="{{.Build.BaseURL}}/{{.Pkg.PkgName}}/checksum.log" class="btn btn-default">checksum</a>
	  <a href="{{.Build.BaseURL}}/{{.Pkg.PkgName}}/depends.log" class="btn btn-default">depends</a>
	  <a href="{{.Build.BaseURL}}/{{.Pkg.PkgName}}/configure.log" class="btn btn-default">configure</a>
	  <a href="{{.Build.BaseURL}}/{{.Pkg.PkgName}}/build.log" class="btn btn-default">build</a>
	  <a href="{{.Build.BaseURL}}/{{.Pkg.PkgName}}/install.log" class="btn btn-default">install</a>
	</div>
      </dd>
      {{end}}
      <dt>Platform</dt>
      <dd>{{.Build.Platform}}</dd>
      <dt>Compiler</dt>
      <dd>{{.Build.Compiler}}</dd>
      <dt>Built on</dt>
      <dd>{{.Build.Date}}</dd>
      <dt>Built by</dt>
      <dd>{{.Build.User}}</dd>
    </dl>
  </div><div class="row">`

var PkgInfo = template.Must(template.New("PkgInfo").Parse(pkgInfo))

const ReindexOK = `<div class="alert alert-success" role="alert">
  Re-index now in progress. This will take about one minute.</div>`

const noDetails = `<div class="alert alert-danger" role="alert">
  No build details found. Try
  <a href="{{.}}?a=reindex" rel="nofollow" class="alert-link">recreating
  the index.</a>
</div>`

var NoDetails = template.Must(template.New("NoDetails").Parse(noDetails)) 

const categoryList = `<h2>Results by category</h2>
  <ul class="list-inline">
  {{$url := .CurrentURL}}{{range $c := .Categories}}
    <li style="width: 14em"><a href="{{$url}}/{{.Category}}">{{.Category}}</a></li>
  {{end}}
  </ul>
`

var CategoryList = template.Must(template.New("CategoryList").Parse(categoryList))

const heading = `<h2>{{.}}</h2>`

var Heading = template.Must(template.New("Heading").Parse(heading))

const DataTable = `
  <script type="text/javascript">
    $(document).ready(function() {
	$('.table').dataTable();
    } );
  </script>
`
