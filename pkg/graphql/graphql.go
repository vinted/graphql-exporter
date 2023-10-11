package graphql

import (
	"fmt"
	"github.com/vinted/graphql-exporter/pkg/config"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"
)

func GraphqlQuery(query string) ([]byte, error) {
	params := url.Values{}
	params.Add("query", query)
	u, err := url.ParseRequestURI(config.Config.GraphqlURL)
	if err != nil {
		log.Printf("Error parsing URL: %s\n", err)
	}
	urlStr := u.String()
	client := &http.Client{}
	req, err := http.NewRequest(http.MethodPost, urlStr, strings.NewReader(params.Encode()))
	if err != nil {
		log.Printf("HTTP requrest error: %s\n", err)
	}
	req.Header.Add("Authorization", config.Config.GraphqlAPIToken)
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	r, err := client.Do(req)
	if r.StatusCode != 200 {
		return nil, fmt.Errorf(r.Status)
	}
	defer r.Body.Close()
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}
	return body, nil
}
