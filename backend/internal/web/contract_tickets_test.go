package web

import (
	"encoding/json"
	"os"
	"reflect"
	"slices"
	"strings"
	"testing"

	"backend_si/internal/domain"
	"backend_si/internal/security"
)

// ticketSchema is the subset of a JSON Schema that the ticket contract
// checks read.
type ticketSchema struct {
	Ref                  string                     `json:"$ref"`
	Type                 any                        `json:"type"`
	AdditionalProperties *bool                      `json:"additionalProperties"`
	Required             []string                   `json:"required"`
	Properties           map[string]json.RawMessage `json:"properties"`
	OneOf                []schemaRef                `json:"oneOf"`
	Discriminator        struct {
		PropertyName string            `json:"propertyName"`
		Mapping      map[string]string `json:"mapping"`
	} `json:"discriminator"`
	Const string   `json:"const"`
	Enum  []string `json:"enum"`
}

// optionalTicketKeys are the "dados" keys that may be left out (spec
// section 4: only the deploy notes are optional).
var optionalTicketKeys = []string{"observacoes"}

func loadTicketSchemas(t *testing.T) map[string]ticketSchema {
	t.Helper()
	data, err := os.ReadFile(contractPath)
	if err != nil {
		t.Fatalf("read contract: %v", err)
	}
	var doc struct {
		Components struct {
			Schemas map[string]ticketSchema `json:"schemas"`
		} `json:"components"`
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatalf("parse contract: %v", err)
	}
	return doc.Components.Schemas
}

func schemaName(t *testing.T, ref string) string {
	t.Helper()
	name, ok := strings.CutPrefix(ref, "#/components/schemas/")
	if !ok {
		t.Fatalf("ref %q is not a component schema", ref)
	}
	return name
}

func propertySchema(t *testing.T, raw json.RawMessage) ticketSchema {
	t.Helper()
	var s ticketSchema
	if err := json.Unmarshal(raw, &s); err != nil {
		t.Fatalf("parse property: %v", err)
	}
	return s
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	return keys
}

func sorted(s []string) []string {
	out := slices.Clone(s)
	slices.Sort(out)
	return out
}

// TestContractTicketRequestMatchesDecoder compares CreateTicketRequest with
// the keys security.DecodeTicket accepts: one schema per type, no other key
// at either level, and the required keys of spec section 4.
func TestContractTicketRequestMatchesDecoder(t *testing.T) {
	t.Parallel()

	schemas := loadTicketSchemas(t)
	keys := security.TicketDataKeys()

	root, ok := schemas["CreateTicketRequest"]
	if !ok {
		t.Fatal("schema CreateTicketRequest is missing")
	}
	if root.Discriminator.PropertyName != "tipo" {
		t.Errorf("discriminator = %q, want tipo", root.Discriminator.PropertyName)
	}
	var types []string
	for typ := range keys {
		types = append(types, string(typ))
	}
	if got := sortedKeys(root.Discriminator.Mapping); !slices.Equal(got, sorted(types)) {
		t.Errorf("discriminator mapping has %v, want the decoder types %v", got, sorted(types))
	}
	var oneOf, mapped []string
	for _, r := range root.OneOf {
		oneOf = append(oneOf, r.Ref)
	}
	for _, ref := range root.Discriminator.Mapping {
		mapped = append(mapped, ref)
	}
	if !slices.Equal(sorted(oneOf), sorted(mapped)) {
		t.Errorf("oneOf %v and the discriminator mapping %v differ", oneOf, mapped)
	}

	for typ, dataKeys := range keys {
		t.Run(string(typ), func(t *testing.T) {
			t.Parallel()
			req := schemas[schemaName(t, root.Discriminator.Mapping[string(typ)])]
			if req.AdditionalProperties == nil || *req.AdditionalProperties {
				t.Error("request schema must have additionalProperties: false")
			}
			if got := sortedKeys(req.Properties); !slices.Equal(got, []string{"dados", "tipo"}) {
				t.Errorf("request properties = %v, want [dados tipo]", got)
			}
			if !slices.Equal(sorted(req.Required), []string{"dados", "tipo"}) {
				t.Errorf("request required = %v, want [dados tipo]", req.Required)
			}
			if c := propertySchema(t, req.Properties["tipo"]).Const; c != string(typ) {
				t.Errorf("tipo const = %q, want %q", c, typ)
			}

			data := schemas[schemaName(t, propertySchema(t, req.Properties["dados"]).Ref)]
			if data.AdditionalProperties == nil || *data.AdditionalProperties {
				t.Error("dados schema must have additionalProperties: false")
			}
			if got := sortedKeys(data.Properties); !slices.Equal(got, sorted(dataKeys)) {
				t.Errorf("dados properties = %v, want the decoder keys %v", got, sorted(dataKeys))
			}
			var required []string
			for _, k := range dataKeys {
				if !slices.Contains(optionalTicketKeys, k) {
					required = append(required, k)
				}
			}
			if !slices.Equal(sorted(data.Required), sorted(required)) {
				t.Errorf("dados required = %v, want %v", data.Required, required)
			}
		})
	}
}

// TestContractTicketEnums compares the contract enums with the domain codes.
func TestContractTicketEnums(t *testing.T) {
	t.Parallel()

	schemas := loadTicketSchemas(t)
	tests := []struct {
		schema string
		codes  []string
		valid  func(string) bool
	}{
		{"TicketType", []string{
			string(domain.TicketTypeAccess), string(domain.TicketTypeBug),
			string(domain.TicketTypeDeploy), string(domain.TicketTypeOther),
		}, func(s string) bool { return domain.TicketType(s).Valid() }},
		{"TicketStatus", []string{
			string(domain.TicketStatusOpen), string(domain.TicketStatusInProgress),
			string(domain.TicketStatusResolved), string(domain.TicketStatusClosed),
		}, func(s string) bool { return domain.TicketStatus(s).Valid() }},
		{"AccessLevel", []string{
			string(domain.AccessLevelRead), string(domain.AccessLevelWrite), string(domain.AccessLevelAdmin),
		}, func(s string) bool { return domain.AccessLevel(s).Valid() }},
		{"Severity", []string{
			string(domain.SeverityLow), string(domain.SeverityMedium),
			string(domain.SeverityHigh), string(domain.SeverityCritical),
		}, func(s string) bool { return domain.Severity(s).Valid() }},
		{"Environment", []string{
			string(domain.EnvironmentDevelopment), string(domain.EnvironmentStaging), string(domain.EnvironmentProduction),
		}, func(s string) bool { return domain.Environment(s).Valid() }},
	}
	for _, tt := range tests {
		t.Run(tt.schema, func(t *testing.T) {
			t.Parallel()
			s, ok := schemas[tt.schema]
			if !ok {
				t.Fatalf("schema %s is missing", tt.schema)
			}
			if !slices.Equal(s.Enum, tt.codes) {
				t.Errorf("enum = %v, want %v", s.Enum, tt.codes)
			}
			for _, code := range s.Enum {
				if !tt.valid(code) {
					t.Errorf("contract code %q is not valid in the domain", code)
				}
			}
		})
	}
}

func TestContractTicketResponseMatchesStruct(t *testing.T) {
	t.Parallel()

	doc := loadContract(t)
	s, ok := doc.Components.Schemas["TicketResponse"]
	if !ok {
		t.Fatal("schema TicketResponse is missing")
	}
	for _, problem := range diffSchema(s, reflect.TypeFor[ticketResponse]()) {
		t.Errorf("TicketResponse vs ticketResponse: %s", problem)
	}
}

// TestDiffParameters checks the parameter comparison on a small inline
// contract (plan 003 section 8: the contract test reads path and query
// parameters for the first time).
func TestDiffParameters(t *testing.T) {
	t.Parallel()

	const components = `"components": {"parameters": {
		"TicketId": {"name": "id", "in": "path", "required": true, "schema": {"type": "integer", "minimum": 1}},
		"OtherId": {"name": "id", "in": "path", "required": true},
		"StatusQuery": {"name": "status", "in": "query"},
		"PaginaQuery": {"name": "pagina", "in": "query"},
		"Csrf": {"name": "X-CSRF-Token", "in": "header", "required": true}
	}}`
	tests := []struct {
		name   string
		path   string
		params string
		query  []string
		want   []string
	}{
		{"no parameters", "/x", ``, nil, nil},
		{"id through TicketId", "/x/{id}", `{"$ref": "#/components/parameters/TicketId"}`, nil, nil},
		{"id without TicketId", "/x/{id}", ``, nil, []string{`path parameter "id" is not declared`}},
		{
			"id inline instead of TicketId", "/x/{id}", `{"name": "id", "in": "path", "required": true}`, nil,
			[]string{`path parameter "id" must be $ref "#/components/parameters/TicketId"`},
		},
		{
			"id through another component", "/x/{id}", `{"$ref": "#/components/parameters/OtherId"}`, nil,
			[]string{`path parameter "id" must be $ref "#/components/parameters/TicketId"`},
		},
		{
			"path parameter not in the path", "/x", `{"$ref": "#/components/parameters/TicketId"}`, nil,
			[]string{`path parameter "id" is not in the path`},
		},
		{
			"optional path parameter", "/x/{slug}", `{"name": "slug", "in": "path"}`, nil,
			[]string{`path parameter "slug" must be required`},
		},
		{
			"query parameters match", "/x", `{"$ref": "#/components/parameters/StatusQuery"}, {"$ref": "#/components/parameters/PaginaQuery"}`,
			[]string{"status", "pagina"},
			nil,
		},
		{
			"missing query parameter", "/x", `{"$ref": "#/components/parameters/StatusQuery"}`,
			[]string{"status", "pagina"},
			[]string{`query parameter "pagina" is not declared`},
		},
		{
			"extra query parameter", "/x", `{"$ref": "#/components/parameters/StatusQuery"}, {"$ref": "#/components/parameters/PaginaQuery"}`,
			[]string{"status"},
			[]string{`query parameter "pagina" is not read by the route`},
		},
		{
			"unresolved $ref", "/x", `{"$ref": "#/components/parameters/Nope"}`, nil,
			[]string{`parameter $ref "#/components/parameters/Nope" does not resolve`},
		},
		{"header parameters are ignored", "/x/{id}", `{"$ref": "#/components/parameters/Csrf"}, {"$ref": "#/components/parameters/TicketId"}`, nil, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			doc := parseContract(t, `{"paths": {"`+tt.path+`": {"get": {"parameters": [`+tt.params+`]}}}, `+components+`}`)
			ops, err := doc.operations()
			if err != nil {
				t.Fatal(err)
			}
			pattern := "GET " + tt.path
			if got := diffParameters(doc, pattern, ops[pattern], tt.query); !slices.Equal(got, tt.want) {
				t.Errorf("diffParameters() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestPathWildcards(t *testing.T) {
	t.Parallel()

	tests := map[string][]string{
		"/api/tickets":      nil,
		"/api/tickets/{id}": {"id"},
		"/api/administradores/tickets/{id}/mensagens": {"id"},
		"/files/{path...}": {"path"},
		"/a/{x}/b/{y}":     {"x", "y"},
	}
	for path, want := range tests {
		if got := pathWildcards(path); !slices.Equal(got, want) {
			t.Errorf("pathWildcards(%q) = %q, want %q", path, got, want)
		}
	}
}
