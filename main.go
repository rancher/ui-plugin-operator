package main

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	_ "net/http/pprof"
	neturl "net/url"
	"strings"

	"github.com/gorilla/mux"
	"github.com/rancher/ui-plugin-operator/pkg/controllers"
	"github.com/rancher/ui-plugin-operator/pkg/controllers/plugin"
	"github.com/rancher/ui-plugin-operator/pkg/crd"
	"github.com/rancher/ui-plugin-operator/pkg/version"
	command "github.com/rancher/wrangler-cli"
	_ "github.com/rancher/wrangler/pkg/generated/controllers/apiextensions.k8s.io"
	_ "github.com/rancher/wrangler/pkg/generated/controllers/networking.k8s.io"
	"github.com/rancher/wrangler/pkg/kubeconfig"
	"github.com/rancher/wrangler/pkg/ratelimit"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	yaml "gopkg.in/yaml.v3"
)

var (
	debugConfig command.DebugConfig
)

type PluginOperator struct {
	Kubeconfig     string `usage:"Kubeconfig file" env:"KUBECONFIG"`
	Namespace      string `usage:"Namespace to watch for UIPlugins" default:"cattle-ui-plugin-system" env:"NAMESPACE"`
	ControllerName string `usage:"Unique name to identify this controller that is added to all UIPlugins tracked by this controller" default:"ui-plugin-operator" env:"CONTROLLER_NAME"`
	NodeName       string `usage:"Name of the node this controller is running on" env:"NODE_NAME"`
}

func (a *PluginOperator) Run(cmd *cobra.Command, args []string) error {
	if len(a.Namespace) == 0 {
		return fmt.Errorf("helm-locker can only be started in a single namespace")
	}

	go func() {
		logrus.Println(http.ListenAndServe(":6060", nil))
	}()
	debugConfig.MustSetupDebug()

	cfg := kubeconfig.GetNonInteractiveClientConfig(a.Kubeconfig)
	clientConfig, err := cfg.ClientConfig()
	if err != nil {
		return err
	}
	clientConfig.RateLimiter = ratelimit.None

	ctx := cmd.Context()
	if err := crd.Create(ctx, clientConfig); err != nil {
		return err
	}

	r := mux.NewRouter()
	r.HandleFunc("/", indexHandler)
	r.HandleFunc("/index.yaml", indexHandler)
	r.HandleFunc("/{name}/{version}/{rest:.*}", pluginHandler)
	http.Handle("/", r)

	go func() {
		log.Println(http.ListenAndServe(":8080", nil))
	}()

	if err := controllers.Register(ctx, a.Namespace, a.ControllerName, a.NodeName, cfg); err != nil {
		return err
	}

	<-cmd.Context().Done()
	return nil
}

func main() {
	cmd := command.Command(&PluginOperator{}, cobra.Command{
		Version: version.FriendlyVersion(),
	})
	cmd = command.AddDebug(cmd, &debugConfig)
	command.Main(cmd)
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	index, err := yaml.Marshal(&plugin.Index)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		logrus.Error(err)
	}
	w.Write(index)
}

func pluginHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	logrus.Debugf("http request vars %s", vars)
	entry, ok := plugin.Index.Entries[vars["name"]]
	if !ok || entry.Version != vars["version"] {
		msg := fmt.Sprintf("plugin [name: %s version: %s] does not exist in index", vars["name"], vars["version"])
		http.Error(w, msg, http.StatusNotFound)
		logrus.Debug(msg)
		return
	}
	if entry.NoCache {
		logrus.Debugf("[noCache: %v] proxying request to [endpoint: %v]\n", entry.NoCache, entry.Endpoint)
		proxyRequest(entry.Endpoint, vars["rest"], w, r)
	} else {
		logrus.Debugf("[noCache: %v] serving plugin files from filesystem cache\n", entry.NoCache)
		http.FileServer(http.Dir(plugin.FSCacheRootDir)).ServeHTTP(w, r)
	}
}

func proxyRequest(target, path string, w http.ResponseWriter, r *http.Request) {
	url, err := neturl.Parse(target)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to parse url [%s]", target), http.StatusInternalServerError)
		return
	}
	if denylist(url) {
		http.Error(w, fmt.Sprintf("url [%s] is forbidden", target), http.StatusForbidden)
		return
	}
	proxy := httputil.NewSingleHostReverseProxy(url)
	r.URL.Host = url.Host
	r.URL.Scheme = url.Scheme
	r.URL.Path = path
	r.Header.Set("X-Forwarded-Host", r.Header.Get("Host"))
	r.Host = url.Host
	proxy.ServeHTTP(w, r)
}

func denylist(url *neturl.URL) bool {
	// temp: is there a way to check if an IP equivalent to localhost is being used?
	denied := map[string]struct{}{
		"localhost": {},
		"127.0.0.1": {},
		"0.0.0.0":   {},
		"":          {},
	}
	host := strings.Split(url.Host, ":")[0]
	_, isDenied := denied[host]

	return isDenied
}
