package schemas

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/goccy/go-graphviz"
	"github.com/goccy/go-graphviz/cgraph"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"

	"github.com/GDVFox/gostreaming/meta_node/api/common"
	"github.com/GDVFox/gostreaming/meta_node/watcher"
	"github.com/GDVFox/gostreaming/util"
	"github.com/GDVFox/gostreaming/util/httplib"
)

const (
	writeWait = 10 * time.Second
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // мы уже прошли слой CORS
	},
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

type socketMessage struct {
	Code string `json:"code"`
	Body string `json:"body"`
}

func newSocketMessage(c string, body string) []byte {
	sockMsg := &socketMessage{
		Code: c,
		Body: body,
	}
	data, err := json.Marshal(sockMsg)
	if err != nil {
		return nil
	}
	return data
}

// SendDashboard периодически отправляет данные по websocket.
func SendDashboard(w http.ResponseWriter, r *http.Request) error {
	logger := r.Context().Value(httplib.RequestLogger).(*util.Logger)

	vars := mux.Vars(r)
	schemeName := vars["scheme_name"]
	if schemeName == "" {
		resp := httplib.NewBadRequestResponse(httplib.NewErrorBody(common.BadNameErrorCode, "scheme_name must be not empty"))
		return resp.WriteTo(w)
	}

	sendPeriodStr := r.FormValue("send_period")
	if sendPeriodStr == "" {
		resp := httplib.NewBadRequestResponse(httplib.NewErrorBody(common.BadPeriodErrorCode, "send_period must be not empty"))
		return resp.WriteTo(w)
	}
	sendPeriod, err := time.ParseDuration(sendPeriodStr)
	if err != nil {
		resp := httplib.NewBadRequestResponse(httplib.NewErrorBody(common.BadPeriodErrorCode, err.Error()))
		return resp.WriteTo(w)
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return err
	}

	go runGraphLoop(conn, schemeName, sendPeriod, logger.WithName("graph loop"))
	return nil
}

func runGraphLoop(conn *websocket.Conn, schemeName string, sendPeriod time.Duration, l *util.Logger) {
	defer conn.Close()
	defer conn.WriteMessage(websocket.CloseMessage, []byte{})

	ticker := time.NewTicker(sendPeriod)
	defer ticker.Stop()
	for {
		<-ticker.C

		telemetry, err := watcher.Watcher.GetPlanTelemetry(schemeName)
		if err != nil {
			l.Warnf("can not get telemetry for %s: %s", schemeName, err)

			jsonErr := newSocketMessage(common.BadSchemeErrorCode, err.Error())
			if err := conn.WriteMessage(websocket.TextMessage, jsonErr); err != nil {
				l.Warnf("can not send error message for %s: %s", schemeName, err)
			}
			return
		}

		graphImg, err := buildGraph(telemetry.Nodes)
		if err != nil {
			l.Warnf("can not generate graph image for %s: %s", schemeName, err)

			jsonErr := newSocketMessage(common.RenderGraphErrorCode, err.Error())
			if err := conn.WriteMessage(websocket.TextMessage, jsonErr); err != nil {
				l.Warnf("can not send error message for %s: %s", schemeName, err)
			}
			return
		}

		graphImgEncoded := base64.StdEncoding.EncodeToString(graphImg)
		conn.SetWriteDeadline(time.Now().Add(writeWait))
		jsonMsg := newSocketMessage("image", graphImgEncoded)
		if err := conn.WriteMessage(websocket.TextMessage, jsonMsg); err != nil {
			l.Warnf("can not write image for %s: %s", schemeName, err)
			return
		}
	}
}

func buildGraph(nodes []*watcher.NodeTelemetry) ([]byte, error) {
	g := graphviz.New()
	graph, err := g.Graph(graphviz.Directed)
	if err != nil {
		return nil, err
	}
	graph.SetRankDir(cgraph.LRRank)

	graphvizNodes := make(map[string]*cgraph.Node)
	for _, node := range nodes {
		graphvizNode, err := graph.CreateNode(node.Name)
		if err != nil {
			return nil, err
		}

		graphvizNode.SetShape(cgraph.RectangleShape)
		graphvizNode.SetLabel(buildLabel(node))

		if node.IsRunning {
			graphvizNode.SetColor("green")
		} else {
			graphvizNode.SetColor("red")
		}

		graphvizNodes[node.Name] = graphvizNode
	}

	for _, node := range nodes {
		to := graphvizNodes[node.Name]
		for _, in := range node.PrevName {
			from := graphvizNodes[in]
			_, err := graph.CreateEdge(in+"-"+node.Name, from, to)
			if err != nil {
				return nil, err
			}
		}
	}

	var buf bytes.Buffer
	if err := g.Render(graph, graphviz.SVG, &buf); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func buildLabel(node *watcher.NodeTelemetry) string {
	b := &strings.Builder{}

	b.WriteString("Name: ")
	b.WriteString(node.Name)
	b.WriteString("\\l") // Выравнивает по левому краю

	b.WriteString("Action: ")
	b.WriteString(node.Action)
	b.WriteString("\\l")

	b.WriteString("Address: ")
	b.WriteString(node.Address)
	b.WriteString("\\l")

	b.WriteString("OldestOutput: ")
	b.WriteString(strconv.Itoa(int(node.OldestOutput)))
	b.WriteString("\\l")

	return b.String()
}
