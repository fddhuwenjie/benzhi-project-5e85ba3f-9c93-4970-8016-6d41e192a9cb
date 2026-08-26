package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

func doJSON(client *http.Client, method, url string, body any) (map[string]any, error) {
	data, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(method, url, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	payload, _ := io.ReadAll(res.Body)
	var value map[string]any
	_ = json.Unmarshal(payload, &value)
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return value, fmt.Errorf("%s %s: %s", method, url, string(payload))
	}
	return value, nil
}

func doJSONStatus(client *http.Client, method, url string, body any, expected int) (map[string]any, error) {
	data, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(method, url, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	payload, _ := io.ReadAll(res.Body)
	var value map[string]any
	_ = json.Unmarshal(payload, &value)
	if res.StatusCode != expected {
		return value, fmt.Errorf("期望状态码 %d，实际 %d", expected, res.StatusCode)
	}
	return value, nil
}

func doRaw(client *http.Client, method, url string, body []byte) ([]byte, error) {
	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	}
	req, err := http.NewRequest(method, url, reader)
	if err != nil {
		return nil, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	data, _ := io.ReadAll(res.Body)
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return data, fmt.Errorf("请求失败: %s", res.Status)
	}
	return data, nil
}
