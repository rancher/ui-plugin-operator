package controllers

import (
	"context"
	"errors"
	"time"

	"github.com/rancher/lasso/pkg/cache"
	"github.com/rancher/lasso/pkg/client"
	"github.com/rancher/lasso/pkg/controller"
	"github.com/rancher/ui-plugin-operator/pkg/controllers/plugin"
	catalog "github.com/rancher/ui-plugin-operator/pkg/generated/controllers/catalog.cattle.io"
	plugincontroller "github.com/rancher/ui-plugin-operator/pkg/generated/controllers/catalog.cattle.io/v1"
	"github.com/rancher/wrangler/pkg/generic"
	"github.com/rancher/wrangler/pkg/leader"
	"github.com/rancher/wrangler/pkg/ratelimit"
	"github.com/rancher/wrangler/pkg/start"
	"github.com/sirupsen/logrus"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/workqueue"
)

type appContext struct {
	plugincontroller.Interface
	K8s      kubernetes.Interface
	starters []start.Starter
}

func (a *appContext) start(ctx context.Context) error {
	return start.All(ctx, 50, a.starters...)
}

func Register(ctx context.Context, systemNamespace, controllerName, _ string, cfg clientcmd.ClientConfig) error {
	if len(systemNamespace) == 0 {
		return errors.New("cannot start controllers on system namespace: system namespace not provided")
	}
	appCtx, err := newContext(ctx, systemNamespace, cfg)
	if err != nil {
		return err
	}
	if len(controllerName) == 0 {
		controllerName = "plugin-operator"
	}
	plugin.Register(ctx,
		systemNamespace,
		controllerName,
		appCtx.UIPlugin(),
		appCtx.UIPlugin().Cache(),
		appCtx.K8s,
	)
	leader.RunOrDie(ctx, systemNamespace, "plugin-operator-lock", appCtx.K8s, func(ctx context.Context) {
		if err := appCtx.start(ctx); err != nil {
			logrus.Fatal(err)
		}
		logrus.Info("All controllers have been started")
	})

	return nil
}

func controllerFactory(rest *rest.Config) (controller.SharedControllerFactory, error) {
	rateLimit := workqueue.NewItemExponentialFailureRateLimiter(5*time.Millisecond, 60*time.Second)
	clientFactory, err := client.NewSharedClientFactory(rest, nil)
	if err != nil {
		return nil, err
	}

	cacheFactory := cache.NewSharedCachedFactory(clientFactory, nil)
	return controller.NewSharedControllerFactory(cacheFactory, &controller.SharedControllerFactoryOptions{
		DefaultRateLimiter: rateLimit,
		DefaultWorkers:     50,
	}), nil
}

func newContext(_ context.Context, systemNamespace string, cfg clientcmd.ClientConfig) (*appContext, error) {
	client, err := cfg.ClientConfig()
	if err != nil {
		return nil, err
	}
	client.RateLimiter = ratelimit.None

	k8s, err := kubernetes.NewForConfig(client)
	if err != nil {
		return nil, err
	}

	scf, err := controllerFactory(client)
	if err != nil {
		return nil, err
	}

	plugin, err := catalog.NewFactoryFromConfigWithOptions(client, &generic.FactoryOptions{
		Namespace:               systemNamespace,
		SharedControllerFactory: scf,
	})
	if err != nil {
		return nil, err
	}
	pluginv := plugin.Catalog().V1()

	return &appContext{
		Interface: pluginv,
		K8s:       k8s,
		starters: []start.Starter{
			plugin,
		},
	}, nil
}
