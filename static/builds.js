// BulkTracker module.
var bt = bt || {};
bt.builds = bt.builds || {};

bt.builds.columns = [
  {
    data: "Timestamp",
    render: function (data) { return data.split("T")[0] }
  },
  {data: "Branch"},
  {data: "Platform"},
  {
    render: function (data, type, row) {
      return "<span class=\"text-danger\">" + row.NumFailed
	+ " failed</span> / <span class=\"text-warning\">"
	+ row.NumIndirectFailed 
	+ " indirect-failed</span> / <span class=\"text-success\">"
	+ row.NumOK
	+ " ok</span>"
    }
  },
  {data: "User"}
];

bt.builds.createdRow = function (row, data) {
  $('td', row).filter(function (i) { return i < 3 })
    .wrapInner('<a href="/build/'+data.Key+'"></a>');
};

bt.builds.init = function () {
  $('.table').dataTable({
    ajax: {
      url: "/json/allbuilds/",
      dataSrc: ""
    },
    columns: bt.builds.columns,
    createdRow: bt.builds.createdRow,
    order: [[0, 'desc']]
  });
};

$(document).ready(bt.builds.init);
