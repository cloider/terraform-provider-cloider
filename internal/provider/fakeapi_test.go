package provider

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/cloider/terraform-provider-cloider/internal/client"
)

// fakeAPI answers the routes the cluster, database, domain, DNS record and
// target resources use (bucket targets without the buckets themselves), the way the api does, quirks included: lists that hide
// Terminated rows, GET by id that still shows them, plain text 404s. Everything
// settles at once, so waits return on their first poll.
type fakeAPI struct {
	mu        sync.Mutex
	nextID    int
	clusters  map[string]*client.Cluster
	databases map[string]*client.Database
	domains   map[string]*client.Domain
	records   map[int64]*client.DnsRecord
	targets   map[string]*client.Target
}

func newFakeAPI(t *testing.T) *httptest.Server {
	t.Helper()
	f := &fakeAPI{
		clusters:  map[string]*client.Cluster{},
		databases: map[string]*client.Database{},
		domains:   map[string]*client.Domain{},
		records:   map[int64]*client.DnsRecord{},
		targets:   map[string]*client.Target{},
	}
	server := httptest.NewServer(f)
	t.Cleanup(server.Close)

	// The provider block below has its own token, the developer's login stays out of it
	t.Setenv("CLOIDER_CONFIG_DIR", t.TempDir())
	client.PollInterval = 10 * time.Millisecond
	return server
}

func fakeProviderConfig(server *httptest.Server) string {
	return fmt.Sprintf("provider \"cloider\" {\n  api_url = %q\n  token   = \"cldr_test\"\n}\n", server.URL+"/api")
}

func (f *fakeAPI) id() string {
	f.nextID++
	return fmt.Sprintf("id%d", f.nextID)
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

func notFound(w http.ResponseWriter) {
	http.Error(w, "Not Found", http.StatusNotFound)
}

func (f *fakeAPI) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if r.Header.Get("Authorization") != "Bearer cldr_test" {
		w.WriteHeader(http.StatusUnauthorized)
		writeJSON(w, map[string]any{"errors": map[string][]string{"_": {"Invalid or revoked API token!"}}})
		return
	}

	parts := strings.Split(strings.Trim(strings.TrimPrefix(r.URL.Path, "/api"), "/"), "/")
	route := r.Method + " " + parts[0]
	if len(parts) > 1 {
		route += "/:id"
	}
	if len(parts) > 2 {
		route += "/" + strings.Join(parts[2:], "/")
	}

	switch route {
	case "GET whoami":
		writeJSON(w, map[string]any{"role": "Owner", "token": map[string]bool{"read_only": false}})
	case "GET options":
		writeJSON(w, client.Options{
			ClusterVersions: []string{"1.34", "1.33"},
			DatabaseEngines: []client.DatabaseEngineOption{{Service: "PostgreSQL", Versions: []string{"18"}, Ssl: true}, {Service: "Redis", Versions: []string{"8"}, Ssl: true}},
			DatabasePlans:   databasePlans,
			SubdomainRoots:  []string{"cloider.app"},
			Nameservers:     []string{"ns1.cloider.com.", "ns2.cloider.com."},
		})

	// Clusters
	case "GET clusters":
		list := []client.Cluster{}
		for _, c := range f.clusters {
			if c.Status != "Terminated" {
				list = append(list, *c)
			}
		}
		writeJSON(w, list)
	case "POST clusters":
		var body struct {
			Name    string `json:"cluster_name"`
			Version string `json:"cluster_version"`
		}
		json.NewDecoder(r.Body).Decode(&body)
		id := f.id()
		f.clusters[id] = &client.Cluster{HashedId: id, ClusterName: body.Name, ClusterLabel: body.Name, ClusterVersion: body.Version, Status: "Ready", CreatedAt: time.Now()}
		writeJSON(w, client.Created{HashedId: id})
	case "GET clusters/:id/status", "GET clusters/:id/kubeconfig", "POST clusters/:id/pause", "POST clusters/:id/resume", "DELETE clusters/:id":
		c, ok := f.clusters[parts[1]]
		if !ok {
			notFound(w)
			return
		}
		switch {
		case strings.HasSuffix(route, "/status"):
			fmt.Fprint(w, c.Status)
		case strings.HasSuffix(route, "/kubeconfig"):
			fmt.Fprintf(w, "apiVersion: v1\ncurrent-context: %[1]s\nclusters:\n- name: %[1]s\n  cluster:\n    server: https://%[1]s.k8s.cloider.com\ncontexts:\n- name: %[1]s\n  context:\n    cluster: %[1]s\n    user: %[1]s\nusers:\n- name: %[1]s\n  user:\n    token: kube-token\n", c.ClusterName)
		case strings.HasSuffix(route, "/pause"):
			c.Status = "Paused"
		case strings.HasSuffix(route, "/resume"):
			c.Status = "Ready"
		default:
			c.Status = "Terminated"
		}

	// Databases
	case "POST databases":
		var form client.DatabaseForm
		json.NewDecoder(r.Body).Decode(&form)
		id := f.id()
		f.databases[id] = &client.Database{
			HashedId: id, DatabaseName: form.DatabaseName, DatabaseLabel: form.DatabaseName, DatabaseUser: form.DatabaseUser,
			Service: form.Service, DatabaseVersion: form.DatabaseVersion, DatabasePlan: form.DatabasePlan,
			TrustedSources: form.TrustedSources, Ssl: form.Ssl, Status: "Ready", CreatedAt: time.Now(),
			DatabasePort: "5432", Host: id + ".db.cloider.com", PrivateHost: id + ".databases.svc",
		}
		writeJSON(w, client.Created{HashedId: id})
	case "GET database", "GET database/:id":
		// /database/password?id= and /database/cert?id=
		db, ok := f.databases[r.URL.Query().Get("id")]
		if !ok || (parts[len(parts)-1] == "cert" && !db.Ssl) {
			notFound(w)
			return
		}
		fmt.Fprint(w, map[string]string{"password": "s3cret", "cert": "CA PEM"}[parts[len(parts)-1]])
	case "GET databases/:id", "GET databases/:id/status", "DELETE databases/:id", "POST databases/:id/pause", "POST databases/:id/resume",
		"PUT databases/:id/label", "PUT databases/:id/ssl", "PUT databases/:id/plan", "PUT databases/:id/trusted-sources":
		db, ok := f.databases[parts[1]]
		if !ok {
			notFound(w)
			return
		}
		var body struct {
			Label   string                 `json:"database_label"`
			Ssl     bool                   `json:"ssl"`
			Plan    string                 `json:"database_plan"`
			Sources []client.TrustedSource `json:"trusted_sources"`
		}
		json.NewDecoder(r.Body).Decode(&body)
		switch route {
		case "GET databases/:id":
			writeJSON(w, db)
		case "GET databases/:id/status":
			fmt.Fprint(w, db.Status)
		case "DELETE databases/:id":
			db.Status = "Terminated"
		case "POST databases/:id/pause":
			db.Status = "Paused"
		case "POST databases/:id/resume":
			db.Status = "Ready"
		case "PUT databases/:id/label":
			db.DatabaseLabel = body.Label
		case "PUT databases/:id/ssl":
			db.Ssl = body.Ssl
		case "PUT databases/:id/plan":
			db.DatabasePlan = body.Plan
		case "PUT databases/:id/trusted-sources":
			db.TrustedSources = body.Sources
		}

	// Domains and DNS records
	case "GET domains/:id", "DELETE domains/:id":
		d, ok := f.domains[parts[1]]
		if !ok {
			notFound(w)
			return
		}
		if r.Method == http.MethodGet {
			writeJSON(w, d)
			return
		}
		delete(f.domains, parts[1])
		for id, record := range f.records {
			if record.DomainName == d.DomainName {
				delete(f.records, id)
			}
		}
		for id, target := range f.targets {
			if target.DomainName == d.DomainName {
				delete(f.targets, id)
			}
		}
	case "POST domains":
		var body struct {
			Name string `json:"domain_name"`
			Type string `json:"domain_type"`
		}
		json.NewDecoder(r.Body).Decode(&body)
		d := &client.Domain{DomainName: body.Name, DomainType: body.Type, Status: "Pending", CreatedAt: time.Now()}
		f.domains[body.Name] = d
		writeJSON(w, d)
	case "POST domains/:id/dns-records":
		if _, ok := f.domains[parts[1]]; !ok {
			notFound(w)
			return
		}
		var form client.DnsRecordForm
		json.NewDecoder(r.Body).Decode(&form)
		f.nextID++
		record := &client.DnsRecord{Id: int64(f.nextID), DomainName: parts[1], RecordType: form.RecordType, RecordName: form.RecordName, RecordValue: form.RecordValue, Ttl: form.Ttl}
		f.records[record.Id] = record
		writeJSON(w, record)
	case "GET dns-records/:id", "PUT dns-records/:id", "DELETE dns-records/:id":
		id, _ := strconv.ParseInt(parts[1], 10, 64)
		record, ok := f.records[id]
		if !ok {
			notFound(w)
			return
		}
		switch r.Method {
		case http.MethodPut:
			var form client.DnsRecordForm
			json.NewDecoder(r.Body).Decode(&form)
			record.RecordType, record.RecordName, record.RecordValue, record.Ttl = form.RecordType, form.RecordName, form.RecordValue, form.Ttl
			writeJSON(w, record)
		case http.MethodDelete:
			delete(f.records, id)
		default:
			writeJSON(w, record)
		}

	// Targets
	case "GET targets":
		domain := r.URL.Query().Get("domain_name")
		if _, ok := f.domains[domain]; !ok {
			w.WriteHeader(http.StatusForbidden)
			writeJSON(w, map[string]any{"errors": map[string][]string{"_": {"You don't have permission to access targets for this domain"}}})
			return
		}
		list := []client.Target{}
		for _, t := range f.targets {
			if t.DomainName == domain {
				list = append(list, *t)
			}
		}
		writeJSON(w, list)
	case "POST targets", "PUT targets/:id", "DELETE targets/:id":
		var t *client.Target
		if r.Method == http.MethodPost {
			t = &client.Target{HashedId: f.id(), CreatedAt: time.Now()}
		} else if t = f.targets[parts[1]]; t == nil {
			notFound(w)
			return
		}
		if r.Method == http.MethodDelete {
			delete(f.targets, t.HashedId)
			return
		}
		var form client.TargetForm
		json.NewDecoder(r.Body).Decode(&form)
		if _, ok := f.domains[form.DomainName]; !ok {
			notFound(w)
			return
		}
		if form.Path == "" {
			form.Path = "/"
		}
		if form.TargetPath == "" {
			form.TargetPath = form.Path
		}
		if form.TargetType == "Bucket" && (form.Path != "/" || form.TargetPath != "/") {
			w.WriteHeader(http.StatusBadRequest)
			writeJSON(w, map[string]any{"errors": map[string][]string{"path": {"A bucket website is served at the root of its domain, the path must be /"}}})
			return
		}
		if form.TargetType == "LoadBalancer" && (form.Path != "/" || form.TargetPath != "/") {
			w.WriteHeader(http.StatusBadRequest)
			writeJSON(w, map[string]any{"errors": map[string][]string{"path": {"A LoadBalancer takes the whole domain, the path must be /"}}})
			return
		}
		t.DomainName, t.TargetType, t.Path, t.TargetPath = form.DomainName, form.TargetType, form.Path, form.TargetPath
		switch form.TargetType {
		case "App":
			// The api routes an app target from the app itself
			t.AppId, t.BucketId, t.ClusterId, t.ClusterName = form.AppId, "", "", ""
			t.Namespace, t.Service, t.Port, t.Protocol = "app-web-"+form.AppId, "app", 8080, "TCP"
		case "Bucket":
			// and a bucket target to the website service, which serves the bucket
			t.AppId, t.BucketId, t.ClusterId, t.ClusterName = "", form.BucketId, "", ""
			t.Namespace, t.Service, t.Port, t.Protocol = "", "", 8006, "TCP"
			t.LoadbalancerIp = ""
		case "LoadBalancer":
			// a LoadBalancer target has no port or protocol, its A record carries them all
			t.AppId, t.BucketId, t.ClusterId = "", "", form.ClusterId
			t.Namespace, t.Service, t.Port, t.Protocol = form.Namespace, form.Service, 0, ""
			t.LoadbalancerIp = "203.0.113.7"
		default:
			t.AppId, t.BucketId, t.ClusterId, t.ClusterName = "", "", form.ClusterId, form.ClusterName
			t.Namespace, t.Service, t.Port, t.Protocol = form.Namespace, form.Service, form.Port, form.Protocol
			t.LoadbalancerIp = ""
		}
		f.targets[t.HashedId] = t
		writeJSON(w, client.Created{HashedId: t.HashedId})

	default:
		http.Error(w, "no fake for "+route, http.StatusNotImplemented)
	}
}
