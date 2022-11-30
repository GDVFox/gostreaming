package schemas

import (
	"bytes"
	"html/template"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/GDVFox/gostreaming/meta_node/api/common"
	"github.com/GDVFox/gostreaming/meta_node/config"
	"github.com/GDVFox/gostreaming/util/httplib"
)

var (
	dashboardTemplate = template.Must(template.New("dashboard").Parse(templateString))
)

// GetDashboard получает описание схемы.
func GetDashboard(r *http.Request) (*httplib.Response, error) {
	vars := mux.Vars(r)
	schemeName := vars["scheme_name"]
	if schemeName == "" {
		return httplib.NewBadRequestResponse(httplib.NewErrorBody(common.BadNameErrorCode, "scheme_name must be not empty")), nil
	}
	sendPeriodStr := r.FormValue("send_period")
	if sendPeriodStr == "" {
		return httplib.NewBadRequestResponse(httplib.NewErrorBody(common.BadPeriodErrorCode, "send_period must be not empty")), nil
	}

	data := map[string]interface{}{
		"Name":   schemeName,
		"Host":   config.Conf.HTTP.Host,
		"Port":   config.Conf.HTTP.Port,
		"Period": sendPeriodStr,
	}

	var buf bytes.Buffer
	if err := dashboardTemplate.Execute(&buf, data); err != nil {
		return nil, err
	}

	return httplib.NewOKResponse(buf.Bytes(), httplib.ContentTypeHTML), nil
}

var (
	templateString = `
<!doctype html>
	<html>
	<head>
	  <meta charset="utf-8">
	  <title>Dashboard: {{ .Name }}</title>
	</head>
	<body>
	  <h1>Dashboard: {{ .Name }}</h1>
	  <h2 id="error_msg" style="visibility: hidden"></h2>
	  <img id="nodes_structure_img" src="">
	</body>
	<script>
	  var graphSocket = new WebSocket("ws://{{ .Host }}:{{ .Port }}/v1/schemas/{{ .Name }}/send_dashboard?send_period={{ .Period }}");
	  graphSocket.onmessage = function (event) {
		var msg = JSON.parse(event.data);
		if (msg["code"] != "image") {
		  document.getElementById("nodes_structure_img").style.visibility = "hidden";
		  document.getElementById("error_msg").innerHTML = msg["body"];
		  document.getElementById("error_msg").style.visibility = "visible";
		} else {
		  document.getElementById("nodes_structure_img").src = "data:image/svg+xml;base64," + msg["body"];
		}
	  }
	</script>
	</html>
`
)
