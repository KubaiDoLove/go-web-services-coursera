package main

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"
)

const (
	InvalidToken                = "InvalidToken"
	TimeoutErrorQuery           = "TimeoutErrorQuery"
	InternalErrorQuery          = "InternalErrorQuery"
	BadRequestErrorQuery        = "BadRequestErrorQuery"
	BadRequestUnknownErrorQuery = "BadRequestUnknownErrorQuery"
	InvalidJsonErrorQuery       = "InvalidJsonErrorQuery"
)

type UserRow struct {
	Id     int    `xml:"id"`
	Name   string `xml:"first_name"`
	Age    int    `xml:"age"`
	About  string `xml:"about"`
	Gender string `xml:"gender"`
}

type Users struct {
	List []UserRow `xml:"row"`
}

type TestCaseWithError struct {
	Request       SearchRequest
	URL           string
	AccessToken   string
	ErrorExact    string
	ErrorContains string
}

type TestCase struct {
	Request SearchRequest
}

func SearchServer(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.FormValue("query") == TimeoutErrorQuery: // timeout error
		time.Sleep(time.Second * 2) //
	case r.Header.Get("AccessToken") == InvalidToken: // 401 error
		w.WriteHeader(http.StatusUnauthorized)
		return
	case r.FormValue("query") == InternalErrorQuery: // 500 error
		w.WriteHeader(http.StatusInternalServerError)
		return
	case r.FormValue("query") == BadRequestErrorQuery: // 400 Error
		w.WriteHeader(http.StatusBadRequest)
		return
	case r.FormValue("query") == BadRequestUnknownErrorQuery: // 400 with Unknown Error
		resp, err := json.Marshal(SearchErrorResponse{"UnknownError"})
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusBadRequest)
		w.Write(resp)
		return
	case r.FormValue("query") == InvalidJsonErrorQuery: // invalid Json
		w.Write([]byte("invalid_json"))
		return
	}

	orderField := r.FormValue("order_field")
	if orderField == "" {
		orderField = "Name"
	}
	if orderField != "Id" && orderField != "Age" && orderField != "Name" {
		resp, err := json.Marshal(SearchErrorResponse{"ErrorBadOrderField"})
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusBadRequest)
		w.Write(resp)
		return
	}

	xmlFile, err := os.Open("dataset.xml")
	if err != nil {
		fmt.Println("cant open file:", err)
		return
	}
	defer xmlFile.Close()

	var data Users
	byteValue, err := ioutil.ReadAll(xmlFile)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	xml.Unmarshal(byteValue, &data)

	offset, err := strconv.Atoi(r.FormValue("offset"))
	if err != nil {
		fmt.Println("cant convert offset to int: ", err)
		return
	}
	limit, err := strconv.Atoi(r.FormValue("limit"))
	if err != nil {
		fmt.Println("cant convert limit to int: ", err)
		return
	}

	resp, err := json.Marshal(data.List[offset:limit])
	if err != nil {
		fmt.Println("cant pack result json:", err)
		return
	}

	w.Write(resp)
}

func TestFindUsersWithErrors(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(SearchServer))
	defer server.Close()

	testCases := []TestCaseWithError{
		{
			Request:    SearchRequest{Limit: -1},
			ErrorExact: "limit must be > 0",
		},
		{
			Request:    SearchRequest{Offset: -1},
			ErrorExact: "offset must be > 0",
		},
		{
			URL:           "http://",
			ErrorContains: "unknown error",
		},
		{
			Request:       SearchRequest{Query: TimeoutErrorQuery},
			ErrorContains: "timeout for",
		},
		{
			AccessToken: InvalidToken,
			ErrorExact:  "Bad AccessToken",
		},
		{
			Request:    SearchRequest{Query: InternalErrorQuery},
			ErrorExact: "SearchServer fatal error",
		},
		{
			Request:       SearchRequest{Query: BadRequestErrorQuery},
			ErrorContains: "cant unpack error json",
		},
		{
			Request:       SearchRequest{Query: BadRequestUnknownErrorQuery},
			ErrorContains: "unknown bad request error",
		},
		{
			Request:    SearchRequest{OrderField: "order_field"},
			ErrorExact: "OrderFeld order_field invalid",
		},
		{
			Request:       SearchRequest{Query: InvalidJsonErrorQuery},
			ErrorContains: "cant unpack result json",
		},
	}

	for i, tCase := range testCases {
		url := server.URL
		if tCase.URL != "" {
			url = tCase.URL
		}

		client := SearchClient{
			URL:         url,
			AccessToken: tCase.AccessToken,
		}
		response, err := client.FindUsers(tCase.Request)

		if response != nil || err == nil {
			t.Errorf("[%d] expected error, got nil", i)
		}

		if tCase.ErrorExact != "" && err.Error() != tCase.ErrorExact {
			t.Errorf("[%d] wrong result, expected %#v, got %#v", i, tCase.ErrorExact, err.Error())
		}

		if tCase.ErrorContains != "" && !strings.Contains(err.Error(), tCase.ErrorContains) {
			t.Errorf("[%d] wrong result, expected %#v to contain %#v", i, err.Error(), tCase.ErrorContains)
		}
	}

}

func TestFindUsers(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(SearchServer))
	defer server.Close()

	testCases := []TestCase{
		{
			SearchRequest{Limit: 1},
		},
		{
			SearchRequest{Limit: 30},
		},
		{
			SearchRequest{Limit: 25, Offset: 1},
		},
	}

	for i, tCase := range testCases {
		client := SearchClient{
			URL: server.URL,
		}
		response, err := client.FindUsers(tCase.Request)

		if response == nil || err != nil {
			t.Errorf("[%d] expected response, got error", i)
		}
	}
}
