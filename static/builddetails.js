// BulkTracker module.
var bt = bt || {};
bt.buildDetails = bt.buildDetails || {};

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
    fixedHeader: true,
    layout: {
      topStart: 'searchBuilder'
    },
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
    fixedHeader: true,
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
      $('td:eq(1)', row).addClass(classes[data.BuildStatus]);
    }
  });

}
