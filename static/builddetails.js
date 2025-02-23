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
  {data: "PkgMaintainer"},
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

bt.buildDetails.init = function (selector, apiName, num) {
  $(selector).dataTable({
    paging: false,
    ajax: {
      url: `${bt.basePath}json/${apiName}/${num}`,
      dataSrc: ""
    },
    columns: bt.buildDetails.columns,
    order: [[4, 'desc']],
    createdRow: function (row, data, dataIndex) {
      $('td:eq(1)', row).wrapInner(`<a href="${bt.basePath}pkg/${data.ResultID}"></a>`);
      $('td:eq(3)', row).addClass(classes[data.BuildStatus]);
    }
  });
};

bt.buildDetails.initSentinels = function (selector, apiName, num) {
  $(selector).dataTable({
    paging: false,
    ajax: {
      url: `${bt.basePath}json/${apiName}/${num}`,
      dataSrc: ""
    },
    columns: [
      {data: "PkgName"},
      {
        data: "BuildStatus",
        render: function (data, type, row, meta) {
          return statuses[data];
        }
      },
      {
        data: "FailedDeps[ ]",
        render: function (data, type, row, meta) {
          s = "";
          row.FailedDeps.forEach(element => {
            s += `<a href="${bt.basePath}pkg/${element.ResultID}">${element.PkgName}</a> `
          });
          return s;
        }
      },
    ],
    createdRow: function (row, data, dataIndex) {
      $('td:eq(0)', row).wrapInner(`<a href="${bt.basePath}pkg/${data.ResultID}"></a>`);
      $('td:eq(2)', row).addClass(classes[data.BuildStatus]);
    }
  });

}
