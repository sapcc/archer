package restapi

import (
	"fmt"
	"github.com/iancoleman/strcase"
	"reflect"
)

func SetModelDefaults(s interface{}) error {
	/*
		This is a workaround for default swagger values, since go-swagger currently doesn't populate default variables
		for nested definitions:
		https://github.com/go-swagger/go-swagger/issues/1393
	*/
	var instanceType string
	if _, err := fmt.Sscanf(fmt.Sprintf("%T", s), "*models.%s", &instanceType); err != nil {
		return err
	}
	for specDefinitionName, specDefinitionModel := range SwaggerSpec.Spec().Definitions {
		if specDefinitionName == instanceType {

			// Found the swagger model
			for propName, property := range specDefinitionModel.SchemaProps.Properties {

				// Check if model has default set
				if property.Default != nil {
					propertyField := reflect.ValueOf(s).Elem().FieldByName(strcase.ToCamel(propName))
					if propertyField.Kind() != reflect.Ptr && propertyField.Kind() != reflect.Uintptr {
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
