$(document).ready(function() {
  $('#pkgresults').submit(function() {
    var pkg = encodeURIComponent($('select#results-pkg').val()).replace(/%2F/gi, "/");
    $(location).attr('href', `/${pkg}`);
    return false;
  });
  
  $('#results-pkg').select2({
    theme: 'bootstrap',
    tags: true,
    ajax: {
      url: '/json/autocomplete/',
      dataType: 'json'
    }
  });
});
