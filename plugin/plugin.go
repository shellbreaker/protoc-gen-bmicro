package plugin

import (
	"errors"
	"fmt"
	"github.com/gogo/protobuf/gogoproto"
	"github.com/gogo/protobuf/protoc-gen-gogo/descriptor"
	"github.com/gogo/protobuf/protoc-gen-gogo/generator"
	"net/url"
	"regexp"
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

	srvGenerators := make([]*serviceGenerator, len(srvs))

	for _, loc := range file.GetSourceCodeInfo().GetLocation() {
		if !gogoproto.IsProto3(file.FileDescriptorProto) {
			continue
		}
		locPath, leadingComments, trailingComments := loc.GetPath(), loc.GetLeadingComments(), loc.GetTrailingComments()
		if len(locPath) >= 4 && (trailingComments != "" || leadingComments != "") {
			switch {
			case locPath[0] == 4 && trailingComments != "":
				(&messageField{
					field:            msgs[locPath[1]].Field[locPath[3]],
					trailingComments: trailingComments,
				}).extract()
			case locPath[0] == 6 && leadingComments != "":
				if g := srvGenerators[locPath[1]]; g == nil {
					srvGenerators[locPath[1]] = &serviceGenerator{
						service:          srvs[locPath[1]],
						methodExtractors: make([]*methodExtractor, len(srvs[locPath[1]].Method)),
					}
				}
				srvGenerators[locPath[1]].methodExtractors[locPath[3]] = &methodExtractor{
					method:  srvs[locPath[1]].Method[locPath[3]],
					comment: leadingComments,
				}
			}
		}
	}

	p.generateGateway(srvGenerators)
	p.generateHelperFuncs()
}

func (p *Plugin) GenerateImports(file *generator.FileDescriptor) {
	p.write("\"context\"")
	p.write("\"strings\"")
	p.write("\"github.com/astaxie/beego\"")
	p.write("bctx \"github.com/astaxie/beego/context\"")
	p.write("jsoniter \"github.com/json-iterator/go\"")
}

func (p *Plugin) generateGateway(sg []*serviceGenerator) {
	fnFingerPrints := make(map[string]bool)
	for _, g := range sg {
		if g != nil {
			srvName := g.service.GetName()
			lowerSrvName := strings.ToLower(srvName)

			//p.write("func Register%sGateway(cli %sService) {", srvName, srvName)
			//p.write("beego.Router(\"%s\", New%sGateway(cli))", "a", srvName)
			//p.write("}")

			//p.write("func New%sGateway(cli %sService) *%sController {", srvName, srvName, lowerSrvName)
			//p.write("return &%sController{", lowerSrvName)
			//p.write("microClient: cli,")
			//p.write("}")
			//p.write("}")

			p.write("type %sController struct {", lowerSrvName)
			p.write("GatewayController")
			p.write("microClient %sService", srvName)
			p.write("}")

			for _, m := range g.methodExtractors {
				if m.Extract() == nil {
					ccMethod, uri := generator.CamelCase(m.gateway.Method), m.gateway.URI
					ffp := fmt.Sprintf("%s%s%s", srvName, ccMethod, uri)
					inputType, outputType := m.method.GetInputType(), m.method.GetOutputType()
					if _, ex := fnFingerPrints[ffp]; !ex {
						fnFingerPrints[ffp] = true
						input, methodName := inputType[strings.LastIndex(inputType, ".")+1:], m.method.GetName()
						output := outputType[strings.LastIndex(outputType, ".")+1:]
						p.write("//output:%s", output)

						p.write("type %s_%s struct {", lowerSrvName, methodName)
						p.write("%s", input)
						p.write("microClient %sService", srvName)
						p.write("}")

						p.write("func (i *%s_%s) Exec(_ *bctx.Context) (interface{}, error) {", lowerSrvName, methodName)
						p.write("return i.microClient.%s(context.TODO(), &i.%s)", methodName, input)
						p.write("}\n")

						p.write("func (c *%sController) %s() {", strings.ToLower(srvName), ccMethod)
						p.write("api := new(%s_%s)", lowerSrvName, methodName)
						p.write("api.microClient = c.microClient")
						p.write("c.ServeJson(api)")
						p.write("}")
					}
				}
			}
		}
	}
}

func (p *Plugin) generateHelperFuncs() {
	defines := `var json = jsoniter.ConfigCompatibleWithStandardLibrary

	type JsonApi interface {
		Exec(*bctx.Context) (interface{}, error)
	}

	type GatewayController struct {
		beego.Controller
	}

	func (g *GatewayController) ServeJson(impl JsonApi) {
		var r interface{}
		var err error
		defer func() {
			if err != nil {

			} else {

			}
			g.Data["json"] = r
			g.ServeJSON()
		}()
		if strings.Contains(strings.ToLower(g.Ctx.Input.Header("content-type")), "json") {
			if e := json.Unmarshal(g.Ctx.Input.RequestBody, impl); e != nil {
				//code, msg
				return
			}
		} else {
			if e := g.ParseForm(impl); e != nil {
				//code, msg
				return
			}
		}
		r, err = impl.Exec(g.Ctx)
	}
	
	`

	p.write(defines)

}

func (p *Plugin) write(s string, args ...interface{}) { p.core.P(fmt.Sprintf(s, args...)) }

type messageField struct {
	field            *descriptor.FieldDescriptorProto
	trailingComments string
}

func (m *messageField) extract() {
	//fmt.Println(">>>>>>>>>>>>>>>>>>>>>>>>", m.field, ":::: message ::::", m.trailingComments, "<<<<<<<<<<<<<<<<<<<<<<<<<<<                         ")

}

type ServiceExtractor interface {
	Extract() error
	GatewayMethod() string
	GatewayURI() string
}

type serviceGenerator struct {
	service          *descriptor.ServiceDescriptorProto
	methodExtractors []*methodExtractor
}

type methodExtractor struct {
	method  *descriptor.MethodDescriptorProto
	comment string
	gateway struct {
		Method, URI string
	}
}

func (m *methodExtractor) Extract() error {
	if rows := strings.Split(m.comment, "\n"); len(rows) > 0 {
		for _, row := range rows {
			if strings.HasPrefix(strings.TrimSpace(row), "@router") {
				if cols := strings.Split(strings.Replace(row, "@router", "", -1), " "); len(cols) > 0 {
					for _, col := range cols {
						col = strings.TrimSpace(col)
						if col == "" {
							continue
						}
						if f := m.extractHttpMethod(col); f != "" {
							m.gateway.Method = f
						}
						if u := m.extractURI(col); u != "" {
							m.gateway.URI = u
						}
					}
				}
			}
		}
	}
	if m.gateway.URI == "" || m.gateway.Method == "" {
		return errors.New("can not extract from comment")
	}
	return nil
}

func (m *methodExtractor) GatewayURI() string {
	return m.gateway.URI
}

func (m *methodExtractor) GatewayMethod() string {
	return m.gateway.Method
}

func (m *methodExtractor) extractURI(u string) string {
	if uParsed, err := url.ParseRequestURI(u); err == nil {
		return uParsed.EscapedPath()
	}
	return ""
}

func (m *methodExtractor) extractHttpMethod(f string) string {
	if exp := regexp.MustCompile(`[\w]+`).FindAllString(f, 1); len(exp) == 1 {
		method := strings.ToUpper(exp[0])
		switch method {
		case "GET", "HEAD", "POST", "PUT", "PATCH", "DELETE", "CONNECT", "OPTIONS", "TRACE":
			return strings.ToLower(method)
		}
	}
	return ""
}
