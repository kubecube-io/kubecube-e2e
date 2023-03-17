/*
Copyright 2022 KubeCube Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package worker

import (
	"time"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/workqueue"

	"github.com/kubecube-io/kubecube/pkg/clog"
)

const (
	// maxRetries is the number of times a resource will be retried before it is dropped out of the queue.
	// With the current rate-limiter in use (5ms*2^(maxRetries-1)) the following numbers represent the times
	// a resource is going to be requeued:
	//
	// 5ms, 10ms, 20ms, 40ms, 80ms, 160ms, 320ms, 640ms, 1.3s, 2.6s, 5.1s, 10.2s, 20.4s, 41s, 82s
	maxRetries = 15
)

// Interface is a worker to process resources periodic with a rateLimitingQueue.
type Interface interface {
	// AddRateLimited adds item to queue.
	AddRateLimited(item interface{})
	// EnqueueRateLimited generates the key for objects then adds the key as an item to queue.
	EnqueueRateLimited(obj runtime.Object)
	Run(workerNumber int, stopChan <-chan struct{})
}

// QueueKey is the item key that stores in queue.
// The key could be arbitrary types.
//
// In some cases, people would like store different resources in a same queue, the traditional full-qualified key,
// such as '<namespace>/<name>', can't distinguish which resource the key belongs to, the key might carry more information
// of a resource, such as GVK(Group Version Kind), in that cases people need to use self-defined key, e.g. a struct.
type QueueKey interface{}

// KeyFunc knows how to make a key from an object. Implementations should be deterministic.
type KeyFunc func(obj interface{}) (QueueKey, error)

// ReconcileFunc knows how to consume items(key) from the queue.
type ReconcileFunc func(key QueueKey) error

type workerImpl struct {
	// keyFunc is the function that make keys for API objects.
	keyFunc KeyFunc
	// reconcileFunc is the function that process keys from the queue.
	reconcileFunc ReconcileFunc
	// queue allowing parallel processing of resources.
	queue workqueue.RateLimitingInterface
	// interval is the interval for process object in the queue.
	interval time.Duration
}

// New returns a workerImpl which can process resource periodic.
func New(name string, interval time.Duration, keyFunc KeyFunc, reconcileFunc ReconcileFunc) Interface {
	return &workerImpl{
		keyFunc:       keyFunc,
		reconcileFunc: reconcileFunc,
		queue:         workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), name),
		interval:      interval,
	}
}

func (w *workerImpl) EnqueueRateLimited(obj runtime.Object) {
	key, err := w.keyFunc(obj)
	if err != nil {
		clog.Warn("Failed to generate key for obj: %s", obj.GetObjectKind().GroupVersionKind())
		return
	}

	w.AddRateLimited(key)
}

func (w *workerImpl) AddRateLimited(item interface{}) {
	if item == nil {
		clog.Warn("Ignore nil item from queue")
		return
	}

	w.queue.AddRateLimited(item)
}

func (w *workerImpl) handleError(err error, key interface{}) {
	if err == nil || errors.HasStatusCause(err, v1.NamespaceTerminatingCause) {
		w.queue.Forget(key)
		return
	}

	if w.queue.NumRequeues(key) < maxRetries {
		w.queue.AddRateLimited(key)
		return
	}

	clog.Info("Dropping resource %q out of the queue: %v", key, err)
	w.queue.Forget(key)
}

func (w *workerImpl) worker() {
	key, quit := w.queue.Get()
	if quit {
		return
	}
	defer w.queue.Done(key)

	err := w.reconcileFunc(key)
	w.handleError(err, key)
}

func (w *workerImpl) Run(workerNumber int, stopChan <-chan struct{}) {
	for i := 0; i < workerNumber; i++ {
		go wait.Until(w.worker, w.interval, stopChan)
	}
	// Ensure all goroutines are cleaned up when the stop channel closes
	go func() {
		<-stopChan
		w.queue.ShutDown()
	}()
}
