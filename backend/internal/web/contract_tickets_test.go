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

// TestContractTrackingSchemasMatchStructs compares the answer schemas of
// spec 003 with the structs the handlers write (S-13: the keys per role).
func TestContractTrackingSchemasMatchStructs(t *testing.T) {
	t.Parallel()

	doc := loadContract(t)
	tests := []struct {
		schema string
		typ    reflect.Type
	}{
		{"TicketSummary", reflect.TypeFor[ticketSummary]()},
		{"TicketPage", reflect.TypeFor[ticketPage]()},
		{"TicketDetail", reflect.TypeFor[ticketDetail]()},
		{"ThreadEntry", reflect.TypeFor[threadEntry]()},
		{"ThreadPage", reflect.TypeFor[threadPage]()},
		{"AdminTicketSummary", reflect.TypeFor[adminTicketSummary]()},
		{"StatusCounts", reflect.TypeFor[statusCounts]()},
		{"QueuePage", reflect.TypeFor[queuePage]()},
		{"AuthorList", reflect.TypeFor[authorList]()},
		{"AdminTicketDetail", reflect.TypeFor[adminTicketDetail]()},
		{"AdminThreadEntry", reflect.TypeFor[adminThreadEntry]()},
		{"AdminThreadPage", reflect.TypeFor[adminThreadPage]()},
	}
	for _, tt := range tests {
		t.Run(tt.schema, func(t *testing.T) {
			t.Parallel()
			s, ok := doc.Components.Schemas[tt.schema]
			if !ok {
				t.Fatalf("schema %s is missing from api/openapi.json", tt.schema)
			}
			for _, problem := range diffSchema(s, tt.typ) {
				t.Errorf("%s vs %s: %s", tt.schema, tt.typ.Name(), problem)
			}
		})
	}
}

// TestContractTicketDetailData checks that data of TicketDetail is one
// schema per type, whose properties are the keys of the type
// (security.TicketDataKeys, plan P-10), all required: an empty optional
// field is null, never left out.
func TestContractTicketDetailData(t *testing.T) {
	t.Parallel()

	schemas := loadTicketSchemas(t)
	detail, ok := schemas["TicketDetail"]
	if !ok {
		t.Fatal("schema TicketDetail is missing")
	}
	data := propertySchema(t, detail.Properties["data"])
	// AdminTicketDetail.data is the same oneOf (plan T5.3).
	if admin, ok := schemas["AdminTicketDetail"]; ok {
		if got := propertySchema(t, admin.Properties["data"]).OneOf; !slices.Equal(got, data.OneOf) {
			t.Errorf("AdminTicketDetail.data oneOf = %v, want the one of TicketDetail %v", got, data.OneOf)
		}
	} else {
		t.Error("schema AdminTicketDetail is missing")
	}
	byKeys := map[string]ticketSchema{}
	for _, r := range data.OneOf {
		s := schemas[schemaName(t, r.Ref)]
		byKeys[strings.Join(sortedKeys(s.Properties), ",")] = s
	}
	keys := security.TicketDataKeys()
	if len(data.OneOf) != len(keys) {
		t.Errorf("data has %d oneOf schemas, want one per type (%d)", len(data.OneOf), len(keys))
	}
	for typ, want := range keys {
		s, ok := byKeys[strings.Join(sorted(want), ",")]
		if !ok {
			t.Errorf("%s: no data schema with the keys %v", typ, sorted(want))
			continue
		}
		if !slices.Equal(sorted(s.Required), sorted(want)) {
			t.Errorf("%s: data required = %v, want every key %v", typ, s.Required, sorted(want))
		}
		if s.AdditionalProperties == nil || *s.AdditionalProperties {
			t.Errorf("%s: data schema must have additionalProperties: false", typ)
		}
	}
}

// TestContractThreadEntryKinds checks the thread entry codes: EntryKind is
// the domain's, EntrySide the web's, and the oneOf of ThreadEntry has one
// branch per kind with the keys that kind carries.
func TestContractThreadEntryKinds(t *testing.T) {
	t.Parallel()

	schemas := loadTicketSchemas(t)
	kinds := []string{string(domain.EntryKindMessage), string(domain.EntryKindStatusChange)}
	if got := schemas["EntryKind"].Enum; !slices.Equal(got, kinds) {
		t.Errorf("EntryKind enum = %v, want %v", got, kinds)
	}
	for _, k := range schemas["EntryKind"].Enum {
		if !domain.EntryKind(k).Valid() {
			t.Errorf("EntryKind %q is not valid in the domain", k)
		}
	}
	if got := schemas["EntrySide"].Enum; !slices.Equal(got, []string{sideAuthor, sideSupport}) {
		t.Errorf("EntrySide enum = %v, want [%s %s]", got, sideAuthor, sideSupport)
	}

	var branches []struct {
		Properties struct {
			Kind struct {
				Const string `json:"const"`
			} `json:"kind"`
		} `json:"properties"`
		Required []string `json:"required"`
	}
	data, err := os.ReadFile(contractPath)
	if err != nil {
		t.Fatal(err)
	}
	var doc struct {
		Components struct {
			Schemas map[string]json.RawMessage `json:"schemas"`
		} `json:"components"`
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatal(err)
	}
	var entry struct {
		OneOf json.RawMessage `json:"oneOf"`
	}
	if err := json.Unmarshal(doc.Components.Schemas["ThreadEntry"], &entry); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(entry.OneOf, &branches); err != nil {
		t.Fatalf("ThreadEntry oneOf: %v", err)
	}
	want := map[string][]string{
		string(domain.EntryKindMessage):      {"text"},
		string(domain.EntryKindStatusChange): {"from", "to"},
	}
	if len(branches) != len(want) {
		t.Fatalf("ThreadEntry has %d oneOf branches, want %d", len(branches), len(want))
	}
	for _, b := range branches {
		if keys, ok := want[b.Properties.Kind.Const]; !ok || !slices.Equal(sorted(b.Required), keys) {
			t.Errorf("branch kind %q requires %v, want %v", b.Properties.Kind.Const, b.Required, keys)
		}
	}
}

// TestContractMessageRequest checks CreateMessageRequest against the
// decoder: only texto, required, no other key (S-8), and the limit of
// security.MessageMaxLength.
func TestContractMessageRequest(t *testing.T) {
	t.Parallel()

	schemas := loadTicketSchemas(t)
	s, ok := schemas["CreateMessageRequest"]
	if !ok {
		t.Fatal("schema CreateMessageRequest is missing")
	}
	if s.AdditionalProperties == nil || *s.AdditionalProperties {
		t.Error("CreateMessageRequest must have additionalProperties: false")
	}
	if got := sortedKeys(s.Properties); !slices.Equal(got, []string{"texto"}) || !slices.Equal(s.Required, []string{"texto"}) {
		t.Errorf("properties = %v, required = %v; want only texto, required", got, s.Required)
	}
	var texto struct {
		MaxLength int `json:"maxLength"`
	}
	if err := json.Unmarshal(s.Properties["texto"], &texto); err != nil || texto.MaxLength != security.MessageMaxLength {
		t.Errorf("texto maxLength = %d (%v), want %d", texto.MaxLength, err, security.MessageMaxLength)
	}
}

// TestContractListParameters checks the list parameters: their names are
// the ones the security parsers read, their codes come from the domain
// enums, and the page size is the server's.
func TestContractListParameters(t *testing.T) {
	t.Parallel()

	data, err := os.ReadFile(contractPath)
	if err != nil {
		t.Fatal(err)
	}
	var doc struct {
		Components struct {
			Parameters map[string]struct {
				Name     string       `json:"name"`
				In       string       `json:"in"`
				Required bool         `json:"required"`
				Schema   ticketSchema `json:"schema"`
			} `json:"parameters"`
			Schemas map[string]struct {
				Properties map[string]struct {
					Const    any  `json:"const"`
					MaxItems *int `json:"maxItems"`
				} `json:"properties"`
			} `json:"schemas"`
		} `json:"components"`
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		param, name, ref string
	}{
		{"StatusQuery", "status", "#/components/schemas/TicketStatus"},
		{"TipoQuery", "tipo", "#/components/schemas/TicketType"},
		{"PaginaQuery", "pagina", ""},
	}
	for _, tt := range tests {
		p, ok := doc.Components.Parameters[tt.param]
		if !ok {
			t.Errorf("parameter %s is missing", tt.param)
			continue
		}
		if p.Name != tt.name || p.In != "query" || p.Required {
			t.Errorf("%s = %q in %s (required %v), want the optional query parameter %q", tt.param, p.Name, p.In, p.Required, tt.name)
		}
		if p.Schema.Ref != tt.ref {
			t.Errorf("%s schema $ref = %q, want %q", tt.param, p.Schema.Ref, tt.ref)
		}
	}

	// The thread page bound is the one of service and repository (100,
	// plan P-4); TestOwnThreadPages checks it on the answers.
	if m := doc.Components.Schemas["ThreadPage"].Properties["entries"].MaxItems; m == nil || *m != 100 {
		t.Errorf("ThreadPage.entries maxItems = %v, want 100", m)
	}
	if p := doc.Components.Parameters["AntesQuery"]; p.Name != "antes" || p.In != "query" || p.Required {
		t.Errorf("AntesQuery = %+v, want the optional query parameter antes", p)
	}

	page := doc.Components.Schemas["TicketPage"].Properties
	if c := page["page_size"].Const; c != float64(listPageSize) {
		t.Errorf("TicketPage.page_size const = %v, want %d", c, listPageSize)
	}
	if m := page["items"].MaxItems; m == nil || *m != listPageSize {
		t.Errorf("TicketPage.items maxItems = %v, want %d", m, listPageSize)
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
