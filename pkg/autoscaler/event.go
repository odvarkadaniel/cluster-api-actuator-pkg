package autoscaler

import (
	"sync"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

type matchEventFunc func(event *corev1.Event) bool
type eventHandlerFunc func(event *corev1.Event)

type eventWatcher struct {
	stopCh          chan struct{}
	informerFactory informers.SharedInformerFactory
	eventInformer   cache.SharedIndexInformer
	startTime       metav1.Time

	eventHandlerLock sync.Mutex
	eventHandlers    []*eventHandler
}

type eventHandler struct {
	sync.Mutex

	matcher matchEventFunc
	handler eventHandlerFunc
	enabled bool
}

func newEventWatcher(clientset kubernetes.Interface) *eventWatcher {
	w := eventWatcher{
		stopCh:          make(chan struct{}),
		startTime:       metav1.Now(),
		informerFactory: informers.NewSharedInformerFactory(clientset, 0),
	}

	w.eventInformer = w.informerFactory.Core().V1().Events().Informer()
	w.eventInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			event, ok := obj.(*corev1.Event)
			if !ok {
				panic("expected to get an of object of type corev1.Event")
			}

			if event.CreationTimestamp.Before(&w.startTime) {
				return
			}

			w.eventHandlerLock.Lock()
			defer w.eventHandlerLock.Unlock()

			for _, h := range w.eventHandlers {
				h.Lock()
				if h.enabled && h.matcher(event) {
					h.handler(event)
				}
				h.Unlock()
			}
		},
	})

	return &w
}

func (w *eventWatcher) run() bool {
	w.informerFactory.Start(w.stopCh)
	return cache.WaitForCacheSync(w.stopCh, w.eventInformer.HasSynced)
}

func (w *eventWatcher) stop() {
	close(w.stopCh)
}

func (w *eventWatcher) onEvent(matcher matchEventFunc, handler eventHandlerFunc) *eventHandler {
	h := &eventHandler{
		matcher: matcher,
		handler: handler,
	}

	w.eventHandlerLock.Lock()
	defer w.eventHandlerLock.Unlock()
	w.eventHandlers = append(w.eventHandlers, h)

	return h
}

func (h *eventHandler) enable() {
	h.Lock()
	defer h.Unlock()
	h.enabled = true
}

func matchAnyEvent(_ *corev1.Event) bool {
	return true
}
