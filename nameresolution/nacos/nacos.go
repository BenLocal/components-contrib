/*
Copyright 2021 The Dapr Authors
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

package nacos

import (
	"encoding/json"
	"fmt"

	"github.com/dapr/components-contrib/nameresolution"
	nr "github.com/dapr/components-contrib/nameresolution"
	dc "github.com/dapr/kit/config"
	"github.com/dapr/kit/logger"
	nacos "github.com/nacos-group/nacos-sdk-go/v2/clients"
	nacos_name "github.com/nacos-group/nacos-sdk-go/v2/clients/naming_client"
	"github.com/nacos-group/nacos-sdk-go/v2/common/constant"
	"github.com/nacos-group/nacos-sdk-go/v2/vo"
)

type config struct {
	Servers  []*serverConfig
	Client   *clientConfig
	Register *registerConfig
}

type serverConfig struct {
	Addr        string
	Port        uint64
	Scheme      string
	ContextPath string
}

type clientConfig struct {
	NamespaceId         string
	TimeoutMs           uint64
	NotLoadCacheAtStart bool
	LogDir              string
	CacheDir            string
	LogLevel            string
}

type registerConfig struct {
	Ip          string
	Port        uint64
	Weight      float64
	Enable      bool
	Healthy     bool
	Metadata    map[string]string
	ClusterName string
	ServiceName string
	GroupName   string
	Ephemeral   bool
}

func (c *config) decode(in interface{}) error {
	n, err := dc.Normalize(in)
	if err != nil {
		return err
	}

	data, err := json.Marshal(n)
	if err != nil {
		return err
	}

	return json.Unmarshal(data, &c)
}

type resolver struct {
	logger logger.Logger
	client nacos_name.INamingClient
}

// NewResolver creates nacos name resolver.
func NewResolver(logger logger.Logger) nr.Resolver {
	return &resolver{
		logger: logger,
	}
}

func (r *resolver) Init(metadata nameresolution.Metadata) error {
	config := config{}

	err := config.decode(metadata.Configuration)
	if err != nil {
		return err
	}

	sc := []constant.ServerConfig{}
	for _, s := range config.Servers {
		sc = append(sc, constant.ServerConfig{
			IpAddr:      s.Addr,
			Port:        s.Port,
			Scheme:      s.Scheme,
			ContextPath: s.ContextPath,
		})
	}

	cc := &constant.ClientConfig{
		NamespaceId:         config.Client.NamespaceId,
		TimeoutMs:           config.Client.TimeoutMs,
		NotLoadCacheAtStart: config.Client.NotLoadCacheAtStart,
		LogDir:              config.Client.LogDir,
		CacheDir:            config.Client.CacheDir,
		LogLevel:            config.Client.LogLevel,
	}

	r.client, err = nacos.NewNamingClient(vo.NacosClientParam{
		ClientConfig:  cc,
		ServerConfigs: sc,
	})
	if err != nil {
		return fmt.Errorf("nacos name error: create name client failed. %w ", err)
	}

	// Registe
	if config.Register != nil {
		r.client.RegisterInstance(vo.RegisterInstanceParam{
			Ip:          config.Register.Ip,
			Port:        config.Register.Port,
			Weight:      config.Register.Weight,
			Enable:      config.Register.Enable,
			Healthy:     config.Register.Healthy,
			Metadata:    config.Register.Metadata,
			ClusterName: config.Register.ClusterName,
			ServiceName: config.Register.ServiceName,
			GroupName:   config.Register.GroupName,
			Ephemeral:   config.Register.Ephemeral,
		})
	}

	return nil
}

func (r *resolver) ResolveID(req nameresolution.ResolveRequest) (string, error) {
	params := vo.SelectOneHealthInstanceParam{}
	service, err := r.client.SelectOneHealthyInstance(params)
	if err != nil {
		return "", fmt.Errorf("failed to query healthy nacos services: %w", err)
	}

	if service == nil {
		return "", fmt.Errorf("no healthy services found with AppID:%s", req.ID)
	}
	return "", nil
}
