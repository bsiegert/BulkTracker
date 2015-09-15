// AddCollapseHandler adds an on-click handler to the collapse link for
// name. The handler loads a list of packages in that category via AJAX
// and builds up a list.
function AddCollapseHandler(name) {
  name = name.split("/")[0];
  $('a#'+name+'-collapse').click(function() {
    $.ajax({
      url: '/json/dir/'+name+'/',
    }).done(function(data) {
      _a = '<ul class="list-inline">';
      for (var i in data) {
	_a += '<li class="column-item"><a href="/pkgresults/'
	  + name + '/' + data[i] + '">' + data[i] + '</a></li>';
      }
      _a += '</ul>';
      $('#'+name+'-body').html(_a);
    })
    .fail(function() {
      $('#'+name+'-body').html('<p class="text-danger">Failed to load navigation.</p>');
    });
  });
}

function AddCollapsePanel(name) {
  name = name.split("/")[0];
  _a = '<li class="column-item">'
    + '<a id="' + name + '-collapse" data-toggle="collapse" href="#'
    + name + '">' + name + '/</a></li>'
    + '<div id="' + name + '" class="panel panel-default panel-collapse collapse">'
    + '<div class="panel-heading">' + name + '</div>'
    + '<div id="' + name + '-body" class="panel-body">'
    + '<p class="text-muted">Loading ...</p></div></div>';
  $('#categories').append(_a);
}
