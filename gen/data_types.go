package gen

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// The api descriptor struct contains the results of the analyzer Analyze() method.
// This includes all the API resources, actions and types.
type ApiDescriptor struct {
	Version       string                     // API Version
	Resources     map[string]*Resource       // Resources indexed by name
	Types         map[string]*ObjectDataType // Types used by resource actions indexed by name
	ResourceNames []string                   // Resource names ordered alphabetically
	TypeNames     []string                   // Type names ordered alphabetically
	NeedTime      bool                       // Whether generated code uses the time.Time package
	NeedJson      bool                       // Whether generated code uses encoding/json package
}

// Merge two descriptors together, make sure there are no duplicate resource names and that
// common types are compatible.
func (d *ApiDescriptor) Merge(other *ApiDescriptor) error {
	if d.Version != other.Version {
		return fmt.Errorf("Can't merge API descriptors with different versions")
	}
	for _, name := range d.ResourceNames {
		for _, otherName := range other.ResourceNames {
			if name == otherName {
				return fmt.Errorf("%s is a resource that exists in multiple APIs, generate separate clients.", name)
			}
		}
	}
	for _, name := range d.TypeNames {
		for i, otherName := range other.TypeNames {
			if name == otherName {
				newName := MakeUniq(otherName, append(d.TypeNames, other.TypeNames...))
				first := other.TypeNames[:i]
				last := append([]string{newName}, other.TypeNames[i+1:]...)
				other.TypeNames = append(first, last...)
				type_ := other.Types[name]
				delete(other.Types, name)
				type_.TypeName = newName
				other.Types[newName] = type_
			}
		}
	}
	for name, resource := range other.Resources {
		d.Resources[name] = resource
	}
	for name, type_ := range other.Types {
		d.Types[name] = type_
	}
	d.ResourceNames = append(d.ResourceNames, other.ResourceNames...)
	d.TypeNames = append(d.TypeNames, other.TypeNames...)
	return nil
}

// Go through all the types generated by the analyzer and generate unique names
func (d *ApiDescriptor) FinalizeTypeNames(rawTypes map[string][]*ObjectDataType) {

	// 1. Make sure data type names don't clash with resource names
	rawTypeNames := make([]string, len(rawTypes))
	idx := 0
	for n, _ := range rawTypes {
		rawTypeNames[idx] = n
		idx += 1
	}
	sort.Strings(rawTypeNames)
	for _, tn := range rawTypeNames {
		types := rawTypes[tn]
		for rn, _ := range d.Resources {
			if tn == rn {
				oldTn := tn
				if strings.HasSuffix(tn, "Param") {
					tn = fmt.Sprintf("%s2", tn)
				} else {
					tn = fmt.Sprintf("%sParam", tn)
				}
				for _, ty := range types {
					ty.TypeName = tn
				}
				rawTypes[tn] = types
				delete(rawTypes, oldTn)
			}
		}
	}

	// 2. Make data type names unique
	idx = 0
	for n, _ := range rawTypes {
		rawTypeNames[idx] = n
		idx += 1
	}
	sort.Strings(rawTypeNames)
	for _, tn := range rawTypeNames {
		types := rawTypes[tn]
		first := types[0]
		d.Types[tn] = first
		if len(types) > 1 {
			for i, ty := range types[1:] {
				found := false
				for j := 0; j < i+1; j++ {
					if ty.IsEquivalent(types[j]) {
						found = true
						break
					}
				}
				if !found {
					newName := d.uniqueTypeName(tn)
					ty.TypeName = newName
					d.Types[newName] = ty
				}
			}
		}
	}

	// 3. Finally initialize .ResourceNames and .TypeNames
	idx = 0
	resourceNames := make([]string, len(d.Resources))
	for n, _ := range d.Resources {
		resourceNames[idx] = n
		idx += 1
	}
	sort.Strings(resourceNames)
	d.ResourceNames = resourceNames

	typeNames := make([]string, len(d.Types))
	idx = 0
	for tn, _ := range d.Types {
		typeNames[idx] = tn
		idx += 1
	}
	sort.Strings(typeNames)
	d.TypeNames = typeNames
}

// Build unique type name by appending "next available index" to given prefix
func (d *ApiDescriptor) uniqueTypeName(prefix string) string {
	u := fmt.Sprintf("%s%d", prefix, 2)
	taken := false
	for _, tn := range d.TypeNames {
		if tn == u {
			taken = true
			break
		}
	}
	idx := 3
	for taken {
		u = fmt.Sprintf("%s%d", prefix, idx)
		taken = false
		for _, tn := range d.TypeNames {
			if tn == u {
				taken = true
				break
			}
		}
		if taken {
			idx += 1
		}
	}
	return u
}

// Data structure used to describe API resources.
type Resource struct {
	Name        string       // Resource name, e.g. "ServerArray"
	ClientName  string       // Name of go client struct, e.g. "Api"
	Description string       // Resource description
	Attributes  []*Attribute // Resource attributes
	Actions     []*Action    // Resource actions, e.g. "index", "show", "update" ...
	LocatorFunc string       // Source code for Locator factory method if any
}

// Resource attributes used to generate resource type definition.
// There may also be a ObjectDataType describing the resource media type for example to use as
// action parameter. The below is solely to generate the go struct corresponding to the resource.
type Attribute struct {
	Name      string // Attribute name, e.g. "elasticity_params"
	FieldName string // Corresponding go struct field name, e.g. "ElasticityParams"
	FieldType string // Corresponding go struct type, e.g. "*ElasticityParams"
}

// Resource actions
type Action struct {
	Name              string         // Action name, e.g. "create", "multi_terminate"
	MethodName        string         // Go method name, e.g. "Create", "MultiTerminate"
	Description       string         // Action description
	ResourceName      string         // Name of resource that contains this action
	PathPatterns      []*PathPattern // Action path patterns
	Payload           DataType       // Payload type if any
	Params            []*ActionParam // Action method parameters
	LeafParams        []*ActionParam // Action parameter leaves (for command line)
	Return            string         // Type of method results, e.g. "*ServerArray"
	ReturnLocation    bool           // Whether API returns a location header
	PathParamNames    []string       // Name of path parameters if any (e.g. :id in /clouds/:id)
	QueryParamNames   []string       // Name of query string parameters if any
	PayloadParamNames []string       // Name of payload parameter names if any (payload top level keys)
}

// MandatoryParams returns the list of all action mandatory parameters
func (a *Action) MandatoryParams() []*ActionParam {
	m := make([]*ActionParam, len(a.Params))
	i := 0
	for _, p := range a.Params {
		if p.Mandatory {
			m[i] = p
			i += 1
		}
	}
	return m[:i]
}

// Whether action takes optional parameters
func (a *Action) HasOptionalParams() bool {
	for _, param := range a.Params {
		if !param.Mandatory {
			return true
		}
	}
	return false
}

// A path pattern represents a possible path for a given action.
type PathPattern struct {
	HttpMethod string   // Action HTTP method, e.g. "GET", "POST"
	Path       string   // Path as it appears in metadata
	Pattern    string   // Actual pattern, e.g. "/clouds/%s/instances/%s"
	Variables  []string // Pattern variable names in order of appearance in pattern, e.g. "cloud_id", "id"
	Regexp     string   // Regular expression used to match href
}

// Possible param locations
const (
	PathParam    = 0
	QueryParam   = 1
	PayloadParam = 2
)

// Data structure used to render method params
type ActionParam struct {
	Name        string        // Name of parameter
	QueryName   string        // Query string style parameter name
	Description string        // Description of parameter
	VarName     string        // go variable name
	Location    int           // 0 for path param, 1 for query param, 2 for payload param
	Type        DataType      // Type of parameter
	Mandatory   bool          // Whether parameter is mandatory
	NonBlank    bool          // Whether parameter must not be blank if provided (string or array)
	Regexp      string        // Regular expression string parameter must match
	ValidValues []interface{} // Allowed values (if not empty)
	Min         int           // Minimum value (int)
	Max         int           // Maximum value (int)
}

// Generate signature used e.g. when specifying param in function signatures
func (p *ActionParam) Signature() (sig string) {
	switch t := p.Type.(type) {
	case *BasicDataType:
		sig = string(*t)
	case *ArrayDataType:
		cs := t.ElemType.Signature()
		sig = fmt.Sprintf("[]%s", cs)
	case *ObjectDataType:
		sig = fmt.Sprintf("*%s", t.TypeName)
	case *EnumerableDataType:
		sig = "map[string]interface{}"
	case *UploadDataType:
		sig = "*rsapi.FileUpload"
	case *SourceUploadDataType:
		sig = "*rsapi.SourceUpload"
	}
	return
}

// true if action params have the same name and type
func (p *ActionParam) IsEquivalent(other *ActionParam) bool {
	return p.Name == other.Name && p.Type.IsEquivalent(other.Type)
}

// Data type interface
type DataType interface {
	IsEquivalent(other DataType) bool // true if datatype and other represent the same data structure
	Name() string                     // Type name
}

// A basic data type only has a name, i.e. "int", "string"...
type BasicDataType string

// true if both b and other represent the same type
func (b *BasicDataType) IsEquivalent(other DataType) bool {
	t, ok := other.(*BasicDataType)
	if !ok {
		return false
	}
	return *t == *b
}

// Name is type itself
func (b *BasicDataType) Name() string {
	return string(*b)
}

// An array data type defines the type of its elements
type ArrayDataType struct {
	ElemType *ActionParam
}

// true if other is also a array data type and element types of both arrays are equivalent
func (a *ArrayDataType) IsEquivalent(other DataType) bool {
	t, ok := other.(*ArrayDataType)
	if !ok {
		return false
	}
	return a.ElemType.IsEquivalent(t.ElemType)
}

// Name is name of element appended with "[]"
func (a *ArrayDataType) Name() string {
	return a.ElemType.Type.Name() + "[]"
}

// An object data type has a name and fields
type ObjectDataType struct {
	TypeName string
	Fields   []*ActionParam
}

// true if other is a object data type and each field is equivalent recursively
func (o *ObjectDataType) IsEquivalent(other DataType) bool {
	a, ok := other.(*ObjectDataType)
	if !ok {
		return false
	}
	if o.TypeName != a.TypeName {
		return false
	}
	if len(o.Fields) != len(a.Fields) {
		return false
	}
	for i, f := range o.Fields {
		if !f.IsEquivalent(a.Fields[i]) {
			return false
		}
	}
	return true
}

// Implement DataType
func (o *ObjectDataType) Name() string {
	return o.TypeName
}

// An enumerable is just a map
type EnumerableDataType int

// true if other is also an enumerable data type
func (e *EnumerableDataType) IsEquivalent(other DataType) bool {
	_, ok := other.(*EnumerableDataType)
	return ok
}

// Implement DataType
func (e *EnumerableDataType) Name() string {
	return "map[string]interface{}"
}

// SourceUploadDataType -- just a string we use as the body
type SourceUploadDataType string

func (e *SourceUploadDataType) IsEquivalent(other DataType) bool {
	_, ok := other.(*SourceUploadDataType)
	return ok
}

// Implement DataType
func (e *SourceUploadDataType) Name() string {
	return "string"
}

// UploadDataType represents the payload of a multipart request with a file part.
type UploadDataType struct {
	TypeName string
}

// true if other is a upload data type and the backing type is the same.
func (u *UploadDataType) IsEquivalent(other DataType) bool {
	o, ok := other.(*UploadDataType)
	if !ok {
		return false
	}
	if o.TypeName != u.TypeName {
		return false
	}
	return true
}

// Implement DataType
func (u *UploadDataType) Name() string {
	return "Upload"
}

// Make it possible to sort action parameters by name
type ByName []*ActionParam

func (b ByName) Len() int           { return len(b) }
func (b ByName) Swap(i, j int)      { b[i], b[j] = b[j], b[i] }
func (b ByName) Less(i, j int) bool { return b[i].Name < b[j].Name }

// Make a unique name given a prefix and a set of names
func MakeUniq(base string, taken []string) string {
	idx := 1
	uniq := base
	inuse := true
	for inuse {
		inuse = false
		for _, gn := range taken {
			if gn == uniq {
				inuse = true
				break
			}
		}
		if inuse {
			idx += 1
			uniq = base + strconv.Itoa(idx)
		}
	}
	return uniq
}
