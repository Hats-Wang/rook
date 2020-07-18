/*
Copyright The Kubernetes Authors.

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

// Code generated by lister-gen. DO NOT EDIT.

package v1alpha1

import (
	v1alpha1 "github.com/rook/rook/pkg/apis/chubao.rook.io/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"
)

// ChubaoMonitorLister helps list ChubaoMonitors.
type ChubaoMonitorLister interface {
	// List lists all ChubaoMonitors in the indexer.
	List(selector labels.Selector) (ret []*v1alpha1.ChubaoMonitor, err error)
	// ChubaoMonitors returns an object that can list and get ChubaoMonitors.
	ChubaoMonitors(namespace string) ChubaoMonitorNamespaceLister
	ChubaoMonitorListerExpansion
}

// chubaoMonitorLister implements the ChubaoMonitorLister interface.
type chubaoMonitorLister struct {
	indexer cache.Indexer
}

// NewChubaoMonitorLister returns a new ChubaoMonitorLister.
func NewChubaoMonitorLister(indexer cache.Indexer) ChubaoMonitorLister {
	return &chubaoMonitorLister{indexer: indexer}
}

// List lists all ChubaoMonitors in the indexer.
func (s *chubaoMonitorLister) List(selector labels.Selector) (ret []*v1alpha1.ChubaoMonitor, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*v1alpha1.ChubaoMonitor))
	})
	return ret, err
}

// ChubaoMonitors returns an object that can list and get ChubaoMonitors.
func (s *chubaoMonitorLister) ChubaoMonitors(namespace string) ChubaoMonitorNamespaceLister {
	return chubaoMonitorNamespaceLister{indexer: s.indexer, namespace: namespace}
}

// ChubaoMonitorNamespaceLister helps list and get ChubaoMonitors.
type ChubaoMonitorNamespaceLister interface {
	// List lists all ChubaoMonitors in the indexer for a given namespace.
	List(selector labels.Selector) (ret []*v1alpha1.ChubaoMonitor, err error)
	// Get retrieves the ChubaoMonitor from the indexer for a given namespace and name.
	Get(name string) (*v1alpha1.ChubaoMonitor, error)
	ChubaoMonitorNamespaceListerExpansion
}

// chubaoMonitorNamespaceLister implements the ChubaoMonitorNamespaceLister
// interface.
type chubaoMonitorNamespaceLister struct {
	indexer   cache.Indexer
	namespace string
}

// List lists all ChubaoMonitors in the indexer for a given namespace.
func (s chubaoMonitorNamespaceLister) List(selector labels.Selector) (ret []*v1alpha1.ChubaoMonitor, err error) {
	err = cache.ListAllByNamespace(s.indexer, s.namespace, selector, func(m interface{}) {
		ret = append(ret, m.(*v1alpha1.ChubaoMonitor))
	})
	return ret, err
}

// Get retrieves the ChubaoMonitor from the indexer for a given namespace and name.
func (s chubaoMonitorNamespaceLister) Get(name string) (*v1alpha1.ChubaoMonitor, error) {
	obj, exists, err := s.indexer.GetByKey(s.namespace + "/" + name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(v1alpha1.Resource("chubaomonitor"), name)
	}
	return obj.(*v1alpha1.ChubaoMonitor), nil
}