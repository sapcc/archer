// Copyright 2023 SAP SE
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package as3

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
)

// AS3

type AS3 struct {
	Schema      string `json:"$schema,omitempty"`
	Persist     bool   `json:"persist"`
	Class       string `json:"class"`
	Action      string `json:"action,omitempty"`
	Declaration any    `json:"declaration,omitempty"`
}

// ADC

type ADC struct {
	Class         string `json:"class"`
	SchemaVersion string `json:"schemaVersion"`
	UpdateMode    string `json:"updateMode"`
	Id            string `json:"id"`

	Tenants map[string]Tenant
}

func (a ADC) MarshalJSON() ([]byte, error) {
	adc, err := struct2map(a)
	if err != nil {
		return nil, err
	}

	for name, tenant := range a.Tenants {
		adc[name] = tenant
	}
	return json.Marshal(adc)
}

// Tenant

type Tenant struct {
	Class  string `json:"class"`
	Label  string `json:"label,omitempty"`
	Remark string `json:"remark,omitempty"`

	Applications map[string]Application
}

func (t Tenant) MarshalJSON() ([]byte, error) {
	tenant, err := struct2map(t)
	if err != nil {
		return nil, err
	}

	for name, application := range t.Applications {
		tenant[name] = application
	}
	return json.Marshal(tenant)
}

// Application

type Application struct {
	Class    string `json:"class"`
	Label    string `json:"label,omitempty"`
	Remark   string `json:"remark,omitempty"`
	Template string `json:"template"`

	// Application Services
	Services map[string]any
}

func (a Application) MarshalJSON() ([]byte, error) {
	application, err := struct2map(a)
	if err != nil {
		return nil, err
	}

	for name, service := range a.Services {
		application[name] = service
	}
	return json.Marshal(application)
}

// Application SnatPools

type SnatPool struct {
	Class         string   `json:"class"`
	Label         string   `json:"label,omitempty"`
	Remark        string   `json:"remark,omitempty"`
	SnatAddresses []string `json:"snatAddresses"`
}

// Application Pools

type PoolMember struct {
	RouteDomain     int      `json:"routeDomain"`
	ServicePort     int32    `json:"servicePort"`
	ServerAddresses []string `json:"serverAddresses"`
	Enable          bool     `json:"enable"`
	Remark          string   `json:"remark,omitempty"`
}

type Pool struct {
	Class    string       `json:"class"`
	Label    string       `json:"label,omitempty"`
	Remark   string       `json:"remark,omitempty"`
	Members  []PoolMember `json:"members"`
	Monitors []Pointer    `json:"monitors"`
}

// Application Service_L4

type ServiceL4 struct {
	Label               string    `json:"label,omitempty"`
	Remark              string    `json:"remark,omitempty"`
	Class               string    `json:"class"`
	AllowVlans          []string  `json:"allowVlans"`
	IRules              []Pointer `json:"iRules"`
	Mirroring           string    `json:"mirroring"`
	PersistanceMethods  []string  `json:"persistenceMethods"`
	Pool                Pointer   `json:"pool"`
	ProfileL4           Pointer   `json:"profileL4"`
	Snat                any       `json:"snat,omitempty"`
	VirtualAddresses    []string  `json:"virtualAddresses"`
	TranslateServerPort bool      `json:"translateServerPort"`
	VirtualPort         int32     `json:"virtualPort"`
}

type IRule struct {
	Label  string `json:"label,omitempty"`
	Remark string `json:"remark,omitempty"`
	Class  string `json:"class"`
	IRule  string `json:"iRule"`
}

// Generic Pointer

type Pointer struct {
	Use   string `json:"use,omitempty"`
	BigIP string `json:"bigip,omitempty"`
}

// Helper

func struct2map(in any) (map[string]any, error) {
	m := make(map[string]any)

	v := reflect.ValueOf(in)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	if v.Kind() != reflect.Struct { // Non-structural return error
		return nil, fmt.Errorf("struct2map only accepts struct or struct pointer; got %T", v)
	}

	t := v.Type()
	// Traversing structure fields
	// Specify the tagName value as the key in the map; the field value as the value in the map
	for i := 0; i < v.NumField(); i++ {
		fi := t.Field(i)
		tags := strings.Split(fi.Tag.Get("json"), ",")
		if tags[0] != "" {
			if len(tags) == 2 && tags[1] == "omitempty" {
				value := fmt.Sprint(v.Field(i).Interface())
				if value != "" {
					m[tags[0]] = value
				}
			} else {
				m[tags[0]] = v.Field(i).Interface()
			}
		}
	}

	return m, nil
}
