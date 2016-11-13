// BulkTracker module.
var bt = bt || {};
bt.categories = bt.categories || {};

// init fires off an AJAX request to populate the categories table.
bt.categories.init = function () {
  $.ajax({url: '/json/dir/'}).done(function (data) {
    for (var i = 0; i < data.length; i++) {
      bt.categories.addPanel(data[i]);
      bt.categories.addHandler(data[i]);
    }
  });
}

// addHandler adds an on-click handler to the collapse link for
// name. The handler loads a list of packages in that category via AJAX
// and builds up a list.
bt.categories.addHandler = function (name) {
  name = name.split("/")[0];
  $('a#'+name+'-collapse').click(function() {
    $.ajax({
      url: '/json/dir/'+name+'/',
    }).done(function (data) {
      var a = ['<ul class="list-inline">'];
      for (var i in data) {
	a.push('<li class="column-item"><a href="/pkgresults/');
	a.push(name, '/', data[i], '">', data[i], '</a></li>');
      }
      a.push('</ul>');
      $('#'+name+'-body').html(a.join(""));
    })
    .fail(function () {
      $('#'+name+'-body').html('<p class="text-danger">Failed to load navigation.</p>');
    });
  });
}

bt.categories.addPanel = function (name) {
  name = name.split("/")[0];
  var a = [
	  '<li class="column-item">',
          '<a id="', name,
	  '-collapse" data-toggle="collapse" href="#', name,
	  '">',
	  name,
	  '/</a></li>',
          '<div id="', name,
	  '" class="panel panel-default panel-collapse collapse">',
          '<div class="panel-heading">',
	  name,
	  '</div>',
          '<div id="', name, '-body" class="panel-body">',
          '<p class="text-muted">Loading ...</p></div></div>',
  ];
  $('#categories').append(a.join(""));
}
