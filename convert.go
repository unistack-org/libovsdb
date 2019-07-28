package libovsdb

import (
	"fmt"
	"reflect"
)

func ovsdbToGo(value interface{}) interface{} {
	var val interface{}

	switch value.(type) {
	case []interface{}:
		slice := value.([]interface{})
		switch slice[0] {
		case "uuid":
			return slice[1].(string)
		case "set":
			values := slice[1].([]interface{})
			switch len(values) {
			case 0:
				return nil
			case 1:
				return ovsdbToGo(values[0])
			default:
				mp := make([]interface{}, len(values))
				for _, val := range values {
					mp = append(mp, ovsdbToGo(val))
				}
				return mp
			}
		case "pair":
			panic("not implemented pair")
		case "map":
			values := slice[1].([]interface{})
			mp := make(map[string]string)
			for _, mv := range values {
				v := mv.([]interface{})
				mp[v[0].(string)] = v[1].(string)
			}
			return mp
		}
	}

	return val
}

func setField(obj interface{}, tag string, name string, value interface{}) error {
	structValue := reflect.ValueOf(obj).Elem()
	fieldVal := structValue.FieldByName(name)

	if !fieldVal.IsValid() {
		return fmt.Errorf("no such field: %s in obj", name)
	}

	if !fieldVal.CanSet() {
		return fmt.Errorf("cannot set %s field value", name)
	}

	switch value.(type) {
	case []interface{}:
		newvalue := ovsdbToGo(value)
		return setField(obj, tag, name, newvalue)
	}

	if value == nil {
		return nil
	}
	val := reflect.ValueOf(value)

	switch fieldVal.Kind() {
	case reflect.Bool:
		if value == nil || value == "true" {
			fieldVal.SetBool(true)
		}
	case reflect.Slice:
		if value != nil {
			fieldVal.Set(reflect.Append(fieldVal, val))
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if val.Kind() == reflect.Float64 {
			fieldVal.SetInt(int64(value.(float64)))
		} else {
			fieldVal.Set(val)
		}
	case reflect.Ptr: // not possible, ovsdb not supports nested objects
		if m, ok := value.(map[string]interface{}); ok {
			if fieldVal.Type().Elem().Kind() == reflect.Struct {
				if fieldVal.IsNil() {
					fieldVal.Set(reflect.New(fieldVal.Type().Elem()))
				}
				return MapToStruct(m, tag, fieldVal.Interface())
			}
		}
	case reflect.Struct: // not possible, ovsdb not supports nested objects
		if m, ok := value.(map[string]interface{}); ok {
			return MapToStruct(m, tag, fieldVal.Addr().Interface())
		}
	default:
		fieldVal.Set(val)
	}

	return nil
}

func MapToStruct(m map[string]interface{}, tag string, s interface{}) error {
	var err error

	structValue := reflect.ValueOf(s)
	if structValue.Kind() == reflect.Ptr {
		structValue = structValue.Elem()
	}

	// we only accept structs
	if structValue.Kind() != reflect.Struct {
		return fmt.Errorf("allow only structs, got %T", structValue)
	}

	structType := structValue.Type()
	for i := 0; i < structValue.NumField(); i++ {
		structField := structType.Field(i)
		if fieldTag := structField.Tag.Get(tag); len(fieldTag) > 0 {
			if mapValue, ok := m[fieldTag]; ok {
				if err = setField(s, tag, structField.Name, mapValue); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func StructToMap(iface interface{}, tag string) (map[string]interface{}, error) {
	mp := make(map[string]interface{})

	v := reflect.ValueOf(iface)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	// we only accept structs
	if v.Kind() != reflect.Struct {
		return nil, fmt.Errorf("allow only structs, got %T", v)
	}

	typ := v.Type()
	for i := 0; i < v.NumField(); i++ {
		// gets us a StructField
		fi := typ.Field(i)
		if tagv := fi.Tag.Get(tag); len(tagv) > 0 {
			// set key of map to value in struct field
			mp[tagv] = v.Field(i).Interface()
		}
	}

	return mp, nil
}
