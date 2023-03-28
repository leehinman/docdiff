package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/esapi"
	"github.com/sergi/go-diff/diffmatchpatch"
	"github.com/spf13/pflag"
)

var fieldsToDrop = []string{
	"_id",
	"_source.@timestamp",
	"_source.agent.ephemeral_id",
	"_source.agent.id",
	"_source.elastic_agent.id",
}

type config struct {
	esAddr       string
	apiKey       string
	index        string
	uniqueField  string
	uniqueValue  string
	ignoreFields []string
}

type ESResponse struct {
	Hits HitsOuter
}

type Total struct {
	Relation string
	Value    int
}

type HitsOuter struct {
	Hits     []mapstr.M
	MaxScore int
	Total    Total
}

func main() {
	c := flagsToConfig()
	esCfg := elasticsearch.Config{
		Addresses: []string{c.esAddr},
		APIKey:    c.apiKey,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}
	es, err := elasticsearch.NewClient(esCfg)
	if err != nil {
		log.Fatalf("error creating client: %s", err)
	}

	req := esapi.SearchRequest{
		Index: []string{c.index},
		Query: fmt.Sprintf("%s: \"%s\"", c.uniqueField, c.uniqueValue),
	}
	res, err := req.Do(context.Background(), es)
	if err != nil {
		log.Fatalf("error doing search request: %s", err)
	}
	defer res.Body.Close()
	if res.IsError() {
		log.Fatalf("response error: %s", res.String())
	}
	var esRes ESResponse
	if err := json.NewDecoder(res.Body).Decode(&esRes); err != nil {
		log.Fatalf("error parsing response body: %s", err)
	}

	if esRes.Hits.Total.Value != 2 {
		log.Fatalf("expected 2 matches got %d", esRes.Hits.Total.Value)
	}

	doc0 := removeIgnoredFields(esRes.Hits.Hits[0], c.ignoreFields)
	doc1 := removeIgnoredFields(esRes.Hits.Hits[1], c.ignoreFields)
	dmp := diffmatchpatch.New()

	diffs := dmp.DiffMain(doc0, doc1, false)

	if len(diffs) != 1 {
		fmt.Println(dmp.DiffPrettyText(diffs))
	}
}

func flagsToConfig() config {
	c := config{}
	pflag.StringVar(&c.esAddr, "addr", "127.0.0.1:9200", "elasticsearch address")
	pflag.StringVar(&c.apiKey, "apikey", "", "ApiKey to connect to elasticsearch")
	pflag.StringVar(&c.index, "index", "", "index to search in")
	pflag.StringVar(&c.uniqueField, "field", "message", "field name to search on")
	pflag.StringVar(&c.uniqueValue, "value", "", "value that must be the same for field")
	pflag.StringSliceVar(&c.ignoreFields, "ignore", nil, "fields to ignore")
	pflag.Parse()
	return c
}

func removeIgnoredFields(m mapstr.M, ignoredFields []string) string {
	fDoc := m.Flatten()
	for _, key := range fieldsToDrop {
		_ = fDoc.Delete(key)
	}
	for _, key := range ignoredFields {
		_ = fDoc.Delete(key)
	}
	return fDoc.StringToPrint()
}
