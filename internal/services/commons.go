
package services

type PlatformResourceUsage interface {
	 // cannot be error
	 RefResource(runId string,  resourceId string) (interface{},APIError)
	 // should never error
	 UnRefResource(runId string,resourceId string) APIError
}
