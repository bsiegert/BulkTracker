// BulkTracker module.
var bt = bt || {};
bt.builds = bt.builds || {};

bt.builds.columns = [
  {
    data: "BuildTs",
    render: function (data) { 
      if (!data) return;
      return data.split("T")[0]
    }
  },
  {data: "Branch"},
  {data: "Platform"},
  {
    render: (data, type, row) =>
      `<span class=\"text-danger\">${row.NumFailed} failed</span> / <span class=\"text-warning\">${row.NumIndirectFailed} indirect-failed</span> / <span class=\"text-success\">${row.NumOk} ok</span>`
  },
  {data: "BuildUser"}
];

bt.builds.createdRow = function (row, data) {
  $('td', row).filter(function (i) { return i < 3 })
    .wrapInner(`<a href="${bt.basePath}build/${data.BuildID}"></a>`);
};

bt.builds.init = function () {
  $('.table').dataTable({
    ajax: {
      url: `${bt.basePath}json/allbuilds/`,
      dataSrc: ""
    },
    columns: bt.builds.columns,
    createdRow: bt.builds.createdRow,
    order: [[0, 'desc']]
  });
};

$(document).ready(bt.builds.init);
