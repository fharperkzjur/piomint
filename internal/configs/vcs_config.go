
package configs


type VcsExtranet struct {
	Host     string
	SshHost  string
	Prefix   string
}

type VcsConfig struct{
	Url       string
	User      string
	Passwd    string
	Host      string
	SshHost   string
	UseHttp   bool
	Extranet  VcsExtranet
}

type VCSConfigTable map[string]VcsConfig

