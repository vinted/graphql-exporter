package graphql

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"text/template"
	"time"

	"github.com/vinted/graphql-exporter/internal/config"
)

type GraphqlRequest struct {
	Query string `json:"query"`
}

var funcMap = template.FuncMap{
	"NOW": func(t string) (string, error) {
		d, err := time.ParseDuration(t)
		return time.Now().UTC().Add(d).Format(time.RFC3339), err
	},
}

func GraphqlQuery(ctx context.Context, query string) ([]byte, error) {
	tpl, err := template.New("query").Funcs(funcMap).Parse(query)
	if err != nil {
		return nil, fmt.Errorf("%s", err)
	}

	var templateBuffer bytes.Buffer
	err = tpl.Execute(&templateBuffer, nil)
	if err != nil {
		return nil, fmt.Errorf("template error %s", err)
	}

	data := GraphqlRequest{
		Query: templateBuffer.String(),
	}
	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("JSON marshal error: %s", err)
	}

	u, err := url.ParseRequestURI(config.Config.GraphqlURL)
	if err != nil {
		return nil, fmt.Errorf("error parsing URL: %s", err)
	}

	urlStr := u.String()
	client := &http.Client{}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, urlStr, bytes.NewReader(jsonData))
	if err != nil {
		return nil, fmt.Errorf("HTTP request error: %s", err)
	}

	req.Header.Add("Authorization", config.Config.GraphqlAPIToken)
	req.Header.Add("Content-Type", "application/json")
	r, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	if r.StatusCode != 200 {
		return nil, fmt.Errorf(r.Status)
	}

	defer r.Body.Close()
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}
	return body, nil
}
