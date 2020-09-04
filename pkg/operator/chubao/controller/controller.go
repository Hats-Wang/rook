package controller

import (
	"fmt"
	"github.com/coreos/pkg/capnslog"
	rookScheme "github.com/rook/rook/pkg/client/clientset/versioned/scheme"
	rookinformers "github.com/rook/rook/pkg/client/informers/externalversions"
	"github.com/rook/rook/pkg/clusterd"
	"github.com/rook/rook/pkg/operator/chubao/cluster"
	"github.com/rook/rook/pkg/operator/chubao/constants"
	"github.com/rook/rook/pkg/operator/chubao/monitor"
	"github.com/rook/rook/pkg/operator/chubao/objectstore"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/informers/internalinterfaces"
	"k8s.io/client-go/kubernetes/scheme"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"time"
)

var logger = capnslog.NewPackageLogger("github.com/rook/rook", "rook-chubao-controller")

// ClusterController encapsulates all the tools the controller needs
// in order to talk to the Kubernetes API
type ClusterController struct {
	context                 *clusterd.Context
	rookInformerFactory     rookinformers.SharedInformerFactory
	kubeInformerFactory     kubeinformers.SharedInformerFactory
	recorder                record.EventRecorder
	clusterHandler          *cluster.ClusterEventHandler
	clusterListerSynced     cache.InformerSynced
	monitorHandler          *monitor.MonitorEventHandler
	monitorListerSynced     cache.InformerSynced
	objectStoreHandler      *objectstore.ObjectStoreEventHandler
	objectStoreListerSynced cache.InformerSynced
}

// New returns a new ClusterController
func New(context *clusterd.Context, operatorNamespace string) *ClusterController {
	// Add sample-controller types to the default Kubernetes Scheme so Events can be
	// logged for sample-controller types.
	if err := rookScheme.AddToScheme(scheme.Scheme); err != nil {
		logger.Errorf("failed to add to the default kubernetes scheme. %v", err)
	}

	// Only watch kubernetes resources relevant to our app
	var tweakListOptionsFunc internalinterfaces.TweakListOptionsFunc
	tweakListOptionsFunc = func(options *metav1.ListOptions) {
		options.LabelSelector = fmt.Sprintf("%s=%s", "app", constants.AppName)
	}

	kubeInformerFactory := kubeinformers.NewSharedInformerFactoryWithOptions(context.Clientset, 0, kubeinformers.WithTweakListOptions(tweakListOptionsFunc))
	rookInformerFactory := rookinformers.NewSharedInformerFactory(context.RookClientset, 0)
	clusterInformer := rookInformerFactory.Chubao().V1alpha1().ChubaoClusters()
	objectStoreInformer := rookInformerFactory.Chubao().V1alpha1().ChubaoObjectStores()
	monitorInformer := rookInformerFactory.Chubao().V1alpha1().ChubaoMonitors()

	// Create event broadcaster
	logger.Infof("creating event broadcaster...")
	eventBroadcaster := record.NewBroadcaster()
	//eventBroadcaster.StartLogging(logger.Infof)
	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: context.Clientset.CoreV1().Events("")})
	recorder := eventBroadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: operatorNamespace})

	// Add event handling functions
	clusterHandler := cluster.New(context, kubeInformerFactory, clusterInformer, recorder)
	clusterInformer.Informer().AddEventHandler(clusterHandler)

	monitorHandler := monitor.New(context, recorder)
	monitorInformer.Informer().AddEventHandler(monitorHandler)

	objectStoreHandler := objectstore.New(context, recorder)
	objectStoreInformer.Informer().AddEventHandler(objectStoreHandler)

	cc := &ClusterController{
		context:                 context,
		recorder:                recorder,
		rookInformerFactory:     rookInformerFactory,
		kubeInformerFactory:     kubeInformerFactory,
		clusterHandler:          clusterHandler,
		clusterListerSynced:     clusterInformer.Informer().HasSynced,
		monitorHandler:          monitorHandler,
		monitorListerSynced:     monitorInformer.Informer().HasSynced,
		objectStoreHandler:      objectStoreHandler,
		objectStoreListerSynced: objectStoreInformer.Informer().HasSynced,
	}

	return cc
}

// Run starts the ClusterController process loop
func (cc *ClusterController) Run(threadnum int, stopCh <-chan struct{}) error {
	defer runtime.HandleCrash()
	go cc.kubeInformerFactory.Start(stopCh)
	go cc.rookInformerFactory.Start(stopCh)

	// 	Start the informer factories to begin populating the informer caches
	logger.Info("starting chubao controller")

	// Wait for the caches to be synced before starting workers
	logger.Info("waiting for informers caches to sync...")
	if ok := cache.WaitForCacheSync(
		stopCh,
		cc.clusterListerSynced,
		cc.monitorListerSynced,
		cc.objectStoreListerSynced,
	); !ok {
		return fmt.Errorf("failed to wait for caches to sync")
	}

	logger.Info("starting workers")
	for i := 0; i < threadnum; i++ {
		go wait.Until(cc.clusterHandler.RunWorker, time.Second, stopCh)
		go wait.Until(cc.monitorHandler.RunWorker, time.Second, stopCh)
		go wait.Until(cc.objectStoreHandler.RunWorker, time.Second, stopCh)
	}
	logger.Info("started workers")
	return nil
}
