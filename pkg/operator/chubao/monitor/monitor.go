package monitor

import (
	"fmt"
	"github.com/coreos/pkg/capnslog"
	"github.com/pkg/errors"
	chubaorookio "github.com/rook/rook/pkg/apis/chubao.rook.io"
	chubaoapi "github.com/rook/rook/pkg/apis/chubao.rook.io/v1alpha1"
	informers "github.com/rook/rook/pkg/client/informers/externalversions/chubao.rook.io/v1alpha1"
	listers "github.com/rook/rook/pkg/client/listers/chubao.rook.io/v1alpha1"
	"github.com/rook/rook/pkg/clusterd"
	"github.com/rook/rook/pkg/operator/chubao/commons"
	"github.com/rook/rook/pkg/operator/chubao/monitor/grafana"
	"github.com/rook/rook/pkg/operator/chubao/monitor/prometheus"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/runtime"
	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
	"reflect"
)

const (
	DefaultNamespace = "default"
	componentName    = "monitor"
	monitorQueueName = "chubao-monitor-queue"
)

var logger = capnslog.NewPackageLogger("github.com/rook/rook", "chubao-controller-monitor")

type MonitorEventHandler struct {
	cache.ResourceEventHandler

	context             *clusterd.Context
	monitorInformer     informers.ChubaoMonitorInformer
	monitorLister       listers.ChubaoMonitorLister
	kubeInformerFactory kubeinformers.SharedInformerFactory
	queue               workqueue.RateLimitingInterface
	recorder            record.EventRecorder
}

func New(
	context *clusterd.Context,
	kubeInformerFactory kubeinformers.SharedInformerFactory,
	monitorInformer informers.ChubaoMonitorInformer,
	recorder record.EventRecorder,
) *MonitorEventHandler {
	return &MonitorEventHandler{
		context:             context,
		monitorInformer:     monitorInformer,
		monitorLister:       monitorInformer.Lister(),
		kubeInformerFactory: kubeInformerFactory,
		recorder:            recorder,
		queue:               workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), monitorQueueName),
	}
}

// OnAdd calls AddFunc if it's not nil.
func (e *MonitorEventHandler) OnAdd(obj interface{}) {
	newChubaomonitor, ok := obj.(*chubaoapi.ChubaoMonitor)
	if !ok {
		return
	}
	commons.EnqueueItem(e.queue, newChubaomonitor)
}

// OnUpdate calls UpdateFunc if it's not nil.
func (e *MonitorEventHandler) OnUpdate(oldObj, newObj interface{}) {
	newMonitor, ok := newObj.(*chubaoapi.ChubaoMonitor)
	if !ok {
		return
	}
	oldMonitor, ok := oldObj.(*chubaoapi.ChubaoMonitor)
	if !ok {
		return
	}

	// If the Spec is the same as the one in our cache, there aren't
	// any changes we are interested in.
	if reflect.DeepEqual(newMonitor.Spec, oldMonitor.Spec) {
		return
	}

	commons.EnqueueItem(e.queue, newMonitor)
}

// OnDelete calls DeleteFunc if it's not nil.
func (e *MonitorEventHandler) OnDelete(obj interface{}) {
	newMonitor, ok := obj.(*chubaoapi.ChubaoCluster)
	if !ok {
		return
	}

	key, _ := cache.MetaNamespaceKeyFunc(newMonitor)
	logger.Infof("delete Monitor object: %s", key)
}

func (e *MonitorEventHandler) RunWorker() {
	for commons.ProcessNextWorkItem(e.queue, e.workFunc) {
	}
}

func (e *MonitorEventHandler) workFunc(key string) error {
	// Convert the namespace/name string into a distinct namespace and name.
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		runtime.HandleError(fmt.Errorf("invalid resource key: %s\n", key))
		return nil
	}

	lister := e.monitorInformer.Lister()
	// Get the Monitor resource with this namespace/name
	monitor, err := lister.ChubaoMonitors(namespace).Get(name)
	if err != nil {
		// The Monitor resource may no longer exist, in which case we stop processing.
		if apierrors.IsNotFound(err) {
			runtime.HandleError(fmt.Errorf("monitor '%s' in work queue no longer exists\n", key))
			return nil
		}
		return fmt.Errorf("Unexpected error while getting monitor object: %s\n", err)
	}
	monitor.Status.Prometheus = chubaoapi.PrometheusStatusUnknown
	monitor.Status.Grafana = chubaoapi.GrafanaStatusUnknown

	logger.Infof("handling monitor object: %s", key)
	// DeepCopy here to ensure nobody messes with the cache.
	oldObj, newObj := monitor, monitor.DeepCopy()
	// If sync was successful and Status has changed, update the Monitor.
	if err = e.sync(newObj); err == nil && !reflect.DeepEqual(oldObj.Status, newObj.Status) {
		oldObj.Status = newObj.Status
		clientSet := e.context.RookClientset
		_, err := clientSet.ChubaoV1alpha1().ChubaoMonitors(namespace).Update(oldObj)
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("failed to update prometheus and grafana status"))
		}
	}

	return err
}

func (e *MonitorEventHandler) sync(monitor *chubaoapi.ChubaoMonitor) error {
	var err error
	if monitor.DeletionTimestamp.IsZero() {
		// new
		err = e.createMonitor(monitor)
	} else {
		// delete
		err = e.deleteMonitor(monitor)
	}

	return err
}

func (e *MonitorEventHandler) deleteMonitor(monitor *chubaoapi.ChubaoMonitor) error {
	monitor.Status.Grafana = chubaoapi.GrafanaStatusFailure
	monitor.Status.Prometheus = chubaoapi.PrometheusStatusFailure
	fmt.Printf("deleteMonitor: %v\n", monitor)
	return nil
}

func (e *MonitorEventHandler) createMonitor(monitor *chubaoapi.ChubaoMonitor) error {
	ownerRef := newMonitorOwnerRef(monitor)
	prom := prometheus.New(e.context, e.kubeInformerFactory, e.recorder, monitor, ownerRef)
	if err := prom.Deploy(); err != nil {
		monitor.Status.Prometheus = chubaoapi.PrometheusStatusFailure
		return errors.Wrap(err, "failed to start prometheus")
	}
	monitor.Status.Prometheus = chubaoapi.PrometheusStatusReady

	graf := grafana.New(e.context, e.kubeInformerFactory, e.recorder, monitor, ownerRef)
	if err := graf.Deploy(); err != nil {
		monitor.Status.Grafana = chubaoapi.GrafanaStatusFailure
		return errors.Wrap(err, "failed to start grafana")
	}
	monitor.Status.Grafana = chubaoapi.GrafanaStatusReady
	return nil
}

func newMonitorOwnerRef(own metav1.Object) metav1.OwnerReference {
	return *metav1.NewControllerRef(own, schema.GroupVersionKind{
		Group:   chubaorookio.CustomResourceGroupName,
		Version: "v1alpha1",
		Kind:    "ChubaoMonitor",
	})
}
