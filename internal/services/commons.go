
package services

import (
	"bytes"
	"encoding/json"
	"github.com/apulis/bmod/ai-lab-backend/internal/configs"
	"github.com/apulis/bmod/ai-lab-backend/pkg/exports"
	"github.com/sirupsen/logrus"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
	"time"
)

var logger *logrus.Logger

var httpClient *http.Client

func InitHttpClient() *http.Client {
	appConfig := configs.GetAppConfig()
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.MaxIdleConns = appConfig.HttpClient.MaxIdleConns
	transport.MaxConnsPerHost = appConfig.HttpClient.MaxConnsPerHost
	transport.MaxIdleConnsPerHost = appConfig.HttpClient.MaxIdleConnsPerHost

	httpClient = &http.Client{
		Timeout:   time.Duration(appConfig.HttpClient.TimeoutSeconds) * time.Second,
		Transport: transport,
	}
	return httpClient
}

func GetHttpClient() *http.Client {
	return httpClient
}


func wrapHttpClientError(err error) APIError{
	 return exports.RaiseAPIError(exports.AILAB_REMOTE_NETWORK_ERROR,err.Error())
}

func doHttpRequest(url, method string, headers map[string]string, rawBody interface{}) ([]byte, APIError) {

	var body io.Reader = nil
	if rawBody != nil {
		switch t := rawBody.(type) {
		case string:
			body = strings.NewReader(t)

		case []byte:
			body = bytes.NewReader(t)

		default:
			data, err := json.Marshal(rawBody)
			if err != nil {//should never happen
				return nil, exports.ParameterError("doRequest marshal error:" + err.Error())
			}
			if headers==nil {
				headers = make(map[string]string)
			}
			headers["Content-Type"] = "application/json"
			body = bytes.NewReader(data)
		}
	}
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, wrapHttpClientError(err)
	}

	if len(headers) != 0 {
		for k, v := range headers {
			req.Header.Set(k, v)
		}
	}

	resp, err := httpClient.Do(req)

	if err != nil {
		return nil, wrapHttpClientError(err)
	}

	defer resp.Body.Close()

	// read response
	responseData := make([]byte, 0)
	if responseData, err = ioutil.ReadAll(resp.Body);err != nil {
		return nil,wrapHttpClientError(err)
	}else if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusBadRequest {
		// record http status error if status code is not 200
		if resp.StatusCode < http.StatusInternalServerError{
			return responseData, exports.RaiseHttpError(resp.StatusCode,exports.AILAB_REMOTE_REST_ERROR,resp.Status)
		}else{
			return responseData, exports.RaiseAPIError(exports.AILAB_REMOTE_REST_ERROR,resp.Status)
		}
	}else{
		return responseData, nil
	}
}
// wild request
func DoRequest(url, method string, headers map[string]string, rawBody interface{}, output interface{}) APIError {

	rspData, err := doHttpRequest(url, method, headers, rawBody)
	if len(rspData) == 0 {// no data return
		return err
	}
	//@todo: only accpet json response ???
	if e := json.Unmarshal(rspData,output);e != nil {
		logger.Warnf("[%s] :%s with req:%v invalid response json data !!!",method,url,rawBody)
		return exports.RaiseAPIError(exports.AILAB_REMOTE_REST_ERROR,"invalid response data format")
	}
	if err != nil {
		logger.Warnf("[%s] :%s with req:%v error response:%v",method,url,rawBody,string(rspData))
	}

	return err
}

//  return body error code
func Request(url, method string, headers map[string]string, rawBody interface{}, output interface{}) APIError {

	rspData, err := doHttpRequest(url, method, headers, rawBody)
	if len(rspData) == 0 {// no data return
		return err
	}
	response := &exports.CommResponse{
		Data: output,
	}
	if err := json.Unmarshal(rspData,response);err != nil {
		logger.Warnf("[%s] :%s with req:%v invalid response json data !!!",method,url,rawBody)
		return exports.RaiseAPIError(exports.AILAB_REMOTE_REST_ERROR,"invalid response data format")
	}
	if response.Code != 0 {
		logger.Warnf("[%s] :%s with req:%v error response:%v",method,url,rawBody,response)
        return exports.RaiseAPIError(response.Code,response.Msg)
	}else{// may return http error code
		return err
	}
}
