package config

import (
	"fmt"
	"io"
	"reflect"
	"regexp"
	"sort"
	"strings"

	"github.com/ghodss/yaml"
	"github.com/mitchellh/mapstructure"

	sunpikev1alpha1 "github.com/platform9/pf9-qbert/sunpike/apiserver/pkg/apis/sunpike/v1alpha1"
)

const (
	HostIDKey     = "HOSTID"
	kubeEnvTagKey = "kube.env"
)

var (
	floatType                = reflect.TypeOf(float64(0))
	knownKubeEnvKeys         = readKubeEnvKeysFromHostTags()
	charsToEscapeInVarExport = regexp.MustCompile(`(?m)(["'` + "`" + `\\$])`)
)

// KubeEnvMap is a representation of the kube.env-like key-value mapping.
type KubeEnvMap map[string]string

// ToKubeEnv writes the KubeEnvMap to a destination in the kube.env format.
//
// Format:
// export FOO="BAR"
func (k KubeEnvMap) ToKubeEnv(w io.Writer) error {
	for _, key := range k.Keys() {
		val := k[key]
		stringVal := fmt.Sprintf("%v", val)
		escapedVal := strings.ReplaceAll(charsToEscapeInVarExport.ReplaceAllString(stringVal, "\\$1"), "\n", "\\n")
		escapedKey := strings.TrimSpace(key)
		// Note: we need to use double-quotes here because this file is parsed
		// by the pf9-kube/config script, which only expects double-quotes.
		line := fmt.Sprintf("export %s=\"%s\"\n", escapedKey, escapedVal)
		_, err := w.Write([]byte(line))
		if err != nil {
			return err
		}
	}
	return nil
}

// ToYAML writes the KubeEnvMap to a destination in the nodelet/config.yaml format.
func (k KubeEnvMap) ToYAML(w io.Writer) error {
	bs, err := yaml.Marshal(k)
	if err != nil {
		return err
	}

	_, err = w.Write(bs)
	if err != nil {
		return err
	}

	return nil
}

// ToConfig converts the KubeEnvMap into a Nodelet Config, without setting any defaults.
func (k KubeEnvMap) ToConfig() (*Config, error) {
	cfg := &Config{}
	err := mapstructure.WeakDecode(k, cfg)
	if err != nil {
		return nil, err
	}
	return cfg, nil
}

// ToHost converts the KubeEnvMap back into a Sunpike Host.
//
// It fills the Spec and the Name of the Host. Because we cannot assume that the
// types are correct in the KubeEnvMap, the mapstructure package is used internally
// to do the decoding, which tries to infer the type based on the value contents.
//
// Note: this is a best-effort conversion, because we lack the type information
// in the kubeEnvMap. So this should only be used for optional work and
// debugging. Do not rely on this for equality checks!
func (k KubeEnvMap) ToHost() (*sunpikev1alpha1.Host, error) {
	host := &sunpikev1alpha1.Host{}
	err := k.decodeToNestedStruct(reflect.ValueOf(&host.Spec))
	if err != nil {
		return nil, fmt.Errorf("failed to decode kubeEnvMap to Host: %w", err)
	}

	// Put the remaining keys in ExtraCfg
	extraCfg := make(map[string]string)
	for key, val := range k {
		if _, ok := knownKubeEnvKeys[key]; ok {
			// Ignore all keys that have a known field in HostSpec
			continue
		}

		switch key {
		case HostIDKey:
			host.Name = fmt.Sprintf("%v", val)
		default:
			// The val should be a string, but just to be sure we convert it with fmt.
			extraCfg[key] = fmt.Sprintf("%v", val)
		}
	}
	host.Spec.ExtraCfg = extraCfg

	return host, nil
}

// Copy creates a shallow clone of the current KubeEnvMap.
func (k KubeEnvMap) Copy() KubeEnvMap {
	m := KubeEnvMap{}
	for key, val := range k {
		m[key] = val
	}
	return m
}

// Keys returns all keys in the map, sorted alphabetically.
func (k KubeEnvMap) Keys() []string {
	keys := make([]string, len(k))
	var i int
	for key := range k {
		keys[i] = key
		i++
	}
	sort.Strings(keys)
	return keys
}

func (k KubeEnvMap) decodeToNestedStruct(structValPtr reflect.Value) error {
	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		Result:           structValPtr.Interface(),
		WeaklyTypedInput: true,
		TagName:          kubeEnvTagKey,
	})
	if err != nil {
		return fmt.Errorf("error creating decoder for %w", err)
	}
	err = decoder.Decode(k)
	if err != nil {
		return fmt.Errorf("decoding error: %w", err)
	}

	structVal := structValPtr.Elem()
	for i := 0; i < structVal.NumField(); i++ {
		fieldVal := structVal.Field(i)
		fieldType := structVal.Type().Field(i)

		switch fieldVal.Kind() {
		case reflect.Struct:
			fieldValPtr := reflect.New(fieldVal.Type())
			err = k.decodeToNestedStruct(fieldValPtr)
			if err != nil {
				return fmt.Errorf("error decoding field %s: %w", fieldType.Name, err)
			}
			fieldVal.Set(fieldValPtr.Elem())
		default:
			// Ignore other fields, because we don't use them in the HostSpec
			// or in the case of ExtraCfg handle them afterwards.
		}
	}
	return nil
}

// ConvertHostToKubeEnvMap transforms a Sunpike Host into a KubeEnvMap.
//
// For the conversion it looks at the `kube.env` tags of the fields to decide
// to which key the field should be mapped. ExtraCfg is mapped too using its
// key-value pair. However it is not allowed to override existing fields.
func ConvertHostToKubeEnvMap(host *sunpikev1alpha1.Host) KubeEnvMap {
	kubeEnv := KubeEnvMap{}
	structToKubeEnvMap(reflect.ValueOf(host.Spec), kubeEnv)

	// The ID of the host is outside of the spec in the host.Name.
	kubeEnv[HostIDKey] = host.Name

	// Because we want to ensure that the ExtraCfg keys cannot override other
	// keys we handle it here after structToKubeEnvMap
	for k, v := range host.Spec.ExtraCfg {
		// Keys in ExtraCfg should be in uppercase, to ensure that they are,
		// uppercase them here.
		kubeEnvKey := strings.ToUpper(k)

		// Keys in extraCfg are not allowed to override existing keys in the map.
		// Ignore those that do.
		_, ok := kubeEnv[kubeEnvKey]
		if !ok {
			kubeEnv[kubeEnvKey] = v
		}
	}

	return kubeEnv
}

func structToKubeEnvMap(structVal reflect.Value, kubeEnv KubeEnvMap) {
	for i := 0; i < structVal.NumField(); i++ {
		fieldVal := structVal.Field(i)
		fieldType := structVal.Type().Field(i)

		switch fieldVal.Kind() {
		case reflect.Struct:
			structToKubeEnvMap(fieldVal, kubeEnv)
		case reflect.Map, reflect.Slice, reflect.Array:
			// Ignore these fields, because we don't use them in the HostSpec
			// or in the case of ExtraCfg handle them afterwards.
		default:
			kubeEnvKey, ok := fieldType.Tag.Lookup(kubeEnvTagKey)
			if !ok {
				// Ignore any fields without the tag.
				continue
			}

			// Convert numeric types to float64, because that is how
			// unmarshalled YAML represents a number in a map[string]interface{}
			// We need this because we need to compare the kube.env.
			if fieldVal.Type().ConvertibleTo(floatType) {
				fieldVal = fieldVal.Convert(floatType)
			}

			kubeEnv[kubeEnvKey] = fmt.Sprintf("%v", fieldVal.Interface())
		}
	}
}

func readKubeEnvKeysFromHostTags() map[string]struct{} {
	keySet := map[string]struct{}{}
	readKubeEnvKeysFromVal(reflect.ValueOf(sunpikev1alpha1.Host{}), keySet)
	return keySet
}

func readKubeEnvKeysFromVal(structVal reflect.Value, keySet map[string]struct{}) {
	for i := 0; i < structVal.NumField(); i++ {
		fieldType := structVal.Type().Field(i)
		fieldVal := structVal.Field(i)

		switch fieldVal.Kind() {
		case reflect.Struct:
			readKubeEnvKeysFromVal(fieldVal, keySet)
		case reflect.Map, reflect.Slice, reflect.Array:
			// Ignore these fields, because we don't use them in the HostSpec
			// or in the case of ExtraCfg handle them afterwards.
		default:
			kubeEnvKey, ok := fieldType.Tag.Lookup(kubeEnvTagKey)
			if !ok {
				// Ignore any fields without the tag.
				continue
			}
			keySet[kubeEnvKey] = struct{}{}
		}
	}
}
