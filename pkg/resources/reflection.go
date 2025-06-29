package resources

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/traego/scaled-mcp/pkg/protocol"
)

func GenerateSchemaFromStruct(structType reflect.Type) (protocol.InputSchema, error) {
	if structType.Kind() == reflect.Ptr {
		structType = structType.Elem()
	}

	if structType.Kind() != reflect.Struct {
		return protocol.InputSchema{}, fmt.Errorf("expected struct type, got %s", structType.Kind())
	}

	schema := protocol.InputSchema{
		Type:       "object",
		Properties: make(map[string]protocol.SchemaProperty),
		Required:   []string{},
	}

	for i := 0; i < structType.NumField(); i++ {
		field := structType.Field(i)

		if !field.IsExported() {
			continue
		}

		fieldName, property, required, err := parseStructField(field)
		if err != nil {
			return protocol.InputSchema{}, fmt.Errorf("error parsing field %s: %w", field.Name, err)
		}

		if fieldName == "" {
			continue
		}

		schema.Properties[fieldName] = property
		if required {
			schema.Required = append(schema.Required, fieldName)
		}
	}

	return schema, nil
}

func parseStructField(field reflect.StructField) (string, protocol.SchemaProperty, bool, error) {
	mcpTag := field.Tag.Get("mcp")
	if mcpTag == "-" {
		return "", protocol.SchemaProperty{}, false, nil
	}

	fieldName := strings.ToLower(field.Name)
	property := protocol.SchemaProperty{}
	required := false

	if mcpTag != "" {
		parts := strings.Split(mcpTag, ",")
		if len(parts) > 0 && parts[0] != "" {
			fieldName = parts[0]
		}
		if len(parts) > 1 && parts[1] != "" {
			property.Description = parts[1]
		}
		for i := 2; i < len(parts); i++ {
			part := strings.TrimSpace(parts[i])
			if part == "required" {
				required = true
			} else if strings.HasPrefix(part, "default=") {
				defaultValue := strings.TrimPrefix(part, "default=")
				property.Default = parseDefaultValue(defaultValue, field.Type)
			}
		}
	}

	schemaType, err := goTypeToSchemaType(field.Type)
	if err != nil {
		return "", protocol.SchemaProperty{}, false, err
	}
	property.Type = schemaType

	return fieldName, property, required, nil
}

func goTypeToSchemaType(t reflect.Type) (string, error) {
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	switch t.Kind() {
	case reflect.String:
		return "string", nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return "integer", nil
	case reflect.Float32, reflect.Float64:
		return "number", nil
	case reflect.Bool:
		return "boolean", nil
	case reflect.Slice, reflect.Array:
		return "array", nil
	case reflect.Map, reflect.Struct:
		return "object", nil
	case reflect.Interface:
		return "object", nil
	default:
		return "", fmt.Errorf("unsupported type: %s", t.Kind())
	}
}

func parseDefaultValue(value string, t reflect.Type) interface{} {
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	switch t.Kind() {
	case reflect.String:
		return value
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if i, err := strconv.ParseInt(value, 10, 64); err == nil {
			return i
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		if i, err := strconv.ParseUint(value, 10, 64); err == nil {
			return i
		}
	case reflect.Float32, reflect.Float64:
		if f, err := strconv.ParseFloat(value, 64); err == nil {
			return f
		}
	case reflect.Bool:
		if b, err := strconv.ParseBool(value); err == nil {
			return b
		}
	}
	return value
}

func UnmarshalParams(params map[string]interface{}, target interface{}) error {
	targetValue := reflect.ValueOf(target)
	if targetValue.Kind() != reflect.Ptr {
		return fmt.Errorf("target must be a pointer")
	}

	targetValue = targetValue.Elem()
	if targetValue.Kind() != reflect.Struct {
		return fmt.Errorf("target must be a pointer to struct")
	}

	targetType := targetValue.Type()

	for i := 0; i < targetType.NumField(); i++ {
		field := targetType.Field(i)
		fieldValue := targetValue.Field(i)

		if !field.IsExported() || !fieldValue.CanSet() {
			continue
		}

		fieldName, _, _, err := parseStructField(field)
		if err != nil {
			return fmt.Errorf("error parsing field %s: %w", field.Name, err)
		}

		if fieldName == "" {
			continue
		}

		paramValue, exists := params[fieldName]
		if !exists {
			continue
		}

		if err := setFieldValue(fieldValue, paramValue); err != nil {
			return fmt.Errorf("error setting field %s: %w", field.Name, err)
		}
	}

	return nil
}

func setFieldValue(fieldValue reflect.Value, paramValue interface{}) error {
	if paramValue == nil {
		return nil
	}

	paramReflectValue := reflect.ValueOf(paramValue)
	fieldType := fieldValue.Type()

	if fieldType.Kind() == reflect.Ptr {
		if fieldValue.IsNil() {
			fieldValue.Set(reflect.New(fieldType.Elem()))
		}
		return setFieldValue(fieldValue.Elem(), paramValue)
	}

	if paramReflectValue.Type().AssignableTo(fieldType) {
		fieldValue.Set(paramReflectValue)
		return nil
	}

	if paramReflectValue.Type().ConvertibleTo(fieldType) {
		fieldValue.Set(paramReflectValue.Convert(fieldType))
		return nil
	}

	switch fieldType.Kind() {
	case reflect.String:
		fieldValue.SetString(fmt.Sprintf("%v", paramValue))
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if f, ok := paramValue.(float64); ok {
			fieldValue.SetInt(int64(f))
		} else if i, ok := paramValue.(int64); ok {
			fieldValue.SetInt(i)
		} else {
			return fmt.Errorf("cannot convert %T to %s", paramValue, fieldType.Kind())
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		if f, ok := paramValue.(float64); ok {
			fieldValue.SetUint(uint64(f))
		} else if i, ok := paramValue.(uint64); ok {
			fieldValue.SetUint(i)
		} else {
			return fmt.Errorf("cannot convert %T to %s", paramValue, fieldType.Kind())
		}
	case reflect.Float32, reflect.Float64:
		if f, ok := paramValue.(float64); ok {
			fieldValue.SetFloat(f)
		} else {
			return fmt.Errorf("cannot convert %T to %s", paramValue, fieldType.Kind())
		}
	case reflect.Bool:
		if b, ok := paramValue.(bool); ok {
			fieldValue.SetBool(b)
		} else {
			return fmt.Errorf("cannot convert %T to %s", paramValue, fieldType.Kind())
		}
	default:
		return fmt.Errorf("unsupported field type: %s", fieldType.Kind())
	}

	return nil
}
