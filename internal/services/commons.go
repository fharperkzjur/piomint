
package services

import (
	"bytes"
	"github.com/apulis/bmod/ai-lab-backend/pkg/exports"
	"github.com/sirupsen/logrus"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
	"encoding/json"
)

var logger *logrus.Logger



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

	client := http.DefaultClient
	resp, err := client.Do(req)

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
		return responseData, exports.RaiseAPIError(exports.AILAB_REMOTE_REST_ERROR,resp.Status)
	}else{
		return responseData, nil
	}
}
// wild request
func DoRequest(url, method string, headers map[string]string, rawBody interface{}, output interface{}) APIError {

	rspData, err := doHttpRequest(url, method, headers, rawBody)
	//@todo: only accpet json response ???
	if err := json.Unmarshal(rspData,output);err != nil {
		return exports.RaiseAPIError(exports.AILAB_REMOTE_REST_ERROR,"invalid response data format")
	}
	return err
}

//  return body error code
func Request(url, method string, headers map[string]string, rawBody interface{}, output interface{}) APIError {

	rspData, err := doHttpRequest(url, method, headers, rawBody)
	response := &exports.CommResponse{
		Data: output,
	}
	if err := json.Unmarshal(rspData,response);err != nil {
		return exports.RaiseAPIError(exports.AILAB_REMOTE_REST_ERROR,"invalid response data format")
	}
	if response.Code != 0 {
        return exports.RaiseAPIError(response.Code,response.Msg)
	}else{// may return http error code
		return err
	}
}
