/* ******************************************************************************
* 2019 - present Contributed by Apulis Technology (Shenzhen) Co. LTD
*
* This program and the accompanying materials are made available under the
* terms of the MIT License, which is available at
* https://www.opensource.org/licenses/MIT
*
* See the NOTICE file distributed with this work for additional
* information regarding copyright ownership.
* Unless required by applicable law or agreed to in writing, software
* distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
* WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the
* License for the specific language governing permissions and limitations
* under the License.
*
* SPDX-License-Identifier: MIT
******************************************************************************/
package exports

const (
	AICODE_ERROR_BEGIN           = AILAB_MODULE_ID*100000 + 400 + iota
	AICODE_CONNECTOR_AUTH_ERROR
	AICODE_CONNECTOR_ERROR
	AICODE_CONNECTOR_ALREADY_EXISTS
	AICODE_CONNECTOR_NETWORK_ERROR
	AICODE_CONNECTOR_NOT_FOUND
	AICODE_CONNECTOR_NOT_IMPLEMENT
	AICODE_CONNECTOR_INVALID_STATUS
)

type ReqCreateRepo struct{
	Bind        string                   `json:"bind" `     // user defined apps (repo resource path)
	Creator     string                   `json:"creator"`
	UserId      uint64                   `json:"userId"`
    RepoId      string                   `json:"repoId"`    // exists repoId
    IsMultiBind bool                     `json:"multi"`     // instruct this app can bind multiple repos
    IsSsoAuth   bool                     `json:"sso"`       // instruct this app need sso user to access repo
    Connector   string                   `json:"connector"` // default to gitea
    HttpUrl     string                   `json:"http_url"`  // if not empty , attach repo only
	SshUrl      string                   `json:"ssh_url"`
	Description string                   `json:"description"`
}

const (
	AICODE_OWN_TYPE_NONE = 0   // none own repo , attach&detah only
	AICODE_OWN_TYPE_SYS  = 1   // use system admin user to manage repo
	AICODE_OWN_TYPE_SSO  = 2   // use platform user to manage repo
)

const (
	AICODE_CONNECTOR_GITEA  = "gitea"
	AICODE_CONNECTOR_GITLAB = "gitlab"
	AICODE_CONNECTOR_GITHUB = "github"
	AICODE_CONNECTOR_SVN    = "svn"
)

const (
	AICODE_REPO_STATUS_INIT = "init"
	AICODE_REPO_STATUS_READY = "ready"
	AICODE_REPO_STATUS_DELETE = "delete"
)

