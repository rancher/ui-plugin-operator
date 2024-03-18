package plugin

import (
	"context"

	v1 "github.com/rancher/ui-plugin-operator/pkg/apis/catalog.cattle.io/v1"
	plugincontroller "github.com/rancher/ui-plugin-operator/pkg/generated/controllers/catalog.cattle.io/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
)

func Register(
	ctx context.Context,
	systemNamespace, managedBy string,
	plugin plugincontroller.UIPluginController,
	pluginCache plugincontroller.UIPluginCache,
	_ kubernetes.Interface,
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

func (h *handler) OnPluginChange(_ string, plugin *v1.UIPlugin) (*v1.UIPlugin, error) {
	cachedPlugins, err := h.pluginCache.List(h.systemNamespace, labels.Everything())
	if err != nil {
		return plugin, err
	}
	err = Index.Generate(cachedPlugins)
	if err != nil {
		return plugin, err
	}
	pattern := FSCacheRootDir + "/*/*"
	fsCacheFiles, err := fsCacheFilepathGlob(pattern)
	if err != nil {
		return plugin, err
	}
	FsCache.SyncWithIndex(&Index, fsCacheFiles)
	if plugin == nil {
		return plugin, nil
	}
	defer h.plugin.UpdateStatus(plugin)
	if plugin.Spec.Plugin.NoCache {
		plugin.Status.CacheState = Disabled
	} else {
		plugin.Status.CacheState = Pending
	}
	err = FsCache.SyncWithControllersCache(cachedPlugins)
	if err != nil {
		return plugin, err
	}
	if !plugin.Spec.Plugin.NoCache {
		plugin.Status.CacheState = Cached
	}

	return plugin, nil
}
