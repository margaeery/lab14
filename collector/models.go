package main

type Country struct {
	ID   int64  `json:"id"`
	Code string `json:"code"`
	Name string `json:"name"`
	URL  string `json:"url"`
}

type League struct {
	ID      int64    `json:"id"`
	Name    string   `json:"name"`
	URL     string   `json:"url"`
	Country *Country `json:"country"`
}

type apiResponse[T any] struct {
	Status     string `json:"status"`
	Count      *int   `json:"count"`
	Data       []T    `json:"data"`
	Message    string `json:"message"`
	Offset     *int   `json:"offset"`
	TotalCount *int   `json:"TotalCount"`
	TraceID    string `json:"traceId"`
}