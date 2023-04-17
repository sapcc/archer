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

package controller

import (
	"fmt"
	"reflect"

	"github.com/iancoleman/strcase"
)

func (c *Controller) SetModelDefaults(s any) error {
	/*
		This is a workaround for default swagger values, since go-swagger currently doesn't populate default variables
		for nested definitions:
		https://github.com/go-swagger/go-swagger/issues/1393
	*/
	var instanceType string
	if _, err := fmt.Sscanf(fmt.Sprintf("%T", s), "*models.%s", &instanceType); err != nil {
		return err
	}
	for specDefinitionName, specDefinitionModel := range c.spec.Spec().Definitions {
		if specDefinitionName == instanceType {

			// Found the swagger model
			for propName, property := range specDefinitionModel.SchemaProps.Properties {

				// Check if model has default set
				if property.Default != nil {
					propertyField := reflect.ValueOf(s).Elem().FieldByName(strcase.ToCamel(propName))
					if propertyField.Kind() != reflect.Ptr && propertyField.Kind() != reflect.Uintptr && propertyField.Kind() != reflect.Slice {
						return fmt.Errorf("unexpected field %s for specDefinitionModel %s", propName, specDefinitionName)
					}

					if !propertyField.IsNil() {
						continue
					}

					// Generate correct Value
					vp := reflect.New(propertyField.Type())
					if property.Type.Contains("boolean") {
						val := property.Default.(bool)
						vp.Elem().Set(reflect.ValueOf(&val))
					} else if property.Type.Contains("string") {
						val := property.Default.(string)
						vp.Elem().Set(reflect.ValueOf(&val))
					} else if property.Type.Contains("integer") {
						val := int64(property.Default.(float64))
						vp.Elem().Set(reflect.ValueOf(&val))
					} else if property.Type.Contains("number") {
						val := float32(property.Default.(float64))
						vp.Elem().Set(reflect.ValueOf(&val))
					} else if property.Type.Contains("array") {
						val := property.Default.([]any)
						if len(val) == 0 {
							vp.Elem().Set(reflect.MakeSlice(vp.Elem().Type(), 0, 0))
						}
					} else {
						return fmt.Errorf("unexpected type %T for property %s", property.Default, propName)
					}
					propertyField.Set(vp.Elem())
				}
			}
			break
		}
	}
	return nil
}
