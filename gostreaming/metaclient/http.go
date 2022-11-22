package metaclient

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"time"

	"github.com/GDVFox/gostreaming/meta_node/api/actions"
	"github.com/GDVFox/gostreaming/meta_node/api/schemas"
	"github.com/GDVFox/gostreaming/meta_node/planner"
	"github.com/GDVFox/gostreaming/util/httplib"
)

var (
	metaScheme       = "http"
	schemasListPath  = "/v1/schemas"
	getSchemePath    = "/v1/schemas/"
	createSchemePath = "/v1/schemas"
	deleteSchemePath = "/v1/schemas/"
	runSchemePath    = "/v1/schemas/%s/run"
	stopSchemePath   = "/v1/schemas/%s/stop"
	actionsListPath  = "/v1/actions"
	getActionPath    = "/v1/actions/"
	createActionPath = "/v1/actions"
	deleteActionPath = "/v1/actions/"
)

var (
	// MetaNode клиент для доступа.
	MetaNode *MetaNodeClient
	// MetaNodeAddress адрес MetaNode.
	MetaNodeAddress string
)

// MetaNodeClientConfig набор настроек для MetaNodeClient.
type MetaNodeClientConfig struct {
	Address string
}

// MetaNodeClient клиент для подключения к meta_node
type MetaNodeClient struct {
	client *http.Client
	cfg    *MetaNodeClientConfig
}

// OpenMetaNodeClient открывает meta_node client.
func OpenMetaNodeClient(cfg *MetaNodeClientConfig) {
	MetaNode = NewMetaNodeClient(cfg)
	MetaNodeAddress = cfg.Address
}

// NewMetaNodeClient возвращает новый MetaNodeClient
func NewMetaNodeClient(cfg *MetaNodeClientConfig) *MetaNodeClient {
	return &MetaNodeClient{
		client: &http.Client{Timeout: 1 * time.Minute},
		cfg:    cfg,
	}
}

// GetSchemasList возвращает список схем.
func (c *MetaNodeClient) GetSchemasList() (*schemas.SchemeList, error) {
	metaURL := url.URL{
		Scheme: metaScheme,
		Host:   c.cfg.Address,
		Path:   schemasListPath,
	}

	schemeList := &schemas.SchemeList{}
	if err := c.get(metaURL.String(), schemeList); err != nil {
		return nil, err
	}

	return schemeList, nil
}

// GetScheme возвращает описание схемы.
func (c *MetaNodeClient) GetScheme(schemeName string) (*planner.Scheme, error) {
	metaURL := url.URL{
		Scheme: metaScheme,
		Host:   c.cfg.Address,
		Path:   getSchemePath + schemeName,
	}

	scheme := &planner.Scheme{}
	if err := c.get(metaURL.String(), scheme); err != nil {
		return nil, err
	}
	return scheme, nil
}

// CreateScheme создает новую схему.
func (c *MetaNodeClient) CreateScheme(scheme *planner.Scheme) error {
	metaURL := url.URL{
		Scheme: metaScheme,
		Host:   c.cfg.Address,
		Path:   createSchemePath,
	}

	return c.post(metaURL.String(), scheme)
}

// DeleteScheme удаление заданной схеме.
func (c *MetaNodeClient) DeleteScheme(schemeName string) error {
	metaURL := url.URL{
		Scheme: metaScheme,
		Host:   c.cfg.Address,
		Path:   deleteSchemePath + schemeName,
	}

	return c.delete(metaURL.String())
}

// RunScheme запускает схему в работу.
func (c *MetaNodeClient) RunScheme(schemeName string) error {
	metaURL := url.URL{
		Scheme: metaScheme,
		Host:   c.cfg.Address,
		Path:   fmt.Sprintf(runSchemePath, schemeName),
	}

	return c.put(metaURL.String())
}

// StopScheme останавливает работу схемы.
func (c *MetaNodeClient) StopScheme(schemeName string) error {
	metaURL := url.URL{
		Scheme: metaScheme,
		Host:   c.cfg.Address,
		Path:   fmt.Sprintf(stopSchemePath, schemeName),
	}

	return c.put(metaURL.String())
}

// GetActionsList возвращает список загруженных действий.
func (c *MetaNodeClient) GetActionsList() (*actions.ActionList, error) {
	metaURL := url.URL{
		Scheme: metaScheme,
		Host:   c.cfg.Address,
		Path:   actionsListPath,
	}

	actionsList := &actions.ActionList{}
	if err := c.get(metaURL.String(), actionsList); err != nil {
		return nil, err
	}
	return actionsList, nil
}

// GetAction возвращает бинарный файл действия.
func (c *MetaNodeClient) GetAction(actionName string) ([]byte, error) {
	metaURL := url.URL{
		Scheme: metaScheme,
		Host:   c.cfg.Address,
		Path:   getActionPath + actionName,
	}

	return c.getBinary(metaURL.String())
}

// CreateAction создает новое действие.
func (c *MetaNodeClient) CreateAction(actionName string, actionBinary []byte) error {
	metaURL := url.URL{
		Scheme: metaScheme,
		Host:   c.cfg.Address,
		Path:   createActionPath,
	}

	var buff bytes.Buffer
	w := multipart.NewWriter(&buff)

	if err := w.WriteField("name", actionName); err != nil {
		return err
	}
	fw, err := w.CreateFormFile("action", actionName)
	if err != nil {
		return err
	}
	if _, err := fw.Write(actionBinary); err != nil {
		return err
	}
	if err := w.Close(); err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, metaURL.String(), &buff)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", w.FormDataContentType())

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusNoContent {
		return c.handleError(resp.Body)
	}
	return nil
}

// DeleteAction удаление заданного действия.
func (c *MetaNodeClient) DeleteAction(actionName string) error {
	metaURL := url.URL{
		Scheme: metaScheme,
		Host:   c.cfg.Address,
		Path:   deleteActionPath + actionName,
	}

	return c.delete(metaURL.String())
}

func (c *MetaNodeClient) get(url string, respData interface{}) error {
	resp, err := c.client.Get(url)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return c.handleError(resp.Body)
	}
	return json.NewDecoder(resp.Body).Decode(respData)
}

func (c *MetaNodeClient) getBinary(url string) ([]byte, error) {
	resp, err := c.client.Get(url)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, c.handleError(resp.Body)
	}
	return io.ReadAll(resp.Body)
}

func (c *MetaNodeClient) post(url string, body interface{}) error {
	reqBodyEncoded, err := json.Marshal(body)
	if err != nil {
		return err
	}
	return c.doNoContent(http.MethodPost, url, reqBodyEncoded)
}

func (c *MetaNodeClient) put(url string) error {
	return c.doNoContent(http.MethodPut, url, nil)
}

func (c *MetaNodeClient) delete(url string) error {
	return c.doNoContent(http.MethodDelete, url, nil)
}

func (c *MetaNodeClient) doNoContent(method string, url string, body []byte) error {
	req, err := http.NewRequest(method, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusNoContent {
		return c.handleError(resp.Body)
	}
	return nil
}

func (c *MetaNodeClient) handleError(r io.Reader) error {
	metaError := &httplib.ErrorBody{}
	if err := json.NewDecoder(r).Decode(metaError); err != nil {
		return fmt.Errorf("can not decode error response: %w", err)
	}
	return fmt.Errorf("%s", metaError.Message)
}
