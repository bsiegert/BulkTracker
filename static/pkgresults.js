function PkgResultsTable(event) {
  var pkgname = null;
  var pkgname_re = /^[A-Za-z0-9+\-_]+\/[A-Za-z0-9+\-_]+$/;
  var fragment = window.location.pathname;
  if (fragment.startsWith(bt.basePath)) {
    fragment = fragment.substring(bt.basePath.length);
  }
  if (fragment.startsWith("pkgresults/")) {
    fragment = fragment.split("pkgresults/")[1];
  }
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
      url: `${bt.basePath}json/${event.data}/${pkgname}`,
      dataSrc: ""
    },
    columns: [
      { data: "PkgName" },
      {
        data: "BuildStatus",
        render: function (data, type, row, meta) {
          return statuses[data];
        }
      },
      {
        data: "BuildTs",
        render: function (data, type, row, meta) {
          if (!data) return;
          return data.split("T")[0];
        }
      },
      { data: "Branch" },
      { data: "Platform" },
      { data: "Compiler" }
    ],
    order: [0, "desc"],
    createdRow: function (row, data, dataIndex) {
      $('td:eq(0)', row).wrapInner(`<a href="${bt.basePath}pkg/${data.ResultID}"></a>`);
      $('td:eq(1)', row).addClass(classes[data.BuildStatus]);
      $('td:eq(2)', row).wrapInner(`<a href="${bt.basePath}build/${data.BuildID}"></a>`);
    }
  });
}

$(document).ready(function () {
  PkgResultsTable({ data: "pkgresults" });
  $("#latest").on("click", null, "pkgresults", PkgResultsTable);
  $("#all").on("click", null, "allpkgresults", PkgResultsTable);
});

