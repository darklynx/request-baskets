package main

const (
	basketPageContentTemplate = `<!DOCTYPE html>
<html>
<head lang="en">
  <title>Request Basket: {{.}}</title>
  <link rel="stylesheet" href="https://maxcdn.bootstrapcdn.com/bootstrap/3.3.7/css/bootstrap.min.css" integrity="sha384-BVYiiSIFeK1dGmJRAkycuHAHRg32OmUcww7on3RYdg4Va+PmSTsz/K68vbdEjh4u" crossorigin="anonymous">
  <link rel="stylesheet" href="https://maxcdn.bootstrapcdn.com/bootstrap/3.3.7/css/bootstrap-theme.min.css" integrity="sha384-rHyoN1iRsVXV4nD0JutlnGaslCJuC7uwjduW9SVrLvRYooPp2bWYgmgJQIXwl/Sp" crossorigin="anonymous">
  <link rel="stylesheet" href="https://cdnjs.cloudflare.com/ajax/libs/highlight.js/9.12.0/styles/default.min.css">
  <script src="https://code.jquery.com/jquery-3.2.1.min.js" integrity="sha256-hwg4gsxgFZhOsEEamdOYGBf13FyQuiTwlAQgxVSNgt4=" crossorigin="anonymous"></script>
  <script src="https://maxcdn.bootstrapcdn.com/bootstrap/3.3.7/js/bootstrap.min.js" integrity="sha384-Tc5IQib027qvyjSMfHjOMaLkfuWVxZxUPnCJA7l2mCWNIpG9mGCD8wGNIcPD7Txa" crossorigin="anonymous"></script>
  <script src="https://cdnjs.cloudflare.com/ajax/libs/highlight.js/9.12.0/highlight.min.js"></script>

  <style>
    body { padding-top: 70px; }
    h1 { margin-top: 2px; }
    #more { margin-left: 100px; }
  </style>

  <script>
  (function($) {
    var fetchedCount = 0;
    var totalCount = 0;
    var currentConfig;

    var autoRefresh = false;
    var autoRefreshId;

    function getParam(name) {
      var params = new RegExp("[\\?&]" + name + "=([^&#]*)").exec(window.location.search);
      return (params && params.length) ? params[1] : null;
    }

    function getBasketToken() {
      return localStorage.getItem("basket_{{.}}");
    }

    function getToken() {
      var token = getBasketToken();
      if (!token) { // fall back to master token if provided
        token = sessionStorage.getItem("master_token");
      }
      return token;
    }

    function onAjaxError(jqXHR) {
      if (jqXHR.status == 401) {
        localStorage.removeItem("basket_{{.}}");
        enableAutoRefresh(false);
        $("#token_dialog").modal({ keyboard : false });
      } else {
        $("#error_message_label").html("HTTP " + jqXHR.status + " - " + jqXHR.statusText);
        $("#error_message_text").html(jqXHR.responseText);
        $("#error_message").modal();
      }
    }

    function escapeHTML(value) {
      return value.replace(/&/g,"&amp;").replace(/</g,"&lt;").replace(/>/g,"&gt;").replace(/"/g,"&quot;");
    }

    function renderRequest(id, request) {
      var path = request.path;
      if (request.query) {
        path += "?";
        if (request.query.length > 70) {
          path += request.query.substring(0, 67) + "...";
        } else {
          path += request.query;
        }
      }

      var headers = [];
      for (header in request.headers) {
        headers.push(header + ": " + request.headers[header].join(","));
      }

      var headerClass = "default";
      switch(request.method) {
        case "GET":
          headerClass = "success";
          break;
        case "PUT":
          headerClass = "info";
          break;
        case "POST":
          headerClass = "primary";
          break;
        case "DELETE":
          headerClass = "danger";
          break;
      }

      var date = new Date(request.date);

      var html = '<div class="row"><div class="col-md-2"><h4 class="text-' + headerClass +
        '">[' + request.method + ']</h4><div><i class="glyphicon glyphicon-time" title="' + date.toString() + '"></i> ' + date.toLocaleTimeString() +
        '</div><div><i class="glyphicon glyphicon-calendar" title="' + date.toString() + '"></i> ' + date.toLocaleDateString() +
        '</div></div><div class="col-md-10"><div class="panel-group" id="' + id + '">' +
        '<div class="panel panel-' + headerClass + '"><div class="panel-heading"><h4 class="panel-title">' + escapeHTML(path) + '</h4></div></div>' +
        '<div class="panel panel-default"><div class="panel-heading"><h4 class="panel-title">' +
        '<a class="collapsed" data-toggle="collapse" data-parent="#' + id + '" href="#' + id + '_headers">Headers</a></h4></div>' +
        '<div id="' + id + '_headers" class="panel-collapse collapse">' +
        '<div class="panel-body"><pre>' + escapeHTML(headers.join('\n')) + '</pre></div></div></div>';

      if (request.query) {
        html += '<div class="panel panel-default"><div class="panel-heading"><h4 class="panel-title">' +
          '<a class="collapsed" data-toggle="collapse" data-parent="#' + id + '" href="#' + id + '_query">Query Params</a></h4></div>' +
          '<div id="' + id + '_query" class="panel-collapse collapse">' +
          '<div class="panel-body"><pre>' + escapeHTML(request.query.split('&').join('\n')) + '</pre></div></div></div>';
      }

      if (request.body) {
        html += '<div class="panel panel-default"><div class="panel-heading"><h4 class="panel-title">' +
          '<a class="collapsed" data-toggle="collapse" data-parent="#' + id + '" href="#' + id + '_body">Body</a></h4></div>' +
          '<div id="' + id + '_body" class="panel-collapse collapse in">' +
          '<div class="panel-body"><pre>' + escapeHTML(request.body) + '</pre></div></div></div>';
      }

      html += '</div></div></div><hr/>';

      return html;
    }

    function addRequests(data) {
      totalCount = data.total_count;
      $("#requests_count").html(data.count + " (" + totalCount + ")");
      if (data.count > 0) {
        $("#empty_basket").addClass("hide");
        $("#requests_link").removeClass("hide");
      } else {
        $("#empty_basket").removeClass("hide");
        $("#requests_link").addClass("hide");
      }

      if (data && data.requests) {
        var requests = $("#requests");
        var index, request;
        for (index = 0; index < data.requests.length; ++index) {
          request = data.requests[index];
          requestId = "req" + fetchedCount;
          requests.append(renderRequest(requestId, request));

          if (request.body) {
            var format = getContentFormat(request.headers["Content-Type"]);
            if (format !== "UNKNOWN") {
              var button = $('<button id="' + requestId + '_body_format_btn" for="' + requestId +
                '" format="' + format + '" type="button" class="btn btn-default">Format Content</button>');
              $("#" + requestId + "_body div pre").after(button);

              button.on("click", function(event) {
                  formatBody(this);
              });
            }
          }

          fetchedCount++;
        }
      }

      if (data.has_more) {
        $("#more").removeClass("hide");
        $("#more_count").html(data.count - fetchedCount);
      } else {
        $("#more").addClass("hide");
        $("#more_count").html("");
      }
    }

    function getContentFormat(contentType) {
      // array of header values to string
      contentType = (contentType || []).join(",");
      if ("application/x-www-form-urlencoded" === contentType.toLowerCase()) {
        return "FORM";
      } else if (/^(application|text)\/(json|.+\+json).*/i.test(contentType)) {
        return "JSON";
      } else if (/^(application|text)\/(xml|.+\+xml).*/i.test(contentType)) {
        return "XML";
      } else {
        return "UNKNOWN";
      }
    }

    function formatBody(btn) {
      var button = $(btn);
      var requestId = button.attr("for");
      var format = button.attr("format");

      var body = $("#" + requestId + "_body div pre");
      var code = $('<code class="' + getHighlightLang(format) + '"></code>');
      code.text(formatText(body.text(), format));
      body.empty();
      body.append(code);

      // highlight
      hljs.highlightBlock(code.get(0));

      // avoid further formatting
      button.remove();
    }

    function getHighlightLang(format) {
      switch(format) {
        case "JSON":
        case "XML":
          return "language-" + format.toLowerCase();
        default:
          return "no-highlight";
      }
    }

    function formatText(text, format) {
      switch (format) {
        case "JSON":
          return JSON.stringify(JSON.parse(text), null, 2);
        case "XML":
          return formatXml(text, "  ");
        case "FORM":
          return text.split("&").map(decodeURIComponent).join("\n");
      }
    }

    function formatXml(xml, tab) {
      var formatted = "";
      // minify first
      xml = xml.replace(/(>)\s*(<)/g, "$1$2");
      xml = xml.replace(/(>)(<)(\/*)/g, "$1\r\n$2$3");
      var pad = 0;
      $.each(xml.split("\r\n"), function(index, node) {
        var indent = 0;
        if (node.match(/.+<\/\w[^>]*>$/)) {
          indent = 0;
        } else if (node.match(/^<\/\w/)) {
          if (pad != 0) {
              pad -= 1;
          }
        } else if (node.match(/^<\w[^>]*[^\/]>.*$/)) {
          indent = 1;
        } else {
          indent = 0;
        }

        var padding = "";
        for (var i = 0; i < pad; i++) {
          padding += tab;
        }

        formatted += padding + node + "\r\n";
        pad += indent;
      });

      return formatted;
    }

    function fetchRequests() {
      $.ajax({
        method: "GET",
        url: "/baskets/{{.}}/requests?skip=" + fetchedCount,
        headers: {
          "Authorization" : getToken()
        }
      }).done(addRequests).fail(onAjaxError);
    }

    function fetchTotalCount() {
      $.ajax({
        method: "GET",
        url: "/baskets/{{.}}/requests?max=0",
        headers: {
          "Authorization" : getToken()
        }
      }).done(function(data) {
        if (data && (data.total_count != totalCount)) {
          refresh();
        }
      }).fail(onAjaxError);
    }

    function fetchResponse(method) {
      // keep in sync during page refresh
      $("#response_method").val(method);
      $.ajax({
        method: "GET",
        url: "/baskets/{{.}}/responses/" + method,
        headers: {
          "Authorization" : getToken()
        }
      }).done(function(data) {
        displayResponse(data);
      }).fail(onAjaxError);
    }

    function displayResponse(response) {
      $("#response_status").val(response.status);
      $("#response_body").val(response.body);
      $("#response_is_template").prop("checked", response.is_template);

      // headers
      $("#response_headers").html(""); // reset

      var row;
      var index = 0;
      for (header in response.headers) {
        row = $('<div class="row"></div>');
        row.append('<div class="col-md-3"><input type="input" class="form-control" id="header_name_' + index +
          '" value="' + header + '" placeholder="name"></div>');
        // multi-value headers are not supported, simply join values through comma
        row.append('<div class="col-md-7"><input type="input" class="form-control" id="header_value_' + index +
          '" value="' + response.headers[header].join(",") + '" placeholder="value"></div>');
        row.appendTo($("#response_headers"));
        index++;
      }

      // button or new header
      if (index > 0) {
        addHeaderButton(row);
      } else {
        addHeader();
      }
    }

    function addHeader() {
      $("#headers_add").remove();

      var index = $("#response_headers > div.row").length;
      var row = $('<div class="row"><div class="col-md-3"><input type="input" class="form-control" id="header_name_' + index +
        '" placeholder="name"></div><div class="col-md-7"><input type="input" class="form-control" id="header_value_' + index +
        '" placeholder="value"></div></div>');
      row.appendTo($("#response_headers"));
      addHeaderButton(row);
    }

    function addHeaderButton(row) {
      row.append('<div id="headers_add" class="col-md-1"><button id="headers_add_btn" type="button" title="Add Header" class="btn btn-success">' +
        '<span class="glyphicon glyphicon-plus-sign"></span></button></div>');
      $("#headers_add_btn").on("click", function(event) {
        addHeader();
      });
    }

    function updateResponse() {
      var method = $("#response_method").val();
      var response = {};
      response.status = parseInt($("#response_status").val());
      response.body = $("#response_body").val();
      response.is_template = $("#response_is_template").prop("checked");
      response.headers = {};
      $("#response_headers > div.row").each( function(index) {
        var name = $("#header_name_" + index).val();
        var value = $("#header_value_" + index).val();
        if (name && name.length > 0 && value && value.length > 0) {
          response.headers[name] = [ value ];
        }
      });

      $.ajax({
        method: "PUT",
        url: "/baskets/{{.}}/responses/" + method,
        dataType: "json",
        data: JSON.stringify(response),
        headers: {
          "Authorization" : getToken()
        }
      }).done(function(data) {
        alert("Response for HTTP " + method + " is updated");
      }).fail(onAjaxError);
    }

    function updateConfig() {
      if (currentConfig && (
        currentConfig.forward_url != $("#basket_forward_url").val() ||
        currentConfig.proxy_response != $("#basket_proxy_response").prop("checked") ||
        currentConfig.expand_path != $("#basket_expand_path").prop("checked") ||
        currentConfig.insecure_tls != $("#basket_insecure_tls").prop("checked") ||
        currentConfig.capacity != $("#basket_capacity").val()
      )) {
        currentConfig.forward_url = $("#basket_forward_url").val();
        currentConfig.proxy_response = $("#basket_proxy_response").prop("checked");
        currentConfig.expand_path = $("#basket_expand_path").prop("checked");
        currentConfig.insecure_tls = $("#basket_insecure_tls").prop("checked");
        currentConfig.capacity = parseInt($("#basket_capacity").val());

        $.ajax({
          method: "PUT",
          url: "/baskets/{{.}}",
          dataType: "json",
          data: JSON.stringify(currentConfig),
          headers: {
            "Authorization" : getToken()
          }
        }).done(function(data) {
          alert("Basket is reconfigured");
        }).fail(onAjaxError);
      }
    }

    function refresh() {
      $("#requests").html(""); // reset
      fetchedCount = 0;
      fetchRequests(); // fetch latest
    }

    function enableAutoRefresh(enable) {
      if (autoRefresh != enable) {
        var btn = $("#auto_refresh");
        if (enable) {
          autoRefreshId = setInterval(fetchTotalCount, 3000);
          btn.removeClass("btn-default");
          btn.addClass("btn-success");
          btn.attr("title", "Auto-Refresh is Enabled");
        } else {
          clearInterval(autoRefreshId);
          btn.removeClass("btn-success");
          btn.addClass("btn-default");
          btn.attr("title", "Auto-Refresh is Disabled");
        }
        autoRefresh = enable;
      }
    }

    function config() {
      $.ajax({
        method: "GET",
        url: "/baskets/{{.}}",
        headers: {
          "Authorization" : getToken()
        }
      }).done(function(data) {
        if (data) {
          currentConfig = data;
          $("#basket_forward_url").val(currentConfig.forward_url);
          $("#basket_proxy_response").prop("checked", currentConfig.proxy_response);
          $("#basket_expand_path").prop("checked", currentConfig.expand_path);
          $("#basket_insecure_tls").prop("checked", currentConfig.insecure_tls);
          $("#basket_capacity").val(currentConfig.capacity);
          $("#config_dialog").modal();
        }
      }).fail(onAjaxError);
    }

    function responses() {
      $("#responses_dialog").modal();
    }

    function deleteRequests() {
      $.ajax({
        method: "DELETE",
        url: "/baskets/{{.}}/requests",
        headers: {
          "Authorization" : getToken()
        }
      }).done(function(data) {
        refresh();
      }).fail(onAjaxError);
    }

    function destroyBasket() {
      $("#destroy_dialog").modal("hide");
      enableAutoRefresh(false);

      $.ajax({
        method: "DELETE",
        url: "/baskets/{{.}}",
        headers: {
          "Authorization" : getToken()
        }
      }).done(function(data) {
        localStorage.removeItem("basket_{{.}}");
        window.location.href = "/web";
      }).fail(onAjaxError);
    }

    function copyToClipboard(text) {
      var textArea = document.createElement("textarea");
      textArea.value = text;
      document.body.insertBefore(textArea, document.body.firstChild);
      textArea.focus();
      textArea.select();

      var copied;
      try {
        copied = document.execCommand("copy");
      } catch (err) {
        copied = false;
      }

      document.body.removeChild(textArea);
      return copied;
    }

    function shareBasket() {
      var token = getBasketToken();
      if (token && copyToClipboard(window.location + "?token=" + token)) {
        alert("A link to share this basket was copied to your clipboard.");
      }
    }

    function acceptSharedBasket() {
      var token = getParam("token");
      if (token) {
        var currentToken = getBasketToken();
        if (!currentToken) {
          // remember basket token
          localStorage.setItem("basket_{{.}}", token);
        } else if (currentToken !== token) {
          if (confirm("The access token for the '{{.}}' basket \n" +
              "from query parameter is different to the token that is \n" +
              "already stored in your browser.\n\n" +
              "If you trust this link choose 'OK' and existing token will be \n" +
              "replaced with the new one, otherwise choose 'Cancel'.\n\n" +
              "Do you want to replace the access token of this basket?")) {
            localStorage.setItem("basket_{{.}}", token);
          }
        }
        // remove token from location URL
        window.location.href = "/web/{{.}}";
      }
    }

    // Initialization
    $(document).ready(function() {
      $(".basket_uri").html(window.location.protocol + "//" + window.location.host + "/{{.}}");
      // dialogs
      $("#token_dialog").on("hidden.bs.modal", function (event) {
        localStorage.setItem("basket_{{.}}", $("#basket_token").val());
        fetchRequests();
      });
      $("#config_form").on("submit", function(event) {
        $("#config_dialog").modal("hide");
        updateConfig();
        event.preventDefault();
      });
      // buttons
      $("#refresh").on("click", function(event) {
        refresh();
      });
      $("#auto_refresh").on("click", function(event) {
        enableAutoRefresh(!autoRefresh);
      });
      $("#config").on("click", function(event) {
        config();
      });
      $("#responses").on("click", function(event) {
        responses();
      });
      $("#share").on("click", function(event) {
        shareBasket();
      });
      $("#delete").on("click", function(event) {
        deleteRequests();
      });
      $("#destroy").on("click", function(event) {
        $("#destroy_dialog").modal();
      });
      $("#destroy_confirmed").on("click", function(event) {
        destroyBasket();
      });
      $("#fetch_more").on("click", function(event) {
        fetchRequests();
      });
      $("#response_method").on("change", function(event) {
        fetchResponse($(this).val());
      });
      $("#update_response").on("click", function(event) {
        updateResponse();
      });
      // shared basket link
      acceptSharedBasket();
      // hide share basket button
      if (!getBasketToken()) {
        $("#share").hide();
      }
      // autorefresh and initial fetch
      if (getToken()) {
        enableAutoRefresh(true);
      }
      fetchRequests();
      fetchResponse("GET");
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
          <button id="refresh" type="button" title="Refresh" class="btn btn-default">
            <span class="glyphicon glyphicon-refresh"></span>
          </button>
          <button id="auto_refresh" type="button" title="Auto Refresh" class="btn btn-default">
            <span class="glyphicon glyphicon-repeat"></span>
          </button>
          &nbsp;
          <button id="config" type="button" title="Settings" class="btn btn-default">
            <span class="glyphicon glyphicon-cog"></span>
          </button>
          <button id="responses" type="button" title="Responses" class="btn btn-default">
            <!-- glyphicon-tags | glyphicon-transfer -->
            <span class="glyphicon glyphicon-transfer"></span>
          </button>
          &nbsp;
          <button id="share" type="button" title="Share Basket" class="btn btn-default">
            <span class="glyphicon glyphicon-link"></span>
          </button>
          &nbsp;
          <button id="delete" type="button" title="Delete Requests" class="btn btn-warning">
            <span class="glyphicon glyphicon-fire"></span>
          </button>
          <button id="destroy" type="button" title="Destroy Basket" class="btn btn-danger">
            <span class="glyphicon glyphicon-trash"></span>
          </button>
        </form>
      </div>
    </div>
  </nav>

  <!-- Login dialog -->
  <form>
  <div class="modal fade" id="token_dialog" tabindex="-1">
    <div class="modal-dialog">
      <div class="modal-content panel-warning">
        <div class="modal-header panel-heading">
          <h4 class="modal-title">Token required</h4>
        </div>
        <div class="modal-body">
          <p>You are not authorized to access this basket. Please enter this basket token or choose another basket.</p>
          <div class="form-group">
            <label for="basket_token" class="control-label">Token:</label>
            <input type="password" class="form-control" id="basket_token">
          </div>
        </div>
        <div class="modal-footer">
          <a href="/web" class="btn btn-default">Back to list of Baskets</a>
          <button type="submit" class="btn btn-success" data-dismiss="modal">Authorize</button>
        </div>
      </div>
    </div>
  </div>
  </form>

  <!-- Config dialog -->
  <form id="config_form">
  <div class="modal fade" id="config_dialog" tabindex="-1">
    <div class="modal-dialog">
      <div class="modal-content panel-default">
        <div class="modal-header panel-heading">
          <button type="button" class="close" data-dismiss="modal">&times;</button>
          <h4 class="modal-title" id="config_dialog_label">Configuration Settings</h4>
        </div>
        <div class="modal-body">
          <div class="form-group">
            <label for="basket_forward_url" class="control-label">Forward URL:</label>
            <input type="input" class="form-control" id="basket_forward_url">
          </div>
          <div class="checkbox">
            <label><input type="checkbox" id="basket_insecure_tls">
              <abbr class="text-danger" title="Warning! Enabling this feature will bypass certificate verification">Insecure TLS</abbr>
              only affects forwarding to URLs like <kbd>https://...</kbd>
            </label>
          </div>
          <div class="checkbox">
            <label><input type="checkbox" id="basket_proxy_response">
              <abbr title="Proxies the response from the forward URL back to the client">Proxy Response</abbr>
            </label>
          </div>
          <div class="checkbox">
            <label><input type="checkbox" id="basket_expand_path"> Expand Forward Path</label>
          </div>
          <div class="form-group">
            <label for="basket_capacity" class="control-label">Basket Capacity:</label>
            <input type="input" class="form-control" id="basket_capacity">
          </div>
        </div>
        <div class="modal-footer">
          <button type="button" class="btn btn-default" data-dismiss="modal">Cancel</button>
          <button type="submit" class="btn btn-primary">Apply</button>
        </div>
      </div>
    </div>
  </div>
  </form>

  <!-- Responses dialog -->
  <form id="response_form">
  <div class="modal fade" id="responses_dialog" tabindex="-1">
    <div class="modal-dialog">
      <div class="modal-content panel-default">
        <div class="modal-header panel-heading">
          <button type="button" class="close" data-dismiss="modal">&times;</button>
          <h4 class="modal-title" id="config_dialog_label">Basket Responses</h4>
        </div>
        <div class="modal-body">
          <div class="form-group">
            <label for="response_method" class="control-label">HTTP method:</label>
            <select class="form-control" id="response_method">
              <option>GET</option>
              <option>HEAD</option>
              <option>POST</option>
              <option>PUT</option>
              <option>PATCH</option>
              <option>DELETE</option>
              <option>CONNECT</option>
              <option>OPTIONS</option>
              <option>TRACE</option>
            </select>
          </div>
          <div class="form-group">
            <label for="response_status" class="control-label">HTTP status:</label>
            <input type="input" class="form-control" id="response_status">
          </div>
          <div class="form-group">
            <label class="control-label">HTTP headers:</label>
            <div id="response_headers">
              <!-- response headers -->
            </div>
          </div>
          <div class="form-group">
            <label for="response_body" class="control-label">Response Body:</label>
            <textarea class="form-control" id="response_body" rows="10"></textarea>
          </div>
          <div class="checkbox">
            <label><input type="checkbox" id="response_is_template"> Process body as HTML template</label>
          </div>
        </div>
        <div class="modal-footer">
          <button type="button" class="btn btn-default" data-dismiss="modal">Close</button>
          <button type="button" class="btn btn-primary" id="update_response">Apply</button>
        </div>
      </div>
    </div>
  </div>
  </form>

  <!-- Destroy dialog -->
  <div class="modal fade" id="destroy_dialog" tabindex="-1">
    <div class="modal-dialog">
      <div class="modal-content panel-danger">
        <div class="modal-header panel-heading">
          <button type="button" class="close" data-dismiss="modal">&times;</button>
          <h4 class="modal-title">Destroy This Basket</h4>
        </div>
        <div class="modal-body">
          <p>Are you sure you want to <strong>permanently destroy</strong> this basket and delete all collected requests?</p>
        </div>
        <div class="modal-footer">
          <button type="button" class="btn btn-default" data-dismiss="modal">Cancel</button>
          <button type="button" class="btn btn-danger" id="destroy_confirmed">Destroy</button>
        </div>
      </div>
    </div>
  </div>

  <!-- Error message -->
  <div class="modal fade" id="error_message" tabindex="-1">
    <div class="modal-dialog">
      <div class="modal-content panel-danger">
        <div class="modal-header panel-heading">
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

  <div class="container">
    <div class="row">
      <div class="col-md-8">
        <h1>Basket: {{.}}</h1>
        <p id="requests_link" class="hide">Requests are collected at <kbd class="basket_uri"></kbd></p>
      </div>
      <div class="col-md-3 col-md-offset-1">
        <h4><abbr title="Current requests count (Total count)">Requests</abbr>: <span id="requests_count"></span></h4>
      </div>
    </div>
    <hr/>
    <div id="requests">
    </div>
    <div id="more" class="hide">
      <button id="fetch_more" type="button" class="btn btn-default">
        More <span id="more_count" class="badge"></span>
      </button>
    </div>

    <!-- Empty basket -->
    <div class="jumbotron text-center hide" id="empty_basket">
      <h1>Empty basket!</h1>
      <p>This basket is empty, send requests to <kbd class="basket_uri"></kbd> and they will appear here.</p>
    </div>
  </div>

  <p>&nbsp;</p>
</body>
</html>`
)
