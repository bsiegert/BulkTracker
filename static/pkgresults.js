var statuses = {
  0: "ok",
  1: "prefailed",
  2: "failed",
  3: "indirect-failed",
  4: "indirect-prefailed"
};
var classes = {
  0: "success text-success",
  1: "info text-info",
  2: "danger text-danger",
  3: "warning text-warning",
  4: "info text-info"
};

function PkgResultsTable(event) {
  var pkgname = null;
  var pkgname_re = /^[A-Za-z0-9+\-_]+\/[A-Za-z0-9+\-_]+$/;
  var fragment = window.location.pathname.split("/pkgresults/")[1];
  if (fragment && pkgname_re.test(fragment)) {
    pkgname = fragment;
  } else {
    let err = $('#error');
    err.html("Failed to decode the package name from the URL.");
    err.show();
    $('#results').hide();
  }
  $('#pkgname-header').text(pkgname);

  $('.table').dataTable({
    destroy: true,
    paging: false,
    ajax: {
      url:     `/json/${event.data}/${pkgname}`,
      dataSrc: ""
    },
    columns: [
      {data: "Pkg.PkgName"},
      {
	data:   "Pkg.BuildStatus",
	render: function(data, type, row, meta) {
	  return statuses[data];
	}
      },
      {
	data:   "Build.Timestamp",
	render: function(data, type, row, meta) {
	  return data.split("T")[0];
	}
      },
      {data: "Build.Branch"},
      {data: "Build.Platform"},
      {data: "Build.Compiler"}
    ],
    createdRow: function(row, data, dataIndex) {
      $('td:eq(0)', row).wrapInner('<a href="/pkg/'+data.Pkg.Key+'"></a>');
      $('td:eq(1)', row).addClass(classes[data.Pkg.BuildStatus]);
      $('td:eq(4)', row).wrapInner('<a href="/build/'+data.Build.Key+'"></a>');
    }
  });  
}

$(document).ready(function() {
  PkgResultsTable({data: "pkgresults"});
  $("#latest").on("click", null, "pkgresults", PkgResultsTable);
  $("#all").on("click", null, "allpkgresults", PkgResultsTable);
});

