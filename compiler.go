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

//go:generate ./COMPILE-PROTOS.sh

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"

	"github.com/golang/protobuf/proto"
	"github.com/googleapis/openapi-compiler/OpenAPIv2"
	"github.com/googleapis/openapi-compiler/compiler"
)

func main() {
	var rawInput = flag.Bool("raw", false, "Output the raw json input")
	var textProtoFileName = flag.String("text_out", "", "Output location for writing the text proto")
	var jsonProtoFileName = flag.String("json_out", "", "Output location for writing the json proto")
	var binaryProtoFileName = flag.String("pb_out", "", "Output location for writing the binary proto")
	var keepReferences = flag.Bool("keep-refs", false, "Disable resolution of $ref references")
	var logErrors = flag.Bool("errors", false, "Log errors to a file")

	flag.Parse()

	flag.Usage = func() {
		fmt.Printf("Usage: openapi-compiler [OPTION] <INPUT_OPEN_API_FILE>\n")
		fmt.Printf("INPUT_OPEN_API_FILE is the path to the input swagger " +
			"file to parse.\n")
		fmt.Printf("Output is generated based on the options given (default " +
			"output files are generated in the current directory if no " +
			"options are provided):\n")
		flag.PrintDefaults()
	}

	flag.Parse()

	var input string

	if len(flag.Args()) == 1 {
		input = flag.Arg(0)
	} else {
		flag.Usage()
		return
	}

	fmt.Printf("Compiling %s (%s)\n", input, openapi_v2.Version())

	raw, err := compiler.ReadFile(input)
	if err != nil {
		fmt.Printf("Error: No Specification\n%+v\n", err)
		os.Exit(-1)
	}

	if *rawInput {
		rawDescription := compiler.DescribeMap(raw, "")
		rawFileName := strings.TrimSuffix(path.Base(input), path.Ext(input)) + ".raw"
		ioutil.WriteFile(rawFileName, []byte(rawDescription), 0644)
	}

	errorFileName := strings.TrimSuffix(path.Base(input), path.Ext(input)) + ".errors"

	document, err := openapi_v2.NewDocument(raw, compiler.NewContext("$root", nil))
	if err != nil {
		fmt.Printf("%+v\n", err)
		if *logErrors {
			ioutil.WriteFile(errorFileName, []byte(err.Error()), 0644)
		}
		os.Exit(-1)
	}

	if !*keepReferences {
		_, err = document.ResolveReferences(input)
		if err != nil {
			fmt.Printf("%+v\n", err)
			if *logErrors {
				ioutil.WriteFile(errorFileName, []byte(err.Error()), 0644)
			}
			os.Exit(-1)
		}
	}

	if *textProtoFileName == "" {
		*textProtoFileName = strings.TrimSuffix(path.Base(input), path.Ext(input)) + ".text"
	}
	ioutil.WriteFile(*textProtoFileName, []byte(proto.MarshalTextString(document)), 0644)
	fmt.Printf("Text output file: %s\n", *textProtoFileName)

	if *jsonProtoFileName == "" {
		*jsonProtoFileName = strings.TrimSuffix(path.Base(input), path.Ext(input)) + ".json"
	}
	jsonBytes, _ := json.Marshal(document)
	ioutil.WriteFile(*jsonProtoFileName, jsonBytes, 0644)
	fmt.Printf("Json output file: %s\n", *jsonProtoFileName)

	if *binaryProtoFileName == "" {
		*binaryProtoFileName = strings.TrimSuffix(path.Base(input), path.Ext(input)) + ".pb"
	}
	protoBytes, _ := proto.Marshal(document)
	ioutil.WriteFile(*binaryProtoFileName, protoBytes, 0644)
	fmt.Printf("Protobuf output file: %s\n", *binaryProtoFileName)
}
