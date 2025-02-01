package helm

import (
	"fmt"

	jsonpatch "github.com/evanphx/json-patch"
	"github.com/k0sproject/dig"
	"github.com/ohler55/ojg/jp"
	"gopkg.in/yaml.v3"
	k8syaml "sigs.k8s.io/yaml"
)

func UnmarshalValues(valuesYaml string) (map[string]interface{}, error) {
	newValuesMap := map[string]interface{}{}
	if err := yaml.Unmarshal([]byte(valuesYaml), &newValuesMap); err != nil {
		return nil, fmt.Errorf("yaml unmarshal: %w", err)
	}
	return newValuesMap, nil
}

func MarshalValues(values map[string]interface{}) (string, error) {
	newValuesYaml, err := yaml.Marshal(values)
	if err != nil {
		return "", fmt.Errorf("yaml marshal: %w", err)
	}
	return string(newValuesYaml), nil
}

// SetValue sets the value at the given path in the values map.
// NOTE: this function does not support creating new maps. It only supports setting values in
// existing ones.
func SetValue(values map[string]interface{}, path string, newValue interface{}) (map[string]interface{}, error) {
	newValuesMap := dig.Mapping(values)

	x, err := jp.ParseString(path)
	if err != nil {
		return nil, fmt.Errorf("parse json path %q: %w", path, err)
	}

	err = x.Set(newValuesMap, newValue)
	if err != nil {
		return nil, fmt.Errorf("set json path %q to %q: %w", path, newValue, err)
	}

	return newValuesMap, nil
}

func GetValue(values map[string]interface{}, path string) (interface{}, error) {
	x, err := jp.ParseString(path)
	if err != nil {
		return nil, fmt.Errorf("parse json path %q: %w", path, err)
	}
	v := x.Get(values)
	if len(v) == 0 {
		return nil, fmt.Errorf("value not found in path %q", path)
	}
	return v[0], nil
}

func PatchValues(values map[string]interface{}, patchYAML string) (map[string]interface{}, error) {
	if len(patchYAML) == 0 {
		return values, nil
	}

	// convert original values to JSON
	originalYAML, err := MarshalValues(values)
	if err != nil {
		return nil, fmt.Errorf("marshal original values: %w", err)
	}
	originalJSON, err := k8syaml.YAMLToJSON([]byte(originalYAML))
	if err != nil {
		return nil, fmt.Errorf("convert original values to json: %w", err)
	}

	// convert patch to JSON
	patchJSON, err := k8syaml.YAMLToJSON([]byte(patchYAML))
	if err != nil {
		return nil, fmt.Errorf("convert patch to json: %w", err)
	}

	// apply as JSON merge patch
	resultJSON, err := jsonpatch.MergePatch(originalJSON, patchJSON)
	if err != nil {
		return nil, fmt.Errorf("patch values: %w", err)
	}

	// convert result back to YAML
	resultYAML, err := k8syaml.JSONToYAML(resultJSON)
	if err != nil {
		return nil, fmt.Errorf("convert patched values to yaml: %w", err)
	}

	// unmarshal result back to map
	result, err := UnmarshalValues(string(resultYAML))
	if err != nil {
		return nil, fmt.Errorf("unmarshal patched values: %w", err)
	}

	return result, nil
}
