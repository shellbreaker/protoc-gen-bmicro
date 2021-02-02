package main

import (
	"github.com/gogo/protobuf/proto"
	"github.com/gogo/protobuf/protoc-gen-gogo/generator"
	"github.com/shellbreaker/protoc-gen-bmicro/plugin"
	"io/ioutil"
	"os"
	"strings"
)

func main() {
	g := generator.New()

	data, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		g.Error(err, "reading stdin")
	}

	err = proto.Unmarshal(data, g.Request)
	if err != nil {
		g.Error(err, "unmashalling data")
	}

	if len(g.Request.FileToGenerate) == 0 {
		g.Fail("no files to generate")
	}

	if p := g.Request.Parameter; p != nil {
		g.CommandLineParameters(*p)
	}

	g.WrapTypes()
	g.SetPackageNames()
	g.BuildTypeNameMap()
	g.GeneratePlugin(plugin.New())

	for i := 0; i < len(g.Response.File); i++ {
		g.Response.File[i].Name = proto.String(strings.Replace(*g.Response.File[i].Name, ".pb.go", ".pb.bmicro.go", -1))
	}

	// Send back the results.
	data, err = proto.Marshal(g.Response)
	if err != nil {
		g.Error(err, "failed to marshal output proto")
	}
	_, err = os.Stdout.Write(data)
	if err != nil {
		g.Error(err, "failed to write output proto")
	}
}
