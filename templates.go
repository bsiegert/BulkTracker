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
    <!--<style type="text/css">
      .btn-primary {
        color: #fff;
	background-color: #f62711;
	border-color: #e6230b;
      }
    </style>-->
  </head>
  <body>
  <!-- jQuery (necessary for Bootstrap's JavaScript plugins) -->
  <script src="https://ajax.googleapis.com/ajax/libs/jquery/1.11.1/jquery.min.js"></script>
  <!-- Include all compiled plugins (below), or include individual files as needed -->
  <script src="/static/bootstrap.min.js"></script>
  <div style="background:#F26711; padding: 20px">
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

  <h2>Recent Builds</h2>
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
