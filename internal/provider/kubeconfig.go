package provider

import (
	"encoding/base64"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"gopkg.in/yaml.v3"
)

var kubeConfigAttrTypes = map[string]attr.Type{
	"raw":                    types.StringType,
	"host":                   types.StringType,
	"cluster_ca_certificate": types.StringType,
	"client_certificate":     types.StringType,
	"client_key":             types.StringType,
	"token":                  types.StringType,
}

type kubeContext struct {
	Name    string `yaml:"name"`
	Context struct {
		Cluster string `yaml:"cluster"`
		User    string `yaml:"user"`
	} `yaml:"context"`
}

type kubeCluster struct {
	Name    string `yaml:"name"`
	Cluster struct {
		Server                   string `yaml:"server"`
		CertificateAuthorityData string `yaml:"certificate-authority-data"`
	} `yaml:"cluster"`
}

type kubeUser struct {
	Name string `yaml:"name"`
	User struct {
		ClientCertificateData string `yaml:"client-certificate-data"`
		ClientKeyData         string `yaml:"client-key-data"`
		Token                 string `yaml:"token"`
	} `yaml:"user"`
}

type kubeconfigFile struct {
	CurrentContext string        `yaml:"current-context"`
	Contexts       []kubeContext `yaml:"contexts"`
	Clusters       []kubeCluster `yaml:"clusters"`
	Users          []kubeUser    `yaml:"users"`
}

// kubeConfigValue splits a kubeconfig into the parts the kubernetes and helm
// providers take, PEM decoded, for its current context. Null until the cluster
// is ready and has one. A kubeconfig that doesn't parse still gives raw.
func kubeConfigValue(raw string) types.Object {
	if raw == "" {
		return types.ObjectNull(kubeConfigAttrTypes)
	}
	values := map[string]attr.Value{
		"raw":                    types.StringValue(raw),
		"host":                   types.StringNull(),
		"cluster_ca_certificate": types.StringNull(),
		"client_certificate":     types.StringNull(),
		"client_key":             types.StringNull(),
		"token":                  types.StringNull(),
	}

	var file kubeconfigFile
	if yaml.Unmarshal([]byte(raw), &file) != nil {
		return types.ObjectValueMust(kubeConfigAttrTypes, values)
	}

	clusterName, userName := "", ""
	for _, c := range file.Contexts {
		if c.Name == file.CurrentContext {
			clusterName, userName = c.Context.Cluster, c.Context.User
			break
		}
	}

	for i, c := range file.Clusters {
		if c.Name == clusterName || (clusterName == "" && i == 0) {
			values["host"] = stringOrNull(c.Cluster.Server)
			values["cluster_ca_certificate"] = stringOrNull(decodeBase64(c.Cluster.CertificateAuthorityData))
			break
		}
	}
	for i, u := range file.Users {
		if u.Name == userName || (userName == "" && i == 0) {
			values["client_certificate"] = stringOrNull(decodeBase64(u.User.ClientCertificateData))
			values["client_key"] = stringOrNull(decodeBase64(u.User.ClientKeyData))
			values["token"] = stringOrNull(u.User.Token)
			break
		}
	}

	return types.ObjectValueMust(kubeConfigAttrTypes, values)
}

func decodeBase64(data string) string {
	decoded, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		return ""
	}
	return string(decoded)
}
