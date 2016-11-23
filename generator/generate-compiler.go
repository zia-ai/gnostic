// Copyright 2016 Google Inc. All Rights Reserved.
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

package main

import (
	"fmt"
	"sort"
	"strings"

	"github.com/googleapis/openapi-compiler/printer"
)

func (classes *ClassCollection) generateCompiler(packageName string, license string) string {
	code := printer.Code{}
	code.Print(license)
	code.Print("// THIS FILE IS AUTOMATICALLY GENERATED.")
	code.Print()
	code.Print("package %s", packageName)
	code.Print()
	code.Print("import (")
	code.Print("\"fmt\"")
	code.Print("\"log\"")
	code.Print("\"github.com/googleapis/openapi-compiler/helpers\"")
	code.Print(")")
	code.Print()
	code.Print("func Version() string {")
	code.Print("  return \"%s\"", packageName)
	code.Print("}")
	code.Print()

	classNames := classes.sortedClassNames()
	for _, className := range classNames {
		code.Print("func Build%s(in interface{}) *%s {", className, className)

		classModel := classes.ClassModels[className]
		parentClassName := className

		if classModel.IsStringArray {
			code.Print("value, ok := in.(string)")
			code.Print("x := &TypeItem{}")
			code.Print("if ok {")
			code.Print("x.Value = make([]string, 0)")
			code.Print("x.Value = append(x.Value, value)")
			code.Print("} else {")
			code.Print("log.Printf(\"unexpected: %+v\", in)")
			code.Print("}")
			code.Print("return x")
			code.Print("}")
			code.Print()
			continue
		}

		if classModel.IsBlob {
			code.Print("x := &Any{}")
			code.Print("x.Value = fmt.Sprintf(\"%%+v\", in)")
			code.Print("return x")
			code.Print("}")
			code.Print()
			continue
		}

		if classModel.Name == "StringArray" {
			code.Print("a, ok := in.([]interface{})")
			code.Print("if ok {")
			code.Print("x := &StringArray{}")
			code.Print("x.Value = make([]string, 0)")
			code.Print("for _, s := range a {")
			code.Print("x.Value = append(x.Value, s.(string))")
			code.Print("}")
			code.Print("return x")
			code.Print("} else {")
			code.Print("return nil")
			code.Print("}")
			code.Print("}")
			code.Print()
			continue
		}
		code.Print("m, ok := helpers.UnpackMap(in)")
		code.Print("if (!ok) {")
		code.Print("log.Printf(\"unexpected argument to Build%s: %%+v\", in)", className)
		code.Print("log.Printf(\"%%d\\n\", len(m))")
		code.Print("return nil")
		code.Print("}")
		oneOfWrapper := classModel.OneOfWrapper

		if len(classModel.Required) > 0 {
			// verify that map includes all required keys
			keyString := ""
			sort.Strings(classModel.Required)
			for _, k := range classModel.Required {
				if keyString != "" {
					keyString += ","
				}
				keyString += "\""
				keyString += k
				keyString += "\""
			}
			code.Print("requiredKeys := []string{%s}", keyString)
			code.Print("if !helpers.MapContainsAllKeys(m, requiredKeys) {")
			code.Print("return nil")
			code.Print("}")
		}

		if !classModel.Open {
			// verify that map has no unspecified keys
			allowedKeys := make([]string, 0)
			for _, property := range classModel.Properties {
				if !property.Implicit {
					allowedKeys = append(allowedKeys, property.Name)
				}
			}
			sort.Strings(allowedKeys)
			allowedKeyString := ""
			for _, allowedKey := range allowedKeys {
				if allowedKeyString != "" {
					allowedKeyString += ","
				}
				allowedKeyString += "\""
				allowedKeyString += allowedKey
				allowedKeyString += "\""
			}
			allowedPatternString := ""
			if classModel.OpenPatterns != nil {
				for _, pattern := range classModel.OpenPatterns {
					if allowedPatternString != "" {
						allowedPatternString += ","
					}
					allowedPatternString += "\""
					allowedPatternString += pattern
					allowedPatternString += "\""
				}
			}
			// verify that map includes only allowed keys and patterns
			code.Print("allowedKeys := []string{%s}", allowedKeyString)
			code.Print("allowedPatterns := []string{%s}", allowedPatternString)
			code.Print("if !helpers.MapContainsOnlyKeysAndPatterns(m, allowedKeys, allowedPatterns) {")
			code.Print("return nil")
			code.Print("}")
		}

		code.Print("  x := &%s{}", className)

		var fieldNumber = 0
		for _, propertyModel := range classModel.Properties {
			propertyName := propertyModel.Name
			fieldNumber += 1
			propertyType := propertyModel.Type
			if propertyType == "int" {
				propertyType = "int64"
			}
			var displayName = propertyName
			if displayName == "$ref" {
				displayName = "_ref"
			}
			if displayName == "$schema" {
				displayName = "_schema"
			}
			displayName = camelCaseToSnakeCase(displayName)

			var line = fmt.Sprintf("%s %s = %d;", propertyType, displayName, fieldNumber)
			if propertyModel.Repeated {
				line = "repeated " + line
			}
			code.Print("// " + line)

			fieldName := strings.Title(propertyName)
			if propertyName == "$ref" {
				fieldName = "XRef"
			}

			classModel, classFound := classes.ClassModels[propertyType]
			if classFound && !classModel.IsPair {
				if propertyModel.Repeated {
					code.Print("if helpers.MapHasKey(m, \"%s\") {", propertyName)
					code.Print("// repeated class %s", classModel.Name)
					code.Print("x.%s = make([]*%s, 0)", fieldName, classModel.Name)
					code.Print("a, ok := helpers.MapValueForKey(m, \"%s\").([]interface{})", propertyName)
					code.Print("if ok {")
					code.Print("for _, item := range a {")
					code.Print("x.%s = append(x.%s, Build%s(item))", fieldName, fieldName, classModel.Name)
					code.Print("}")
					code.Print("}")
					code.Print("}")
				} else {
					if oneOfWrapper {
						code.Print("{")
						code.Print("t := Build%s(m)", classModel.Name)
						code.Print("if t != nil {")
						code.Print("x.Oneof = &%s_%s{%s: t}", parentClassName, classModel.Name, classModel.Name)
						code.Print("}")
						code.Print("}")
					} else {
						code.Print("if helpers.MapHasKey(m, \"%s\") {", propertyName)
						code.Print("x.%s = Build%s(helpers.MapValueForKey(m,\"%v\"))", fieldName, classModel.Name, propertyName)
						code.Print("}")
					}
				}
			} else if propertyType == "string" {
				if propertyModel.Repeated {
					code.Print("if helpers.MapHasKey(m, \"%s\") {", propertyName)
					code.Print("v, ok := helpers.MapValueForKey(m, \"%v\").([]interface{})", propertyName)
					code.Print("if ok {")
					code.Print("x.%s = helpers.ConvertInterfaceArrayToStringArray(v)", fieldName)
					code.Print("} else {")
					code.Print(" log.Printf(\"unexpected: %%+v\", helpers.MapValueForKey(m,\"%v\"))", propertyName)
					code.Print("}")
					code.Print("}")
				} else {
					code.Print("if helpers.MapHasKey(m, \"%s\") {", propertyName)
					code.Print("x.%s = helpers.MapValueForKey(m,\"%v\").(string)", fieldName, propertyName)
					code.Print("}")
				}
			} else if propertyType == "float" {
				code.Print("if helpers.MapHasKey(m, \"%s\") {", propertyName)
				code.Print("x.%s = helpers.MapValueForKey(m, \"%v\").(float64)", fieldName, propertyName)
				code.Print("}")
			} else if propertyType == "int64" {
				code.Print("if helpers.MapHasKey(m, \"%s\") {", propertyName)
				code.Print("x.%s = helpers.MapValueForKey(m, \"%v\").(int64)", fieldName, propertyName)
				code.Print("}")
			} else if propertyType == "bool" {
				code.Print("if helpers.MapHasKey(m, \"%s\") {", propertyName)
				code.Print("x.%s = helpers.MapValueForKey(m, \"%v\").(bool)", fieldName, propertyName)
				code.Print("}")
			} else {
				mapTypeName := propertyModel.MapType
				isMap := mapTypeName != ""
				if isMap {
					code.Print("// MAP: %s %s", mapTypeName, propertyModel.Pattern)
					if mapTypeName == "string" {
						code.Print("x.%s = make([]*NamedString, 0)", fieldName)
					} else {
						code.Print("x.%s = make([]*Named%s, 0)", fieldName, mapTypeName)
					}
					code.Print("for _, item := range m {")
					code.Print("k := item.Key.(string)")
					code.Print("v := item.Value")
					if propertyModel.Pattern != "" {
						code.Print("if helpers.PatternMatches(\"%s\", k) {", propertyModel.Pattern)
					}
					code.Print("pair := &Named" + strings.Title(mapTypeName) + "{}")
					code.Print("pair.Name = k")
					if mapTypeName == "string" {
						code.Print("pair.Value = v.(string)")
					} else {
						code.Print("pair.Value = Build%v(v)", mapTypeName)
					}
					code.Print("x.%s = append(x.%s, pair)", fieldName, fieldName)
					if propertyModel.Pattern != "" {
						code.Print("}")
					}
					code.Print("}")
				} else {
					code.Print("// TODO: %s", propertyType)
				}
			}
		}
		code.Print("  return x")
		code.Print("}\n")
	}
	return code.String()
}
