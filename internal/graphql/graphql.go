package graphql

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"text/template"
	"time"

	"log/slog"

	"github.com/vinted/graphql-exporter/internal/config"
)

var funcMap = template.FuncMap{
	"NOW": func(t string) (string, error) {
		d, err := time.ParseDuration(t)
		return time.Now().UTC().Add(d).Format(time.RFC3339), err
	},
}

func GraphqlQuery(ctx context.Context, query string, previousTimestapm time.Time) ([]byte, error) {
	params := url.Values{}
	tpl, err := template.New("query").Funcs(funcMap).Parse(query)
	if err != nil {
		return nil, fmt.Errorf("%s", err)
	}

	var templateBuffer bytes.Buffer
	err = tpl.Execute(&templateBuffer, struct{ PreviousRun, Now string }{
		PreviousRun: previousTimestapm.Format(time.RFC3339),
		Now:         time.Now().Format(time.RFC3339),
	})
	if err != nil {
		return nil, fmt.Errorf("template error %s", err)
	}
	q := templateBuffer.String()
	params.Add("query", q)
	slog.Debug(fmt.Sprintf("run query: %s", q))
	u, err := url.ParseRequestURI(config.Config.GraphqlURL)
	if err != nil {
		return nil, fmt.Errorf("error parsing URL: %s", err)
	}

	urlStr := u.String()
	client := &http.Client{}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, urlStr, strings.NewReader(params.Encode()))
	if err != nil {
		return nil, fmt.Errorf("HTTP request error: %s", err)
	}
	header := config.Config.GraphqlCustomHeader
	if header == "" {
		header = "Authorization"
	}
	req.Header.Add(header, config.Config.GraphqlAPIToken)
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
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
