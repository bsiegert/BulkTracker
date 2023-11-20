// BulkTracker module.
var bt = bt || {};
bt.buildDetails = bt.buildDetails || {};

// These are from pkgresults.js. TODO: have only one copy
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


bt.buildDetails.columns = [
  {data: "PkgPath"},
  {data: "PkgName"},
  {
    data: "BuildStatus",
    render: function (data, type, row, meta) {
      return statuses[data];
    }
  },
  {data: "Breaks"},
];

bt.buildDetails.createdRow = function (row, data) {
  $('td', row).filter((i) => i < 2)
    .wrapInner(`<a href="${bt.basePath}build/${data.ResultID}"></a>`);
};

bt.buildDetails.init = function () {
  var buildNo = window.location.pathname.slice(-2).replaceAll("/", "");
  $('.table').dataTable({
    ajax: {
      url: `${bt.basePath}json/pkgsbreakingmostothers/${buildNo}`,
      dataSrc: ""
    },
    columns: bt.buildDetails.columns,
    order: [[3, 'desc']],
    createdRow: function (row, data, dataIndex) {
      $('td:eq(1)', row).wrapInner(`<a href="${bt.basePath}pkg/${data.ResultID}"></a>`);
      $('td:eq(2)', row).addClass(classes[data.BuildStatus]);
    }
  });
};

$(document).ready(bt.buildDetails.init);
