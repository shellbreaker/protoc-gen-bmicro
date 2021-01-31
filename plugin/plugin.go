package plugin

import (
	"fmt"
	"github.com/gogo/protobuf/gogoproto"
	"github.com/gogo/protobuf/protoc-gen-gogo/descriptor"
	"github.com/gogo/protobuf/protoc-gen-gogo/generator"
	"strings"
)

type Plugin struct {
	core     *generator.Generator
	importer generator.PluginImports
}

func New() generator.Plugin {
	return &Plugin{}
}

func (p *Plugin) Name() string {
	return "bmicro"
}

func (p *Plugin) Init(g *generator.Generator) {
	p.core = g
	p.importer = generator.NewPluginImports(p.core)
}

func (p *Plugin) Generate(file *generator.FileDescriptor) {
	if len(file.FileDescriptorProto.Service) == 0 {
		return
	}
	msgs, srvs := file.Messages(), file.GetService()
	for _, loc := range file.GetSourceCodeInfo().GetLocation() {
		if !gogoproto.IsProto3(file.FileDescriptorProto) {
			continue
		}
		locPath, leadingComments, trailingComments := loc.GetPath(), loc.GetLeadingComments(), loc.GetTrailingComments()
		if len(locPath) >= 4 && (trailingComments != "" || leadingComments != "") {
			switch locPath[0] {
			case 4:
				//fmt.Println(">>>>>>>>>>>>>>>>>>>>>>>>", msgs[locPath[1]].Field[locPath[3]], ":::: message ::::", leadingComments, trailingComments, "<<<<<<<<<<<<<<<<<<<<<<<<<<<                         ")
				(&messageField{
					field:            msgs[locPath[1]].Field[locPath[3]],
					trailingComments: trailingComments,
				}).extract()
			case 6:
				(&serviceMethod{
					method:          srvs[locPath[1]].Method[locPath[3]],
					leadingComments: leadingComments,
				}).extract()
			}
		}
	}

	/*for _, msg := range file.Messages() {
		if msg.DescriptorProto.GetOptions().GetMapEntry() {
			continue
		}


		if gogoproto.IsProto3(file.FileDescriptorProto) {
			//for _, field := range msg.Field {
			//	fmt.Println(field, "<=======================")
			//
			//}
		}
	}*/
}

func (p *Plugin) GenerateImports(file *generator.FileDescriptor) {

}

type messageField struct {
	field            *descriptor.FieldDescriptorProto
	trailingComments string
}

func (m *messageField) extract() {
	//fmt.Println(">>>>>>>>>>>>>>>>>>>>>>>>", m.field, ":::: message ::::", m.trailingComments, "<<<<<<<<<<<<<<<<<<<<<<<<<<<                         ")

}

type serviceMethod struct {
	method          *descriptor.MethodDescriptorProto
	leadingComments string
}

func (s *serviceMethod) extract() {
	//fmt.Println(">>>>>>>>>>>>>>>>>>>>>>>>", s.method, ":::: service ::::", s.leadingComments, "<<<<<<<<<<<<<<<<<<<<<<<<<<<                         ")
	if rows := strings.Split(s.leadingComments, "\n"); len(rows) > 0 {
		for _, row := range rows {
			if strings.HasPrefix(strings.TrimSpace(row), "@router") {
				if cols := strings.Split(strings.Replace(row, "@router", "", -1), " "); len(cols) > 0 {
					for idx, col := range cols {
						col = strings.TrimSpace(col)
						if col == "" {
							continue
						}
						fmt.Printf("========= %d -> %v =========            \n", idx, col)
					}
				}
			}

		}
	}
}
