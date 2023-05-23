package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/gookit/color"
	"github.com/graphql-go/graphql/language/ast"
	"github.com/graphql-go/graphql/language/kinds"
	"github.com/graphql-go/graphql/language/parser"
	"github.com/graphql-go/graphql/language/source"
	"github.com/iancoleman/strcase"
	"github.com/jinzhu/copier"
	"github.com/jinzhu/inflection"
	"github.com/kenshaw/snaker"
)

var (
	schemaFile     = flag.String("schema", "", "input schema file")
	kind           = flag.String("kind", "gql", "override kind")
	queryType      = flag.String("query-type", "Query", "override Query type")
	mutationType   = flag.String("mutation-type", "Mutation", "override Mutation type")
	optional       = flag.Bool("optional", false, "all optional type except input")
	removeComments = flag.Bool("remove-comments", false, "remove comments")
	out            = flag.String("o", "", "output file")
	debug          = flag.Bool("debug", false, "debug")
	verbose        = flag.Bool("v", false, "verbose")
	force          = flag.Bool("f", false, "force update")
	changedFlag    = flag.Bool("changed", false, "exit code 2 if changed")
	gqlgen         = flag.Bool("gqlgen", false, "use gqlgen case")
	typeMappings   = flag.String("type-mappings", "", "type mappings use key1=value1,key2=value2 format")
)
var baseTypes = map[string]string{
	"ID":       "string",
	"String":   "string",
	"Boolean":  "bool",
	"Float":    "float64",
	"Int":      "int",
	"DateTime": "time.Time",
}
var goTypes = map[string]struct{}{
	"string":    {},
	"bool":      {},
	"int":       {},
	"datetime":  {},
	"time.Time": {},
}
var enumTypes = map[string]struct{}{}
var notOptionalTypes = map[string]struct{}{
	"Time":     {},
	"Date":     {},
	"Password": {},
	"DateTime": {},
}

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
	Arguments []*Argument `json:"args"`
}

func (t *Directive) GetKind() string {
	return t.Kind
}

type Argument struct {
	Kind string `json:"kind"`
	Name string `json:"name"`
}

func (t *Argument) GetKind() string {
	return t.Kind
}

type MyTypeImpl struct {
	Kind    string
	Name    string
	Src     string
	SrcX    string
	IsArray bool
	NotNull bool
}

func (m MyTypeImpl) String() string {
	goType := m.Name
	if !m.NotNull {
		goType = "*" + goType
	} else if _, ok := enumTypes[m.Name]; ok {
	} else if _, ok := notOptionalTypes[m.Name]; ok {
	} else if _, ok := goTypes[goType]; !ok && *optional && !strings.HasPrefix(goType, "Input") {
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

type DirectiveDefinition struct {
	Kind        string                  `json:"kind"`
	Name        *Name                   `json:"-"`
	GoName      string                  `json:"Name"`
	Description *StringValue            `json:"-"`
	Key         string                  `json:"key"`
	Arguments   []*InputValueDefinition `json:"args,omitempty"`
}

func (def *DirectiveDefinition) GetNodeKind() string {
	return def.Kind
}
func (def *DirectiveDefinition) GetNodeKey() string {
	return def.Key
}

type FieldDefinition struct {
	Kind          string                  `json:"-"`
	Name          *Name                   `json:"-"`
	GoName        string                  `json:"Name"`
	NameJson      string                  `json:"nameJson"`
	GoVarName     string                  `json:"name"`
	GoNames       string                  `json:"Names"`
	GoVarNames    string                  `json:"names"`
	GoShortName   string                  `json:"n"`
	NameDb        string                  `json:"nameDb"`
	NameExactDb   string                  `json:"nameExactDb"`
	NamesDb       string                  `json:"namesDb"`
	NameExact     string                  `json:"NameExact"`
	NameExactJson string                  `json:"nameExact"`
	NameCamel     string                  `json:"NameCamel"`
	NameOrig      string                  `json:"nameOrig"`
	Description   *StringValue            `json:"-"`
	Type          MyType                  `json:"-"`
	Alias         string                  `json:"alias"`
	SrcType       string                  `json:"srcType"`
	SrcTypeX      string                  `json:"srcTypeX"`
	GoType        *MyTypeImpl             `json:"Type,omitempty"`
	BaseType      string                  `json:"baseType"`
	TypeDb        string                  `json:"typeDb"`
	IsArray       bool                    `json:"isArray"`
	NotNull       bool                    `json:"notNull"`
	Arguments     []*InputValueDefinition `json:"args,omitempty"`
	MyDirectives  MyDirective             `json:"directives"`
	Key           string                  `json:"key"`
	Fields        []*FieldDefinition      `json:"fields"`
}

func (t *FieldDefinition) GetKind() string {
	return t.Kind
}

type InputValueDefinition struct {
	Kind           string                  `json:"-"`
	Name           *Name                   `json:"-"`
	GoName         string                  `json:"Name"`
	NameJson       string                  `json:"nameJson"`
	GoVarName      string                  `json:"name"`
	GoNames        string                  `json:"Names"`
	GoVarNames     string                  `json:"names"`
	GoShortName    string                  `json:"n"`
	NameDb         string                  `json:"nameDb"`
	NameExactDb    string                  `json:"nameExactDb"`
	NamesDb        string                  `json:"namesDb"`
	NameExact      string                  `json:"NameExact"`
	NameExactJson  string                  `json:"nameExact"`
	NameOrig       string                  `json:"nameOrig"`
	NameInput      string                  `json:"nameInput,omitempty"`
	NameInputExact string                  `json:"nameInputExact,omitempty"`
	Description    *StringValue            `json:"-"`
	Type           MyType                  `json:"-"`
	SrcType        string                  `json:"srcType"`
	SrcTypeX       string                  `json:"srcTypeX"`
	GoType         *MyTypeImpl             `json:"Type,omitempty"`
	BaseType       string                  `json:"baseType"`
	TypeDb         string                  `json:"typeDb"`
	IsArray        bool                    `json:"isArray"`
	NotNull        bool                    `json:"notNull"`
	DefaultValue   Value                   `json:"-"`
	MyDirectives   MyDirective             `json:"directives"`
	Fields         []*InputValueDefinition `json:"fields"`
	Key            string                  `json:"key"`
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
func (o ScalarDefinition) GetNodeKey() string {
	return o.Name.Value
}

type MyDirective map[string]map[string]interface{}
type EnumDefinition struct {
	Name          *Name                  `json:"-"`
	GoName        string                 `json:"Name"`
	NameJson      string                 `json:"nameJson"`
	Key           string                 `json:"key"`
	GoVarName     string                 `json:"name"`
	GoNames       string                 `json:"Names"`
	GoVarNames    string                 `json:"names"`
	GoShortName   string                 `json:"n"`
	NameDb        string                 `json:"nameDb"`
	NamesDb       string                 `json:"namesDb"`
	NameExact     string                 `json:"NameExact"`
	NameExactJson string                 `json:"nameExact"`
	NameOrig      string                 `json:"nameOrig"`
	Kind          string                 `json:"kind"`
	Directives    []*Directive           `json:"-"`
	Values        []*EnumValueDefinition `json:"fields"`
	MyDirectives  MyDirective            `json:"directives"`
}

func (o EnumDefinition) GetNodeKind() string {
	return o.Kind
}
func (o EnumDefinition) GetNodeKey() string {
	return o.Key
}

type EnumValueDefinition struct {
	Name          *Name       `json:"-"`
	GoName        string      `json:"Name"`
	NameJson      string      `json:"nameJson"`
	Key           string      `json:"key"`
	GoVarName     string      `json:"name"`
	NameExactJson string      `json:"nameExact"`
	NameOrig      string      `json:"nameOrig"`
	GoType        *MyTypeImpl `json:"Type"`
	MyDirectives  MyDirective `json:"directives"`
}
type ObjectDefinition struct {
	Name           *Name              `json:"-"`
	GoName         string             `json:"Name"`
	Key            string             `json:"key"`
	GoVarName      string             `json:"name"`
	GoNames        string             `json:"Names"`
	GoVarNames     string             `json:"names"`
	GoShortName    string             `json:"n"`
	NameDb         string             `json:"nameDb"`
	NameExactDb    string             `json:"nameExactDb"`
	NamesDb        string             `json:"namesDb"`
	NameExact      string             `json:"NameExact"`
	NameExactJson  string             `json:"nameExact"`
	NamesExactJson string             `json:"namesExact"`
	NameOrig       string             `json:"nameOrig"`
	Comment        string             `json:"comment"`
	Operation      string             `json:"operation,omitempty"`
	OperationName  string             `json:"Operation,omitempty"`
	Kind           string             `json:"kind"`
	Description    *StringValue       `json:"-"`
	MyDirectives   MyDirective        `json:"directives"`
	Fields         []*FieldDefinition `json:"fields"`
	Interfaces     []*FieldDefinition `json:"interfaces,omitempty"`
}

func (o ObjectDefinition) GetNodeKind() string {
	return o.Kind
}
func (o ObjectDefinition) GetNodeKey() string {
	return o.Key
}

type OperationDefinition struct {
	Name                *Name                   `json:"-"`
	GoName              string                  `json:"Name"`
	Key                 string                  `json:"key"`
	GoVarName           string                  `json:"name"`
	GoNames             string                  `json:"Names"`
	GoVarNames          string                  `json:"names"`
	GoShortName         string                  `json:"n"`
	NameDb              string                  `json:"nameDb"`
	NameExactDb         string                  `json:"nameExactDb"`
	NamesDb             string                  `json:"namesDb"`
	NameExact           string                  `json:"NameExact"`
	NameExactJson       string                  `json:"nameExact"`
	NameOrig            string                  `json:"nameOrig"`
	Comment             string                  `json:"comment"`
	Kind                string                  `json:"kind"`
	Operation           string                  `json:"operation"`
	OperationName       string                  `json:"Operation"`
	VariableDefinitions []*InputValueDefinition `json:"vars,omitempty"`
	MyDirectives        MyDirective             `json:"directives"`
	Fields              []*FieldDefinition      `json:"fields,omitempty"`
}

func (o OperationDefinition) GetNodeKind() string {
	return o.Kind
}
func (o OperationDefinition) GetNodeKey() string {
	return o.Key
}

type InputObjectDefinition struct {
	Name           *Name                   `json:"-"`
	GoName         string                  `json:"Name"`
	Key            string                  `json:"key"`
	GoVarName      string                  `json:"name"`
	GoNames        string                  `json:"Names"`
	GoVarNames     string                  `json:"names"`
	GoShortName    string                  `json:"n"`
	NameDb         string                  `json:"nameDb"`
	NameExactDb    string                  `json:"nameExactDb"`
	NamesDb        string                  `json:"namesDb"`
	NameExact      string                  `json:"NameExact"`
	NameExactJson  string                  `json:"nameExact"`
	NamesExactJson string                  `json:"namesExact"`
	NameOrig       string                  `json:"nameOrig"`
	NameInput      string                  `json:"NameInput"`
	Comment        string                  `json:"comment"`
	Kind           string                  `json:"kind"`
	Description    *StringValue            `json:"-"`
	MyDirectives   MyDirective             `json:"directives"`
	Fields         []*InputValueDefinition `json:"fields"`
}

func (o InputObjectDefinition) GetNodeKind() string {
	return o.Kind
}
func (o InputObjectDefinition) GetNodeKey() string {
	return o.Key
}

type Node interface {
	GetNodeKind() string
	GetNodeKey() string
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
func parseField(v *ast.Field, frag map[string][]ast.Selection, colMap map[string]map[string]*FieldDefinition, refKey string, path string, varMap map[string]*InputValueDefinition) *FieldDefinition {
	o := &FieldDefinition{
		Kind: v.Kind,
	}
	if v.Alias != nil {
		o.Alias = v.Alias.Value
	}
	o.NameOrig = v.Name.Value
	o.NameExact = snaker.ForceCamelIdentifier(v.Name.Value)
	o.NameExactJson = lowerCamel(v.Name.Value)
	o.GoName = snaker.ForceCamelIdentifier(v.Name.Value)
	o.NameJson = lowerCamel(inflection.Singular(v.Name.Value))
	o.NameDb = snaker.CamelToSnake(inflection.Singular(o.GoName))
	o.NameExactDb = snaker.CamelToSnake(v.Name.Value)
	o.NameCamel = strcase.ToCamel(o.NameExactDb)
	o.NamesDb = plural(o.NameDb)
	o.GoVarName = lowerCamel(inflection.Singular(o.GoName))
	if strings.HasSuffix(o.GoVarName, "IDS") {
		o.GoVarName = o.GoVarName[:len(o.GoVarName)-2] + "Ds"
	}
	o.GoNames = plural(o.GoName)
	if strings.HasSuffix(o.GoNames, "IDS") {
		o.GoNames = o.GoNames[:len(o.GoNames)-2] + "Ds"
	}
	o.GoVarNames = lowerCamel(plural(o.GoVarName))
	o.GoShortName = shortName(o.GoName)
	parseDirectives(&o.MyDirectives, v.Directives, o.GoName)
	curType := o.NameExactJson
	if refKey == "Query" || refKey == "Mutation" {
		o.Key = o.NameExact
		if xx, ok := colMap[refKey][o.Key]; ok {
			curType = xx.BaseType
		}
	} else {
		o.Key = o.NameExactJson
		if xx, ok := colMap[refKey][o.Key]; ok {
			curType = xx.BaseType
		}
	}
	if fieldMap, ok := colMap[refKey]; ok {
		if cc, ok := fieldMap[o.Key]; ok {
			o.Type = cc.Type
			o.SrcType = cc.SrcType
			o.SrcTypeX = cc.SrcTypeX
			o.GoType = cc.GoType
			o.BaseType = cc.BaseType
			o.IsArray = cc.IsArray
			o.NotNull = cc.NotNull
			for k, v := range cc.MyDirectives {
				o.MyDirectives[k] = v
			}
		}
	}
	if o.NameExactJson == "__typename" {
		o.BaseType = refKey
	}
	o.Arguments = make([]*InputValueDefinition, len(v.Arguments))
	for i, a := range v.Arguments {
		m := parseInputField(v, &ast.ObjectField{Kind: a.Kind, Name: a.Name, Value: a.Value})
		m.MyDirectives = make(MyDirective, 0)
		name := a.Name.Value
		m.NameOrig = name
		m.NameExact = snaker.ForceCamelIdentifier(name)
		m.NameExactJson = lowerCamel(name)
		m.NameJson = lowerCamel(inflection.Singular(name))
		m.Key = m.NameJson
		if m.Key == "id" {
			m.Key = o.GoVarName + "Id"
		}
		m.GoName = snaker.ForceCamelIdentifier(m.NameJson)
		m.NameDb = snaker.CamelToSnake(inflection.Singular(m.GoName))
		m.NameExactDb = snaker.CamelToSnake(name)
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
		if m.GoType == nil {
			if v, ok := varMap[m.NameInputExact]; ok && v.GoType != nil && v.GoType.Name != "" {
				m.GoType = v.GoType
			}
		}
		o.Arguments[i] = m
	}
	prevKey := o.Key
	if v.SelectionSet != nil {
		for _, ss := range v.SelectionSet.Selections {
			switch sf := ss.(type) {
			case *ast.FragmentSpread:
				if f, ok := frag[sf.Name.Value]; ok {
					for _, ff := range f {
						switch ssf := ff.(type) {
						case *ast.Field:
							field := parseField(ssf, frag, colMap, curType, path+"."+prevKey, varMap)
							o.Fields = append(o.Fields, field)
						case *ast.FragmentSpread:
							if f1, ok := frag[ssf.Name.Value]; ok {
								for _, ff1 := range f1 {
									switch ssf1 := ff1.(type) {
									case *ast.Field:
										field := parseField(ssf1, frag, colMap, curType, path+"."+prevKey, varMap)
										o.Fields = append(o.Fields, field)
									default:
										fmt.Println(ssf1)
									}
								}
							}
						default:
							fmt.Println(ssf)
						}
					}
				}
			case *ast.Field:
				ff := parseField(sf, frag, colMap, curType, path+"."+prevKey, varMap)
				o.Fields = append(o.Fields, ff)
			}
		}
	}
	sort.Slice(o.Fields, func(i, j int) bool {
		if len(o.Fields[i].Fields) > 0 && len(o.Fields[j].Fields) > 0 {
			return o.Fields[i].Key < o.Fields[j].Key
		} else if len(o.Fields[i].Fields) > 0 {
			return false
		} else if len(o.Fields[j].Fields) > 0 {
			return true
		}
		return o.Fields[i].Key < o.Fields[j].Key
	})
	return o
}
func parseInputField(v *ast.Field, a *ast.ObjectField) *InputValueDefinition {
	m := &InputValueDefinition{
		Kind: a.Kind,
	}
	switch vv := a.Value.(type) {
	case *ast.Variable:
		m.NameInputExact = vv.Name.Value
		m.NameInput = lowerCamel(snaker.ForceCamelIdentifier(m.NameInputExact))
	case *ast.StringValue:
		m.NameInput = "\"" + vv.Value + "\""
		m.GoType = &MyTypeImpl{
			Name:    "string",
			IsArray: false,
			NotNull: true,
		}
	case *ast.EnumValue:
		m.NameInput = "\"" + vv.Value + "\""
		m.GoType = &MyTypeImpl{
			Name:    "string",
			IsArray: false,
			NotNull: true,
		}
	case *ast.IntValue:
		m.NameInput = vv.Value
		m.GoType = &MyTypeImpl{
			Name:    "int",
			IsArray: false,
			NotNull: true,
		}
	case *ast.BooleanValue:
		m.NameInput = fmt.Sprint(vv.Value)
		m.GoType = &MyTypeImpl{
			Name:    "bool",
			IsArray: false,
			NotNull: true,
		}
	case *ast.ObjectValue:
		fields := make([]*InputValueDefinition, len(vv.Fields))
		for i, f := range vv.Fields {
			fields[i] = parseInputField(v, f)
		}
		m.Fields = fields
	default:
		fmt.Println("unknown", reflect.TypeOf(vv))
	}
	return m
}
func convert(nodes []ast.Node, frag map[string][]ast.Selection) ([]Node, error) {
	colMap := make(map[string]map[string]*FieldDefinition)
	onodes := make([]Node, 0, len(nodes))
	for _, n := range nodes {
		if v, ok := n.(*ast.TypeExtensionDefinition); ok {
			n = v.Definition
		}
		switch v := n.(type) {
		case *ast.DirectiveDefinition:
			o := &DirectiveDefinition{Kind: v.Kind}
			o.GoName = v.Name.Value
			o.Key = v.Name.Value
			o.Arguments = make([]*InputValueDefinition, len(v.Arguments))
			for i, a := range v.Arguments {
				m := &InputValueDefinition{
					Kind: a.Kind,
					Type: a.Type,
				}
				name := a.Name.Value
				m.NameOrig = name
				m.NameExact = snaker.ForceCamelIdentifier(name)
				m.NameExactJson = lowerCamel(name)
				m.NameJson = lowerCamel(inflection.Singular(name))
				m.Key = m.NameJson
				m.GoName = snaker.ForceCamelIdentifier(m.NameJson)
				m.NameDb = snaker.CamelToSnake(inflection.Singular(m.GoName))
				m.NameExactDb = snaker.CamelToSnake(name)
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
				m.MyDirectives = make(MyDirective, 0)
				m.GoType = &MyTypeImpl{}
				getGoType(m.Type, m.GoType)
				m.SrcType = m.GoType.Src
				m.SrcTypeX = m.GoType.SrcX
				m.BaseType = m.GoType.Name
				m.IsArray = m.GoType.IsArray
				m.NotNull = m.GoType.NotNull
				m.TypeDb = snaker.CamelToSnake(m.GoType.Name)
				o.Arguments[i] = m
			}
			onodes = append(onodes, o)
		case *ast.EnumDefinition:
			o := &EnumDefinition{Kind: v.Kind}
			o.Key = snaker.ForceCamelIdentifier(v.Name.Value)
			enumTypes[o.Key] = struct{}{}
			o.GoName = v.Name.Value
			o.Kind = kinds.EnumDefinition
			o.NameOrig = v.Name.Value
			o.NameExact = snaker.ForceCamelIdentifier(v.Name.Value)
			o.NameExactJson = v.Name.Value
			o.NameDb = snaker.CamelToSnake(inflection.Singular(o.GoName))
			o.NamesDb = plural(o.NameDb)
			o.GoVarName = lowerCamel(inflection.Singular(o.GoName))
			o.GoNames = plural(o.GoName)
			o.GoVarNames = lowerCamel(plural(o.GoVarName))
			o.GoShortName = shortName(o.GoName)
			colMap[o.Key] = make(map[string]*FieldDefinition)
			o.Values = make([]*EnumValueDefinition, len(v.Values))
			for i, vv := range v.Values {
				x := &EnumValueDefinition{}
				x.GoType = &MyTypeImpl{Name: "string", NotNull: true, Kind: vv.Kind}
				x.NameOrig = vv.Name.Value
				x.NameExactJson = vv.Name.Value
				x.NameJson = strcase.ToCamel(strcase.ToSnake(x.NameExactJson))
				x.GoName = ToGo(x.NameJson)
				x.GoVarName = lowerCamel(x.NameJson)
				x.Key = x.NameExactJson
				o.Values[i] = x
				x.MyDirectives = make(MyDirective, len(vv.Directives))
				parseDirectives(&x.MyDirectives, vv.Directives, vv.Name.Value)
				colMap[o.Key][x.Key] = &FieldDefinition{
					Kind:          vv.Kind,
					GoType:        x.GoType,
					NameExactJson: x.NameExactJson,
					GoName:        x.GoName,
					Key:           x.Key,
				}
			}
			parseDirectives(&o.MyDirectives, v.Directives, v.Name.Value)
			onodes = append(onodes, o)
		case *ast.InputObjectDefinition:
			o := &InputObjectDefinition{}
			o.Kind = v.Kind
			o.Key = deInitialism(snaker.ForceCamelIdentifier(v.Name.Value))
			o.NameInput = v.Name.Value
			o.GoName = strings.TrimPrefix(v.Name.Value, "Input")
			o.NameExact = v.Name.Value
			o.NameOrig = v.Name.Value
			o.NameExactJson = lowerCamel(v.Name.Value)
			o.NamesExactJson = lowerCamel(plural(v.Name.Value))
			o.NameExactDb = snaker.CamelToSnake(v.Name.Value)
			o.NameDb = snaker.CamelToSnake(inflection.Singular(o.GoName))
			o.NamesDb = plural(o.NameDb)
			o.GoVarName = lowerCamel(inflection.Singular(o.GoName))
			o.GoNames = plural(o.GoName)
			o.GoVarNames = lowerCamel(plural(o.GoVarName))
			o.GoShortName = shortName(o.GoName)
			if v.Description != nil {
				o.Comment = v.Description.Value
			}
			o.MyDirectives = make(MyDirective, len(v.Directives))
			parseDirectives(&o.MyDirectives, v.Directives, v.Name.Value)
			if err := copier.Copy(o, v); err != nil {
				return nil, err
			}
			for i, m := range o.Fields {
				m.NameOrig = m.Name.Value
				m.NameExact = snaker.ForceCamelIdentifier(m.Name.Value)
				m.NameJson = lowerCamel(inflection.Singular(m.Name.Value))
				m.NameExactJson = lowerCamel(m.Name.Value)
				m.Key = deInitialism(m.NameJson)
				m.GoName = snaker.ForceCamelIdentifier(m.NameJson)
				m.NameDb = snaker.CamelToSnake(inflection.Singular(m.GoName))
				m.NameExactDb = snaker.CamelToSnake(m.Name.Value)
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
				m.SrcType = m.GoType.Src
				m.SrcTypeX = m.GoType.SrcX
				m.BaseType = m.GoType.Name
				m.IsArray = m.GoType.IsArray
				m.NotNull = m.GoType.NotNull
				m.TypeDb = snaker.CamelToSnake(m.GoType.Name)
				m.MyDirectives = make(MyDirective, len(v.Fields[i].Directives))
				parseDirectives(&m.MyDirectives, v.Fields[i].Directives, v.Fields[i].Name.Value)
			}
			onodes = append(onodes, o)
		case *ast.OperationDefinition:
			o := &OperationDefinition{Kind: v.Kind}
			o.Key = deInitialism(snaker.ForceCamelIdentifier(v.Name.Value))
			o.GoName = v.Name.Value
			o.Operation = v.Operation
			o.OperationName = strings.Title(v.Operation)
			o.NameOrig = v.Name.Value
			o.NameExact = snaker.ForceCamelIdentifier(v.Name.Value)
			o.NameExactJson = lowerCamel(v.Name.Value)
			o.NameDb = snaker.CamelToSnake(inflection.Singular(o.GoName))
			o.NameExactDb = snaker.CamelToSnake(v.Name.Value)
			o.NamesDb = plural(o.NameDb)
			o.GoVarName = lowerCamel(inflection.Singular(o.GoName))
			o.GoNames = plural(o.GoName)
			o.GoVarNames = lowerCamel(plural(o.GoVarName))
			o.GoShortName = shortName(o.GoName)
			o.MyDirectives = make(MyDirective, len(v.Directives))
			parseDirectives(&o.MyDirectives, v.Directives, v.Name.Value)
			varMap := map[string]*InputValueDefinition{}
			o.VariableDefinitions = make([]*InputValueDefinition, len(v.VariableDefinitions))
			for i, d := range v.VariableDefinitions {
				m := &InputValueDefinition{
					Kind: d.Kind,
					Type: d.Type,
				}
				name := d.Variable.Name.Value
				m.NameOrig = name
				m.NameExact = snaker.ForceCamelIdentifier(name)
				m.NameExactJson = lowerCamel(name)
				m.NameJson = lowerCamel(inflection.Singular(name))
				m.Key = m.NameJson
				if m.Key == "id" {
					m.Key = o.GoVarName + "Id"
				}
				m.GoName = snaker.ForceCamelIdentifier(m.NameJson)
				m.NameDb = snaker.CamelToSnake(inflection.Singular(m.GoName))
				m.NameExactDb = snaker.CamelToSnake(name)
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
				m.MyDirectives = make(MyDirective, 0)
				m.GoType = &MyTypeImpl{}
				getGoType(m.Type, m.GoType)
				m.SrcType = m.GoType.Src
				m.SrcTypeX = m.GoType.SrcX
				m.BaseType = m.GoType.Name
				m.IsArray = m.GoType.IsArray
				m.NotNull = m.GoType.NotNull
				o.VariableDefinitions[i] = m
				varMap[m.Key] = m
			}
			if v.SelectionSet != nil {
				for _, ss := range v.SelectionSet.Selections {
					switch sf := ss.(type) {
					case *ast.FragmentSpread:
						if f, ok := frag[sf.Name.Value]; ok {
							for _, ff := range f {
								if sf, ok := ff.(*ast.Field); ok {
									field := parseField(sf, frag, colMap, "", "", varMap)
									o.Fields = append(o.Fields, field)
								}
							}
						}
					case *ast.Field:
						oKey := "Query"
						if o.Operation == "mutation" {
							oKey = "Mutation"
						}
						field := parseField(sf, frag, colMap, oKey, "", varMap)
						o.Fields = append(o.Fields, field)
					default:
						log.Println("unknown type in SelectionSet", ss)
					}
				}
			}
			onodes = append(onodes, o)
		case *ast.ScalarDefinition:
			o := &ObjectDefinition{Kind: v.Kind}
			o.GoName = v.Name.Value
			o.NameExact = v.Name.Value
			o.NameOrig = v.Name.Value
			o.NameExactJson = lowerCamel(v.Name.Value)
			o.NameDb = snaker.CamelToSnake(inflection.Singular(o.GoName))
			o.NameExactDb = snaker.CamelToSnake(v.Name.Value)
			o.NamesDb = plural(o.NameDb)
			o.GoVarName = lowerCamel(inflection.Singular(o.GoName))
			o.GoNames = plural(o.GoName)
			o.GoVarNames = lowerCamel(plural(o.GoVarName))
			o.GoShortName = shortName(o.GoName)
			o.Fields = make([]*FieldDefinition, 0)
			o.MyDirectives = make(MyDirective, len(v.Directives))
			parseDirectives(&o.MyDirectives, v.Directives, v.Name.Value)
			onodes = append(onodes, o)
		case *ast.UnionDefinition:
			o := &ObjectDefinition{Kind: v.Kind}
			o.GoName = v.Name.Value
			o.NameExact = v.Name.Value
			o.NameOrig = v.Name.Value
			o.NameExactJson = lowerCamel(v.Name.Value)
			o.NameDb = snaker.CamelToSnake(inflection.Singular(o.GoName))
			o.NameExactDb = snaker.CamelToSnake(v.Name.Value)
			o.NamesDb = plural(o.NameDb)
			o.GoVarName = lowerCamel(inflection.Singular(o.GoName))
			o.GoNames = plural(o.GoName)
			o.GoVarNames = lowerCamel(plural(o.GoVarName))
			o.GoShortName = shortName(o.GoName)
			o.Fields = make([]*FieldDefinition, 0, len(v.Types))
			for _, t := range v.Types {
				m := &FieldDefinition{}
				m.NameExact = t.Name.Value
				m.NameOrig = t.Name.Value
				m.NameExactJson = lowerCamel(t.Name.Value)
				m.NameJson = lowerCamel(inflection.Singular(t.Name.Value))
				m.Key = m.NameExactJson
				m.GoName = snaker.ForceCamelIdentifier(m.NameJson)
				m.NameDb = snaker.CamelToSnake(inflection.Singular(m.GoName))
				m.NameExactDb = snaker.CamelToSnake(t.Name.Value)
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
				o.Fields = append(o.Fields, m)
			}
			onodes = append(onodes, o)
		case *ast.InterfaceDefinition:
			o := &ObjectDefinition{Kind: v.Kind}
			o.Key = snaker.ForceCamelIdentifier(v.Name.Value)
			o.GoName = v.Name.Value
			o.NameOrig = v.Name.Value
			o.NameExact = v.Name.Value
			o.NameExactJson = lowerCamel(v.Name.Value)
			o.NameDb = snaker.CamelToSnake(inflection.Singular(o.GoName))
			o.NameExactDb = snaker.CamelToSnake(v.Name.Value)
			o.NamesDb = plural(o.NameDb)
			o.GoVarName = lowerCamel(inflection.Singular(o.GoName))
			o.GoNames = plural(o.GoName)
			o.GoVarNames = lowerCamel(plural(o.GoVarName))
			o.GoShortName = shortName(o.GoName)
			if v.Description != nil {
				o.Comment = v.Description.Value
			}
			o.MyDirectives = make(MyDirective, len(v.Directives))
			parseDirectives(&o.MyDirectives, v.Directives, v.Name.Value)
			if err := copier.Copy(o, v); err != nil {
				return nil, err
			}
			colMap[o.Key] = make(map[string]*FieldDefinition)
			for i, m := range o.Fields {
				fd := v.Fields[i]
				parseFieldDefinition(m, o, fd, v.Fields[i].Name.Value)
				colMap[o.Key][m.Key] = m
			}
			onodes = append(onodes, o)
		case *ast.ObjectDefinition:
			o := &ObjectDefinition{Kind: v.Kind}
			o.Key = deInitialism(snaker.ForceCamelIdentifier(v.Name.Value))
			o.GoName = v.Name.Value
			o.NameOrig = v.Name.Value
			o.NameExact = v.Name.Value
			o.NameExactJson = lowerCamel(v.Name.Value)
			o.NamesExactJson = lowerCamel(plural(v.Name.Value))
			o.NameDb = snaker.CamelToSnake(inflection.Singular(o.GoName))
			o.NameExactDb = snaker.CamelToSnake(v.Name.Value)
			o.NamesDb = plural(o.NameDb)
			o.GoVarName = lowerCamel(inflection.Singular(o.GoName))
			o.GoNames = plural(o.GoName)
			o.GoVarNames = lowerCamel(plural(o.GoVarName))
			o.GoShortName = shortName(o.GoName)
			if v.Description != nil {
				o.Comment = v.Description.Value
			}
			o.MyDirectives = make(MyDirective, len(v.Directives))
			parseDirectives(&o.MyDirectives, v.Directives, v.Name.Value)
			if err := copier.Copy(o, v); err != nil {
				return nil, err
			}
			if o.GoName == *queryType {
				o.Kind = "QueryDefinition"
				o.Operation = "query"
				o.OperationName = "Query"
			} else if o.GoName == *mutationType {
				o.Kind = "MutationDefinition"
				o.Operation = "mutation"
				o.OperationName = "Mutation"
			}
			o.Interfaces = make([]*FieldDefinition, 0, len(v.Interfaces))
			for _, oi := range v.Interfaces {
				m := &FieldDefinition{}
				m.NameOrig = oi.Name.Value
				m.NameExact = oi.Name.Value
				m.NameExactJson = lowerCamel(m.NameExact)
				m.NameJson = lowerCamel(inflection.Singular(m.NameExact))
				m.Key = m.NameExactJson
				if m.Key == "id" {
					m.Key = o.GoVarName + "Id"
				}
				m.GoName = snaker.ForceCamelIdentifier(m.NameJson)
				m.NameDb = snaker.CamelToSnake(inflection.Singular(m.GoName))
				m.NameExactDb = snaker.CamelToSnake(m.NameExact)
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
				o.Interfaces = append(o.Interfaces, m)
			}
			colMap[o.Key] = make(map[string]*FieldDefinition)
			for i, m := range o.Fields {
				fd := v.Fields[i]
				parseFieldDefinition(m, o, fd, v.Fields[i].Name.Value)
				colMap[o.Key][m.Key] = m
			}
			onodes = append(onodes, o)
		}
	}
	typeOrder := []string{"ScalarDefinition", "EnumDefinition", "UnionDefinition", "InterfaceDefinition", "ObjectDefinition", "InputObjectDefinition", "QueryDefinition", "MutationDefinition", "OperationDefinition"}
	sort.Slice(onodes, func(i, j int) bool {
		a1, a2 := onodes[i].GetNodeKind(), onodes[j].GetNodeKind()
		k1, k2 := onodes[i].GetNodeKey(), onodes[j].GetNodeKey()
		if a1 == a2 {
			return k1 < k2
		}
		i1, i2 := IndexOf(typeOrder, a1), IndexOf(typeOrder, a2)
		return i1 < i2
	})
	return onodes, nil
}
func IndexOf[T comparable](collection []T, el T) int {
	for i, x := range collection {
		if x == el {
			return i
		}
	}
	return -1
}
func parseFieldDefinition(m *FieldDefinition, o *ObjectDefinition, fd *ast.FieldDefinition, fdName string) {
	if strings.HasSuffix(m.Name.Value, "ID") && !strings.HasSuffix(m.Name.Value, "UID") {
		if len(m.Name.Value) < 3 || !unicode.IsUpper(rune(m.Name.Value[len(m.Name.Value)-3])) {
			color.Yellow.Printf("WARNING: Model '%s', Field '%s' ends with ID, use Id instead\n", o.Name.Value, m.Name.Value)
		}
	}
	m.NameOrig = m.Name.Value
	m.NameExact = snaker.ForceCamelIdentifier(m.Name.Value)
	m.NameExactJson = lowerCamel(m.Name.Value)
	m.NameJson = lowerCamel(inflection.Singular(m.Name.Value))
	if o.Key == "Query" || o.Key == "Mutation" {
		m.Key = m.NameExact
	} else {
		m.Key = m.NameExactJson
	}
	m.Key = deInitialism(m.Key)
	m.MyDirectives = make(MyDirective, len(fd.Directives))
	parseDirectives(&m.MyDirectives, fd.Directives, fdName)
	if m.Key == "id" {
		m.Key = o.GoVarName + "Id"
	}
	m.GoName = snaker.ForceCamelIdentifier(m.NameJson)
	m.NameDb = snaker.CamelToSnake(inflection.Singular(m.GoName))
	m.NameExactDb = snaker.CamelToSnake(m.Name.Value)
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
	m.SrcType = m.GoType.Src
	m.SrcTypeX = m.GoType.SrcX
	if m.NameExact == "__typename" {
		m.BaseType = o.NameExact
	} else {
		m.BaseType = m.GoType.Name
	}
	m.IsArray = m.GoType.IsArray
	m.NotNull = m.GoType.NotNull
	m.TypeDb = snaker.CamelToSnake(m.GoType.Name)
	for _, n := range m.Arguments {
		n.NameOrig = n.Name.Value
		n.NameExact = snaker.ForceCamelIdentifier(n.Name.Value)
		n.NameExactJson = lowerCamel(n.Name.Value)
		n.NameJson = lowerCamel(inflection.Singular(n.Name.Value))
		n.Key = n.NameJson
		if n.Key == "id" {
			n.Key = o.GoVarName + "Id"
		}
		n.GoName = snaker.ForceCamelIdentifier(n.NameJson)
		n.NameDb = snaker.CamelToSnake(inflection.Singular(n.GoName))
		n.NameExactDb = snaker.CamelToSnake(n.Name.Value)
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
		n.MyDirectives = make(MyDirective, 0)
		n.GoType = &MyTypeImpl{}
		getGoType(n.Type, n.GoType)
		n.SrcType = n.GoType.Src
		n.SrcTypeX = n.GoType.SrcX
		n.BaseType = n.GoType.Name
		n.IsArray = n.GoType.IsArray
		n.NotNull = n.GoType.NotNull
		n.TypeDb = snaker.CamelToSnake(n.GoType.Name)
	}
}
func parseDirectives(od *MyDirective, directives []*ast.Directive, name string) {
	if od == nil || *od == nil {
		*od = make(MyDirective, len(directives))
	}
	md := *od
	for _, d := range directives {
		if _, ok := md[name]; !ok {
			md[d.Name.Value] = make(map[string]interface{}, len(d.Arguments))
		}
		for _, a := range d.Arguments {
			switch aa := a.Value.(type) {
			case *ast.ListValue:
				enumValues := make(map[string]int, len(aa.Values))
				for _, value := range aa.Values {
					switch listItem := value.(type) {
					case *ast.EnumValue:
						enumValues[listItem.Value] = 1
					case *ast.StringValue:
						enumValues[listItem.Value] = 1
					default:
						fmt.Println("unknown ListValue", aa)
					}
				}
				md[d.Name.Value][a.Name.Value] = enumValues
			case *ast.Variable:
				md[d.Name.Value][a.Name.Value] = aa.Name.Value
			case *ast.StringValue:
				md[d.Name.Value][a.Name.Value] = aa.Value
			case *ast.IntValue:
				num, err := strconv.ParseUint(aa.Value, 10, 64)
				if err != nil {
					log.Fatalln(err)
				}
				md[d.Name.Value][a.Name.Value] = num
			case *ast.BooleanValue:
				md[d.Name.Value][a.Name.Value] = aa.Value
			case *ast.EnumValue:
				md[d.Name.Value][a.Name.Value] = aa.Value
			default:
				md[d.Name.Value][a.Name.Value] = a.Value.GetValue()
			}
		}
	}
}
func getGoType(m MyType, typ *MyTypeImpl) {
	switch w := m.(type) {
	case *ast.NonNull:
		typ.NotNull = true
		getGoType(w.Type, typ)
	case *ast.Named:
		typ.Src = w.Name.Value
		if typ.NotNull {
			typ.SrcX = typ.Src + "!"
		} else {
			typ.SrcX = typ.Src
		}
		typ.Name = convertGoType(w.Name.Value)
	case *ast.List:
		typ.IsArray = true
		getGoType(w.Type, typ)
	}
}
func convertGoType(s string) string {
	if ss, ok := baseTypes[s]; ok {
		return ss
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
func upperCamel(s string) string {
	if s == "" {
		return ""
	}
	r, n := utf8.DecodeRuneInString(s)
	return string(unicode.ToUpper(r)) + s[n:]
}

var shortNameRe = regexp.MustCompile("[A-Z]")

func shortName(s string) string {
	return strings.ToLower(strings.Join(shortNameRe.FindAllString(s, -1), ""))
}

type FileContent struct {
	FileKind string `json:"kind"`
	SrcKind  string `json:"srcKind"`
	Data     []Node `json:"data"`
}

func process() error {
	files := flag.Args()
	if schemaFile != nil && *schemaFile != "" {
		files = append([]string{*schemaFile}, files...)
	}
	if !*force && *out != "" && *out != "-" {
		inTime, outTime := 0, 0
		if t := mtime(*out); t > 0 && t > outTime {
			outTime = t
		}
		if outTime > 0 {
			for _, f := range files {
				if t := mtime(f); t > inTime {
					inTime = t
				}
			}
			if exe, err := os.Executable(); err == nil {
				if t := mtime(exe); t > inTime {
					inTime = t
				}
			}
			if inTime > 0 && inTime <= outTime {
				if *debug || *verbose {
					log.Println("skip since no file has changed or use -f")
				}
				return nil
			}
		}
	}
	commentRe := regexp.MustCompile(`"""[\s\S]+?"""`)
	unsupportedRe := regexp.MustCompile(`\s*=\s*null\b`)
	nodes := make([]ast.Node, 0)
	for _, f := range files {
		b, err := os.ReadFile(f)
		if err != nil {
			return err
		}
		if *removeComments {
			b = commentRe.ReplaceAll(b, []byte{})
			b = unsupportedRe.ReplaceAll(b, []byte{})
		}
		node, err := parseSchema(b)
		if err != nil {
			fmt.Printf("failed to parse file: %s, err: %v\n", f, err)
			return fmt.Errorf("failed to parse file (%s): %w", f, err)
		}
		nodes = append(nodes, node.Definitions...)
	}
	fmap := make(map[string][]ast.Selection)
	for _, n := range nodes {
		switch v := n.(type) {
		case *ast.FragmentDefinition:
			fmap[v.Name.Value] = v.SelectionSet.Selections
		}
	}
	if *debug {
		for _, f := range nodes {
			fmt.Fprint(os.Stderr, "var _ = ")
			fmt.Fprintln(os.Stderr, f)
		}
	}
	items, err := convert(nodes, fmap)
	if err != nil {
		return err
	}
	fileContent := FileContent{
		FileKind: *kind,
		SrcKind:  "gql",
		Data:     items,
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
	if *changedFlag {
		os.Exit(2)
	}
	return nil
}
func mtime(name string) int {
	if st, err := os.Stat(name); err != nil {
		return -1
	} else {
		return int(st.ModTime().UnixMicro())
	}
}
func main() {
	flag.Parse()
	if *typeMappings != "" {
		kvs := strings.Split(*typeMappings, ",")
		for _, kv := range kvs {
			pair := strings.SplitN(kv, "=", 2)
			baseTypes[pair[0]] = pair[1]
			goTypes[pair[1]] = struct{}{}
		}
	}
	if err := process(); err != nil {
		log.Fatalln(err)
	}
}
