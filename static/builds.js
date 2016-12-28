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
      console.log(row)
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

bt.builds.init = function () {
  $('.table').dataTable({
    ajax: {
      url: "/json/allbuilds/",
      dataSrc: ""
    },
    columns: bt.builds.columns
  });
};

$(document).ready(bt.builds.init);
