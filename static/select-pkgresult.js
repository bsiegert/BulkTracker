$(document).ready(function() {
  $('#pkgresults').submit(function() {
    var pkg = encodeURIComponent($('select#results-pkg').val()).replace("%2F", "/");
    $(location).attr('href', "/pkgresults/" + pkg);
    return false;
  });
  
  $('#results-pkg').select2({
    theme: 'bootstrap',
    ajax: {
      url: '/json/autocomplete/',
      dataType: 'json'
    }
  });
});
