package routers

import (
	"fmt"
	"github.com/apulis/bmod/ai-lab-backend/internal/configs"
	"github.com/apulis/bmod/ai-lab-backend/internal/services"
	"github.com/gin-gonic/gin"
	"path"
)


type IamEndPointType struct {
	Resource     string `json:"resource"`
	Action       string `json:"action"`
	HTTPMethod   string `json:"httpMethod"`
	HTTPEndpoint string `json:"httpEndpoint"`
	Module       string `json:"module"`
	Desc         string `json:"desc"`
}

type IamEndPointAction struct {
	Actions []string `json:"actions"`
	Effect  string   `json:"effect"`
}

type IamPoliciesType struct {
	Module      string              `json:"module"`
	SystemAdmin []IamEndPointAction `json:"systemAdmin"`
	OrgAdmin    []IamEndPointAction `json:"orgAdmin"`
	Developer   []IamEndPointAction `json:"developer"`
}

type RegisterEndpointsReq struct {
	Endpoints []IamEndPointType `json:"endpoints"`
	Policies  IamPoliciesType   `json:"policies"`
}

var g_IAM_EndPoints RegisterEndpointsReq

type IAMRouteGroup  gin.RouterGroup

func (d*IAMRouteGroup)FullPath(relativePath string)string{
	return path.Join((*gin.RouterGroup)(d).BasePath(),relativePath)
}

func (d*IAMRouteGroup)GET(relativePath string, handlers gin.HandlerFunc,name string) gin.IRoutes{
	r := (*gin.RouterGroup)(d)
	registerRoute("get",d.FullPath(relativePath),name)
	return r.GET(relativePath,handlers)
}
func (d*IAMRouteGroup)POST(relativePath string, handlers gin.HandlerFunc,name string) gin.IRoutes{
	r := (*gin.RouterGroup)(d)
	registerRoute("post",d.FullPath(relativePath),name)
	return r.POST(relativePath,handlers)
}

func (d*IAMRouteGroup)PUT(relativePath string, handlers gin.HandlerFunc,name string) gin.IRoutes{
	r := (*gin.RouterGroup)(d)
	registerRoute("put",d.FullPath(relativePath),name)
	return r.PUT(relativePath,handlers)
}

func (d*IAMRouteGroup)DELETE(relativePath string, handlers gin.HandlerFunc,name string) gin.IRoutes{
	r := (*gin.RouterGroup)(d)
	registerRoute("delete",d.FullPath(relativePath),name)
	return r.DELETE(relativePath,handlers)
}


func registerRoute(method string, path string, name string) {

	convert := false
	resultPath := []rune{}
	for _, v := range path {
		if v == ':' {
			convert = true
			resultPath = append(resultPath, '{')
		} else if v == '/' && convert {
			resultPath = append(resultPath, '}')
			resultPath = append(resultPath, '/')
			convert = false
		} else {
			resultPath = append(resultPath, v)
		}
	}
	if convert {
		resultPath = append(resultPath, '}')
	}

	action := fmt.Sprintf("%s:%s",g_IAM_EndPoints.Policies.Module,name)

	g_IAM_EndPoints.Endpoints = append(g_IAM_EndPoints.Endpoints, IamEndPointType{
		Resource:     "*",
		Action:       action,
		HTTPMethod:   method,
		HTTPEndpoint: string(resultPath),
		Module:       g_IAM_EndPoints.Policies.Module,
		Desc:         name,
	})
	g_IAM_EndPoints.Policies.SystemAdmin[0].Actions = append(g_IAM_EndPoints.Policies.SystemAdmin[0].Actions, action)
	g_IAM_EndPoints.Policies.OrgAdmin[0].Actions = append(g_IAM_EndPoints.Policies.OrgAdmin[0].Actions, action)
	g_IAM_EndPoints.Policies.Developer[0].Actions = append(g_IAM_EndPoints.Policies.Developer[0].Actions, action)
}

func init() {
	g_IAM_EndPoints.Policies.Module = "ailab"
	g_IAM_EndPoints.Policies.SystemAdmin = []IamEndPointAction{
		{Actions: []string{}, Effect: "allow"},
	}

	g_IAM_EndPoints.Policies.OrgAdmin = []IamEndPointAction{
		{Actions: []string{}, Effect: "allow"},
	}

	g_IAM_EndPoints.Policies.Developer = []IamEndPointAction{
		{Actions: []string{}, Effect: "allow"},
	}
}

func registerEndpoints(){
		urlpath := configs.GetAppConfig().Resources.Iam + "/endpoints"

		err := services.Request(urlpath,"POST",nil,&g_IAM_EndPoints,nil)

		logger.Infof("register endpoints to iam:%v",err)
}
