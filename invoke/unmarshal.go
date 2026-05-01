package invoke

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"github.com/gotd/td/bin"
)

// unmarshalRequest unmarshals JSON into a gotd request struct.
// It handles the gotd type convention: interface fields like InputUserClass
// are deserialized from objects with a "_" key specifying the constructor name
// (e.g., {"_": "inputUserSelf"}).
func unmarshalRequest(data json.RawMessage, req bin.Object) error {
	// First pass: unmarshal into a generic map.
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("json unmarshal: %w", err)
	}

	// Reflect on the request struct and set fields.
	v := reflect.ValueOf(req).Elem()
	t := v.Type()

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		fieldVal := v.Field(i)

		// Find the JSON key for this field.
		// gotd uses PascalCase; we match case-insensitively.
		key := findKey(raw, field.Name)
		if key == "" {
			continue
		}

		rawVal := raw[key]
		if err := setField(fieldVal, rawVal); err != nil {
			return fmt.Errorf("field %s: %w", field.Name, err)
		}
	}

	return nil
}

// findKey finds a key in the raw map matching the field name case-insensitively.
func findKey(raw map[string]json.RawMessage, fieldName string) string {
	lower := strings.ToLower(fieldName)
	for k := range raw {
		if strings.ToLower(k) == lower {
			return k
		}
	}
	return ""
}

// setField sets a reflect.Value from raw JSON. Handles interface fields
// (like InputUserClass) by looking up the constructor from the "_" key.
func setField(field reflect.Value, raw json.RawMessage) error {
	// Handle interface types (gotd's XxxClass types).
	if field.Kind() == reflect.Interface {
		return setInterfaceField(field, raw)
	}

	// Handle slices of interface types (like []InputUserClass).
	if field.Kind() == reflect.Slice && field.Type().Elem().Kind() == reflect.Interface {
		return setInterfaceSliceField(field, raw)
	}

	// Regular types — standard JSON unmarshal.
	ptr := reflect.New(field.Type())
	if err := json.Unmarshal(raw, ptr.Interface()); err != nil {
		return err
	}
	field.Set(ptr.Elem())
	return nil
}

// setInterfaceField deserializes a JSON object into the correct concrete type
// based on the "_" key (constructor name).
func setInterfaceField(field reflect.Value, raw json.RawMessage) error {
	// Parse to get the constructor name.
	var peek struct {
		Type string `json:"_"`
	}
	if err := json.Unmarshal(raw, &peek); err != nil {
		return err
	}

	if peek.Type == "" {
		return fmt.Errorf("missing \"_\" constructor name in JSON object")
	}

	// Look up the constructor in the registry.
	reg := GlobalRegistry()
	ctor := reg.ConstructorByName(peek.Type)
	if ctor == nil {
		return fmt.Errorf("unknown constructor %q", peek.Type)
	}

	// Create the concrete type.
	obj := ctor()

	// Unmarshal the full JSON into the concrete struct.
	if err := json.Unmarshal(raw, obj); err != nil {
		return fmt.Errorf("unmarshal %s: %w", peek.Type, err)
	}

	// Set the interface value.
	field.Set(reflect.ValueOf(obj))
	return nil
}

// setInterfaceSliceField handles slices of interface types like []InputUserClass.
func setInterfaceSliceField(field reflect.Value, raw json.RawMessage) error {
	var items []json.RawMessage
	if err := json.Unmarshal(raw, &items); err != nil {
		return err
	}

	elemType := field.Type().Elem()
	slice := reflect.MakeSlice(field.Type(), 0, len(items))

	for _, item := range items {
		elem := reflect.New(elemType).Elem()
		if err := setInterfaceField(elem, item); err != nil {
			return err
		}
		slice = reflect.Append(slice, elem)
	}

	field.Set(slice)
	return nil
}
