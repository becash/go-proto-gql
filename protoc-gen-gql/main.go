package main

import (
	"bytes"
	"google.golang.org/protobuf/compiler/protogen"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/vektah/gqlparser/v2/formatter"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/pluginpb"

	"github.com/danielvladco/go-proto-gql/pkg/generator"
)

func main() {
	in, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		log.Fatal(err)
	}
	req := &pluginpb.CodeGeneratorRequest{}
	if err := proto.Unmarshal(in, req); err != nil {
		log.Fatal(err)
	}

	files, err := generate(req)
	res := &pluginpb.CodeGeneratorResponse{
		File:              files,
		SupportedFeatures: proto.Uint64(uint64(pluginpb.CodeGeneratorResponse_FEATURE_PROTO3_OPTIONAL)),
	}
	if err != nil {
		res.Error = proto.String(err.Error())
	}

	out, err := proto.Marshal(res)
	if err != nil {
		log.Fatal(err)
	}
	if _, err := os.Stdout.Write(out); err != nil {
		log.Fatal(err)
	}
}

func generate(req *pluginpb.CodeGeneratorRequest) (outFiles []*pluginpb.CodeGeneratorResponse_File, err error) {
	var genServiceDesc bool
	var merge bool
	var prefix bool
	var extension = generator.DefaultExtension
	var output string
	config := &generator.Config{}
	for _, param := range strings.Split(req.GetParameter(), ",") {
		var value string
		if i := strings.Index(param, "="); i >= 0 {
			value, param = param[i+1:], param[0:i]
		}
		switch param {
		case "svc":
			if genServiceDesc, err = strconv.ParseBool(value); err != nil {
				return nil, err
			}
		case "merge":
			if merge, err = strconv.ParseBool(value); err != nil {
				return nil, err
			}
		case "prefix":
			if prefix, err = strconv.ParseBool(value); err != nil {
				return nil, err
			}
			config.TypePrefix = &prefix
		case "ext":
			extension = strings.Trim(value, ".")
		case "go_model":
			config.GoModel = &value
		case "output":
			config.Output = &value
		}
	}
	p, err := protogen.Options{}.New(req)
	if err != nil {
		log.Fatal(err)
	}
	descs, err := generator.CreateDescriptorsFromProto(req)
	if err != nil {
		return nil, err
	}

	gqlDesc, err := generator.NewSchemas(descs, merge, genServiceDesc, p, config)
	if err != nil {
		return nil, err
	}
	for _, schema := range gqlDesc {
		buff := &bytes.Buffer{}
		formatter.NewFormatter(buff).FormatSchema(schema.AsGraphql())

		if config.Output != nil && *config.Output != "" {
			output = *config.Output
		} else {
			output = resolveGraphqlFilename(schema.FileDescriptors[0].GetName(), merge, extension)
		}

		outFiles = append(outFiles, &pluginpb.CodeGeneratorResponse_File{
			Name:    proto.String(output),
			Content: proto.String(buff.String()),
		})
	}

	return
}

func resolveGraphqlFilename(protoFileName string, merge bool, extension string) string {
	if merge {
		gqlFileName := "schema." + extension
		absProtoFileName, err := filepath.Abs(protoFileName)
		if err == nil {
			protoDirSlice := strings.Split(filepath.Dir(absProtoFileName), string(filepath.Separator))
			if len(protoDirSlice) > 0 {
				gqlFileName = protoDirSlice[len(protoDirSlice)-1] + "." + extension
			}
		}
		protoDir, _ := path.Split(protoFileName)
		return path.Join(protoDir, gqlFileName)
	}

	return strings.TrimSuffix(protoFileName, path.Ext(protoFileName)) + "." + extension
}
