package exports

import (
	"net/http"
)

const AILAB_MODULE_ID = 2800

const (
	AILAB_ERROR_BEGIN    = AILAB_MODULE_ID*100000 + iota

	AILAB_NOT_FOUND
	AILAB_PARAM_ERROR
	AILAB_ALREADY_EXISTS
	AILAB_NO_API
	AILAB_NOT_IMPLEMENT
	AILAB_UNKNOWN_ERROR
	AILAB_FILE_NOT_FOUND
	AILAB_PATH_NOT_FOUND
	AILAB_OS_IO_ERROR
	AILAB_OS_READ_DIR_ERROR
	AILAB_OS_REMOVE_FILE
	AILAB_OS_CREATE_FILE
	AILAB_FILE_TOO_LARGE
	AILAB_FILE_TYPE_ERROR
	AILAB_DATASET_ERROR       // remote dataset server error
	AILAB_CODE_ERROR          // remote code management error
	AILAB_JOB_SCHEDULER_ERROR // remote job scheduler error
	AILAB_DOCKER_IMAGE_ERROR
	AILAB_INVALID_LAB_STATUS
	AILAB_INVALID_RUN_STATUS
	AILAB_REFER_PARENT_ERROR
	AILAB_SERVER_BUSY
	AILAB_WOULD_BLOCK
	AILAB_NO_AUTH
	AILAB_STILL_ACTIVE
	AILAB_LOGIC_ERROR
	AILAB_SINGLETON_RUN_EXISTS
	AILAB_RUN_CANNOT_RESTART
	AILAB_REMOTE_NETWORK_ERROR
	AILAB_REMOTE_REST_ERROR
	AILAB_REMOTE_GRPC_ERROR
	AILAB_CANNOT_COMMIT
	AILAB_EXCEED_LIMIT
	AILAB_INVALID_ENDPOINT_STATUS
)
const (
	AILAB_DB_ERROR      = AILAB_MODULE_ID*100000 + 300 + iota
	AILAB_DB_QUERY_FAILED
	AILAB_DB_EXEC_FAILED
	AILAB_DB_DUPLICATE
	AILAB_DB_UPDATE_UNEXPECT
	AILAB_DB_WRONG_TYPE
	AILAB_DB_READ_ROWS
)


type APIError interface{
	Error()     string
	Errno()     int
}

type APIException struct {
	StatusCode int    `json:"-"`   // http status code ,should be rarely used
	Code       int    `json:"code"`
	Msg        string `json:"msg"`
}

func (e *APIException) Error() string {
	return e.Msg
}
func (e*APIException)  Errno() int {
	return e.Code
}

func NewAPIException(statusCode, code int, msg string) *APIException {
	return &APIException{
		StatusCode: statusCode,
		Code:       code,
		Msg:        msg,
	}
}


func UnAuthorizedError(msg string) *APIException {
	return NewAPIException(http.StatusUnauthorized, AILAB_NO_AUTH, msg)
}

func NotFoundError() * APIException {
	return NewAPIException(http.StatusNotFound, AILAB_NOT_FOUND, http.StatusText(http.StatusNotFound))
}

func UnknownError(msg string) *APIException {
	return NewAPIException(http.StatusForbidden, AILAB_UNKNOWN_ERROR, msg)
}

func ParameterError(msg string) *APIException {
	return NewAPIException(http.StatusBadRequest, AILAB_PARAM_ERROR, msg)
}

func NotImplementError(msg string)*APIException{
	return NewAPIException(http.StatusBadRequest,AILAB_NOT_IMPLEMENT,msg)
}

func DockerImageError()*APIException{
	return NewAPIException(http.StatusBadRequest,AILAB_DOCKER_IMAGE_ERROR,"docker image error")
}

func RaiseReqWouldBlock(msg string) *APIException{
	return NewAPIException(http.StatusOK,AILAB_WOULD_BLOCK,msg)
}

func RaiseAPIError(code int , args ... string) APIError{
	if len(args) > 0 {
		return NewAPIException(http.StatusBadRequest, code, args[0])
	}else{
		return NewAPIException(http.StatusBadRequest, code, "")
	}
}
func RaiseServerError(code int,args ... string) APIError{
	if len(args) > 0 {
		return NewAPIException(http.StatusInternalServerError, code, args[0])
	}else{
		return NewAPIException(http.StatusInternalServerError, code, "")
	}
}
func RaiseHttpError(statusCode int,code int ,status string)APIError{
	return NewAPIException(statusCode,code,status)
}


func CheckWithError(err error,code int) APIError{
	if err != nil {
		return RaiseAPIError(code,err.Error())
	}else if code!=0 {
		return RaiseAPIError(code)
	}else{
		return nil
	}
}

