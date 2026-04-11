$(document).ready(function() {
  $('#pkgresults').submit(function() {
    var pkg = encodeURIComponent($('select#results-pkg').val()).replace(/%2F/gi, "/");
    $(location).attr('href', `${bt.basePath}${pkg}`);
    return false;
  });

  var $pkg = $('#results-pkg');
  $pkg.removeAttr('data-select2-id tabindex aria-hidden')
      .removeClass('select2-hidden-accessible');
  $pkg.next('.select2-container').remove();

  $pkg.select2({
    theme: 'bootstrap',
    tags: true,
    ajax: {
      url: bt.basePath+'json/autocomplete/',
      dataType: 'json'
    },
    insertTag: function(data, tag) {
      // Insert the user-typed tag at the end instead of the beginning
      data.push(tag);
    }
  });

  setTimeout(function() {
    $pkg.select2('open');
    var searchField = document.querySelector('.select2-container--open .select2-search__field');
    if (searchField) {
      searchField.focus();
    }
  }, 500);
});
