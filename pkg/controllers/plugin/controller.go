package plugin

import (
	"context"

	v1 "github.com/rancher/ui-plugin-operator/pkg/apis/catalog.cattle.io/v1"
	plugincontroller "github.com/rancher/ui-plugin-operator/pkg/generated/controllers/catalog.cattle.io/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	FilesTxtFilename = "files.txt"
)

func Register(
	ctx context.Context,
	systemNamespace, managedBy string,
	plugin plugincontroller.UIPluginController,
	pluginCache plugincontroller.UIPluginCache,
	k8s kubernetes.Interface,
) {
	h := &handler{
		systemNamespace: systemNamespace,
		managedBy:       managedBy,
		plugin:          plugin,
		pluginCache:     pluginCache,
	}
	plugin.OnChange(ctx, "on-plugin-change", h.OnPluginChange)
}

type handler struct {
	systemNamespace string
	managedBy       string
	plugin          plugincontroller.UIPluginController
	pluginCache     plugincontroller.UIPluginCache
}

func (h *handler) OnPluginChange(key string, plugin *v1.UIPlugin) (*v1.UIPlugin, error) {
	err := Index.Generate(h.systemNamespace, h.pluginCache)
	if err != nil {
		return plugin, err
	}
	Index.SyncWithFsCache()
	if plugin == nil {
		return plugin, nil
	}
	defer h.plugin.UpdateStatus(plugin)
	if plugin.Spec.Plugin.NoCache {
		plugin.Status.CacheState = Disabled
	} else {
		plugin.Status.CacheState = Pending
	}
	err = FsCache.Sync(h.systemNamespace, h.pluginCache)
	if err != nil {
		return plugin, err
	}
	if !plugin.Spec.Plugin.NoCache {
		plugin.Status.CacheState = Cached
	}

	return plugin, nil
}
