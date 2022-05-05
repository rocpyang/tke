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

package images

import (
	"fmt"
	"reflect"
	"sort"
	"tkestack.io/tke/pkg/util/log"

	"tkestack.io/tke/pkg/util/containerregistry"
)

const (
	// LatestVersion is latest version of addon.
	V1Version     = "v1.0.0"
	LatestVersion = "v1.1.0"
)

type Components struct {
	Tiller  containerregistry.Image
	Swift   containerregistry.Image
	HelmAPI containerregistry.Image
}

func (c Components) GetTag(name string) string {
	v := reflect.ValueOf(c)
	for i := 0; i < v.NumField(); i++ {
		v, _ := v.Field(i).Interface().(containerregistry.Image)
		if v.Name == name {
			return v.Tag
		}
	}
	return ""
}

func (c Components) Get(name string) *containerregistry.Image {
	v := reflect.ValueOf(c)
	for i := 0; i < v.NumField(); i++ {
		v, _ := v.Field(i).Interface().(containerregistry.Image)
		if v.Name == name {
			return &v
		}
	}
	return nil
}

var versionMap = map[string]Components{
	V1Version: {
		Tiller:  containerregistry.Image{Name: "tiller", Tag: "v2.10.0"},
		Swift:   containerregistry.Image{Name: "swift", Tag: "0.9.0"},
		HelmAPI: containerregistry.Image{Name: "helm-api", Tag: "v1.3"},
	},
	// 兼容18集群及以上
	LatestVersion: {
		Tiller:  containerregistry.Image{Name: "tiller", Tag: "v2.16.8"},
		Swift:   containerregistry.Image{Name: "swift", Tag: "0.9.0"},
		HelmAPI: containerregistry.Image{Name: "helm-api", Tag: "v1.3"},
	},
}

func List() []string {
	items := make([]string, 0, len(versionMap))
	keys := make([]string, 0, len(versionMap))
	for key := range versionMap {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		v := reflect.ValueOf(versionMap[key])
		for i := 0; i < v.NumField(); i++ {
			v, _ := v.Field(i).Interface().(containerregistry.Image)
			items = append(items, v.BaseName())
		}
	}

	return items
}

func Validate(version string) error {
	log.Infof("Validate version[%s],allversion[%v]", version, versionMap)
	_, ok := versionMap[version]
	if !ok {
		return fmt.Errorf("the component version definition corresponding to version %s could not be found", version)
	}
	return nil
}

func Get(version string) Components {
	cv, ok := versionMap[version]
	if !ok {
		panic(fmt.Sprintf("the component version definition corresponding to version %s could not be found", version))
	}
	return cv
}
