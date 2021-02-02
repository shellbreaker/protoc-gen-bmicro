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

	p.generateHelpers()
	p.generateGateway(srvGenerators)
}

func (p *Plugin) GenerateImports(file *generator.FileDescriptor) {
	p.write("\"context\"")
	p.write("\"strings\"")
	p.write("\"reflect\"")
	p.write("\"github.com/astaxie/beego\"")
	p.write("bctx \"github.com/astaxie/beego/context\"")
	p.write("jsoniter \"github.com/json-iterator/go\"")
	p.write("microErr \"github.com/micro/go-micro/v2/errors\"")
}

func (p *Plugin) generateGateway(sg []*serviceGenerator) {
	fnFingerPrints := make(map[string]bool)
	for _, g := range sg {
		if g != nil {
			srvName := g.service.GetName()
			//lowerSrvName := strings.ToLower(srvName)

			p.write("func Register%sGateway(cli %sService, opts ...Option) {", srvName, srvName)
			p.write(`settings := new(gatewaySettings)
					for _, opt := range opts {
						opt.fn(settings)
					}
			`)
			for _, m := range g.methodExtractors {
				if m.Extract() == nil {
					ccMethod, uri := generator.CamelCase(m.GatewayMethod()), m.GatewayURI()
					ffp := fmt.Sprintf("%s%s%s", srvName, ccMethod, uri)
					inputType, outputType := m.method.GetInputType(), m.method.GetOutputType()
					if _, ex := fnFingerPrints[ffp]; !ex {
						fnFingerPrints[ffp] = true
						input, methodName := inputType[strings.LastIndex(inputType, ".")+1:], m.method.GetName()
						output := outputType[strings.LastIndex(outputType, ".")+1:]
						p.write("//%s(%s) %s", methodName, input, output)
						p.write(`beego.%s("%s", func(c *bctx.Context) {
							var data *%s
							var e error
							err := settings.CloneError()
							defer func() {
								if e == nil {
									c.Output.JSON(data, true, true)
								} else {
									c.Output.JSON(err, true, true)
								}
							}()
							params := new(%s)
							e = ParseParams(c, params)
							if e == nil {
								data, e = cli.%s(context.TODO(), params)
								if e != nil {
									err.Set(10500, microErr.Parse(e.Error()).GetDetail())
								}
							} else {
								err.Set(10400, e.Error())
							}
					
						})`, ccMethod, m.GatewayURI(), output, input, methodName)
					}
				}
			}
			p.write("}\n")
		}
	}
}

func (p *Plugin) generateHelpers() {
	defines := `var json = jsoniter.ConfigCompatibleWithStandardLibrary

	func ParseParams(c *bctx.Context, obj interface{}) (err error) {
		if strings.Contains(strings.ToLower(c.Input.Header("content-type")), "json") {
			err = json.Unmarshal(c.Input.RequestBody, obj)
		} else {
			err = c.Request.ParseForm()
			if err == nil {
				err = beego.ParseForm(c.Request.Form, obj)
			}
		}
		return
	}

	type gatewaySettings struct {
		CustomError Error
	}

	func (s *gatewaySettings) CloneError() Error {
		if s.CustomError != nil {
			rv := reflect.ValueOf(s.CustomError)
			if rv.Kind() == reflect.Ptr {
				rv = rv.Elem()
			}
			err, ok := reflect.New(rv.Type()).Interface().(Error)
			if ok {
				return err
			}
		}
		return new(ErrorBase)
	}

	type Option struct {
		fn func(s *gatewaySettings)
	}

	func SetCustomError(e Error) Option {
		return Option{func(s *gatewaySettings) {
			s.CustomError = e
		}}
	}

	type Error interface {
		Set(int, string)
	}
	`
	p.write(defines)
	p.write("type ErrorBase struct {")
	p.write("Code int `json:\"code\"`")
	p.write("Msg string `json:\"msg\"`")
	p.write(`}
		func (ei *ErrorBase) Set(c int, m string) {
			ei.Code, ei.Msg = c, m
		}
	`)
}

func (p *Plugin) write(s string, args ...interface{}) {
	p.core.P(fmt.Sprintf(s, args...))
}

//TODO: validation
type messageField struct {
	field            *descriptor.FieldDescriptorProto
	trailingComments string
}

func (m *messageField) extract() {}

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
		return uParsed.Path
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
