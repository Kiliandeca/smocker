package services

import (
	"fmt"
	"net/url"
	"sort"
	"time"

	"github.com/Thiht/smocker/server/types"
)

const (
	ClientHost  = "Client"
	SmockerHost = "Smocker"

	requestType    = "request"
	responseType   = "response"
	processingType = "processing"
)

type Graph interface {
	Generate(cfg types.GraphConfig, history types.History, mocks types.Mocks) types.GraphHistory
}

type graph struct {
}

func NewGraph() Graph {
	return &graph{}
}

func (g *graph) Generate(cfg types.GraphConfig, history types.History, mocks types.Mocks) types.GraphHistory {

	endpointCpt := 0
	endpoints := map[string]string{}

	mocksByID := map[string]*types.Mock{}
	for _, mock := range mocks {
		mocksByID[mock.State.ID] = mock
	}

	graphHistory := types.GraphHistory{}
	for _, entry := range history {
		from := ClientHost
		if src := entry.Request.Headers.Get(cfg.SrcHeader); src != "" {
			from = src
		}
		to := SmockerHost
		if dest := entry.Request.Headers.Get(cfg.DestHeader); dest != "" {
			to = dest
		}

		params := entry.Request.QueryParams.Encode()
		if decoded, err := url.QueryUnescape(params); err == nil {
			params = decoded
		}
		if params != "" {
			params = "?" + params
		}

		requestMessage := entry.Request.Method + " " + entry.Request.Path + params
		graphHistory = append(graphHistory, types.GraphEntry{
			Type:    requestType,
			Message: requestMessage,
			From:    from,
			To:      SmockerHost,
			Date:    entry.Request.Date,
		})

		graphHistory = append(graphHistory, types.GraphEntry{
			Type:    responseType,
			Message: fmt.Sprintf("%d", entry.Response.Status),
			From:    SmockerHost,
			To:      from,
			Date:    entry.Response.Date,
		})

		if entry.Context.MockID != "" {
			if mocksByID[entry.Context.MockID].Proxy != nil {
				host := mocksByID[entry.Context.MockID].Proxy.Host
				u, err := url.Parse(host)
				if err == nil {
					host = u.Host
				}
				if to == SmockerHost {
					if endpoint := endpoints[host]; endpoint == "" {
						endpointCpt++
						endpoints[host] = fmt.Sprintf("Endpoint%d", endpointCpt)
					}
					to = endpoints[host]
				}

				graphHistory = append(graphHistory, types.GraphEntry{
					Type:    requestType,
					Message: requestMessage,
					From:    SmockerHost,
					To:      to,
					Date:    entry.Request.Date.Add(1 * time.Nanosecond),
				})

				graphHistory = append(graphHistory, types.GraphEntry{
					Type:    responseType,
					Message: fmt.Sprintf("%d", entry.Response.Status),
					From:    to,
					To:      SmockerHost,
					Date:    entry.Response.Date.Add(-1 * time.Nanosecond),
				})
			} else {
				graphHistory = append(graphHistory, types.GraphEntry{
					Type:    processingType,
					Message: "use response mock",
					From:    SmockerHost,
					To:      SmockerHost,
					Date:    entry.Response.Date.Add(-1 * time.Nanosecond),
				})
			}
		}

	}
	sort.Sort(graphHistory)
	return graphHistory
}
