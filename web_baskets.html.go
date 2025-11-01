package main

var (
	basketsPageContentTemplate = `<!DOCTYPE html>
<html>
<head lang="en">
  <title>Request Baskets - Administration</title>{{.ThemeCSS}}
  <link rel="stylesheet" href="/static/css/font-awesome-4.6.3.min.css">
  <script src="/static/js/jquery-3.6.4.min.js"></script>
  <script src="/static/js/bootstrap-3.3.7.min.js"></script>

  <style>
    html { position: relative; min-height: 100%; }
    body { padding-top: 70px; margin-bottom: 60px; }
    .footer { position: absolute; bottom: 0; left: 0; min-width: 100%; height: 60px; background-color: #f5f5f5; }
    .container .text-muted { margin: 20px 0; }
    h1 { margin-top: 2px; }
    #more { margin-left: 60px; padding-bottom: 10px; }
    #all_baskets ul { width: 100%; }
    #all_baskets li { float: left; display: inline; position: relative; width: 25%; }
    .baskets li { padding: 0 0 5px 20px; }
    .baskets li:before { content: "\f291"; font-family: "FontAwesome"; position: absolute; left: 0px; top:0px; }
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
          basketRowId = "basket_row_" + basketsCount;

          if (showDetails) {
            baskets.append("<tr id='" + basketRowId + "'><td><a href='{{.Prefix}}/web/" + name + "' title='" + name + "'>" +
              toDisplayName(name) + "</a></td></tr>");
            fetchBasketDetails(name, basketRowId);
          } else {
            baskets.append("<li><a href='{{.Prefix}}/web/" + name + "' title='" + name + "'>" + toDisplayName(name) + "</a></li>");
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

    function showStats(stats) {
      $("#stats_baskets_count").html(toDisplayInt(stats.baskets_count));
      $("#stats_empty_baskets_count").html(toDisplayInt(stats.empty_baskets_count));
      $("#stats_requests_count").html(toDisplayInt(stats.requests_count));
      $("#stats_requests_total_count").html(toDisplayInt(stats.requests_total_count));
      $("#stats_max_basket_size").html(toDisplayInt(stats.max_basket_size));
      $("#stats_avg_basket_size").html(toDisplayInt(stats.avg_basket_size));
      showTopBaskets($("#top_baskets_size"), stats.top_baskets_size);
      showTopBaskets($("#top_baskets_recent"), stats.top_baskets_recent);
    }

    function showTopBaskets(basketsList, baskets) {
      if (basketsList && baskets && baskets.length) {
        var index, basket;
        for (index = 0; index < baskets.length; ++index) {
          basket = baskets[index];
          basketsList.append("<li class='list-group-item'><a href='{{.Prefix}}/web/" + basket.name
            + "' title='" + basket.name + "'>" + toDisplayName(basket.name) + "</a> - " + basket.requests_count
            + " (" + toDisplayInt(basket.requests_total_count) + ")<br>Last Request: "
            + ((basket.last_request_date > 0) ? new Date(basket.last_request_date).toISOString() : "n/a") + "</li>");
        }
      }
    }

    function toDisplayName(name) {
      return (name.length < 25) ? name : name.substring(0, 25) + "...";
    }

    function toDisplayInt(value) {
      return value.toString().replace(/\d(?=(\d{3})+$)/g, "$& ");
    }

    function fetchBaskets() {
      $.ajax({
        method: "GET",
        url: "{{.Prefix}}/api/baskets?skip=" + basketsCount,
        headers: {
          "Authorization" : sessionStorage.getItem("master_token")
        }
      }).done(function(data) {
        addBaskets(data);
      }).fail(onAjaxError);
    }

    function fetchStats() {
      $.ajax({
        method: "GET",
        url: "{{.Prefix}}/api/stats",
        headers: {
          "Authorization" : sessionStorage.getItem("master_token")
        }
      }).done(function(stats) {
        showStats(stats);
        fetchBaskets();
      }).fail(onAjaxError);
    }

    function fetchBasketDetails(name, basketRowId) {
      $.ajax({
        method: "GET",
        url: "{{.Prefix}}/api/baskets/" + name + "/requests?max=1",
        headers: {
          "Authorization" : sessionStorage.getItem("master_token")
        }
      }).done(function(requests) {
        $.ajax({
          method: "GET",
          url: "{{.Prefix}}/api/baskets/" + name,
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
        basketRow.append("<td>" + requests.count + " (" + toDisplayInt(requests.total_count) + ")</td>");
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
      fetchStats();
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

      fetchStats();
    });
  })(jQuery);
  </script>
</head>
<body>
  <!-- Fixed navbar -->
  <nav class="navbar navbar-default navbar-fixed-top">
    <div class="container">
      <div class="navbar-header">
        <a class="navbar-brand" href="{{.Prefix}}/web">Request Baskets</a>
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
          <a href="{{.Prefix}}/web" class="btn btn-default">Back to list of your baskets</a>
          <button type="submit" class="btn btn-success" data-dismiss="modal">Authorize</button>
        </div>
        </form>
      </div>
    </div>
  </div>

  <div class="container">
    <div class="row">
      <div class="col-md-4">
        <div class="panel panel-default">
          <div class="panel-heading">
            <h3 class="panel-title">Basic Statistics</h3>
          </div>
          <ul class="panel-body list-group">
            <li class="list-group-item">
              <span id="stats_baskets_count" class="badge">?</span>
              <b>Total baskets</b>
            </li>
            <li class="list-group-item">
              <span id="stats_empty_baskets_count" class="badge">?</span>
              Empty baskets
            </li>
            <li class="list-group-item">
              <span id="stats_requests_total_count" class="badge">?</span>
              <b>Total requests</b>
            </li>
            <li class="list-group-item">
              <span id="stats_requests_count" class="badge">?</span>
              Current requests
            </li>
            <li class="list-group-item">
              <span id="stats_max_basket_size" class="badge">?</span>
              Max. basket size
            </li>
            <li class="list-group-item">
              <span id="stats_avg_basket_size" class="badge">?</span>
              Avg. basket size
            </li>
          </ul>
        </div>
      </div>
      <div class="col-md-4">
        <div class="panel panel-default">
          <div class="panel-heading">
            <h3 class="panel-title">Top Baskets: by size</h3>
          </div>
          <ul id="top_baskets_size" class="panel-body list-group baskets">
          </ul>
        </div>
      </div>
      <div class="col-md-4">
        <div class="panel panel-default">
          <div class="panel-heading">
            <h3 class="panel-title">Top Baskets: recently active</h3>
          </div>
          <ul id="top_baskets_recent" class="panel-body list-group baskets">
          </ul>
        </div>
      </div>
    </div>
    <div class="row">
      <div class="col-md-4">
        <h3>All Baskets</h3>
      </div>
    </div>
    <hr/>
    <div class="row">
      <ul id="all_baskets" class="baskets">
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
      <p class="text-muted">
        <small>
          Powered by <a href="{{.Version.SourceCode}}">{{.Version.Name}}</a> |
          Version: <abbr title="From commit: {{.Version.CommitShort}} ({{.Version.Commit}})">{{.Version.Version}}</abbr>
        </small>
      </p>
    </div>
  </footer>
</body>
</html>`
)
