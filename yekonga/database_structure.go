package yekonga

import (
	"strings"

	"github.com/robertkonga/yekonga-server-go/helper"
)

type DatabaseStructure = map[string]CollectionStructure

type CollectionStructure struct {
	Name   string
	Fields map[string]CollectionFieldConfig
}

type CollectionFieldConfig struct {
	PrimaryKey   bool
	Name         string
	Kind         string
	DefaultValue interface{}
	Required     bool
	Protected    bool
	Options      []string
	ForeignKey   CollectionFieldConfigForeignKey
}

type CollectionFieldConfigForeignKey struct {
	Model string
	Key   string
}

// databaseCollectionFieldConfigFromMap builds a DatabaseCollectionFieldConfig from the raw
// map[string]interface{} shape used by the external database structure JSON file
// (e.g. {"type": "String", "default": nil, "required": false, "foreignKey": "Tenant.id"}).
func databaseCollectionFieldConfigFromMap(field map[string]interface{}) CollectionFieldConfig {
	result := CollectionFieldConfig{}

	if v, ok := field["type"]; ok {
		if vi, oki := v.(string); oki {
			result.Kind = vi
		}
	}

	if v, ok := field["default"]; ok {
		result.DefaultValue = v
	} else if v, ok := field["defaultValue"]; ok {
		result.DefaultValue = v
	}

	if v, ok := field["required"]; ok {
		if vi, oki := v.(bool); oki {
			result.Required = vi
		}
	}

	if v, ok := field["protected"]; ok {
		if vi, oki := v.(bool); oki {
			result.Protected = vi
		}
	}

	if v, ok := field["primaryKey"]; ok {
		if vi, oki := v.(bool); oki {
			result.PrimaryKey = vi
		}
	}

	if v, ok := field["options"]; ok && helper.IsArray(v) {
		result.Options = helper.ToList[string](v)
	}

	rv, rok := field["foreignKey"]
	if !rok {
		rv, rok = field["relation"]
	}
	if !rok {
		rv, rok = field["source"]
	}

	if rok {
		if vi, oki := rv.(string); oki && helper.IsNotEmpty(vi) {
			ks := strings.SplitN(vi, ".", 2)
			foreignKey := CollectionFieldConfigForeignKey{Model: ks[0]}

			if len(ks) == 2 {
				foreignKey.Key = ks[1]
			}

			result.ForeignKey = foreignKey
		}
	}

	return result
}
