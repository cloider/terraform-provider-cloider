package client

import "time"

// The shapes the provider sends and reads, the subset of the api it manages.

type Whoami struct {
	Account struct {
		Email string `json:"email"`
	} `json:"account"`
	Role  string `json:"role"`
	Token *struct {
		ReadOnly bool `json:"read_only"`
	} `json:"token"`
}

// Options are the choices the create forms offer, from GET /options.
type Options struct {
	ClusterVersions []string               `json:"cluster_versions"`
	DatabaseEngines []DatabaseEngineOption `json:"database_engines"`
	DatabasePlans   []string               `json:"database_plans"`
	SubdomainRoots  []string               `json:"subdomain_roots"`
	Nameservers     []string               `json:"nameservers"`
}

type DatabaseEngineOption struct {
	Service  string   `json:"service"`
	Versions []string `json:"versions"`
	Ssl      bool     `json:"ssl"`
}

// Created is what the creates that answer with the id alone return.
type Created struct {
	HashedId string `json:"hashed_id"`
}

// Clusters

type Cluster struct {
	HashedId       string    `json:"hashed_id"`
	ClusterName    string    `json:"cluster_name"`
	ClusterLabel   string    `json:"cluster_label"`
	ClusterVersion string    `json:"cluster_version"`
	Status         string    `json:"status"`
	CreatedAt      time.Time `json:"created_at"`
}

// Databases

type TrustedSource struct {
	Name string `json:"name"`
	Code string `json:"code"`
	Type string `json:"type"`
}

type Database struct {
	HashedId        string          `json:"hashed_id"`
	DatabaseName    string          `json:"database_name"`
	DatabaseLabel   string          `json:"database_label"`
	DatabaseUser    string          `json:"database_user"`
	Service         string          `json:"service"`
	DatabaseVersion string          `json:"database_version"`
	DatabasePlan    string          `json:"database_plan"`
	TrustedSources  []TrustedSource `json:"trusted_sources"`
	Ssl             bool            `json:"ssl"`
	Status          string          `json:"status"`
	CreatedAt       time.Time       `json:"created_at"`
	// Only on GET /databases/:id
	DatabasePort string `json:"database_port"`
	Host         string `json:"host"`
	PrivateHost  string `json:"private_host"`
}

type DatabaseForm struct {
	DatabaseName    string          `json:"database_name"`
	DatabaseUser    string          `json:"database_user"`
	Service         string          `json:"service"`
	DatabaseVersion string          `json:"database_version"`
	DatabasePlan    string          `json:"database_plan"`
	TrustedSources  []TrustedSource `json:"trusted_sources"`
	Ssl             bool            `json:"ssl"`
}

// App is the part of an app a trusted source names it by.
type App struct {
	HashedId string `json:"hashed_id"`
	AppName  string `json:"app_name"`
}

// Buckets

type BucketRegion struct {
	Code     string `json:"code"`
	Name     string `json:"name"`
	Endpoint string `json:"endpoint"`
}

type Bucket struct {
	HashedId   string    `json:"hashed_id"`
	BucketName string    `json:"bucket_name"`
	Region     string    `json:"region"`
	Public     bool      `json:"public"`
	Status     string    `json:"status"`
	Endpoint   string    `json:"endpoint"`
	URL        string    `json:"url"`
	CreatedAt  time.Time `json:"created_at"`
}

// BucketVersioning is "", Enabled or Suspended: once on, versioning can only be suspended.
type BucketVersioning struct {
	Status string `json:"status"`
}

type BucketKeyForm struct {
	Label      string   `json:"label"`
	AllBuckets bool     `json:"all_buckets"`
	Buckets    []string `json:"buckets"`
	ReadOnly   bool     `json:"read_only"`
}

type BucketKey struct {
	BucketKeyForm
	HashedId  string    `json:"hashed_id"`
	AccessKey string    `json:"access_key"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

type CreatedBucketKey struct {
	Key       BucketKey `json:"key"`
	SecretKey string    `json:"secret_key"`
}

// Container registry projects

type Registry struct {
	HashedId string `json:"hashed_id"`
	Name     string `json:"name"`
	// The account's apps project, where every app's images are built into
	Apps           bool      `json:"apps"`
	Public         bool      `json:"public"`
	StorageLimitGi int       `json:"storage_limit_gi"`
	Status         string    `json:"status"`
	Host           string    `json:"host"`
	URL            string    `json:"url"`
	CreatedAt      time.Time `json:"created_at"`
}

type RegistrySettings struct {
	Public         bool `json:"public"`
	StorageLimitGi int  `json:"storage_limit_gi"`
}

// RegistryImmutability is only read for the Immutable tags setting its rules add up to.
type RegistryImmutability struct {
	ImmutableTags bool `json:"immutable_tags"`
}

type RegistryTokenForm struct {
	Label       string   `json:"label"`
	AllProjects bool     `json:"all_projects"`
	Projects    []string `json:"projects"`
	ReadOnly    bool     `json:"read_only"`
}

type NewRegistryTokenForm struct {
	RegistryTokenForm
	// 0 for never
	ExpiresInDays int `json:"expires_in_days"`
}

type RegistryToken struct {
	RegistryTokenForm
	HashedId  string     `json:"hashed_id"`
	Username  string     `json:"username"`
	ExpiresAt *time.Time `json:"expires_at"`
	Status    string     `json:"status"`
	CreatedAt time.Time  `json:"created_at"`
}

type CreatedRegistryToken struct {
	Token  RegistryToken `json:"token"`
	Secret string        `json:"secret"`
}

// Domains and DNS

type Domain struct {
	DomainName string    `json:"domain_name"`
	DomainType string    `json:"domain_type"`
	Status     string    `json:"status"`
	CreatedAt  time.Time `json:"created_at"`
}

type DnsRecord struct {
	Id          int64  `json:"id"`
	DomainName  string `json:"domain_name"`
	RecordType  string `json:"record_type"`
	RecordName  string `json:"record_name"`
	RecordValue string `json:"record_value"`
	Ttl         int    `json:"ttl"`
}

type DnsRecordForm struct {
	RecordType  string `json:"record_type"`
	RecordName  string `json:"record_name"`
	RecordValue string `json:"record_value"`
	Ttl         int    `json:"ttl"`
}

// Targets route a path of a domain to a cluster service or an app. Cluster
// targets carry the cluster's id and routing fields, app targets only app_id:
// the api fills in the rest from the app.
type Target struct {
	HashedId    string    `json:"hashed_id"`
	DomainName  string    `json:"domain_name"`
	TargetType  string    `json:"target_type"`
	ClusterId   string    `json:"cluster_id"`
	ClusterName string    `json:"cluster_name"`
	AppId       string    `json:"app_id"`
	Namespace   string    `json:"namespace"`
	Service     string    `json:"service"`
	Port        int       `json:"port"`
	Protocol    string    `json:"protocol"`
	Path        string    `json:"path"`
	TargetPath  string    `json:"target_path"`
	CreatedAt   time.Time `json:"created_at"`
}

type TargetForm struct {
	DomainName  string `json:"domain_name"`
	TargetType  string `json:"target_type"`
	Path        string `json:"path"`
	TargetPath  string `json:"target_path"`
	ClusterId   string `json:"cluster_id,omitempty"`
	ClusterName string `json:"cluster_name,omitempty"`
	Namespace   string `json:"namespace,omitempty"`
	Service     string `json:"service,omitempty"`
	Port        int    `json:"port,omitempty"`
	Protocol    string `json:"protocol,omitempty"`
	AppId       string `json:"app_id,omitempty"`
}
