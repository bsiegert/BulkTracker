$(document).ready(function() {
    var activeLogName = "";
    var isFullscreen = false;

    // Disable buttons that have 404 statusCode
    $("a[name=build-stages]").each(function() {
      var text = $(this).text();
      var id = text + ".log";
      var statusCode = urlData[id].statusCode;
      if (statusCode == 404) {
        $(this).addClass("disabled");
      }
    });

    $("a[name=build-stages]").click(function(e) {
      e.preventDefault();
      var text = $(this).text();
      var id = text + ".log";
      var statusCode = urlData[id].statusCode;
      var url = new String(urlData[id].url);
      var displayURL = url.length > 90 ? url.substring(0, 40) + " ... " + url.substring(url.length-40, url.length): url;
      var displayData = urlData[id].data === "" ? "No data available" : urlData[id].data;

      // Add shortcut unicode character to the displayURL
      displayURL = displayURL + " &#x21D7;"; // Unicode character for link

      $("a[name=build-stages]").removeClass("btn-warning").addClass("btn-default");

      if (activeLogName == text) {
        $("#logContainer").hide();
        activeLogName = "";
        return;
      } else {
        $(this).removeClass("btn-default").addClass("btn-warning");
        activeLogName = text;
      }

      $("#logFrame").text(urlData[id].error || displayData);
      $("#logContainer").show();
      $("#original-logfile").attr("href", urlData[id].url);
      $("#original-logfile").html(displayURL);
    });
    
    // Toggle fullscreen function
    $("#toggleFullscreen").click(function() {
      isFullscreen = !isFullscreen;
      if (isFullscreen) {
        $("#logContainer").addClass("fullscreen-log");
        $("#logFrame").addClass("fullscreen-frame");
        $(this).html('&#x1F5D9; Collapse'); // Unicode character for collapse
      } else {
        $("#logContainer").removeClass("fullscreen-log");
        $("#logFrame").removeClass("fullscreen-frame");
        $(this).html('&#x26F6; Fullscreen'); // Unicode character for fullscreen
      }
    });
  });