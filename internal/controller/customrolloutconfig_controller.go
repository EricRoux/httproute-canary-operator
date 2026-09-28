package controller

import (
	"context"
	// "os"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"

	corev1alpha1 "github.com/EricRoux/kub-controller/api/v1alpha1"
)

func (r *CustomRolloutConfigReconciler) syncHttpRoute(
	ctx context.Context,
	rollout *corev1alpha1.CustomRolloutConfig,
	config *corev1alpha1.CustomHttprouteConfig,
) error {
	logger := log.FromContext(ctx)

	routeName := rollout.Spec.TargetRef.Name + "-internal-route"

	var rules []gatewayv1.HTTPRouteRule
	for _, b := range config.Spec.Route.Backends {
		serviceName := rollout.Spec.TargetRef.Name
		for _, h := range config.Spec.Route.HeaderMatches {
			if h.ApplicationType == "" ||
				rollout.Spec.ApplicationType == "" ||
				h.ApplicationType == rollout.Spec.ApplicationType {
				singleMatch := gatewayv1.HTTPRouteMatch{
					Headers: []gatewayv1.HTTPHeaderMatch{
						{
							Name:  gatewayv1.HTTPHeaderName(h.Name),
							Value: h.Value,
							Type:  ptrTo(gatewayv1.HeaderMatchRegularExpression),
						},
					},
					Path: &gatewayv1.HTTPPathMatch{
						Type:  ptrTo(gatewayv1.PathMatchPathPrefix),
						Value: ptrTo(b.PathPrefix),
					},
				}

				singleBackendRef := gatewayv1.HTTPBackendRef{
					BackendRef: gatewayv1.BackendRef{
						BackendObjectReference: gatewayv1.BackendObjectReference{
							Group: ptrTo(gatewayv1.Group("")),
							Kind:  ptrTo(gatewayv1.Kind("Service")),
							Name:  gatewayv1.ObjectName(serviceName + "-" + rollout.Spec.ServiceCanaryPrefix),
							Port:  ptrTo(gatewayv1.PortNumber(b.Port)),
						},
						Weight: ptrTo(int32(100)),
					},
				}

				rules = append(rules, gatewayv1.HTTPRouteRule{
					Matches:     []gatewayv1.HTTPRouteMatch{singleMatch},
					BackendRefs: []gatewayv1.HTTPBackendRef{singleBackendRef},
				})
			}
		}
		if config.Spec.IncludeDefaultRoute {
			singleMatch := gatewayv1.HTTPRouteMatch{
				Path: &gatewayv1.HTTPPathMatch{
					Type:  ptrTo(gatewayv1.PathMatchPathPrefix),
					Value: ptrTo(b.PathPrefix),
				},
			}

			singleBackendRef := gatewayv1.HTTPBackendRef{
				BackendRef: gatewayv1.BackendRef{
					BackendObjectReference: gatewayv1.BackendObjectReference{
						Group: ptrTo(gatewayv1.Group("")),
						Kind:  ptrTo(gatewayv1.Kind("Service")),
						Name:  gatewayv1.ObjectName(serviceName),
						Port:  ptrTo(gatewayv1.PortNumber(b.Port)),
					},
					Weight: ptrTo(int32(100)),
				},
			}

			rules = append(rules, gatewayv1.HTTPRouteRule{
				Matches:     []gatewayv1.HTTPRouteMatch{singleMatch},
				BackendRefs: []gatewayv1.HTTPBackendRef{singleBackendRef},
			})
		}
	}

	labels := make(map[string]string)
	labels["app.kubernetes.io/name"] = routeName
	labels["app.kubernetes.io/instance"] = routeName
	labels["app.kubernetes.io/managed-by"] = "core.argo-rollout.canary.controller.io"
	if rollout.Spec.ApplicationType != "" {
		labels["app.kubernetes.io/component"] = rollout.Spec.ApplicationType
	}

	desiredRoute := &gatewayv1.HTTPRoute{
		ObjectMeta: metav1.ObjectMeta{
			Name:      routeName,
			Namespace: rollout.Namespace,
			Labels:    labels,
		},
		Spec: gatewayv1.HTTPRouteSpec{
			ParentRefs: []gatewayv1.ParentReference{
				{
					Group:     ptrTo(gatewayv1.Group("")),
					Kind:      ptrTo(gatewayv1.Kind("Service")),
					Name:      gatewayv1.ObjectName(rollout.Spec.TargetRef.Name),
					Namespace: (*gatewayv1.Namespace)(&config.Namespace),
				},
			},
			Rules: rules,
		},
	}

	if err := ctrl.SetControllerReference(rollout, desiredRoute, r.Scheme); err != nil {
		return err
	}

	existingRoute := &gatewayv1.HTTPRoute{}
	err := r.Get(ctx, types.NamespacedName{Name: routeName, Namespace: rollout.Namespace}, existingRoute)

	if err != nil {
		if errors.IsNotFound(err) {
			logger.Info("Creating a new HTTPRoute", "name", routeName)
			return r.Create(ctx, desiredRoute)
		}
		return err
	}

	logger.Info("HTTPRoute already exists, syncing spec", "name", routeName)
	existingRoute.Spec = desiredRoute.Spec
	return r.Update(ctx, existingRoute)
}

func ptrTo[T any](v T) *T {
	return &v
}

// CustomRolloutConfigReconciler reconciles a CustomRolloutConfig object
type CustomRolloutConfigReconciler struct {
	client.Client
	APIReader client.Reader
	Scheme    *runtime.Scheme
}

//+kubebuilder:rbac:groups=gateway.networking.k8s.io,resources=httproutes,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core.argo-rollout.canary.controller.io,resources=customrolloutconfigs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core.argo-rollout.canary.controller.io,resources=customrolloutconfigs/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=core.argo-rollout.canary.controller.io,resources=customhttprouteconfigs,verbs=get;list;watch

func (r *CustomRolloutConfigReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// 1. Получаем сам CustomRolloutConfig (из быстрого глобального кэша)
	var rolloutConfig corev1alpha1.CustomRolloutConfig
	if err := r.Get(ctx, req.NamespacedName, &rolloutConfig); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	var httpRouteConfigs corev1alpha1.CustomHttprouteConfigList
	if err := r.List(ctx, &httpRouteConfigs, client.InNamespace(req.Namespace)); err != nil {
		return ctrl.Result{}, err
	}

	if len(httpRouteConfigs.Items) == 0 {
		logger.Info("No CustomHttprouteConfig found in this namespace, waiting for creation")
		return ctrl.Result{}, nil
	}

	if len(httpRouteConfigs.Items) > 1 {
		logger.Error(nil, "Multiple CustomHttprouteConfigs found in the namespace! Only ONE is allowed.",
			"found_count", len(httpRouteConfigs.Items))

		return ctrl.Result{}, nil
	}

	httpRouteConfig := httpRouteConfigs.Items[0]

	if rolloutConfig.Spec.TargetRef.Kind != "Deployment" {
		logger.Error(nil, "Unsupported targetRef Kind, only Deployment is supported", "kind", rolloutConfig.Spec.TargetRef.Kind)
		return ctrl.Result{}, nil
	}

	logger.Info("Successfully fetched all configurations within the namespace",
		"namespace", req.Namespace,
		"targetDeployment", rolloutConfig.Spec.TargetRef.Name,
		"configName", httpRouteConfig.Name,
		"includeDefaultRoute", httpRouteConfig.Spec.IncludeDefaultRoute,
	)

	if err := r.syncHttpRoute(ctx, &rolloutConfig, &httpRouteConfig); err != nil {
		logger.Error(err, "Failed to sync HTTPRoute for rollout config",
			"rolloutConfigName", rolloutConfig.Name,
			"targetDeployment", rolloutConfig.Spec.TargetRef.Name,
		)
		return ctrl.Result{}, err
	}

	logger.Info("HTTPRoute successfully synchronized",
		"namespace", req.Namespace,
		"targetDeployment", rolloutConfig.Spec.TargetRef.Name,
	)

	return ctrl.Result{}, nil
}

func (r *CustomRolloutConfigReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1alpha1.CustomRolloutConfig{}).
		Watches(
			&corev1alpha1.CustomHttprouteConfig{},
			handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, obj client.Object) []reconcile.Request {
				var rolloutList corev1alpha1.CustomRolloutConfigList
				err := mgr.GetClient().List(ctx, &rolloutList, client.InNamespace(obj.GetNamespace()))
				if err != nil {
					return nil
				}

				var requests []reconcile.Request
				for _, item := range rolloutList.Items {
					requests = append(requests, reconcile.Request{
						NamespacedName: types.NamespacedName{
							Name:      item.Name,
							Namespace: item.Namespace,
						},
					})
				}
				return requests
			}),
		).
		Complete(r)
}
