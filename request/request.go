package request

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// get请求（不超时）
func HttpGet(apiUrl string, params interface{}, headers map[string][]string) (body []byte, err error) {
	return HttpGetWithTimeout(apiUrl, params, headers, 0)
}

// get请求（支持超时，timeout <= 0 表示不超时）
func HttpGetWithTimeout(apiUrl string, params interface{}, headers map[string][]string, timeout time.Duration) (body []byte, err error) {
	resChan := make(chan []byte, 1)
	errChan := make(chan error, 1)
	repoUrl := apiUrl
	data, err := json.Marshal(params)
	fmt.Printf("GET请求JSON：%s\n", data)
	request, err := http.NewRequest(http.MethodGet, repoUrl, bytes.NewReader(data))
	if headers != nil {
		request.Header = headers
	}
	if err != nil {
		return nil, err
	}

	var ctx context.Context
	var cancel context.CancelFunc
	if timeout > 0 {
		ctx, cancel = context.WithTimeout(context.Background(), timeout)
		defer cancel()
	} else {
		ctx = context.Background()
	}

	go func() {
		if err := httpRequestWithContext(ctx, request, resChan); err != nil {
			errChan <- err
		}
	}()

	if timeout > 0 {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case err := <-errChan:
			return nil, err
		case body := <-resChan:
			return body, nil
		}
	} else {
		select {
		case err := <-errChan:
			return nil, err
		case body := <-resChan:
			return body, nil
		}
	}
}

// post请求（不超时）
func HttpPost(apiUrl string, params interface{}, headers map[string][]string) (body []byte, err error) {
	return HttpPostWithTimeout(apiUrl, params, headers, 0)
}

// post请求（支持超时，timeout <= 0 表示不超时）
func HttpPostWithTimeout(apiUrl string, params interface{}, headers map[string][]string, timeout time.Duration) (body []byte, err error) {
	resChan := make(chan []byte, 1)
	errChan := make(chan error, 1)
	repoUrl := apiUrl
	data, err := json.Marshal(params)
	fmt.Printf("POST请求JSON：%s\n", data)
	request, err := http.NewRequest(http.MethodPost, repoUrl, bytes.NewReader(data))
	if headers != nil {
		request.Header = headers
	} else {
		request.Header.Set("Content-Type", "application/json;charset=UTF-8")
	}
	if err != nil {
		return nil, err
	}

	var ctx context.Context
	var cancel context.CancelFunc
	if timeout > 0 {
		ctx, cancel = context.WithTimeout(context.Background(), timeout)
		defer cancel()
	} else {
		ctx = context.Background()
	}

	go func() {
		if err := httpRequestWithContext(ctx, request, resChan); err != nil {
			errChan <- err
		}
	}()

	if timeout > 0 {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case err := <-errChan:
			return nil, err
		case body := <-resChan:
			return body, nil
		}
	} else {
		select {
		case err := <-errChan:
			return nil, err
		case body := <-resChan:
			return body, nil
		}
	}
}

func httpRequestWithContext(ctx context.Context, request *http.Request, resChan chan<- []byte) (err error) {
	request = request.WithContext(ctx)
	client := &http.Client{}
	resp, err := client.Do(request)
	if err != nil {
		return fmt.Errorf("client.Do Error: %s", err.Error())
	}
	if resp.StatusCode != 200 {
		return fmt.Errorf("response from weixin with status %v", resp.StatusCode)
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("io.ReadAll Error: %s", err.Error())
	}
	defer resp.Body.Close()
	resChan <- data
	return nil
}
