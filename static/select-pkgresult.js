$(document).ready(function() {
  $('#pkgresults').submit(function() {
    var pkg = encodeURIComponent($('input#results-pkg').val()).replace("%2F", "/");
    $(location).attr('href', "/pkgresults/" + pkg);
    return false;
  });
});
