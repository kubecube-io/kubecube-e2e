/*
Copyright 2023 KubeCube Authors

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

package workloads_new

import (
	"fmt"
	"net/url"
	"strings"
)

const ProxyUrlPrefix = "/api/v1/cube/proxy"

func BuildK8sProxyUrl(host string, cluster string, gv string, namespace string, resource string, name string) string {
	var builder strings.Builder
	builder.Write([]byte(host))
	builder.Write([]byte(ProxyUrlPrefix))
	if len(cluster) > 0 {
		builder.WriteByte('/')
		builder.Write([]byte("clusters"))
		builder.WriteByte('/')
		builder.Write([]byte(cluster))
	}
	builder.WriteByte('/')
	builder.Write([]byte(gv))
	if len(namespace) > 0 {
		builder.WriteByte('/')
		builder.Write([]byte("namespaces"))
		builder.WriteByte('/')
		builder.Write([]byte(namespace))
	}
	builder.WriteByte('/')
	builder.Write([]byte(resource))
	if len(name) > 0 {
		builder.WriteByte('/')
		builder.Write([]byte(name))
	}
	return builder.String()
}

func BuildLogUrl(host string, cluster string, namespace string, podName string, containerName string) string {
	url := "%s/api/v1/cube/extend/clusters/%s/namespaces/%s/logs/%s?containerName=%s&offsetFrom=2000000&offsetTo=2000100&referenceLineNum=0&logFilePosition=end&referenceTimestamp=newest"
	url = fmt.Sprintf(url, host, cluster, namespace, podName, containerName)
	return url
}

func BuildEventUrl(host string, cluster string, namespace string, uid string) string {
	param := "involvedObject.uid=" + uid
	query := url.QueryEscape(param)
	url := host + "/api/v1/cube/proxy/clusters/" + cluster + "/api/v1/namespaces/" + namespace + "/events?fieldSelector" + query
	return url
}
