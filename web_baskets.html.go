package main

var (
	basketsPageContent = []byte(`<!DOCTYPE html>
<html>
<head lang="en">
  <title>Request Baskets - Administration</title>
  <link rel="stylesheet" href="https://maxcdn.bootstrapcdn.com/bootstrap/3.3.7/css/bootstrap.min.css" integrity="sha384-BVYiiSIFeK1dGmJRAkycuHAHRg32OmUcww7on3RYdg4Va+PmSTsz/K68vbdEjh4u" crossorigin="anonymous">
  <link rel="stylesheet" href="https://maxcdn.bootstrapcdn.com/bootstrap/3.3.7/css/bootstrap-theme.min.css" integrity="sha384-rHyoN1iRsVXV4nD0JutlnGaslCJuC7uwjduW9SVrLvRYooPp2bWYgmgJQIXwl/Sp" crossorigin="anonymous">
  <link rel="stylesheet" href="https://maxcdn.bootstrapcdn.com/font-awesome/4.6.3/css/font-awesome.min.css" integrity="sha384-T8Gy5hrqNKT+hzMclPo118YTQO6cYprQmhrYwIiQ/3axmI1hQomh7Ud2hPOy8SP1" crossorigin="anonymous">
  <script src="https://code.jquery.com/jquery-3.2.1.min.js" integrity="sha256-hwg4gsxgFZhOsEEamdOYGBf13FyQuiTwlAQgxVSNgt4=" crossorigin="anonymous"></script>
  <script src="https://maxcdn.bootstrapcdn.com/bootstrap/3.3.7/js/bootstrap.min.js" integrity="sha384-Tc5IQib027qvyjSMfHjOMaLkfuWVxZxUPnCJA7l2mCWNIpG9mGCD8wGNIcPD7Txa" crossorigin="anonymous"></script>

  <style>
    html { position: relative; min-height: 100%; }
    body { padding-top: 70px; margin-bottom: 60px; }
    .footer { position: absolute; bottom: 0; width: 100%; height: 60px; background-color: #f5f5f5; }
    .container .text-muted { margin: 20px 0; }
    h1 { margin-top: 2px; }
    #more { margin-left: 60px; padding-bottom: 10px; }
    #all_baskets ul { width: 100%; }
    #all_baskets li { padding: 0 0 5px 20px; float: left; display: inline; position: relative; width: 25%; }
    #all_baskets li:before { content: "\f291"; font-family: "FontAwesome"; position: absolute; left: 0px; top:0px; }
  </style>

  <script>
  (function($) {
    var showDetails = false;
    var basketsCount = 0;

    function onAjaxError(jqXHR) {
      if (jqXHR.status == 401) {
        $("#master_token_dialog").modal({ keyboard : false });
      } else {
        $("#error_message_label").html("HTTP " + jqXHR.status + " - " + jqXHR.statusText);
        $("#error_message_text").html(jqXHR.responseText);
        $("#error_message").modal();
      }
    }

    function addBaskets(data) {
      if (data && data.names) {
        var baskets = (showDetails) ? $("#all_baskets_details tbody") : $("#all_baskets");
        var index, name, displayName, basketRowId;

        for (index = 0; index < data.names.length; ++index) {
          name = data.names[index];
          displayName = (name.length < 25) ? name : name.substring(0, 25) + "...";
          basketRowId = "basket_row_" + basketsCount;

          if (showDetails) {
            baskets.append("<tr id='" + basketRowId + "'><td><a href='/web/" + name + "' title='" + name + "'>" +
              displayName + "</a></td></tr>");
            fetchBasketDetails(name, basketRowId);
          } else {
            baskets.append("<li><a href='/web/" + name + "' title='" + name + "'>" + displayName + "</a></li>");
          }

          basketsCount++;
        }

        if (data.has_more) {
          $("#more").removeClass("hide");
        } else {
          $("#more").addClass("hide");
        }
      }
    }

    function fetchBaskets() {
      $.ajax({
        method: "GET",
        url: "/baskets?skip=" + basketsCount,
        headers: {
          "Authorization" : sessionStorage.getItem("master_token")
        }
      }).done(function(data) {
        addBaskets(data);
      }).fail(onAjaxError);
    }

    function fetchBasketDetails(name, basketRowId) {
      $.ajax({
        method: "GET",
        url: "/baskets/" + name + "/requests?max=1",
        headers: {
          "Authorization" : sessionStorage.getItem("master_token")
        }
      }).done(function(requests) {
        $.ajax({
          method: "GET",
          url: "/baskets/" + name,
          headers: {
            "Authorization" : sessionStorage.getItem("master_token")
          }
        }).done(function(config) {
          updateBasketDetails(basketRowId, requests, config);
        }).fail(onAjaxError);
      }).fail(onAjaxError);
    }

    function updateBasketDetails(basketRowId, requests, config) {
      var basketRow = $("#" + basketRowId);
      if (requests) {
        basketRow.append("<td>" + requests.count + " (" + requests.total_count + ")</td>");
      } else {
        basketRow.append("<td>failed to retrieve!</td>");
      }
      if (config) {
        var details = "Max capacity: " + config.capacity;
        if (requests.requests && requests.requests.length > 0) {
          details += "; Last Req.: " + new Date(requests.requests[0].date).toISOString();
        }
        if (config.forward_url) {
          details += "; Forward URL: " + config.forward_url;
        }
        basketRow.append("<td>" + details + "</td>");
      } else {
        basketRow.append("<td>failed to retrieve!</td>");
      }
    }

    function saveMasterToken() {
      var token = $("#master_token").val();
      $("#master_token").val("");
      $("#master_token_dialog").modal("hide");
      if (token) {
        sessionStorage.setItem("master_token", token);
      } else {
        sessionStorage.removeItem("master_token");
      }
      fetchBaskets();
    }

    // Initialization
    $(document).ready(function() {
      $("#master_token_dialog").on("hidden.bs.modal", function (event) {
        saveMasterToken();
      });
      $("#fetch_more").on("click", function(event) {
        fetchBaskets();
      });
      $("#list_quick").on("change", function(event) {
        location.reload();
      });
      $("#list_details").on("change", function(event) {
        $("#all_baskets").html("");
        $("#all_baskets_details").removeClass("hide");
        basketsCount = 0;
        showDetails = true;
        fetchBaskets();
      });

      fetchBaskets();
    });
  })(jQuery);
  </script>
</head>
<body>
  <!-- Fixed navbar -->
  <nav class="navbar navbar-default navbar-fixed-top">
    <div class="container">
      <div class="navbar-header">
        <a class="navbar-brand" href="/web">Request Baskets</a>
      </div>
      <div class="collapse navbar-collapse">
        <form class="navbar-form navbar-right">
          <div class="btn-group btn-group-toggle" data-toggle="buttons">
            <label class="btn btn-default active">
              <input type="radio" name="options" id="list_quick" autocomplete="off" checked>
                <span class="glyphicon glyphicon-th" aria-hidden="true"></span>
              </input>
            </label>
            <label class="btn btn-default">
              <input type="radio" name="options" id="list_details" autocomplete="off">
                <span class="glyphicon glyphicon-th-list" aria-hidden="true"></span>
              </input>
            </label>
          </div>
        </form>
      </div>
    </div>
  </nav>

  <!-- Error message -->
  <div class="modal fade" id="error_message" tabindex="-1">
    <div class="modal-dialog">
      <div class="modal-content panel-danger">
        <div class="modal-header panel-heading">
          <button type="button" class="close" data-dismiss="modal">&times;</button>
          <h4 class="modal-title" id="error_message_label">HTTP error</h4>
        </div>
        <div class="modal-body">
          <p id="error_message_text"></p>
        </div>
        <div class="modal-footer">
          <button type="button" class="btn btn-default" data-dismiss="modal">Close</button>
        </div>
      </div>
    </div>
  </div>

  <!-- Master token dialog -->
  <div class="modal fade" id="master_token_dialog" tabindex="-1">
    <div class="modal-dialog">
      <div class="modal-content panel-warning">
        <div class="modal-header panel-heading">
          <h4 class="modal-title">Master Token</h4>
        </div>
        <form id="master_token_form">
        <div class="modal-body">
          <p>By providing the master token you will gain access to all baskets.</p>
          <div class="form-group">
            <label for="master_token" class="control-label">Token:</label>
            <input type="password" class="form-control" id="master_token">
          </div>
        </div>
        <div class="modal-footer">
          <a href="/web" class="btn btn-default">Back to list of your baskets</a>
          <button type="submit" class="btn btn-success" data-dismiss="modal">Authorize</button>
        </div>
        </form>
      </div>
    </div>
  </div>

  <div class="container">
    <div class="row">
      <div class="col-md-4">
        <h1>All Baskets</h1>
      </div>
    </div>
    <hr/>
    <div class="row">
      <ul id="all_baskets">
      </ul>
      <table id="all_baskets_details" class="table hide">
        <thead>
          <tr>
            <th>Basket</th>
            <th>Requests</th>
            <th width="70%">Details</th>
          </tr>
        </thead>
        <tbody>
        </tbody>
      </table>
      <div id="more" class="hide">
        <a id="fetch_more" class="btn btn-default btn-s">more...</a>
      </div>
    </div>
  </div>

  <footer class="footer">
    <div class="container">
      <p class="text-muted"><small>Powered by <a href="https://github.com/darklynx/request-baskets">request-baskets</a></small></p>
    </div>
  </footer>
</body>
</html>`)
)
