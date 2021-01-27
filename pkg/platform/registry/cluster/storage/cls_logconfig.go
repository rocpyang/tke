/*
 * Tencent is pleased to support the open source community by making TKEStack
 * available.
 *
 * Copyright (C) 2012-2019 Tencent. All Rights Reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License"); you may not use
 * this file except in compliance with the License. You may obtain a copy of the
 * License at
 *
 * https://opensource.org/licenses/Apache-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
 * WARRANTIES OF ANY KIND, either express or implied.  See the License for the
 * specific language governing permissions and limitations under the License.
 */

package storage

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"
	"tkestack.io/tke/pkg/util/log"

	"tkestack.io/tke/pkg/platform/util"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	netutil "k8s.io/apimachinery/pkg/util/net"
	"k8s.io/apiserver/pkg/registry/generic/registry"
	"k8s.io/apiserver/pkg/registry/rest"
	platforminternalclient "tkestack.io/tke/api/client/clientset/internalversion/typed/platform/internalversion"
	"tkestack.io/tke/api/platform"
)


// CLSLogConfigREST implements proxy LogConfig request to cluster of user.
type CLSLogConfigREST struct {
	rest.Storage
	store          *registry.Store
	platformClient platforminternalclient.PlatformInterface
}

// ConnectMethods returns the list of HTTP methods that can be proxied
func (r *CLSLogConfigREST) ConnectMethods() []string {
	return []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"}
}

// NewConnectOptions returns versioned resource that represents proxy parameters
func (r *CLSLogConfigREST) NewConnectOptions() (runtime.Object, bool, string) {
	return &platform.CLSLogConfigProxyOptions{}, false, ""
}

// Connect returns a handler for the kube-apiserver proxy
func (r *CLSLogConfigREST) Connect(ctx context.Context, clusterName string, opts runtime.Object, responder rest.Responder) (http.Handler, error) {
	clusterObject, err := r.store.Get(ctx, clusterName, &metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	cluster := clusterObject.(*platform.Cluster)
	if err := util.FilterCluster(ctx, cluster); err != nil {
		return nil, err
	}
	proxyOpts := opts.(*platform.CLSLogConfigProxyOptions)

	location, transport, token, err := util.APIServerLocationByCluster(ctx, cluster, r.platformClient)
	if err != nil {
		return nil, err
	}
	return &clsLogConfigProxyHandler{
		location:  location,
		transport: transport,
		token:     token,
		name:      proxyOpts.Name,
	}, nil
}

// New creates a new LogConfig proxy options object
func (r *CLSLogConfigREST) New() runtime.Object {
	return &platform.CLSLogConfigProxyOptions{}
}

type clsLogConfigProxyHandler struct {
	transport http.RoundTripper
	location  *url.URL
	token     string
	name      string
}

func (h *clsLogConfigProxyHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	log.Errorf("debug-","debug")

	loc := *h.location
	loc.RawQuery = req.URL.RawQuery

	prefix := "/apis/cls.cloud.tencent.com/v1/logconfigs"

	if len(h.name) == 0 {
		loc.Path = fmt.Sprintf("%s", prefix)
	} else {
		loc.Path = fmt.Sprintf("%s/%s", prefix, h.name)
	}

	// WithContext creates a shallow clone of the request with the new context.
	newReq := req.WithContext(context.Background())
	newReq.Header = netutil.CloneHeader(req.Header)
	newReq.URL = &loc
	if h.token != "" {
		newReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", strings.TrimSpace(h.token)))
	}

	reserveProxy := httputil.NewSingleHostReverseProxy(&url.URL{Scheme: h.location.Scheme, Host: h.location.Host})
	reserveProxy.Transport = h.transport
	reserveProxy.FlushInterval = 100 * time.Millisecond
	reserveProxy.ServeHTTP(w, newReq)
}
