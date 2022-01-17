package main

import (
	"encoding/json"
	"flag"
	"io/ioutil"
	"log"
	"os"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/fatih/color"
	"github.com/graphql-go/graphql/language/ast"
	"github.com/graphql-go/graphql/language/kinds"
	"github.com/graphql-go/graphql/language/parser"
	"github.com/graphql-go/graphql/language/source"
	"github.com/jinzhu/copier"
	"github.com/jinzhu/inflection"
	"github.com/kenshaw/snaker"
)

var (
	schemaFile = flag.String("schema", "", "input schema file")
	out        = flag.String("o", "", "output file")
)

type Name struct {
	Kind  string `json:"name"`
	Value string `json:"value"`
}

func (t *Name) GetKind() string {
	return t.Kind
}

type StringValue struct {
	Kind  string `json:"kind"`
	Value string `json:"value"`
}

func (t *StringValue) GetKind() string {
	return t.Kind
}
func (t *StringValue) GetValue() interface{} {
	return t.Value
}

type Named struct {
	Kind string `json:"kind"`
	Name *Name  `json:"name"`
}

func (t *Named) GetKind() string {
	return t.Kind
}
func (t *Named) String() string {
	return t.Kind
}

type Directive struct {
	Kind      string      `json:"kind"`
	Name      *Name       `json:"name"`
	Arguments []*Argument `json:"arguments"`
}

func (t *Directive) GetKind() string {
	return t.Kind
}

type Argument struct {
	Kind  string `json:"kind"`
	Name  *Name  `json:"name"`
	Value Value  `json:"value"`
}

func (t *Argument) GetKind() string {
	return t.Kind
}

type MyTypeImpl struct {
	Name    string
	IsArray bool
	NotNull bool
}

func (m MyTypeImpl) String() string {
	goType := m.Name
	if !m.NotNull {
		goType = "*" + goType
	}
	if m.IsArray {
		goType = "[]" + goType
	}
	return goType
}
func (m MyTypeImpl) MarshalJSON() ([]byte, error) {
	return json.Marshal(m.String())
}

type FieldDefinition struct {
	Kind          string                            `json:"-"`
	Name          *Name                             `json:"-"`
	GoName        string                            `json:"Name"`
	NameJson      string                            `json:"nameJson"`
	GoVarName     string                            `json:"name"`
	GoNames       string                            `json:"Names"`
	GoVarNames    string                            `json:"names"`
	GoShortName   string                            `json:"n"`
	NameDb        string                            `json:"nameDb"`
	NamesDb       string                            `json:"namesDb"`
	NameExact     string                            `json:"NameExact"`
	NameExactJson string                            `json:"nameExact"`
	Description   *StringValue                      `json:"-"`
	Type          MyType                            `json:"-"`
	GoType        *MyTypeImpl                       `json:"Type"`
	BaseType      string                            `json:"baseType"`
	TypeDb        string                            `json:"typeDb"`
	IsArray       bool                              `json:"isArray"`
	NotNull       bool                              `json:"notNull"`
	Arguments     []*InputValueDefinition           `json:"arguments,omitempty"`
	MyDirectives  map[string]map[string]interface{} `json:"directives,omitempty"`
	Key           string                            `json:"key"`
}

func (t *FieldDefinition) GetKind() string {
	return t.Kind
}

type InputValueDefinition struct {
	Kind          string                            `json:"-"`
	Name          *Name                             `json:"-"`
	GoName        string                            `json:"Name"`
	NameJson      string                            `json:"nameJson"`
	GoVarName     string                            `json:"name"`
	GoNames       string                            `json:"Names"`
	GoVarNames    string                            `json:"names"`
	GoShortName   string                            `json:"n"`
	NameDb        string                            `json:"nameDb"`
	NamesDb       string                            `json:"namesDb"`
	NameExact     string                            `json:"NameExact"`
	NameExactJson string                            `json:"nameExact"`
	Description   *StringValue                      `json:"-"`
	Type          MyType                            `json:"-"`
	GoType        *MyTypeImpl                       `json:"Type"`
	BaseType      string                            `json:"baseType"`
	TypeDb        string                            `json:"typeDb"`
	IsArray       bool                              `json:"isArray"`
	NotNull       bool                              `json:"notNull"`
	DefaultValue  Value                             `json:"-"`
	MyDirectives  map[string]map[string]interface{} `json:"directives,omitempty"`
	Key           string                            `json:"key"`
}

func (t *InputValueDefinition) GetKind() string {
	return t.Kind
}

type Variable struct {
	Kind string
	Name *Name
}

func (v *Variable) GetValue() interface{} {
	return v.Name
}
func (v *Variable) GetKind() string {
	return v.Kind
}

type IntValue struct {
	Kind  string
	Value string
}

func (v *IntValue) GetKind() string {
	return v.Kind
}

func (v *IntValue) GetValue() interface{} {
	return v.Value
}

type FloatValue struct {
	Kind  string
	Value string
}

func (v *FloatValue) GetKind() string {
	return v.Kind
}

func (v *FloatValue) GetValue() interface{} {
	return v.Value
}

type BooleanValue struct {
	Kind  string
	Value bool
}

func (v *BooleanValue) GetKind() string {
	return v.Kind
}

func (v *BooleanValue) GetValue() interface{} {
	return v.Value
}

type EnumValue struct {
	Kind  string
	Value string
}

func (v *EnumValue) GetKind() string {
	return v.Kind
}

func (v *EnumValue) GetValue() interface{} {
	return v.Value
}

type ListValue struct {
	Kind   string
	Values []Value
}

func (v *ListValue) GetKind() string {
	return v.Kind
}

func (v *ListValue) GetValue() interface{} {
	return v.GetValues()
}

func (v *ListValue) GetValues() interface{} {
	return v.Values
}

type ObjectValue struct {
	Kind   string
	Fields []*ObjectField
}

func (v *ObjectValue) GetKind() string {
	return v.Kind
}
func (v *ObjectValue) GetValue() interface{} {
	return v.Fields
}

type ObjectField struct {
	Kind  string
	Name  *Name
	Value Value
}

func (f *ObjectField) GetKind() string {
	return f.Kind
}

func (f *ObjectField) GetValue() interface{} {
	return f.Value
}

type Value interface {
	GetValue() interface{}
	GetKind() string
}
type MyType interface {
	GetKind() string
	String() string
}
type ScalarDefinition struct {
	Description *StringValue `json:"-"`
	Name        *Name        `json:"name"`
	Kind        string       `json:"-"`
	Directives  []*Directive `json:"-"`
}

func (o ScalarDefinition) GetNodeKind() string {
	return o.Kind
}

type EnumDefinition struct {
	Name   *Name  `json:"-"`
	GoName string `json:"Name"`
	Key    string `json:"key"`

	GoVarName     string `json:"name"`
	GoNames       string `json:"Names"`
	GoVarNames    string `json:"names"`
	GoShortName   string `json:"n"`
	NameDb        string `json:"nameDb"`
	NamesDb       string `json:"namesDb"`
	NameExact     string `json:"NameExact"`
	NameExactJson string `json:"nameExact"`

	Kind       string                 `json:"kind"`
	Directives []*Directive           `json:"-"`
	Values     []*EnumValueDefinition `json:"fields"`
}

func (o EnumDefinition) GetNodeKind() string {
	return o.Kind
}

type EnumValueDefinition struct {
	Name   *Name  `json:"-"`
	GoName string `json:"Name"`

	NameExactJson string      `json:"nameExact"`
	GoType        *MyTypeImpl `json:"Type"`
	Key           string      `json:"key"`
}

type ObjectDefinition struct {
	Name   *Name  `json:"-"`
	GoName string `json:"Name"`
	Key    string `json:"key"`

	GoVarName     string `json:"name"`
	GoNames       string `json:"Names"`
	GoVarNames    string `json:"names"`
	GoShortName   string `json:"n"`
	NameDb        string `json:"nameDb"`
	NamesDb       string `json:"namesDb"`
	NameExact     string `json:"NameExact"`
	NameExactJson string `json:"nameExact"`
	Comment       string `json:"comment"`

	Kind         string                            `json:"kind"`
	Description  *StringValue                      `json:"-"`
	Interfaces   []*Named                          `json:"-"`
	MyDirectives map[string]map[string]interface{} `json:"directives,omitempty"`
	Fields       []*FieldDefinition                `json:"fields"`
}

func (o ObjectDefinition) GetNodeKind() string {
	return o.Kind
}

type InputObjectDefinition struct {
	Name   *Name  `json:"-"`
	GoName string `json:"Name"`
	Key    string `json:"key"`

	GoVarName     string `json:"name"`
	GoNames       string `json:"Names"`
	GoVarNames    string `json:"names"`
	GoShortName   string `json:"n"`
	NameDb        string `json:"nameDb"`
	NamesDb       string `json:"namesDb"`
	NameExact     string `json:"NameExact"`
	NameExactJson string `json:"nameExact"`
	NameInput     string `json:"NameInput"`
	Comment       string `json:"comment"`

	Kind         string                            `json:"kind"`
	Description  *StringValue                      `json:"-"`
	MyDirectives map[string]map[string]interface{} `json:"directives,omitempty"`
	Fields       []*InputValueDefinition           `json:"fields"`
}

func (o InputObjectDefinition) GetNodeKind() string {
	return o.Kind
}

type Node interface {
	GetNodeKind() string
}
type List struct {
	Kind string `json:"kind"`
	Type MyType `json:"type"`
}

func (t *List) GetKind() string {
	return t.Kind
}
func (t *List) String() string {
	return t.Kind
}

type NonNull struct {
	Kind string
	Type MyType
}

func (t *NonNull) GetKind() string {
	return t.Kind
}
func (t *NonNull) String() string {
	return t.Kind
}

func parseSchema(b []byte) (*ast.Document, error) {
	re := regexp.MustCompile("#[^\n]+\n")
	b = re.ReplaceAll(b, []byte{})
	_ast, err := parser.Parse(parser.ParseParams{
		Source: &source.Source{Body: b},
		Options: parser.ParseOptions{
			NoLocation: true,
			NoSource:   true,
		},
	})
	return _ast, err
}

func plural(s string) string {
	out := inflection.Plural(s)
	if out == "information" {
		return "informations"
	} else if out == "Information" {
		return "Informations"
	}
	return out
}

func convert(nodes []ast.Node) ([]Node, error) {
	warning := color.New(color.FgYellow)
	onodes := make([]Node, 0, len(nodes))
	for _, n := range nodes {
		switch v := n.(type) {
		case *ast.EnumDefinition:
			o := &EnumDefinition{}
			o.Kind = v.Kind
			o.Key = snaker.ForceCamelIdentifier(v.Name.Value)
			o.GoName = v.Name.Value
			o.Kind = kinds.EnumDefinition
			o.NameExactJson = v.Name.Value
			o.NameExact = snaker.ForceCamelIdentifier(o.NameExactJson)
			o.NameDb = snaker.CamelToSnake(inflection.Singular(o.GoName))
			o.NamesDb = plural(o.NameDb)
			o.GoVarName = lowerCamel(inflection.Singular(o.GoName))
			o.GoNames = plural(o.GoName)
			o.GoVarNames = lowerCamel(plural(o.GoVarName))
			o.GoShortName = shortName(o.GoName)
			o.Values = make([]*EnumValueDefinition, len(v.Values))
			for i, vv := range v.Values {
				x := &EnumValueDefinition{}
				x.GoType = &MyTypeImpl{Name: "string", NotNull: true}
				x.NameExactJson = vv.Name.Value
				x.GoName = snaker.ForceCamelIdentifier(strings.ToLower(x.NameExactJson))
				x.Key = x.NameExactJson
				o.Values[i] = x
			}
			onodes = append(onodes, o)
		case *ast.InputObjectDefinition:
			o := &InputObjectDefinition{}
			o.Kind = v.Kind
			o.Key = snaker.ForceCamelIdentifier(v.Name.Value)
			o.NameInput = v.Name.Value
			o.GoName = strings.TrimPrefix(v.Name.Value, "Input")
			o.NameExactJson = v.Name.Value
			o.NameExact = snaker.ForceCamelIdentifier(o.NameExactJson)
			o.NameDb = snaker.CamelToSnake(inflection.Singular(o.GoName))
			o.NamesDb = plural(o.NameDb)
			o.GoVarName = lowerCamel(inflection.Singular(o.GoName))
			o.GoNames = plural(o.GoName)
			o.GoVarNames = lowerCamel(plural(o.GoVarName))
			o.GoShortName = shortName(o.GoName)
			if v.Description != nil {
				o.Comment = v.Description.Value
			}

			o.MyDirectives = make(map[string]map[string]interface{}, len(v.Directives))
			for _, d := range v.Directives {
				if _, ok := o.MyDirectives[d.Name.Value]; !ok {
					o.MyDirectives[d.Name.Value] = make(map[string]interface{}, len(d.Arguments))
				}
				for _, a := range d.Arguments {
					o.MyDirectives[d.Name.Value][a.Name.Value] = a.Value.GetValue()
				}
			}

			if err := copier.Copy(o, v); err != nil {
				return nil, err
			}
			for _, m := range o.Fields {
				m.NameExactJson = m.Name.Value
				m.NameExact = snaker.ForceCamelIdentifier(m.NameExactJson)
				m.NameJson = lowerCamel(inflection.Singular(m.Name.Value))
				m.Key = m.NameJson
				m.GoName = snaker.ForceCamelIdentifier(m.NameJson)

				m.NameDb = snaker.CamelToSnake(inflection.Singular(m.GoName))
				m.NamesDb = plural(m.NameDb)
				if m.GoName == "ID" {
					m.GoVarName = "id"
					m.GoNames = "Ids"
				} else {
					m.GoVarName = lowerCamel(inflection.Singular(m.GoName))
					m.GoNames = plural(m.GoName)
					if strings.HasSuffix(m.GoNames, "IDS") {
						m.GoNames = m.GoNames[:len(m.GoNames)-2] + "ds"
					}
				}
				m.GoVarNames = lowerCamel(plural(m.GoVarName))
				if strings.HasSuffix(m.GoVarNames, "IDS") {
					m.GoVarNames = m.GoVarNames[:len(m.GoVarNames)-2] + "ds"
				}
				m.GoShortName = shortName(m.GoName)
				m.GoType = &MyTypeImpl{}
				getGoType(m.Type, m.GoType)
				m.BaseType = m.GoType.Name
				m.IsArray = m.GoType.IsArray
				m.NotNull = m.GoType.NotNull
				m.TypeDb = snaker.CamelToSnake(m.GoType.Name)
			}
			onodes = append(onodes, o)
		case *ast.ObjectDefinition:
			o := &ObjectDefinition{}
			o.Kind = v.Kind
			o.Key = snaker.ForceCamelIdentifier(v.Name.Value)
			o.GoName = v.Name.Value

			o.NameExactJson = v.Name.Value
			o.NameExact = snaker.ForceCamelIdentifier(o.NameExactJson)
			o.NameDb = snaker.CamelToSnake(inflection.Singular(o.GoName))
			o.NamesDb = plural(o.NameDb)
			o.GoVarName = lowerCamel(inflection.Singular(o.GoName))
			o.GoNames = plural(o.GoName)
			o.GoVarNames = lowerCamel(plural(o.GoVarName))
			o.GoShortName = shortName(o.GoName)
			if v.Description != nil {
				o.Comment = v.Description.Value
			}

			o.MyDirectives = make(map[string]map[string]interface{}, len(v.Directives))
			for _, d := range v.Directives {
				if _, ok := o.MyDirectives[d.Name.Value]; !ok {
					o.MyDirectives[d.Name.Value] = make(map[string]interface{}, len(d.Arguments))
				}
				for _, a := range d.Arguments {
					o.MyDirectives[d.Name.Value][a.Name.Value] = a.Value.GetValue()
				}
			}

			if err := copier.Copy(o, v); err != nil {
				return nil, err
			}
			for i, m := range o.Fields {
				if strings.HasSuffix(m.Name.Value, "ID") {
					warning.Printf("WARNING: Model '%s', Field '%s' ends with ID, use Id instead\n", o.Name.Value, m.Name.Value)
				}
				m.NameExactJson = m.Name.Value
				m.NameExact = snaker.ForceCamelIdentifier(m.NameExactJson)
				m.NameJson = lowerCamel(inflection.Singular(m.Name.Value))
				if o.Key == "Query" || o.Key == "Mutation" {
					m.Key = m.NameExactJson
				} else {
					m.Key = m.NameJson
				}
				m.MyDirectives = make(map[string]map[string]interface{}, len(v.Fields[i].Directives))
				for _, d := range v.Fields[i].Directives {
					if _, ok := m.MyDirectives[d.Name.Value]; !ok {
						m.MyDirectives[d.Name.Value] = make(map[string]interface{}, len(d.Arguments))
					}
					for _, a := range d.Arguments {
						m.MyDirectives[d.Name.Value][a.Name.Value] = a.Value.GetValue()
					}
				}
				if m.Key == "id" {
					m.Key = o.GoVarName + "Id"
				}
				m.GoName = snaker.ForceCamelIdentifier(m.NameJson)

				m.NameDb = snaker.CamelToSnake(inflection.Singular(m.GoName))
				m.NamesDb = plural(m.NameDb)
				if m.GoName == "ID" {
					m.GoVarName = "id"
					m.GoNames = "Ids"
				} else {
					m.GoVarName = lowerCamel(inflection.Singular(m.GoName))
					m.GoNames = plural(m.GoName)
					if strings.HasSuffix(m.GoNames, "IDS") {
						m.GoNames = m.GoNames[:len(m.GoNames)-2] + "ds"
					}
				}
				m.GoVarNames = lowerCamel(plural(m.GoVarName))
				if strings.HasSuffix(m.GoVarNames, "IDS") {
					m.GoVarNames = m.GoVarNames[:len(m.GoVarNames)-2] + "ds"
				}
				m.GoShortName = shortName(m.GoName)

				m.GoType = &MyTypeImpl{}
				getGoType(m.Type, m.GoType)
				m.BaseType = m.GoType.Name
				m.IsArray = m.GoType.IsArray
				m.NotNull = m.GoType.NotNull
				m.TypeDb = snaker.CamelToSnake(m.GoType.Name)
				for _, n := range m.Arguments {
					n.NameExactJson = n.Name.Value
					n.NameExact = snaker.ForceCamelIdentifier(n.NameExactJson)
					n.NameJson = lowerCamel(inflection.Singular(n.Name.Value))
					n.Key = n.NameJson
					if n.Key == "id" {
						n.Key = o.GoVarName + "Id"
					}
					n.GoName = snaker.ForceCamelIdentifier(n.NameJson)

					n.NameDb = snaker.CamelToSnake(inflection.Singular(n.GoName))
					n.NamesDb = plural(n.NameDb)
					if n.GoName == "ID" {
						n.GoVarName = "id"
						n.GoNames = "Ids"
					} else {
						n.GoVarName = lowerCamel(inflection.Singular(n.GoName))
						n.GoNames = plural(n.GoName)
						if strings.HasSuffix(n.GoNames, "IDS") {
							n.GoNames = n.GoNames[:len(n.GoNames)-2] + "ds"
						}
					}
					n.GoVarNames = lowerCamel(plural(n.GoVarName))
					if strings.HasSuffix(n.GoVarNames, "IDS") {
						n.GoVarNames = n.GoVarNames[:len(n.GoVarNames)-2] + "ds"
					}
					n.GoShortName = shortName(n.GoName)

					n.GoType = &MyTypeImpl{}
					getGoType(n.Type, n.GoType)
					n.BaseType = n.GoType.Name
					n.IsArray = n.GoType.IsArray
					n.NotNull = n.GoType.NotNull
					n.TypeDb = snaker.CamelToSnake(n.GoType.Name)
				}
			}
			onodes = append(onodes, o)
		}
	}
	return onodes, nil
}

func getGoType(m MyType, typ *MyTypeImpl) {
	switch w := m.(type) {
	case *ast.NonNull:
		typ.NotNull = true
		getGoType(w.Type, typ)
	case *ast.Named:
		typ.Name = convertGoType(w.Name.Value)
	case *ast.List:
		typ.IsArray = true
		getGoType(w.Type, typ)
	}
}

func convertGoType(s string) string {
	if s == "String" {
		return "string"
	} else if s == "Boolean" {
		return "bool"
	} else if s == "DateTime" {
		return "datetime"
	} else if s == "Int" {
		return "int"
	}
	return s
}

func lowerCamel(s string) string {
	if s == "" {
		return ""
	}
	r, n := utf8.DecodeRuneInString(s)
	return string(unicode.ToLower(r)) + s[n:]
}

var shortNameRe = regexp.MustCompile("[A-Z]")

func shortName(s string) string {
	return strings.ToLower(strings.Join(shortNameRe.FindAllString(s, -1), ""))
}

type FileContent struct {
	FileKind string `json:"kind"`
	Data     []Node `json:"data"`
}

func process() error {
	b, err := os.ReadFile(*schemaFile)
	if err != nil {
		return err
	}
	parsed, err := parseSchema(b)
	if err != nil {
		return err
	}
	parsed2, err := convert(parsed.Definitions)
	if err != nil {
		return err
	}
	fileContent := FileContent{
		FileKind: "gql",
		Data:     parsed2,
	}
	parsedJson, err := json.MarshalIndent(fileContent, "", "\t")
	if err != nil {
		return err
	}

	if *out == "-" {
		if _, err := os.Stdout.Write(parsedJson); err != nil {
			return err
		}
	} else {
		outFile := *out
		if outFile == "" {
			outFile = strings.Replace(*schemaFile, ".graphql", "-graphql.json", 1)
		}
		if err := ioutil.WriteFile(outFile, parsedJson, 0644); err != nil {
			return err
		}
	}

	return nil
}

func main() {
	flag.Parse()

	if err := process(); err != nil {
		log.Fatalln(err)
	}
}
