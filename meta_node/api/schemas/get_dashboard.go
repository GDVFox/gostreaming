package schemas

import (
	"bytes"
	"html/template"
	"net/http"
	"path"

	"github.com/gorilla/mux"

	"github.com/GDVFox/gostreaming/meta_node/api/common"
	"github.com/GDVFox/gostreaming/meta_node/config"
	"github.com/GDVFox/gostreaming/util/httplib"
)

var (
	dashboardTemplateName = "dashboard.tmpl"
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

	dashboardTemplate, err := template.ParseFiles(path.Join(config.Conf.StaticPath, dashboardTemplateName))
	if err != nil {
		return nil, err
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
